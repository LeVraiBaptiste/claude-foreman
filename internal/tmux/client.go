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
	PID            string
	CurrentCommand string
	Active         string
}

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
	return runTmux("list-sessions", "-F", "#{session_name}:#{session_attached}")
}

func (c *RealClient) ListWindows() ([]string, error) {
	return runTmux("list-windows", "-a", "-F", "#{session_name}:#{window_index}:#{window_name}:#{window_active}")
}

func (c *RealClient) ListPanes() ([]string, error) {
	return runTmux("list-panes", "-a", "-F", "#{session_name}:#{window_index}:#{pane_index}:#{pane_pid}:#{pane_current_command}:#{pane_active}")
}

func (c *RealClient) ActiveTarget() (string, error) {
	lines, err := runTmux("display-message", "-p", "#S:#I:#P")
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
