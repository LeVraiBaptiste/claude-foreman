package tmux

import (
	"os/exec"
	"strings"
)

type RawSession struct {
	Name     string
	Attached string
}

type RawWindow struct {
	SessionName string
	Index       string
	Name        string
	Active      string
}

type RawPane struct {
	SessionName    string
	WindowIndex    string
	Index          string
	PaneID         string // tmux #{pane_id}, e.g. "%3" — matches $TMUX_PANE inside the pane
	PID            string
	CurrentCommand string
	Active         string
}

// sep is the field delimiter used in tmux -F format strings. We use the ASCII
// unit separator (0x1f) instead of ':' because session/window names and pane
// commands can legitimately contain ':', which would corrupt a ':'-based split.
const sep = "\x1f"

type Client interface {
	ListSessions() ([]string, error)
	ListWindows() ([]string, error)
	ListPanes() ([]string, error)
	ActiveTarget() (string, error)
	SwitchClient(target string) error
	CapturePane(target string) (string, error)
}

type RealClient struct{}

func (c *RealClient) ListSessions() ([]string, error) {
	return runTmux("list-sessions", "-F", join("#{session_name}", "#{session_attached}"))
}

func (c *RealClient) ListWindows() ([]string, error) {
	return runTmux("list-windows", "-a", "-F", join("#{session_name}", "#{window_index}", "#{window_name}", "#{window_active}"))
}

func (c *RealClient) ListPanes() ([]string, error) {
	return runTmux("list-panes", "-a", "-F", join("#{session_name}", "#{window_index}", "#{pane_index}", "#{pane_id}", "#{pane_pid}", "#{pane_current_command}", "#{pane_active}"))
}

func (c *RealClient) ActiveTarget() (string, error) {
	lines, err := runTmux("display-message", "-p", join("#S", "#I", "#P"))
	if err != nil {
		return "", err
	}
	if len(lines) > 0 {
		return lines[0], nil
	}
	return "", nil
}

func (c *RealClient) SwitchClient(target string) error {
	cmd := exec.Command("tmux", "switch-client", "-t", target)
	return cmd.Run()
}

func (c *RealClient) CapturePane(target string) (string, error) {
	cmd := exec.Command("tmux", "capture-pane", "-t", target, "-p", "-J")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func join(fields ...string) string {
	return strings.Join(fields, sep)
}

func runTmux(args ...string) ([]string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}
