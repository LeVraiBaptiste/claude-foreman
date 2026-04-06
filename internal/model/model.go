package model

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/LeVraiBaptiste/claude-foreman/internal/domain"
	"github.com/LeVraiBaptiste/claude-foreman/internal/polling"
)

// ItemKind distinguishes navigable items in the flattened list.
type ItemKind int

const (
	ItemSession ItemKind = iota
	ItemWindow
)

// NavItem is a navigable element in the tree view.
type NavItem struct {
	Kind        ItemKind
	SessionIdx  int
	WindowIdx   int
	SessionName string
	WindowIndex int
}

type Model struct {
	State     domain.AppState
	Items     []NavItem
	Cursor    int
	Poller    *polling.Poller
	Width     int
	Height    int
	Spinner spinner.Model
}

func New(poller *polling.Poller) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	return Model{
		Poller:  poller,
		Spinner: s,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		pollCmd(m.Poller),
		tea.EnableMouseCellMotion,
		m.Spinner.Tick,
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time

func pollCmd(p *polling.Poller) tea.Cmd {
	return func() tea.Msg {
		state := p.Poll()
		return TmuxStateMsg{State: state}
	}
}

// flattenItems builds a flat list of navigable items from the app state.
func flattenItems(state domain.AppState) []NavItem {
	var items []NavItem
	for si, sess := range state.Sessions {
		items = append(items, NavItem{
			Kind:        ItemSession,
			SessionIdx:  si,
			SessionName: sess.Name,
		})
		for wi, win := range sess.Windows {
			items = append(items, NavItem{
				Kind:        ItemWindow,
				SessionIdx:  si,
				WindowIdx:   wi,
				SessionName: sess.Name,
				WindowIndex: win.Index,
			})
		}
	}
	return items
}
