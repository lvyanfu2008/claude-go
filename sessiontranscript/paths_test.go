package sessiontranscript

import (
	"path/filepath"
	"testing"

	"goc/claudemd"
)

func TestProjectDirForOriginalCwd_sanitize(t *testing.T) {
	home := t.TempDir()
	got := ProjectDirForOriginalCwd("/foo/bar", home)
	want := filepath.Join(home, "projects", claudemd.SanitizePath("/foo/bar"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestTranscriptPath_sessionProjectOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "proj")
	home := "/tmp/cfg"
	id := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	got := TranscriptPath(id, "/cwd", override, home)
	want := filepath.Join(override, id+".jsonl")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
