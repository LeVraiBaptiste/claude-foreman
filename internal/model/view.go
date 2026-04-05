package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/LeVraiBaptiste/claude-foreman/internal/domain"
	"github.com/LeVraiBaptiste/claude-foreman/internal/style"
)

func (m Model) View() string {
	if m.Width == 0 {
		return "  Loading..."
	}

	contentWidth := m.Width - 4 // margin
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Header: title left, summary right
	title := style.TitleStyle.Render("claude-foreman")
	summary := style.SummaryStyle.Render(m.summaryText())
	titleW := lipgloss.Width(title)
	summaryW := lipgloss.Width(summary)
	gap := contentWidth - titleW - summaryW
	if gap < 2 {
		gap = 2
	}
	header := "  " + title + strings.Repeat(" ", gap) + summary

	if len(m.State.Sessions) == 0 {
		content := header + "\n\n  Waiting for tmux data..."
		return lipgloss.Place(m.Width, m.Height, lipgloss.Left, lipgloss.Top, content)
	}

	var sections []string
	sections = append(sections, header)
	sections = append(sections, "")

	// Session cards
	navIdx := 0
	for _, sess := range m.State.Sessions {
		card := m.renderSessionCard(sess, contentWidth, navIdx)
		sections = append(sections, card)
		navIdx += 1 + len(sess.Windows)
	}

	content := strings.Join(sections, "\n")
	return lipgloss.Place(m.Width, m.Height, lipgloss.Left, lipgloss.Top, content)
}

func (m Model) renderSessionCard(sess domain.Session, width int, navStartIdx int) string {
	borderColor := lipgloss.Color("241")
	if sess.Attached {
		borderColor = lipgloss.Color("15")
	}
	bs := lipgloss.NewStyle().Foreground(borderColor)

	innerWidth := width - 4 // 2 border chars + 2 padding spaces
	if innerWidth < 10 {
		innerWidth = 10
	}

	// Top border: ╭─ session-name ─────────╮
	titleText := " " + sess.Name + " "
	titleW := lipgloss.Width(titleText)
	fillCount := innerWidth + 2 - 1 - titleW // +2 for padding, -1 for first ─
	if fillCount < 1 {
		fillCount = 1
	}
	topLine := bs.Render("╭─") + titleText + bs.Render(strings.Repeat("─", fillCount)+"╮")

	// Content lines
	var cardLines []string
	cardLines = append(cardLines, topLine)

	for wi, win := range sess.Windows {
		winNavIdx := navStartIdx + 1 + wi
		winContent := m.renderWindowLine(win, winNavIdx)
		cardLines = append(cardLines, borderLine(bs, winContent, innerWidth))

		// Claude instances under this window
		for _, pane := range win.Panes {
			if pane.Claude != nil {
				claudeContent := m.renderClaudeContent(pane.Claude, innerWidth)
				cardLines = append(cardLines, borderLine(bs, claudeContent, innerWidth))
			}
		}
	}

	// Bottom border: ╰─────────────╯
	bottomLine := bs.Render("╰" + strings.Repeat("─", innerWidth+2) + "╯")
	cardLines = append(cardLines, bottomLine)

	return strings.Join(cardLines, "\n")
}

// borderLine wraps content in side borders: │ content     │
func borderLine(bs lipgloss.Style, content string, innerWidth int) string {
	contentW := lipgloss.Width(content)
	pad := innerWidth - contentW
	if pad < 0 {
		pad = 0
	}
	return bs.Render("│") + " " + content + strings.Repeat(" ", pad) + " " + bs.Render("│")
}

func (m Model) renderWindowLine(win domain.Window, navIdx int) string {
	name := fmt.Sprintf("%d:%s", win.Index, win.Name)
	selected := navIdx == m.Cursor

	if selected {
		if win.Active {
			return style.CursorStyle.Render("▸ " + style.WindowActiveStyle.Render(name))
		}
		return style.CursorStyle.Render("▸ " + style.WindowStyle.Render(name))
	}
	if win.Active {
		return "  " + style.WindowActiveStyle.Render(name)
	}
	return "  " + style.WindowStyle.Render(name)
}

func (m Model) renderClaudeContent(claude *domain.ClaudeSession, innerWidth int) string {
	indicator := claudeStatusIndicator(claude.Status, m.Spinner.View())

	dur := ""
	if claude.StartTime > 0 {
		elapsed := m.Now.Sub(time.Unix(claude.StartTime, 0))
		dur = style.DurationStyle.Render(formatDuration(elapsed))
	}

	indicatorW := lipgloss.Width(indicator)
	durW := lipgloss.Width(dur)
	gap := innerWidth - 4 - indicatorW - durW // 4 = indent
	if gap < 1 {
		gap = 1
	}

	return "    " + indicator + strings.Repeat(" ", gap) + dur
}

func claudeStatusIndicator(status domain.ClaudeStatus, spinnerFrame string) string {
	switch status {
	case domain.ClaudeStatusBusy:
		return spinnerFrame + " " + style.ClaudeBusy.String()
	case domain.ClaudeStatusWaiting:
		dot := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("●")
		return dot + " " + style.ClaudeWaiting.String()
	case domain.ClaudeStatusIdle:
		dot := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("●")
		return dot + " " + style.ClaudeIdle.String()
	case domain.ClaudeStatusDone:
		dot := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("●")
		return dot + " " + style.ClaudeDone.String()
	default:
		return ""
	}
}

func (m Model) summaryText() string {
	var busy, waiting, idle, done int
	for _, s := range m.State.Sessions {
		for _, w := range s.Windows {
			for _, p := range w.Panes {
				if p.Claude != nil {
					switch p.Claude.Status {
					case domain.ClaudeStatusBusy:
						busy++
					case domain.ClaudeStatusWaiting:
						waiting++
					case domain.ClaudeStatusIdle:
						idle++
					case domain.ClaudeStatusDone:
						done++
					}
				}
			}
		}
	}
	total := busy + waiting + idle + done
	if total == 0 {
		return fmt.Sprintf("%d sessions", len(m.State.Sessions))
	}
	parts := []string{fmt.Sprintf("%d instances", total)}
	if busy > 0 {
		parts = append(parts, fmt.Sprintf("%d busy", busy))
	}
	if waiting > 0 {
		parts = append(parts, fmt.Sprintf("%d waiting", waiting))
	}
	if idle > 0 {
		parts = append(parts, fmt.Sprintf("%d idle", idle))
	}
	if done > 0 {
		parts = append(parts, fmt.Sprintf("%d done", done))
	}
	return strings.Join(parts, " · ")
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	mins := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if mins >= 60 {
		h := mins / 60
		mins = mins % 60
		return fmt.Sprintf("%dh %02dm", h, mins)
	}
	return fmt.Sprintf("%dm %02ds", mins, s)
}
