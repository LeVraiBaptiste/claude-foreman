package claude

import (
	"regexp"
	"strings"

	"github.com/LeVraiBaptiste/claude-foreman/internal/domain"
)

// Analyzer determines Claude status by analyzing captured tmux pane content.
type Analyzer struct{}

var (
	waitingPattern = regexp.MustCompile(`(?i)\(y\s*=\s*yes|Allow|Approve|Do you want|yes.*to proceed`)
	busyPattern    = regexp.MustCompile(`[✶✢✻✦✳✽∴⊛★☆※·⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏*].*\w+ing\b`)
)

// Analyze examines the last ~10 non-empty lines of pane content and returns a ClaudeStatus.
func (a *Analyzer) Analyze(paneContent string) domain.ClaudeStatus {
	lines := strings.Split(strings.TrimRight(paneContent, "\n"), "\n")

	var tail []string
	for i := len(lines) - 1; i >= 0 && len(tail) < 10; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			tail = append(tail, lines[i])
		}
	}

	text := strings.Join(tail, "\n")

	if waitingPattern.MatchString(text) {
		return domain.ClaudeStatusWaiting
	}

	if busyPattern.MatchString(text) {
		return domain.ClaudeStatusBusy
	}

	return domain.ClaudeStatusIdle
}
