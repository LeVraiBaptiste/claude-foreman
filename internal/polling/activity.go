package polling

import (
	"hash/fnv"
	"strings"
	"time"

	"github.com/LeVraiBaptiste/claude-foreman/internal/tmux"
)

// activityWindow is how long a pane stays "busy" on the strength of a content
// change alone. Polls run every 500ms and Claude's spinner (glyph + elapsed
// counter) moves several times a second while it works, so any genuine busy
// spell produces a change every poll; the window only smooths the odd frame
// that happens to hash-collide. Kept short so idle is reported promptly.
const activityWindow = 2 * time.Second

// paneActivity is the last observed transcript hash for a pane and when it last
// changed. A zero changedAt means "seen once, never observed changing" — not
// busy.
type paneActivity struct {
	hash      uint64
	changedAt time.Time
}

// sawRecentChange reports whether the pane's transcript region (everything above
// the input box: the scrollback plus the spinner line) changed within
// activityWindow. This is the primary busy signal: the spinner animates for the
// whole time Claude works — including long tool runs, thinking and streaming,
// the exact gaps the status-line freshness and the /proc shell scan miss.
//
// The input box and the status bar are excluded from the hash, so neither the
// user typing at an idle prompt nor a ticking status bar reads as activity.
func (p *Poller) sawRecentChange(paneID, content string, now time.Time) bool {
	h := activityHash(content)
	if p.activity == nil {
		p.activity = make(map[string]paneActivity)
	}
	prev, ok := p.activity[paneID]
	switch {
	case !ok:
		// First sighting: record the hash but claim nothing — we have no
		// previous frame to compare against, so we can't tell busy from idle.
		p.activity[paneID] = paneActivity{hash: h}
		return false
	case prev.hash != h:
		// The transcript moved since the last poll: active right now.
		p.activity[paneID] = paneActivity{hash: h, changedAt: now}
		return true
	default:
		// Unchanged this poll: still busy only if the last change is recent.
		return !prev.changedAt.IsZero() && now.Sub(prev.changedAt) < activityWindow
	}
}

// pruneActivity drops per-pane state for panes that no longer exist so the map
// can't grow without bound.
func (p *Poller) pruneActivity(panes []tmux.RawPane) {
	if p.activity == nil {
		return
	}
	live := make(map[string]bool, len(panes))
	for _, rp := range panes {
		live[rp.PaneID] = true
	}
	for id := range p.activity {
		if !live[id] {
			delete(p.activity, id)
		}
	}
}

// activityHash hashes the transcript region — the pane content above the input
// box. Anchoring on the box border rather than matching the spinner's glyphs or
// the status-bar format keeps this robust to Claude Code display changes: box
// borders are among the most stable elements of the UI, and we never inspect
// what the changing text actually is.
func activityHash(content string) uint64 {
	lines := strings.Split(content, "\n")
	if cut := inputBoxTop(lines); cut >= 0 {
		lines = lines[:cut]
	}
	h := fnv.New64a()
	for _, l := range lines {
		h.Write([]byte(l))
		h.Write([]byte{'\n'})
	}
	return h.Sum64()
}

// inputBoxTop returns the index of the top border of Claude's input box, or -1
// if no box is found (fall back to hashing the whole capture). It finds the
// bottom-most rule line (the box's bottom border) then the highest rule line
// within a few lines above it (the top border), so everything from the box down
// — box body and the status bar below it — is excluded.
func inputBoxTop(lines []string) int {
	bottom := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if isRuleLine(lines[i]) {
			bottom = i
			break
		}
	}
	if bottom < 0 {
		return -1
	}
	top := bottom
	for i := bottom - 1; i >= 0 && bottom-i <= 6; i-- {
		if isRuleLine(lines[i]) {
			top = i
		}
	}
	return top
}

// isRuleLine reports whether a line is a horizontal rule / box border: mostly
// box-drawing glyphs (U+2500–U+257F) or ASCII dashes. The threshold on the
// glyph count keeps the input's own vertical borders ("│ > text │", only two
// box runes) from matching.
func isRuleLine(s string) bool {
	s = strings.TrimSpace(s)
	box, nonSpace := 0, 0
	for _, r := range s {
		if r == ' ' {
			continue
		}
		nonSpace++
		if (r >= 0x2500 && r <= 0x257F) || r == '-' {
			box++
		}
	}
	return box >= 3 && box*100 >= nonSpace*60
}
