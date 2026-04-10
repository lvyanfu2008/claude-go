package slashresolve

import (
	"strings"
	"testing"

	"goc/types"
)

func TestResolveBundledSkill_simplifyAppendsArgs(t *testing.T) {
	src := "bundled"
	cmd := types.Command{
		CommandBase: types.CommandBase{
			Name:       "simplify",
			LoadedFrom: &src,
		},
		Type:   "prompt",
		Source: &src,
	}
	res, err := ResolveBundledSkill(cmd, "focus on tests", "sid", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != types.SlashResolveBundledEmbed {
		t.Fatalf("source: %v", res.Source)
	}
	if !strings.Contains(res.UserText, "## Additional Focus") || !strings.Contains(res.UserText, "focus on tests") {
		t.Fatalf("expected TS simplify.ts args section, got len=%d", len(res.UserText))
	}
}

func TestResolveBundledSkill_loopUsageWhenEmpty(t *testing.T) {
	src := "bundled"
	cmd := types.Command{
		CommandBase: types.CommandBase{Name: "loop", LoadedFrom: &src},
		Type:        "prompt",
		Source:      &src,
	}
	res, err := ResolveBundledSkill(cmd, "", "sid", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.UserText, "Usage: /loop") {
		t.Fatalf("expected usage message")
	}
}

func TestResolveBundledSkill_verifyMaterializes(t *testing.T) {
	src := "bundled"
	cmd := types.Command{
		CommandBase: types.CommandBase{Name: "verify", LoadedFrom: &src},
		Type:        "prompt",
		Source:      &src,
	}
	res, err := ResolveBundledSkill(cmd, "check auth", "sid", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.MaterializedPaths) != 1 {
		t.Fatalf("materialized: %v", res.MaterializedPaths)
	}
	if !strings.HasPrefix(res.UserText, "Base directory for this skill:") {
		t.Fatalf("expected base dir prefix")
	}
}

func TestIsBundledPrompt(t *testing.T) {
	src := "bundled"
	if !IsBundledPrompt(types.Command{CommandBase: types.CommandBase{Name: "x"}, Type: "prompt", Source: &src}) {
		t.Fatal("expected bundled via source")
	}
	if IsBundledPrompt(types.Command{CommandBase: types.CommandBase{Name: "x"}, Type: "prompt"}) {
		t.Fatal("not bundled")
	}
}
