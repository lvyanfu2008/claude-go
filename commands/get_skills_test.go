package commands

import (
	"context"
	"testing"

	"goc/types"
)

func TestGetSkills_nonNilSlices(t *testing.T) {
	t.Cleanup(ClearBuiltinPluginsRegistry)
	ClearBuiltinPluginsRegistry()

	b := GetSkills(context.Background(), t.TempDir(), DefaultLoadOptions())
	if b.SkillDirCommands == nil || b.PluginSkills == nil || b.BundledSkills == nil || b.BuiltinPluginSkills == nil {
		t.Fatalf("want non-nil slices, got %#v", b)
	}
	wantBundled := loadBundledSkills()
	if len(b.BundledSkills) != len(wantBundled) {
		t.Fatalf("bundled len %d vs loadBundledSkills %d", len(b.BundledSkills), len(wantBundled))
	}
}

func TestGetSkills_concatMatchesLoadAllCommandsPrefix(t *testing.T) {
	t.Cleanup(ClearLoadAllCommandsCache)
	t.Cleanup(ClearBuiltinPluginsRegistry)
	ClearBuiltinPluginsRegistry()

	ctx := context.Background()
	cwd := t.TempDir()
	opts := DefaultLoadOptions()

	ClearLoadAllCommandsCache()
	full, err := LoadAllCommands(ctx, cwd, opts)
	if err != nil {
		t.Fatal(err)
	}
	b := GetSkills(ctx, cwd, opts)
	prefixLen := len(b.BundledSkills) + len(b.BuiltinPluginSkills) + len(b.SkillDirCommands)
	if prefixLen > len(full) {
		t.Fatalf("prefixLen %d > full %d", prefixLen, len(full))
	}
	var concat []types.Command
	concat = append(concat, b.BundledSkills...)
	concat = append(concat, b.BuiltinPluginSkills...)
	concat = append(concat, b.SkillDirCommands...)
	for i := range concat {
		if concat[i].Name != full[i].Name {
			t.Fatalf("index %d: concat %q full %q", i, concat[i].Name, full[i].Name)
		}
	}
}
