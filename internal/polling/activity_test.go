package polling

import (
	"strings"
	"testing"
	"time"

	"github.com/LeVraiBaptiste/claude-foreman/internal/tmux"
)

// pane renders a Claude Code capture matching the real layout (verified against
// live captures): transcript, the spinner line (above the input), the input
// framed by two full-width rules (the top one carries an embedded session
// label), then foreman's two-line status bar with its ticking cost/duration
// clock. transcript and spinner vary; input and status vary independently to
// model typing at the prompt and a ticking clock.
func pane(transcript, spinner, input, status string) string {
	return strings.Join([]string{
		transcript,
		spinner,
		"──────────────────────────────────── my-session ──",
		"❯ " + input,
		"───────────────────────────────────────────────────",
		"  Opus 4.8 · 🧠 █░░░░░░░ 7%  " + status,
		"  main ~1 ?2 · +286/-1 · 5h 6% 7d 7%",
	}, "\n")
}

func TestIsRuleLine(t *testing.T) {
	rules := []string{
		"╭────────────────────────────────────────╮",
		"╰────────────────────────────────────────╯",
		"──────────────",
		"------------",
	}
	for _, s := range rules {
		if !isRuleLine(s) {
			t.Errorf("expected rule line: %q", s)
		}
	}
	notRules := []string{
		"│ > hello world",              // input line: only 1 box glyph
		" 42% | $1.23 | claude-opus",   // status bar
		"✶ Herding… (esc to interrupt)", // spinner
		"",
		"── x",
	}
	for _, s := range notRules {
		if isRuleLine(s) {
			t.Errorf("did not expect rule line: %q", s)
		}
	}
}

func TestActivityHashExcludesInputAndStatus(t *testing.T) {
	base := pane("wrote foo.go", "✶ Herding…", "hel", " 42% | $1.20 | 1m00s")

	// Typing in the input must not change the hash.
	typing := pane("wrote foo.go", "✶ Herding…", "hello", " 42% | $1.20 | 1m00s")
	if activityHash(base) != activityHash(typing) {
		t.Error("typing in the input box changed the transcript hash")
	}

	// A ticking status bar must not change the hash.
	ticked := pane("wrote foo.go", "✶ Herding…", "hel", " 42% | $1.99 | 2m30s")
	if activityHash(base) != activityHash(ticked) {
		t.Error("status-bar change altered the transcript hash")
	}

	// The spinner moving must change the hash.
	spun := pane("wrote foo.go", "✻ Herding…", "hel", " 42% | $1.20 | 1m00s")
	if activityHash(base) == activityHash(spun) {
		t.Error("spinner movement did not change the transcript hash")
	}

	// New transcript output must change the hash.
	out := pane("wrote foo.go\nwrote bar.go", "✶ Herding…", "hel", " 42% | $1.20 | 1m00s")
	if activityHash(base) == activityHash(out) {
		t.Error("new transcript output did not change the transcript hash")
	}
}

func TestSawRecentChange(t *testing.T) {
	p := &Poller{}
	t0 := time.Unix(1_700_000_000, 0)
	idle := pane("done", "", "", " 42% | $1.20 | 1m00s")

	// First sighting: no history, not busy.
	if p.sawRecentChange("%1", idle, t0) {
		t.Fatal("first sighting should not be busy")
	}
	// Same idle content next poll: still not busy.
	if p.sawRecentChange("%1", idle, t0.Add(500*time.Millisecond)) {
		t.Fatal("static idle pane should not be busy")
	}
	// Typing at the idle prompt (transcript unchanged): still not busy.
	typed := pane("done", "", "typing here", " 42% | $1.20 | 1m00s")
	if p.sawRecentChange("%1", typed, t0.Add(time.Second)) {
		t.Fatal("typing at idle prompt should not read as busy")
	}
}

func TestSawRecentChangeBusyAndHysteresis(t *testing.T) {
	p := &Poller{}
	t0 := time.Unix(1_700_000_000, 0)

	spin1 := pane("working", "✶ Herding…", "", " 42% | $1.20 | 1m00s")
	spin2 := pane("working", "✻ Herding…", "", " 42% | $1.20 | 1m01s")

	// Seed, then a spinner move → busy.
	p.sawRecentChange("%2", spin1, t0)
	if !p.sawRecentChange("%2", spin2, t0.Add(500*time.Millisecond)) {
		t.Fatal("spinner movement should be busy")
	}
	// Same frame 1s later (within the 2s window): hysteresis keeps it busy.
	if !p.sawRecentChange("%2", spin2, t0.Add(1500*time.Millisecond)) {
		t.Fatal("hysteresis should hold busy within the window")
	}
	// Still the same frame past the window: now idle.
	if p.sawRecentChange("%2", spin2, t0.Add(3*time.Second)) {
		t.Fatal("stale content past the window should be idle")
	}
}

func TestPruneActivity(t *testing.T) {
	p := &Poller{}
	t0 := time.Unix(1_700_000_000, 0)
	p.sawRecentChange("%1", "a", t0)
	p.sawRecentChange("%2", "b", t0)

	p.pruneActivity([]tmux.RawPane{{PaneID: "%1"}})

	if _, ok := p.activity["%1"]; !ok {
		t.Error("live pane %1 was pruned")
	}
	if _, ok := p.activity["%2"]; ok {
		t.Error("dead pane %2 was not pruned")
	}
}
