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

type ClaudeSession struct {
	Status    ClaudeStatus
	StartTime int64 // Unix timestamp (seconds), 0 if unknown
}

type Pane struct {
	Index          int
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
