package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"goc/types"
	"goc/utils"
)

func indexOfCommandName(out []types.Command, name string) int {
	for i, c := range out {
		if c.Name == name {
			return i
		}
	}
	return -1
}

// indexStartOfBuiltinTail is the first index of the COMMANDS() block in [LoadAllCommands] output
// (bundled may reuse the same name, e.g. "update-config").
func indexStartOfBuiltinTail(out []types.Command) int {
	b := loadBuiltinCommands()
	if len(b) == 0 {
		return len(out)
	}
	idx := len(out) - len(b)
	if idx < 0 {
		return 0
	}
	return idx
}

// lenEmbeddedStaticTail: bundled + builtin plugin skills + COMMANDS() when skill-dir / workflow / plugin sources are empty.
func lenEmbeddedStaticTail() int {
	bp, _ := loadBuiltinPluginSkills(".")
	return len(loadBundledSkills()) + len(bp) + len(loadBuiltinCommands())
}

func TestLoadBuiltinPluginSkills_handwritten(t *testing.T) {
	cmds, err := loadBuiltinPluginSkills(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cmds == nil {
		t.Fatal("expected non-nil slice (may be empty)")
	}
}

func TestLoadBundledSkills_handwritten(t *testing.T) {
	cmds := loadBundledSkills()
	if len(cmds) < 1 {
		t.Fatalf("expected embedded bundled skills, got %d", len(cmds))
	}
	for _, c := range cmds {
		if c.Source == nil || *c.Source != "bundled" {
			t.Fatalf("expected source bundled, got %#v", c.Source)
		}
		if c.Type != "prompt" {
			t.Fatalf("expected prompt type, got %q", c.Type)
		}
	}
}

func TestLoadBuiltinCommands_handwritten(t *testing.T) {
	cmds := loadBuiltinCommands()
	if len(cmds) < 40 {
		t.Fatalf("expected embedded COMMANDS snapshot, got len=%d", len(cmds))
	}
	seen := make(map[string]struct{})
	for _, c := range cmds {
		if c.Name == "" {
			t.Fatal("empty command name")
		}
		seen[c.Name] = struct{}{}
	}
	for _, need := range []string{"help", "config", "skills", "clear", "mcp"} {
		if _, ok := seen[need]; !ok {
			t.Fatalf("missing builtin slash command %q", need)
		}
	}
}

func TestLoadAllCommands_IncludesBuiltinTail(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	f := false
	out, err := LoadAllCommands(context.Background(), tmp, LoadOptions{BareMode: &f})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 40 {
		t.Fatalf("expected builtins + empty sources, got len=%d", len(out))
	}
}

func TestLoadAllCommands_EmptySources(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	f := false
	opts := LoadOptions{BareMode: &f}
	out, err := LoadAllCommands(context.Background(), tmp, opts)
	if err != nil {
		t.Fatal(err)
	}
	want := lenEmbeddedStaticTail()
	if len(out) != want {
		t.Fatalf("expected bundled+builtin-plugin+builtin-only list len=%d, got %d", want, len(out))
	}
}

func TestLoadAllCommands_ProjectSkillFirstWhenEarlierSourcesEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	skillRoot := filepath.Join(tmp, "proj", ".claude", "skills", "myskill")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	md := filepath.Join(skillRoot, "SKILL.md")
	content := "---\ndescription: test skill\n---\n\n# Hello\n"
	if err := os.WriteFile(md, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	opts := LoadOptions{BareMode: &f}
	out, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "proj"), opts)
	if err != nil {
		t.Fatal(err)
	}
	i := indexOfCommandName(out, "myskill")
	if i < 0 {
		t.Fatal("expected myskill in load order before builtins tail")
	}
	c := out[i]
	if c.Type != "prompt" {
		t.Fatalf("type: got %q", c.Type)
	}
	if c.LoadedFrom == nil || *c.LoadedFrom != "skills" {
		t.Fatalf("loadedFrom: %+v", c.LoadedFrom)
	}
	if c.Source == nil || *c.Source != "projectSettings" {
		t.Fatalf("source: %+v", c.Source)
	}
	if ib := indexStartOfBuiltinTail(out); i >= ib {
		t.Fatalf("expected project skill before builtin tail (myskill@%d tail@%d)", i, ib)
	}
}

func TestLoadAllCommands_CacheByCwd(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	f := false
	opts := LoadOptions{BareMode: &f}
	ctx := context.Background()
	a, err := LoadAllCommands(ctx, tmp, opts)
	if err != nil {
		t.Fatal(err)
	}
	b, err := LoadAllCommands(ctx, tmp, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) {
		t.Fatalf("cache mismatch len %d vs %d", len(a), len(b))
	}
	other := filepath.Join(tmp, "other")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	c, err := LoadAllCommands(ctx, other, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(c) != len(a) {
		t.Fatalf("other cwd should match len (both skill-dir empty): %d vs %d", len(c), len(a))
	}
}

func TestLoadAllCommands_BareSkipsUnlessAddDir(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	skillRoot := filepath.Join(tmp, "proj", ".claude", "skills", "x")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte("---\ndescription: x\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tr := true
	opts := LoadOptions{BareMode: &tr}
	out, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "proj"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if want := lenEmbeddedStaticTail(); len(out) != want {
		t.Fatalf("bare: expected only embedded static tail len=%d, got %d", want, len(out))
	}
	opts2 := LoadOptions{BareMode: &tr, AddSkillDirs: []string{filepath.Join(tmp, "proj")}}
	ClearLoadAllCommandsCache()
	out2, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "proj"), opts2)
	if err != nil {
		t.Fatal(err)
	}
	if ix := indexOfCommandName(out2, "x"); ix < 0 {
		t.Fatalf("bare+add-dir: missing skill x in %#v", out2)
	}
	if len(out2) != lenEmbeddedStaticTail()+1 {
		t.Fatalf("bare+add-dir: want embedded static tail+1, got %d", len(out2))
	}
}

func TestCommandFromSkillMD_AllowedToolsSlice(t *testing.T) {
	cmd, err := commandFromSkillMD("t", "/skill", "/x/SKILL.md", []byte("---\nallowed-tools:\n  - Bash\n  - Read\n---\n"), "userSettings")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.AllowedTools) != 2 || cmd.AllowedTools[0] != "Bash" {
		t.Fatalf("allowedTools: %#v", cmd.AllowedTools)
	}
}

func TestParseEffortInSkill(t *testing.T) {
	cmd, err := commandFromSkillMD("t", "/s", "/p", []byte("---\neffort: high\n---\n# x\n"), "userSettings")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Effort == nil {
		t.Fatal("expected effort")
	}
	lvl, ok := cmd.Effort.Level()
	if !ok || lvl != utils.EffortHigh {
		t.Fatalf("effort: %+v", cmd.Effort)
	}
}

func TestLoadAllCommands_ManagedSkillsBeforeUserSkills(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	t.Setenv("CLAUDE_CODE_MANAGED_SETTINGS_PATH", filepath.Join(tmp, "managed"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills", "us"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "cfg", "skills", "us", "SKILL.md"), []byte("---\ndescription: u\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "managed", ".claude", "skills", "ms"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "managed", ".claude", "skills", "ms", "SKILL.md"), []byte("---\ndescription: m\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	out, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "cwd"), LoadOptions{BareMode: &f})
	if err != nil {
		t.Fatal(err)
	}
	ims := indexOfCommandName(out, "ms")
	ius := indexOfCommandName(out, "us")
	if ims < 0 || ius < 0 {
		t.Fatalf("missing ms or us: ms@%d us@%d", ims, ius)
	}
	if ims >= ius {
		t.Fatalf("want managed ms before user us: ms@%d us@%d", ims, ius)
	}
	if out[ims].Source == nil || *out[ims].Source != "policySettings" {
		t.Fatalf("managed source: %+v", out[ims].Source)
	}
	if ib := indexStartOfBuiltinTail(out); ims >= ib {
		t.Fatalf("skills should precede builtin tail (tail@%d)", ib)
	}
}

func TestLoadAllCommands_DisablePolicySkillsEnvSkipsManagedSkillsDir(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	t.Setenv("CLAUDE_CODE_MANAGED_SETTINGS_PATH", filepath.Join(tmp, "managed"))
	t.Setenv("CLAUDE_CODE_DISABLE_POLICY_SKILLS", "1")
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills", "us"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "cfg", "skills", "us", "SKILL.md"), []byte("---\ndescription: u\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "managed", ".claude", "skills", "ms"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "managed", ".claude", "skills", "ms", "SKILL.md"), []byte("---\ndescription: m\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	out, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "cwd"), LoadOptions{BareMode: &f})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range out {
		if c.Name == "ms" {
			t.Fatal("managed skill should be skipped when CLAUDE_CODE_DISABLE_POLICY_SKILLS")
		}
	}
}

func TestLoadAllCommands_LegacyCommandsMarkdown(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmdDir := filepath.Join(tmp, "proj", ".claude", "commands")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "hello.md"), []byte("---\ndescription: legacy hi\n---\n\n# Body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	out, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "proj"), LoadOptions{BareMode: &f})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, c := range out {
		if c.Name == "hello" {
			found = true
			if c.LoadedFrom == nil || *c.LoadedFrom != "commands_DEPRECATED" {
				t.Fatalf("loadedFrom: %+v", c.LoadedFrom)
			}
			if c.Source == nil || *c.Source != "projectSettings" {
				t.Fatalf("source: %+v", c.Source)
			}
		}
	}
	if !found {
		t.Fatal("expected legacy hello command")
	}
}

func TestLoadAllCommands_ConditionalSkillOmittedUnlessIncludeConditionalSkills(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	skillRoot := filepath.Join(tmp, "proj", ".claude", "skills", "cond")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	md := filepath.Join(skillRoot, "SKILL.md")
	content := "---\ndescription: c\npaths: src/**\n---\n\n# Hello\n"
	if err := os.WriteFile(md, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	out, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "proj"), LoadOptions{BareMode: &f})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range out {
		if c.Name == "cond" {
			t.Fatal("conditional skill should be omitted when IncludeConditionalSkills is false")
		}
	}
	ClearLoadAllCommandsCache()
	out2, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "proj"), LoadOptions{BareMode: &f, IncludeConditionalSkills: true})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, c := range out2 {
		if c.Name == "cond" {
			found = true
			if len(c.Paths) == 0 {
				t.Fatal("expected paths when included")
			}
		}
	}
	if !found {
		t.Fatal("expected cond skill when IncludeConditionalSkills is true")
	}
}

func TestLoadAllCommands_EmptyEnabledSettingSourcesSkipsUserSkills(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills", "u"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "cfg", "skills", "u", "SKILL.md"), []byte("---\ndescription: u\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	out, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "cwd"), LoadOptions{BareMode: &f, EnabledSettingSources: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range out {
		if c.Name == "u" {
			t.Fatal("user skill should be skipped when EnabledSettingSources is empty (isolation)")
		}
	}
}

func TestLoadAllCommands_SkillsPluginOnlyLockedSkipsUserNotManaged(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	t.Setenv("CLAUDE_CODE_MANAGED_SETTINGS_PATH", filepath.Join(tmp, "managed"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills", "us"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "cfg", "skills", "us", "SKILL.md"), []byte("---\ndescription: u\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "managed", ".claude", "skills", "ms"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "managed", ".claude", "skills", "ms", "SKILL.md"), []byte("---\ndescription: m\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	out, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "cwd"), LoadOptions{BareMode: &f, SkillsPluginOnlyLocked: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != lenEmbeddedStaticTail()+1 {
		t.Fatalf("want managed skill + embedded static tail, got len=%d", len(out))
	}
	if indexOfCommandName(out, "ms") < 0 {
		t.Fatal("missing managed ms")
	}
	if indexOfCommandName(out, "us") >= 0 {
		t.Fatal("user skill should be skipped when SkillsPluginOnlyLocked")
	}
}

func TestDefaultLoadOptions_nonBareNilSources(t *testing.T) {
	o := DefaultLoadOptions()
	if o.BareMode == nil || *o.BareMode {
		t.Fatalf("expected BareMode false: %#v", o.BareMode)
	}
	if o.EnabledSettingSources != nil {
		t.Fatalf("expected nil EnabledSettingSources, got %#v", o.EnabledSettingSources)
	}
	if !o.isSettingSourceEnabled("projectSettings") {
		t.Fatal("expected projectSettings enabled by default")
	}
}
