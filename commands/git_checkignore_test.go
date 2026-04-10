package commands

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsPathGitignored_failOpenOutsideRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}
	tmp := t.TempDir()
	p := filepath.Join(tmp, "some", "path")
	if IsPathGitignored(p, tmp) {
		t.Fatal("expected fail-open false outside git repo")
	}
}
