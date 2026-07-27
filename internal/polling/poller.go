package polling

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LeVraiBaptiste/claude-foreman/internal/claude"
	"github.com/LeVraiBaptiste/claude-foreman/internal/domain"
	"github.com/LeVraiBaptiste/claude-foreman/internal/process"
	"github.com/LeVraiBaptiste/claude-foreman/internal/state"
	"github.com/LeVraiBaptiste/claude-foreman/internal/tmux"
)

type Poller struct {
	Tmux     tmux.Client
	Process  process.Inspector
	Analyzer *claude.Analyzer

	// activity tracks the last transcript hash per pane id, for the
	// content-diff busy signal. Written only from the (serialized) poll loop.
	activity map[string]paneActivity
}

func (p *Poller) Poll() domain.AppState {
	sessionLines, err := p.Tmux.ListSessions()
	if err != nil {
		return domain.AppState{}
	}
	windowLines, _ := p.Tmux.ListWindows()
	paneLines, _ := p.Tmux.ListPanes()
	activeTarget, _ := p.Tmux.ActiveTarget()

	rawSessions := tmux.ParseSessions(sessionLines)
	rawWindows := tmux.ParseWindows(windowLines)
	rawPanes := tmux.ParsePanes(paneLines)
	activeSess, activeWin, activePane := tmux.ParseActiveTarget(activeTarget)

	p.pruneActivity(rawPanes)

	return p.assemble(rawSessions, rawWindows, rawPanes, activeSess, activeWin, activePane)
}

func (p *Poller) assemble(
	rawSessions []tmux.RawSession,
	rawWindows []tmux.RawWindow,
	rawPanes []tmux.RawPane,
	activeSess string, activeWin int, activePane int,
) domain.AppState {
	windowsBySession := make(map[string][]tmux.RawWindow)
	for _, w := range rawWindows {
		windowsBySession[w.SessionName] = append(windowsBySession[w.SessionName], w)
	}

	panesByKey := make(map[string][]tmux.RawPane)
	for _, rp := range rawPanes {
		key := rp.SessionName + ":" + rp.WindowIndex
		panesByKey[key] = append(panesByKey[key], rp)
	}

	var sessions []domain.Session
	for _, rs := range rawSessions {
		sess := domain.Session{
			Name:     rs.Name,
			Attached: rs.Attached != "0",
			Active:   rs.Name == activeSess,
		}

		for _, rw := range windowsBySession[rs.Name] {
			winIdx, _ := strconv.Atoi(rw.Index)
			win := domain.Window{
				Index:  winIdx,
				Name:   rw.Name,
				Active: rw.Active == "1" && rs.Name == activeSess,
			}

			key := rs.Name + ":" + rw.Index
			for _, rp := range panesByKey[key] {
				paneIdx, _ := strconv.Atoi(rp.Index)
				panePID, _ := strconv.Atoi(rp.PID)

				pane := domain.Pane{
					Index:          paneIdx,
					PaneID:         rp.PaneID,
					PID:            panePID,
					Active:         rp.Active == "1" && win.Active && paneIdx == activePane,
					CurrentCommand: rp.CurrentCommand,
				}

				children, _ := p.Process.Children(panePID)
				for _, c := range children {
					pane.Processes = append(pane.Processes, domain.Process{
						PID:     c.PID,
						Command: c.Command,
					})
				}

				if isClaudePane(rp.CurrentCommand, children) {
					pane.Claude = p.claudeSession(rp, children)
				}

				win.Panes = append(win.Panes, pane)
			}

			sess.Windows = append(sess.Windows, win)
		}

		sessions = append(sessions, sess)
	}

	return domain.AppState{
		Sessions:     sessions,
		ActiveTarget: activeSess + ":" + strconv.Itoa(activeWin) + ":" + strconv.Itoa(activePane),
	}
}

// busyWindow is how long after the last status line invocation we still
// consider the session "busy" on freshness alone (covers gaps between the
// assistant messages that trigger the status line).
const busyWindow = 10 * time.Second

// toolShellComm lists shell process names. Claude Code runs every Bash tool via
// a shell (`bash -c "…"`), so a shell descendant of the Claude process means a
// tool is executing — the signal that keeps "busy" accurate during long tool
// runs the status line can't see. Matching shells (rather than "any non-runtime
// process") avoids false positives from persistent MCP servers (node/python).
// Verified against a real session: idle tree is just the Claude process; during
// a Bash tool it gains "bash" + the command.
var toolShellComm = map[string]bool{
	"bash": true,
	"sh":   true,
	"zsh":  true,
	"dash": true,
	"ksh":  true,
	"fish": true,
}

// claudeSession builds a ClaudeSession for a pane. When the status line has
// written telemetry for this pane (keyed by pane id), status is derived from
// that + process activity + a narrow permission-prompt regex; otherwise it
// falls back to full regex scraping of the pane content.
func (p *Poller) claudeSession(rp tmux.RawPane, children []process.Process) *domain.ClaudeSession {
	target := rp.SessionName + ":" + rp.WindowIndex + "." + rp.Index

	if m, ok := state.Read(rp.PaneID); ok {
		cs := &domain.ClaudeSession{
			Source:       domain.ClaudeSourceState,
			ContextPct:   m.ContextPct,
			Model:        m.Model,
			CostUSD:      m.CostUSD,
			LinesAdded:   m.LinesAdded,
			LinesRemoved: m.LinesRemoved,
			Rate5hPct:    m.Rate5hPct,
			Rate7dPct:    m.Rate7dPct,
			Elapsed:      formatElapsed(m.DurationMs),
		}
		// waiting > busy > idle. "waiting" (a permission prompt) is the only
		// state the status line can't reveal, so we check the rendered pane.
		content, _ := p.Tmux.CapturePane(target)
		// A moving transcript is the primary busy signal (see sawRecentChange);
		// status-line freshness and a shell child are cheap backstops that can
		// only extend "busy", never flip it off, so they can't re-introduce the
		// flicker they used to cause on their own.
		busy := p.sawRecentChange(rp.PaneID, content, time.Now()) ||
			fresh(m.UpdatedAt) || hasActiveToolChild(children)
		switch {
		case p.Analyzer.IsWaiting(content):
			cs.Status = domain.ClaudeStatusWaiting
		case busy:
			cs.Status = domain.ClaudeStatusBusy
		default:
			cs.Status = domain.ClaudeStatusIdle
		}
		return cs
	}

	// Fallback: full regex over captured pane content (no integration here).
	result := claude.AnalyzeResult{Status: domain.ClaudeStatusIdle, ContextPct: -1}
	if content, err := p.Tmux.CapturePane(target); err == nil {
		result = p.Analyzer.Analyze(content)
	}
	return &domain.ClaudeSession{
		Source:     domain.ClaudeSourceScrape,
		Status:     result.Status,
		ContextPct: result.ContextPct,
		Elapsed:    result.Elapsed,
		Rate5hPct:  -1,
		Rate7dPct:  -1,
	}
}

// fresh reports whether a unix-seconds timestamp is within busyWindow of now.
func fresh(updatedAt int64) bool {
	return updatedAt > 0 && time.Since(time.Unix(updatedAt, 0)) < busyWindow
}

// hasActiveToolChild reports whether Claude has a shell descendant, i.e. it is
// currently running a Bash tool.
func hasActiveToolChild(children []process.Process) bool {
	for _, c := range children {
		if toolShellComm[strings.ToLower(c.Command)] {
			return true
		}
	}
	return false
}

// formatElapsed renders a duration in ms as e.g. "2m00s" / "1h05m" / "45s".
func formatElapsed(ms int64) string {
	if ms <= 0 {
		return ""
	}
	total := ms / 1000
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

func isClaudePane(currentCommand string, children []process.Process) bool {
	if strings.Contains(strings.ToLower(currentCommand), "claude") {
		return true
	}
	for _, c := range children {
		if strings.Contains(strings.ToLower(c.Command), "claude") {
			return true
		}
	}
	return false
}
