package statusline

import (
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gitCacheMaxAge = 10 * time.Second

// gitInfo returns a compact git summary ("branch +s ~m ?u") for dir, cached
// briefly on disk to keep the status line fast on large repos. branchHint (from
// the status line JSON) is used when present to skip a git call. Returns "" when
// dir is not a git work tree or git is unavailable.
func gitInfo(dir, branchHint string) string {
	if dir == "" {
		return ""
	}

	cache := filepath.Join(os.TempDir(), fmt.Sprintf("claude-foreman-git-%d", hashStr(dir)))
	if fi, err := os.Stat(cache); err == nil && time.Since(fi.ModTime()) < gitCacheMaxAge {
		if data, err := os.ReadFile(cache); err == nil {
			return string(data)
		}
	}

	info := computeGitInfo(dir, branchHint)
	_ = os.WriteFile(cache, []byte(info), 0o644) // cache empties too, to avoid re-probing non-repos
	return info
}

func computeGitInfo(dir, branchHint string) string {
	if strings.TrimSpace(runGit(dir, "rev-parse", "--is-inside-work-tree")) != "true" {
		return ""
	}

	branch := branchHint
	if branch == "" {
		branch = strings.TrimSpace(runGit(dir, "branch", "--show-current"))
	}
	if branch == "" {
		branch = "detached"
	}

	var staged, modified, untracked int
	for _, line := range strings.Split(runGit(dir, "status", "--porcelain=v1"), "\n") {
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]
		switch {
		case x == '?' && y == '?':
			untracked++
		default:
			if x != ' ' {
				staged++
			}
			if y == 'M' || y == 'D' {
				modified++
			}
		}
	}

	out := branch
	if staged > 0 {
		out += fmt.Sprintf(" +%d", staged)
	}
	if modified > 0 {
		out += fmt.Sprintf(" ~%d", modified)
	}
	if untracked > 0 {
		out += fmt.Sprintf(" ?%d", untracked)
	}
	return out
}

func runGit(dir string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func hashStr(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}
