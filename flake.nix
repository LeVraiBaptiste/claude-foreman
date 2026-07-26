{
  description = "claude-foreman — TUI to monitor tmux sessions and Claude Code status";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs supportedSystems (system: f {
        pkgs = nixpkgs.legacyPackages.${system};
        inherit system;
      });
    in
    {
      packages = forAllSystems ({ pkgs, ... }: {
        default = pkgs.buildGoModule {
          pname = "claude-foreman";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-hRNEhso0bRTig9rncCMjxX10KtULnei7dC3fsyQayr0=";
          subPackages = [ "cmd" ];
          nativeBuildInputs = [ pkgs.makeWrapper ];
          postInstall = ''
            mv $out/bin/cmd $out/bin/claude-foreman
            wrapProgram $out/bin/claude-foreman \
              --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.tmux ]}
          '';
          meta = with pkgs.lib; {
            description = "TUI to monitor tmux sessions and Claude Code status";
            license = licenses.mit;
            mainProgram = "claude-foreman";
          };
        };
      });

      apps = forAllSystems ({ system, ... }: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/claude-foreman";
        };
      });

      overlays.default = final: prev: {
        claude-foreman = self.packages.${prev.system}.default;
      };

      # Settings helper: returns the Claude Code `statusLine` block pointing at
      # the Foreman binary. Use it so your Claude Code config is a one-liner and
      # Foreman owns the behaviour:
      #   statusLine = inputs.claude-foreman.lib.statusLine pkgs;
      lib.statusLine = pkgs: {
        type = "command";
        command = "${self.packages.${pkgs.system}.default}/bin/claude-foreman statusline";
        padding = 2;
      };

      # Home Manager module: installs the binary and points Claude Code's status
      # line at it, so Foreman derives status from a stable JSON contract instead
      # of scraping rendered output.
      homeManagerModules.default = { config, lib, pkgs, ... }:
        let
          cfg = config.programs.claude-foreman;
          bin = "${cfg.package}/bin/claude-foreman";
          integration = {
            statusLine = { type = "command"; command = "${bin} statusline"; padding = 2; };
          };
        in
        {
          options.programs.claude-foreman = {
            enable = lib.mkEnableOption "claude-foreman TUI and Claude Code integration";
            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.system}.default;
              description = "The claude-foreman package to use.";
            };
            manageClaudeSettings = lib.mkOption {
              type = lib.types.bool;
              default = true;
              description = ''
                Write ~/.claude/settings.json with the statusLine + hooks wiring.
                When enabled, this module owns that file: fold any other Claude Code
                settings you keep into `extraClaudeSettings`.
              '';
            };
            extraClaudeSettings = lib.mkOption {
              type = lib.types.attrs;
              default = { };
              description = "Extra Claude Code settings merged under the Foreman integration.";
            };
          };

          config = lib.mkIf cfg.enable (lib.mkMerge [
            { home.packages = [ cfg.package ]; }
            (lib.mkIf cfg.manageClaudeSettings {
              home.file.".claude/settings.json".text =
                builtins.toJSON (lib.recursiveUpdate cfg.extraClaudeSettings integration);
            })
          ]);
        };

      devShells = forAllSystems ({ pkgs, ... }: {
        default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            tmux
            jq
          ];
        };
      });
    };
}
