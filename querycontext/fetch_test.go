package querycontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goc/commands"
	"goc/tscontext"
	"goc/types"
)

func TestFetchSystemPromptParts_CustomSkipsDefaultAndSystemCtx(t *testing.T) {
	t.Setenv("CLAUDE_CODE_OVERRIDE_DATE", "2030-06-15")
	ctx := context.Background()
	res, err := FetchSystemPromptParts(ctx, FetchOpts{
		CustomSystemPrompt: "only custom",
		Gou: commands.GouDemoSystemOpts{
			Cwd: t.TempDir(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.DefaultSystemPrompt) != 0 {
		t.Fatalf("default parts: %#v", res.DefaultSystemPrompt)
	}
	if len(res.SystemContext) != 0 {
		t.Fatalf("system ctx: %#v", res.SystemContext)
	}
	if res.UserContext["currentDate"] != "Today's date is 2030-06-15." {
		t.Fatalf("user ctx date: %q", res.UserContext["currentDate"])
	}
}

func TestFetchSystemPromptParts_TSSnapshot_skipsGoBuild(t *testing.T) {
	ctx := context.Background()
	snap := &tscontext.Snapshot{
		DefaultSystemPrompt: []string{"from-bridge"},
		UserContext:         map[string]string{"k": "v"},
		SystemContext:       map[string]string{"git": "clean"},
	}
	res, err := FetchSystemPromptParts(ctx, FetchOpts{
		Gou: commands.GouDemoSystemOpts{
			Cwd: t.TempDir(),
		},
		TSSnapshot: snap,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.DefaultSystemPrompt) != 1 || res.DefaultSystemPrompt[0] != "from-bridge" {
		t.Fatalf("default: %#v", res.DefaultSystemPrompt)
	}
	if res.UserContext["k"] != "v" || res.SystemContext["git"] != "clean" {
		t.Fatalf("ctx user=%#v sys=%#v", res.UserContext, res.SystemContext)
	}
}

func TestFetchSystemPromptParts_TSSnapshot_customClearsDefaultAndSystem(t *testing.T) {
	ctx := context.Background()
	snap := &tscontext.Snapshot{
		DefaultSystemPrompt: []string{"ignored-when-custom"},
		UserContext:         map[string]string{"stale": "ok"},
		SystemContext:       map[string]string{"ignored": "yes"},
	}
	res, err := FetchSystemPromptParts(ctx, FetchOpts{
		CustomSystemPrompt: "custom only",
		Gou: commands.GouDemoSystemOpts{
			Cwd: t.TempDir(),
		},
		TSSnapshot: snap,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.DefaultSystemPrompt) != 0 {
		t.Fatalf("default: %#v", res.DefaultSystemPrompt)
	}
	if len(res.SystemContext) != 0 {
		t.Fatalf("system: %#v", res.SystemContext)
	}
	if res.UserContext["stale"] != "ok" {
		t.Fatalf("user: %#v", res.UserContext)
	}
}

func TestFetchSystemPromptParts_DefaultBuildsPrompt(t *testing.T) {
	t.Setenv("CLAUDE_CODE_OVERRIDE_DATE", "2026-01-01")
	dir := t.TempDir()
	ctx := context.Background()
	res, err := FetchSystemPromptParts(ctx, FetchOpts{
		Gou: commands.GouDemoSystemOpts{
			Cwd:              dir,
			ModelID:          "test-model",
			Language:         "",
			EnabledToolNames: map[string]struct{}{"Bash": {}},
			SkillToolCommands: []types.Command{
				{Type: "prompt", CommandBase: types.CommandBase{Name: "skill-a", Description: "d"}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.DefaultSystemPrompt) != 1 || !strings.Contains(res.DefaultSystemPrompt[0], "interactive agent") {
		t.Fatalf("unexpected default prompt: %q", res.DefaultSystemPrompt[0])
	}
	if res.UserContext["currentDate"] == "" {
		t.Fatal("missing currentDate")
	}
}

func TestBuildUserContext_DisableClaudeMds(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_CLAUDE_MDS", "1")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	uc, err := BuildUserContext(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := uc["claudeMd"]; ok {
		t.Fatalf("claudeMd should be omitted: %#v", uc)
	}
}

func TestBuildSystemContext_CacheBreaker(t *testing.T) {
	t.Setenv("FEATURE_BREAK_CACHE_COMMAND", "1")
	t.Setenv("CLAUDE_CODE_REMOTE", "1") // skip git
	dir := t.TempDir()
	inj := "debug-token"
	sc := BuildSystemContext(context.Background(), dir, &inj)
	if sc["cacheBreaker"] != "[CACHE_BREAKER: debug-token]" {
		t.Fatalf("%#v", sc)
	}
}
