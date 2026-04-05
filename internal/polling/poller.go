package polling

import (
	"strconv"
	"strings"

	"github.com/LeVraiBaptiste/claude-foreman/internal/claude"
	"github.com/LeVraiBaptiste/claude-foreman/internal/domain"
	"github.com/LeVraiBaptiste/claude-foreman/internal/process"
	"github.com/LeVraiBaptiste/claude-foreman/internal/tmux"
)

type Poller struct {
	Tmux     tmux.Client
	Process  process.Inspector
	Analyzer *claude.Analyzer
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
					target := rp.SessionName + ":" + rp.WindowIndex + "." + rp.Index
					content, err := p.Tmux.CapturePane(target)
					status := domain.ClaudeStatusIdle
					if err == nil {
						status = p.Analyzer.Analyze(content)
					}
					startTime := claudeStartTime(rp.CurrentCommand, panePID, children)
					pane.Claude = &domain.ClaudeSession{Status: status, StartTime: startTime}
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

// claudeStartTime finds the PID of the Claude process and reads its start time.
func claudeStartTime(currentCommand string, panePID int, children []process.Process) int64 {
	if strings.Contains(strings.ToLower(currentCommand), "claude") {
		t, _ := process.ReadStartTime(panePID)
		return t
	}
	for _, c := range children {
		if strings.Contains(strings.ToLower(c.Command), "claude") {
			t, _ := process.ReadStartTime(c.PID)
			return t
		}
	}
	return 0
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
