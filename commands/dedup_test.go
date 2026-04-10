package commands

import (
	"testing"

	"goc/types"
)

func TestDedupeSkillEntries_SamePathTwice(t *testing.T) {
	lf := "skills"
	cmd := types.Command{
		CommandBase: types.CommandBase{Name: "a", LoadedFrom: &lf},
		Type:        "prompt",
	}
	e := []SkillLoadEntry{
		{Cmd: cmd, MarkdownPath: "/same/path/SKILL.md"},
		{Cmd: cmd, MarkdownPath: "/same/path/SKILL.md"},
	}
	out := dedupeSkillEntries(e)
	if len(out) != 1 {
		t.Fatalf("dedupe: want 1 got %d", len(out))
	}
}
