package main

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LeVraiBaptiste/claude-foreman/internal/claude"
	"github.com/LeVraiBaptiste/claude-foreman/internal/model"
	"github.com/LeVraiBaptiste/claude-foreman/internal/polling"
	"github.com/LeVraiBaptiste/claude-foreman/internal/process"
	"github.com/LeVraiBaptiste/claude-foreman/internal/state"
	"github.com/LeVraiBaptiste/claude-foreman/internal/statusline"
	"github.com/LeVraiBaptiste/claude-foreman/internal/tmux"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runTUI()
		return
	}

	switch args[0] {
	case "statusline":
		runStatusline()
	case "-h", "--help", "help":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", args[0])
		usage(os.Stderr)
		os.Exit(2)
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `claude-foreman — monitor tmux sessions and Claude Code status

Usage:
  claude-foreman              Launch the TUI
  claude-foreman statusline   Status line command (reads Claude Code JSON on stdin)
`)
}

func runTUI() {
	poller := &polling.Poller{
		Tmux:     &tmux.RealClient{},
		Process:  &process.ProcInspector{},
		Analyzer: &claude.Analyzer{},
	}

	m := model.New(poller)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runStatusline reads the status line JSON on stdin, persists the telemetry
// keyed by tmux pane, and prints the rendered bar. It must never fail loudly:
// a broken status line would degrade the user's Claude Code display.
func runStatusline() {
	in, err := statusline.Parse(os.Stdin)
	if err != nil {
		return
	}
	if pane := os.Getenv("TMUX_PANE"); pane != "" {
		_ = state.WriteMeta(pane, in.Meta())
	}
	fmt.Println(statusline.Render(in))
}
