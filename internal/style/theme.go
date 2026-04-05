package style

import "github.com/charmbracelet/lipgloss"

var (
	SessionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	SessionActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("33"))

	WindowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	WindowActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true)

	CursorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("237"))

	ClaudeBusy = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			SetString("busy")

	ClaudeWaiting = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			SetString("waiting")

	ClaudeIdle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			SetString("idle")

	ClaudeDone = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			SetString("done")

	ClaudeDot = lipgloss.NewStyle().
			SetString("●")

	DurationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	FooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("33"))

	SummaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// Session card borders
	SessionCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("241")).
				Padding(0, 1)

	SessionCardAttachedStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("15")).
					Padding(0, 1)
)
