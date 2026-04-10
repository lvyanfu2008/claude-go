package slashresolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goc/types"
)

func TestResolveDiskSkill_fixture(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	md := `---
name: my-skill
description: test
arguments: who what
allowed-tools: Read
---
# Title

Hello $who $what ${CLAUDE_SKILL_DIR} sid=${CLAUDE_SESSION_ID}
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	root := skillDir
	cmd := types.Command{
		CommandBase: types.CommandBase{Name: "my-skill", Description: "test"},
		Type:        "prompt",
		SkillRoot:   &root,
		AllowedTools: []string{"Read"},
	}
	res, err := ResolveDiskSkill(cmd, "a b", "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != types.SlashResolveDisk {
		t.Fatalf("source %q", res.Source)
	}
	for _, p := range []string{"Base directory for this skill:", skillDir, "Hello a b", "sess-1", filepath.ToSlash(skillDir)} {
		if !strings.Contains(res.UserText, p) {
			t.Fatalf("missing %q in:\n%s", p, res.UserText)
		}
	}
}
