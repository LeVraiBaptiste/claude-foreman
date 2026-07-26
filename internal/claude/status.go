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
	// Active spinner: gerund verb followed by "..." (done messages use past tense, no ellipsis).
	busyPattern = regexp.MustCompile(`[✶✢✻✦✳✽∴⊛★☆※·⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏*].*\w+ing\b\.\.\.`)
	// Matches Claude Code status bar: "... | ░░░█████ 42% | $1.23 | 5m30s"
	statusBarPattern = regexp.MustCompile(`(\d+)%\s*\|\s*(\$[\d.]+)\s*\|\s*([\dm]+[\d.]*s)`)
)

// AnalyzeResult contains both the status and metadata extracted from pane content.
type AnalyzeResult struct {
	Status     domain.ClaudeStatus
	ContextPct int // -1 if not found
	Elapsed    string
}

// IsWaiting reports whether the pane content shows a permission prompt. Used on
// the status-line path, where busy/idle come from other signals but "waiting"
// (a distinct, blocking state) is only visible in the rendered prompt.
func (a *Analyzer) IsWaiting(paneContent string) bool {
	return waitingPattern.MatchString(tailText(paneContent, 50))
}

// tailText returns the last n non-empty lines of s joined by newlines.
func tailText(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	var tail []string
	for i := len(lines) - 1; i >= 0 && len(tail) < n; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			tail = append(tail, lines[i])
		}
	}
	return strings.Join(tail, "\n")
}

// Analyze examines the last ~50 non-empty lines of pane content and returns status + metadata.
// The window must be large enough to cover the busy spinner and status bar even when a long
// tool output or todo list is rendered between them.
func (a *Analyzer) Analyze(paneContent string) AnalyzeResult {
	text := tailText(paneContent, 50)

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
