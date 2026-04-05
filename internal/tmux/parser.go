package tmux

import (
	"strconv"
	"strings"
)

func ParseSessions(lines []string) []RawSession {
	result := make([]RawSession, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		result = append(result, RawSession{
			Name:     parts[0],
			Attached: parts[1],
		})
	}
	return result
}

func ParseWindows(lines []string) []RawWindow {
	result := make([]RawWindow, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 4)
		if len(parts) != 4 {
			continue
		}
		result = append(result, RawWindow{
			SessionName: parts[0],
			Index:       parts[1],
			Name:        parts[2],
			Active:      parts[3],
		})
	}
	return result
}

func ParsePanes(lines []string) []RawPane {
	result := make([]RawPane, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 6)
		if len(parts) != 6 {
			continue
		}
		result = append(result, RawPane{
			SessionName:    parts[0],
			WindowIndex:    parts[1],
			Index:          parts[2],
			PID:            parts[3],
			CurrentCommand: parts[4],
			Active:         parts[5],
		})
	}
	return result
}

func ParseActiveTarget(raw string) (session string, window int, pane int) {
	parts := strings.SplitN(raw, ":", 3)
	if len(parts) != 3 {
		return "", 0, 0
	}
	session = parts[0]
	window, _ = strconv.Atoi(parts[1])
	pane, _ = strconv.Atoi(parts[2])
	return
}
