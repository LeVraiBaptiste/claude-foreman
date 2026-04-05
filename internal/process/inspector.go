package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Inspector interface {
	Children(pid int) ([]Process, error)
}

type Process struct {
	PID     int
	Command string
}

type ProcInspector struct{}

func (p *ProcInspector) Children(pid int) ([]Process, error) {
	var result []Process
	err := p.walk(pid, &result)
	return result, err
}

func (p *ProcInspector) walk(pid int, result *[]Process) error {
	children, err := childPIDs(pid)
	if err != nil {
		return nil // silently ignore unreadable procs
	}
	for _, cpid := range children {
		cmd := readComm(cpid)
		if cmd != "" {
			*result = append(*result, Process{PID: cpid, Command: cmd})
		}
		p.walk(cpid, result)
	}
	return nil
}

func childPIDs(pid int) ([]int, error) {
	taskDir := fmt.Sprintf("/proc/%d/task", pid)
	tids, err := os.ReadDir(taskDir)
	if err != nil {
		return nil, err
	}
	seen := make(map[int]bool)
	var children []int
	for _, tid := range tids {
		childFile := filepath.Join(taskDir, tid.Name(), "children")
		data, err := os.ReadFile(childFile)
		if err != nil {
			continue
		}
		for _, field := range strings.Fields(string(data)) {
			cpid, err := strconv.Atoi(field)
			if err != nil {
				continue
			}
			if !seen[cpid] {
				seen[cpid] = true
				children = append(children, cpid)
			}
		}
	}
	return children, nil
}

func readComm(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// readBootTime reads the system boot time from /proc/stat.
func readBootTime() (int64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return strconv.ParseInt(fields[1], 10, 64)
			}
		}
	}
	return 0, fmt.Errorf("btime not found")
}

// ReadStartTime returns the start time of a process as a Unix timestamp (seconds).
func ReadStartTime(pid int) (int64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, err
	}
	// /proc/pid/stat format: pid (comm) state ... field22=starttime
	// comm can contain spaces/parens, so find the last ')' first
	s := string(data)
	idx := strings.LastIndex(s, ")")
	if idx < 0 {
		return 0, fmt.Errorf("malformed /proc/%d/stat", pid)
	}
	fields := strings.Fields(s[idx+2:]) // skip ") "
	// field 0 after ')' = state, field 19 = starttime (0-indexed from after comm)
	if len(fields) < 20 {
		return 0, fmt.Errorf("not enough fields in /proc/%d/stat", pid)
	}
	startTicks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil {
		return 0, err
	}
	btime, err := readBootTime()
	if err != nil {
		return 0, err
	}
	// CLK_TCK is 100 on Linux
	return btime + startTicks/100, nil
}
