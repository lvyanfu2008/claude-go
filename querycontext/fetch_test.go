package querycontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goc/commands"
	"goc/tscontext"
)

// UserContext must match TS getUserContext() (live only); snapshot UserContext keys are ignored.
func TestFetchSystemPromptParts_tsSnapshotUserContextLiveOnlyLikeTS(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Title\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CODE_SYSTEM_PROMPT_MODEL_ID", "test-model")
	t.Cleanup(func() { _ = os.Unsetenv("CLAUDE_CODE_SYSTEM_PROMPT_MODEL_ID") })

	snap := &tscontext.Snapshot{
		DefaultSystemPrompt: []string{"frozen-from-ts"},
		UserContext: map[string]string{
			"currentDate": `Today's date is 1900-01-01.`,
		},
		SystemContext: map[string]string{},
	}
	res, err := FetchSystemPromptParts(context.Background(), FetchOpts{
		TSSnapshot: snap,
		Gou:        commands.GouDemoSystemOpts{Cwd: dir, ModelID: "test-model"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.UserContext["claudeMd"], "CLAUDE.md") {
		t.Fatalf("expected live claudeMd in user context, got keys=%v", keysOf(res.UserContext))
	}
	if strings.Contains(res.UserContext["currentDate"], "1900") {
		t.Fatalf("expected live currentDate to overwrite snapshot, got %q", res.UserContext["currentDate"])
	}
	if len(res.DefaultSystemPrompt) != 1 || !strings.Contains(res.DefaultSystemPrompt[0], "test-model") {
		t.Fatalf("expected rebuilt Go default system, got %#v", res.DefaultSystemPrompt)
	}
}

// TS: customSystemPrompt skips default system + systemContext, but getUserContext() still runs (live only).
func TestFetchSystemPromptParts_tsSnapshotCustomUserContextLiveOnlyLikeTS(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# SnapTitle\nfrom-disk"), 0o644); err != nil {
		t.Fatal(err)
	}
	snap := &tscontext.Snapshot{
		DefaultSystemPrompt: []string{"ignored-under-custom"},
		UserContext: map[string]string{
			"currentDate": `Today's date is 1900-01-01.`,
		},
		SystemContext: map[string]string{"git": "stale"},
	}
	res, err := FetchSystemPromptParts(context.Background(), FetchOpts{
		CustomSystemPrompt: "custom system only",
		TSSnapshot:         snap,
		Gou:                commands.GouDemoSystemOpts{Cwd: dir},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.DefaultSystemPrompt) != 0 {
		t.Fatalf("expected empty default system under custom, got %#v", res.DefaultSystemPrompt)
	}
	if len(res.SystemContext) != 0 {
		t.Fatalf("expected empty system context under custom, got %#v", res.SystemContext)
	}
	if !strings.Contains(res.UserContext["claudeMd"], "from-disk") {
		t.Fatalf("expected live claudeMd merged under custom+tssnapshot, got keys=%v", keysOf(res.UserContext))
	}
	if strings.Contains(res.UserContext["currentDate"], "1900") {
		t.Fatalf("expected live currentDate to overwrite snapshot, got %q", res.UserContext["currentDate"])
	}
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
