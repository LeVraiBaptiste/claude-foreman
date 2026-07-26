// Package state persists per-pane Claude Code telemetry derived from Claude
// Code's own status line, keyed by tmux pane id ($TMUX_PANE == #{pane_id}).
//
// The status line command writes one file per pane (<key>.meta.json). The TUI
// reads it to display metadata and to derive busy/idle from its freshness.
// Every write is atomic (temp file + rename).
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Meta is the telemetry written by the status line command from the JSON it
// receives on stdin. Numeric "unknown" values use -1 where a zero would be a
// valid reading. UpdatedAt doubles as the activity heartbeat: the TUI treats a
// recently-written file as "busy".
type Meta struct {
	SessionID    string  `json:"session_id"`
	SessionName  string  `json:"session_name"`
	Cwd          string  `json:"cwd"`
	Model        string  `json:"model"`
	ContextPct   int     `json:"context_pct"` // -1 if unknown
	CostUSD      float64 `json:"cost_usd"`
	DurationMs   int64   `json:"duration_ms"`
	LinesAdded   int     `json:"lines_added"`
	LinesRemoved int     `json:"lines_removed"`
	Rate5hPct    int     `json:"rate_5h_pct"` // -1 if unknown
	Rate7dPct    int     `json:"rate_7d_pct"` // -1 if unknown
	UpdatedAt    int64   `json:"updated_at"`  // unix seconds
}

// Dir returns the state directory (~/.claude/foreman/state), creating it.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".claude", "foreman", "state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// fileKey turns a tmux pane id (e.g. "%3") into a filesystem-safe key. The same
// transform is applied to $TMUX_PANE (writer) and #{pane_id} (reader), so keys
// always match.
func fileKey(paneID string) string {
	var b strings.Builder
	for _, r := range paneID {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// WriteMeta atomically persists m for the given pane id, stamping UpdatedAt.
func WriteMeta(paneID string, m Meta) error {
	m.UpdatedAt = time.Now().Unix()
	return writeJSON(fileKey(paneID)+".meta.json", m)
}

func writeJSON(name string, v any) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+name+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filepath.Join(dir, name))
}

// Read returns the meta for a pane id. A missing file is not an error: ok is
// false.
func Read(paneID string) (m Meta, ok bool) {
	dir, err := Dir()
	if err != nil {
		return Meta{}, false
	}
	data, err := os.ReadFile(filepath.Join(dir, fileKey(paneID)+".meta.json"))
	if err != nil {
		return Meta{}, false
	}
	if json.Unmarshal(data, &m) != nil {
		return Meta{}, false
	}
	return m, true
}
