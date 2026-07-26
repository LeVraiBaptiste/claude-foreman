# claude-foreman

TUI that monitors your tmux sessions and shows Claude Code status in real time.

Displays the tree of tmux sessions, windows, and panes. Detects Claude Code instances via process inspection and scrapes pane content to determine their status (busy, waiting for permission, idle). Navigate and switch tmux windows with keyboard or mouse.

## Status detection

The only integration point is Claude Code's **status line**. No hooks.

`claude-foreman statusline` is registered as the status line command. Claude Code pipes a
structured JSON (model, context %, cost, duration, lines added/removed, rate limits, …) to
it on stdin — a documented, stable contract, not rendered output. Foreman renders the bar
*and* records that telemetry to `~/.claude/foreman/state/<pane>.meta.json`, keyed by tmux
pane id (`$TMUX_PANE`, inherited by the status line command).

The TUI derives status from three signals, none of which parse the status bar's text:

- **busy** — the status line was invoked recently (fresh `meta.json`, i.e. Claude is
  emitting messages) **or** the pane has an active child process (Claude is running a
  tool — covers long builds/tests the status line can't see).
- **waiting** — a permission prompt is detected by a narrow regex over the pane. This is
  the one state the status line cannot expose.
- **idle** — otherwise.

Context %, cost, model and rate limits come straight from the JSON.

**Fallback.** For any Claude pane without a `meta.json` (status line not installed, or a
Claude running elsewhere), Foreman falls back to full `tmux capture-pane` + regex. No
integration is *required* to run the TUI — it just makes detection robust.

See [Claude Code integration](#claude-code-integration) to wire it up.

## Install

### Nix (flake)

```
nix run github:LeVraiBaptiste/claude-foreman
```

Or install permanently:

```
nix profile install github:LeVraiBaptiste/claude-foreman
```

### NixOS / Home Manager

Add the input to your flake:

```nix
inputs.claude-foreman.url = "github:LeVraiBaptiste/claude-foreman";
```

Then either use the package directly:

```nix
environment.systemPackages = [
  inputs.claude-foreman.packages.${system}.default
];
```

Or use the overlay:

```nix
nixpkgs.overlays = [ inputs.claude-foreman.overlays.default ];
environment.systemPackages = [ pkgs.claude-foreman ];
```

### From source

```
go build -o claude-foreman ./cmd
```

Requires `tmux` in PATH.

## Claude Code integration

Point Claude Code's status line at Foreman — that's the whole integration.

### Nix flake (recommended)

The flake exposes a `lib.statusLine` helper so your config stays a one-liner and Foreman
owns the behaviour:

```nix
# in the module that builds ~/.claude/settings.json
{ pkgs, inputs, ... }:
{
  home.file.".claude/settings.json".text = builtins.toJSON {
    statusLine = inputs.claude-foreman.lib.statusLine pkgs;
    # ...your other Claude Code settings...
  };
}
```

There's also a Home Manager module (`homeManagerModules.default`) that installs the binary
and manages the settings file, for standalone Home Manager setups.

### Manual

Add to `~/.claude/settings.json` (adjust the path to the `claude-foreman` binary):

```json
{
  "statusLine": { "type": "command", "command": "claude-foreman statusline", "padding": 2 }
}
```

## Usage

```
claude-foreman
```

Run it in any terminal — it doesn't need to be inside tmux.

### Keys

| Key | Action |
|---|---|
| `j` / `k` or arrows | Navigate |
| `Enter` | Switch to selected window |
| Click | Select and switch |
| `q` / `Ctrl+C` | Quit |

## How it works

Every 500ms, claude-foreman:

1. Queries tmux for all sessions, windows, and panes (incl. `#{pane_id}`)
2. Inspects `/proc` to find child processes of each pane
3. Identifies panes running Claude Code (by process name)
4. For each Claude pane, reads its `~/.claude/foreman/state/<pane>.meta.json`
   (written by the status line) and derives status from freshness + process activity
5. If no `meta.json` exists, falls back to `tmux capture-pane` + regex

## Requirements

- Linux (uses `/proc` for process inspection)
- tmux
