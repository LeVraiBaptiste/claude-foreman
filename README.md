# claude-foreman

TUI that monitors your tmux sessions and shows Claude Code status in real time.

Displays the tree of tmux sessions, windows, and panes. Detects Claude Code instances via process inspection and scrapes pane content to determine their status (busy, waiting for permission, idle). Navigate and switch tmux windows with keyboard or mouse.

## Status detection

Claude Code status is determined by scraping the visible content of tmux panes.

- **busy** — Claude is executing a tool (spinner + action detected)
- **waiting** — Claude is waiting for user permission
- **idle** — Claude is running but not doing anything

Context usage percentage and elapsed time are extracted from Claude Code's status bar, which looks like:

```
Opus 4.6 (1M context) | ░░░░░░░░░░░░░░░ 3% | $0.26 | 2m00s
```

This means claude-foreman requires a version of Claude Code that displays this status bar format. If the status bar is absent or has a different format, context and time info will simply not be shown.

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

1. Queries tmux for all sessions, windows, and panes
2. Inspects `/proc` to find child processes of each pane
3. Identifies panes running Claude Code (by process name)
4. Captures the visible content of Claude panes via `tmux capture-pane`
5. Analyzes the last lines with regex to determine status

## Requirements

- Linux (uses `/proc` for process inspection)
- tmux
