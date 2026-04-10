package querycontext

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goc/commands"
	"goc/types"
)

func TestParseAdditionalWorkingDirsJSON_Array(t *testing.T) {
	raw, _ := json.Marshal([]string{" /a ", "b"})
	got := ParseAdditionalWorkingDirsJSON(raw)
	if len(got) != 2 || got[0] != "/a" || got[1] != "b" {
		t.Fatalf("got %#v", got)
	}
}

func TestParseAdditionalWorkingDirsJSON_ObjectKeys(t *testing.T) {
	raw := json.RawMessage(`{"/proj/extra":{},"/other":{}}`)
	got := ParseAdditionalWorkingDirsJSON(raw)
	if len(got) != 2 || got[0] != "/other" || got[1] != "/proj/extra" {
		t.Fatalf("got %#v", got)
	}
}

func TestExtraClaudeMdRootsForFetch_RuntimeAndEnv(t *testing.T) {
	dir := t.TempDir()
	extra := filepath.Join(dir, "extra")
	if err := os.MkdirAll(extra, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal([]string{extra})
	rc := &types.ProcessUserInputContextData{
		ToolPermissionContext: &types.ToolPermissionContextData{
			Mode:                         types.PermissionDefault,
			AdditionalWorkingDirectories: raw,
		},
	}
	t.Setenv("GOU_DEMO_EXTRA_CLAUDE_MD_ROOTS", extra)
	got := ExtraClaudeMdRootsForFetch(rc)
	if len(got) != 1 {
		t.Fatalf("expected deduped single abs path, got %#v", got)
	}
}

func TestFetchSystemPromptParts_ExtraClaudeMdRoots(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD", "1")
	t.Setenv("CLAUDE_CODE_OVERRIDE_DATE", "2026-04-08")
	primary := t.TempDir()
	extra := t.TempDir()
	const marker = "CLAUDE_MD_EXTRA_ROOT_UNIQUE_MARKER"
	if err := os.WriteFile(filepath.Join(extra, "CLAUDE.md"), []byte("# X\n"+marker), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	res, err := FetchSystemPromptParts(ctx, FetchOpts{
		Gou: commands.GouDemoSystemOpts{
			Cwd:              primary,
			ModelID:          "m",
			EnabledToolNames: map[string]struct{}{"Bash": {}},
		},
		ExtraClaudeMdRoots: []string{extra},
	})
	if err != nil {
		t.Fatal(err)
	}
	cm, ok := res.UserContext["claudeMd"]
	if !ok {
		t.Fatalf("missing claudeMd: %#v", res.UserContext)
	}
	if !strings.Contains(cm, marker) {
		t.Fatalf("claudeMd should include extra root file; got %q", cm)
	}
}
