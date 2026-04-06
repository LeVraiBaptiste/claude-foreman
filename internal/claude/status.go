package claude

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/LeVraiBaptiste/claude-foreman/internal/domain"
)

// Analyzer determines Claude status by analyzing captured tmux pane content.
type Analyzer struct{}

var (
	waitingPattern = regexp.MustCompile(`(?i)\(y\s*=\s*yes|Allow|Approve|Do you want|yes.*to proceed`)
	busyPattern    = regexp.MustCompile(`[✶✢✻✦✳✽∴⊛★☆※·⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏*].*\w+ing\b`)
	// Matches Claude Code status bar: "... | ░░░█████ 42% | $1.23 | 5m30s"
	statusBarPattern = regexp.MustCompile(`(\d+)%\s*\|\s*(\$[\d.]+)\s*\|\s*([\dm]+[\d.]*s)`)
)

// AnalyzeResult contains both the status and metadata extracted from pane content.
type AnalyzeResult struct {
	Status     domain.ClaudeStatus
	ContextPct int    // -1 if not found
	Elapsed    string
}

// Analyze examines the last ~10 non-empty lines of pane content and returns status + metadata.
func (a *Analyzer) Analyze(paneContent string) AnalyzeResult {
	lines := strings.Split(strings.TrimRight(paneContent, "\n"), "\n")

	var tail []string
	for i := len(lines) - 1; i >= 0 && len(tail) < 10; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			tail = append(tail, lines[i])
		}
	}

	text := strings.Join(tail, "\n")

	status := domain.ClaudeStatusIdle
	if waitingPattern.MatchString(text) {
		status = domain.ClaudeStatusWaiting
	} else if busyPattern.MatchString(text) {
		status = domain.ClaudeStatusBusy
	}

	result := AnalyzeResult{Status: status, ContextPct: -1}

	// Extract context %, cost, and elapsed from status bar
	if m := statusBarPattern.FindStringSubmatch(text); m != nil {
		if pct, err := strconv.Atoi(m[1]); err == nil {
			result.ContextPct = pct
		}
		result.Elapsed = m[3]
	}

	return result
}
