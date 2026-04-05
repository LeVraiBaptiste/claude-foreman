package model

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case TmuxStateMsg:
		m.State = msg.State
		m.Now = time.Now()
		m.Items = flattenItems(m.State)
		if m.Cursor >= len(m.Items) && len(m.Items) > 0 {
			m.Cursor = len(m.Items) - 1
		}
		return m, tickCmd()

	case tickMsg:
		return m, pollCmd(m.Poller)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		m.Now = time.Now()
		return m, cmd

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
			return m, nil
		case "down", "j":
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
			}
			return m, nil
		case "enter":
			return m, m.switchToSelected()
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if idx := m.lineToNavItem(msg.Y); idx >= 0 {
				m.Cursor = idx
				if m.Items[idx].Kind == ItemWindow {
					return m, m.switchToSelected()
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// lineToNavItem maps a screen Y coordinate to a NavItem index.
// Returns -1 if the line is not clickable.
func (m Model) lineToNavItem(y int) int {
	// Layout: line 0 = header, line 1 = blank
	line := 2
	navIdx := 0
	for _, sess := range m.State.Sessions {
		// Top border — click selects session
		if y == line && navIdx < len(m.Items) {
			return navIdx
		}
		line++
		for wi, win := range sess.Windows {
			itemIdx := navIdx + 1 + wi
			// Window line
			if y == line && itemIdx < len(m.Items) {
				return itemIdx
			}
			line++
			// Claude instance lines (not clickable as nav items)
			for _, pane := range win.Panes {
				if pane.Claude != nil {
					line++
				}
			}
		}
		// Bottom border
		line++
		// Blank line between cards
		line++
		navIdx += 1 + len(sess.Windows)
	}
	return -1
}

func (m Model) switchToSelected() tea.Cmd {
	if m.Cursor >= len(m.Items) {
		return nil
	}
	item := m.Items[m.Cursor]
	var target string
	switch item.Kind {
	case ItemSession:
		target = item.SessionName
	case ItemWindow:
		target = fmt.Sprintf("%s:%d", item.SessionName, item.WindowIndex)
	}
	return func() tea.Msg {
		m.Poller.Tmux.SwitchClient(target)
		return nil
	}
}
