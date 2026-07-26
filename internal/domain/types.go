package domain

// ClaudeStatus represents the current state of a Claude Code session.
type ClaudeStatus int

const (
	ClaudeStatusNone    ClaudeStatus = iota // no claude in this pane
	ClaudeStatusIdle                        // claude running but idle
	ClaudeStatusBusy                        // claude executing a tool
	ClaudeStatusWaiting                     // claude waiting for user permission
	ClaudeStatusDone                        // claude session finished
)

func (s ClaudeStatus) String() string {
	switch s {
	case ClaudeStatusIdle:
		return "idle"
	case ClaudeStatusBusy:
		return "busy"
	case ClaudeStatusWaiting:
		return "waiting"
	case ClaudeStatusDone:
		return "done"
	default:
		return ""
	}
}

type Process struct {
	PID     int
	Command string
}

// ClaudeSource indicates how a ClaudeSession's data was obtained.
type ClaudeSource int

const (
	ClaudeSourceScrape ClaudeSource = iota // regex fallback over captured pane content
	ClaudeSourceState                      // status line + hooks state files (robust)
)

type ClaudeSession struct {
	Status     ClaudeStatus
	Source     ClaudeSource
	ContextPct int    // context window usage percentage (0-100), -1 if unknown
	Elapsed    string // elapsed time string e.g. "2m00s", empty if unknown

	// Rich metadata, populated from the status line state file (empty/zero if unknown).
	Model        string  // model display name, e.g. "Opus 4.8"
	CostUSD      float64 // session cost in USD
	LinesAdded   int
	LinesRemoved int
	Rate5hPct    int // 5-hour rate limit usage %, -1 if unknown
	Rate7dPct    int // 7-day rate limit usage %, -1 if unknown
}

type Pane struct {
	Index          int
	PaneID         string // tmux #{pane_id}, e.g. "%3"
	PID            int
	Active         bool
	CurrentCommand string
	Processes      []Process
	Claude         *ClaudeSession
}

type Window struct {
	Index  int
	Name   string
	Active bool
	Panes  []Pane
}

type Session struct {
	Name     string
	Attached bool
	Active   bool
	Windows  []Window
}

type AppState struct {
	Sessions     []Session
	ActiveTarget string // "session:window:pane"
}
