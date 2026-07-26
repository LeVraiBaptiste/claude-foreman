// Package statusline parses the JSON Claude Code sends to a status line command
// on stdin and renders a two-line status bar. It reads a documented, stable
// contract (never Claude Code's rendered output), so it does not break when the
// terminal display changes across versions.
package statusline

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/LeVraiBaptiste/claude-foreman/internal/state"
)

// Input mirrors the subset of the status line stdin JSON we consume. Fields that
// may be absent/null are pointers or tolerated as zero values.
type Input struct {
	SessionID   string `json:"session_id"`
	SessionName string `json:"session_name"`
	Cwd         string `json:"cwd"`

	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`

	Workspace struct {
		CurrentDir string `json:"current_dir"`
		ProjectDir string `json:"project_dir"`
	} `json:"workspace"`

	Cost struct {
		TotalCostUSD      float64 `json:"total_cost_usd"`
		TotalDurationMs   float64 `json:"total_duration_ms"`
		TotalLinesAdded   int     `json:"total_lines_added"`
		TotalLinesRemoved int     `json:"total_lines_removed"`
	} `json:"cost"`

	ContextWindow struct {
		UsedPercentage    *float64 `json:"used_percentage"`
		TotalInputTokens  int      `json:"total_input_tokens"`
		ContextWindowSize int      `json:"context_window_size"`
		CurrentUsage      struct {
			InputTokens              int `json:"input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`

	RateLimits struct {
		FiveHour struct {
			UsedPercentage *float64 `json:"used_percentage"`
		} `json:"five_hour"`
		SevenDay struct {
			UsedPercentage *float64 `json:"used_percentage"`
		} `json:"seven_day"`
	} `json:"rate_limits"`

	Effort struct {
		Level string `json:"level"`
	} `json:"effort"`

	Thinking struct {
		Enabled bool `json:"enabled"`
	} `json:"thinking"`

	Worktree struct {
		Branch string `json:"branch"`
	} `json:"worktree"`
}

// Parse decodes the status line JSON from r.
func Parse(r io.Reader) (Input, error) {
	var in Input
	data, err := io.ReadAll(r)
	if err != nil {
		return in, err
	}
	if err := json.Unmarshal(data, &in); err != nil {
		return in, err
	}
	return in, nil
}

// WorkDir returns the best-effort working directory for the session.
func (in Input) WorkDir() string {
	if in.Cwd != "" {
		return in.Cwd
	}
	return in.Workspace.CurrentDir
}

func pct(p *float64) int {
	if p == nil {
		return -1
	}
	return int(*p)
}

// Meta converts the parsed input into the persisted telemetry record.
func (in Input) Meta() state.Meta {
	return state.Meta{
		SessionID:    in.SessionID,
		SessionName:  in.SessionName,
		Cwd:          in.WorkDir(),
		Model:        in.Model.DisplayName,
		ContextPct:   pct(in.ContextWindow.UsedPercentage),
		CostUSD:      in.Cost.TotalCostUSD,
		DurationMs:   int64(in.Cost.TotalDurationMs),
		LinesAdded:   in.Cost.TotalLinesAdded,
		LinesRemoved: in.Cost.TotalLinesRemoved,
		Rate5hPct:    pct(in.RateLimits.FiveHour.UsedPercentage),
		Rate7dPct:    pct(in.RateLimits.SevenDay.UsedPercentage),
	}
}

// ANSI colors (raw codes: the status line stdout is captured, not a TTY).
const (
	cRed    = "\x1b[31m"
	cGreen  = "\x1b[32m"
	cYellow = "\x1b[33m"
	cBlue   = "\x1b[34m"
	cCyan   = "\x1b[36m"
	cDim    = "\x1b[2m"
	cBold   = "\x1b[1m"
	cReset  = "\x1b[0m"
)

func colorByPct(p int) string {
	switch {
	case p >= 80:
		return cRed
	case p >= 50:
		return cYellow
	default:
		return cGreen
	}
}

func progressBar(p int) string {
	const width = 15
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	filled := p * width / 100
	return colorByPct(p) + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + cReset
}

func formatDuration(ms int64) string {
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

func cacheHitPct(in Input) int {
	u := in.ContextWindow.CurrentUsage
	denom := u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
	if denom <= 0 {
		return -1
	}
	return u.CacheReadInputTokens * 100 / denom
}

func dim(s string) string { return cDim + s + cReset }

// Render builds the two-line status bar (the "enriched" layout).
func Render(in Input) string {
	ctx := pct(in.ContextWindow.UsedPercentage)

	// --- Line 1: model · effort · thinking  <bar> pct%  $cost · duration ---
	var l1 strings.Builder
	l1.WriteString(cBold + cCyan + orUnknown(in.Model.DisplayName) + cReset)
	if lvl := in.Effort.Level; lvl != "" && lvl != "medium" {
		l1.WriteString(dim(" · ") + cYellow + "⚡" + lvl + cReset)
	}
	if in.Thinking.Enabled {
		l1.WriteString(dim(" · ") + "🧠")
	}
	l1.WriteString("  " + progressBar(ctx))
	if ctx >= 0 {
		l1.WriteString(fmt.Sprintf(" %d%%", ctx))
	}
	l1.WriteString("  " + cGreen + fmt.Sprintf("$%.2f", in.Cost.TotalCostUSD) + cReset)
	l1.WriteString(dim(" · ") + dim(formatDuration(int64(in.Cost.TotalDurationMs))))

	// --- Line 2: branch counts · cache% · +added/-removed · 5h/7d ---
	var segs []string
	if g := gitInfo(in.WorkDir(), in.Worktree.Branch); g != "" {
		segs = append(segs, cBlue+" "+g+cReset)
	}
	if c := cacheHitPct(in); c >= 0 {
		segs = append(segs, dim("cache ")+fmt.Sprintf("%d%%", c))
	}
	if in.Cost.TotalLinesAdded > 0 || in.Cost.TotalLinesRemoved > 0 {
		segs = append(segs, cGreen+fmt.Sprintf("+%d", in.Cost.TotalLinesAdded)+cReset+
			cRed+fmt.Sprintf("/-%d", in.Cost.TotalLinesRemoved)+cReset)
	}
	if rl := rateLimits(in); rl != "" {
		segs = append(segs, rl)
	}

	out := l1.String()
	if len(segs) > 0 {
		out += "\n" + strings.Join(segs, dim("  ·  "))
	}
	return out
}

func rateLimits(in Input) string {
	var parts []string
	if p := pct(in.RateLimits.FiveHour.UsedPercentage); p >= 0 {
		parts = append(parts, colorByPct(p)+fmt.Sprintf("5h %d%%", p)+cReset)
	}
	if p := pct(in.RateLimits.SevenDay.UsedPercentage); p >= 0 {
		parts = append(parts, colorByPct(p)+fmt.Sprintf("7d %d%%", p)+cReset)
	}
	return strings.Join(parts, " ")
}

func orUnknown(s string) string {
	if s == "" {
		return "Unknown"
	}
	return s
}
