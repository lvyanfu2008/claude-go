package debugpath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveLogPath_relativeLogsDirUnderConfigHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	t.Setenv("CLAUDE_CODE_DEBUG_LOG_FILE", "")
	t.Setenv("CLAUDE_CODE_DEBUG_LOGS_DIR", "debug/goc")
	p := ResolveLogPath()
	if !strings.HasSuffix(p, ".txt") {
		t.Fatalf("want .txt suffix: %q", p)
	}
	wantPrefix := filepath.Join(root, "debug", "goc") + string(os.PathSeparator)
	if !strings.HasPrefix(p, wantPrefix) {
		t.Fatalf("want prefix %q, got %q", wantPrefix, p)
	}
}

func TestLatestLinkPathFor(t *testing.T) {
	if got := LatestLinkPathFor(""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
	dir := t.TempDir()
	logf := filepath.Join(dir, "sub", "sess.txt")
	want := filepath.Join(dir, "sub", "latest")
	if got := LatestLinkPathFor(logf); got != want {
		t.Fatalf("LatestLinkPathFor: got %q want %q", got, want)
	}
}

func TestMaybeUpdateLatestSymlink_createsLatest(t *testing.T) {
	dir := t.TempDir()
	logf := filepath.Join(dir, "abc123.txt")
	if err := os.WriteFile(logf, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	MaybeUpdateLatestSymlink(logf)
	p := filepath.Join(dir, "latest")
	fi, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("latest: want symlink")
	}
	got, err := os.Readlink(p)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(logf)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("latest target: got %q want %q", got, want)
	}
}
