// Git check-ignore helper (TS isPathGitignored).
package commands

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsPathGitignored mirrors src/utils/git/gitignore.ts isPathGitignored:
// runs `git check-ignore <path>` with working directory gitCwd.
// Exit 0 → true; 1 → false; 128 / other → false (fail open, same as TS).
func IsPathGitignored(path, gitCwd string) bool {
	path = strings.TrimSpace(path)
	gitCwd = strings.TrimSpace(gitCwd)
	if path == "" || gitCwd == "" {
		return false
	}
	path = filepath.Clean(path)
	gitCwd = filepath.Clean(gitCwd)
	cmd := exec.Command("git", "check-ignore", "-q", path)
	cmd.Dir = gitCwd
	err := cmd.Run()
	if err == nil {
		return true
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return false
	}
	return false
}
