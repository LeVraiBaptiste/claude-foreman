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
