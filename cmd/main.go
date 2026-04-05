package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LeVraiBaptiste/claude-foreman/internal/claude"
	"github.com/LeVraiBaptiste/claude-foreman/internal/model"
	"github.com/LeVraiBaptiste/claude-foreman/internal/polling"
	"github.com/LeVraiBaptiste/claude-foreman/internal/process"
	"github.com/LeVraiBaptiste/claude-foreman/internal/tmux"
)

func main() {
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
