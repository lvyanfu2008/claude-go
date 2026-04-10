package commands

import (
	"testing"

	"goc/types"
)

func ptrBool(v bool) *bool { return &v }
func ptrStr(v string) *string { return &v }

func TestSkillToolCommands_mirrorsTSFilter(t *testing.T) {
	all := []types.Command{
		{CommandBase: types.CommandBase{Name: "a"}, Type: "local"},
		{
			CommandBase: types.CommandBase{
				Name:                   "b",
				DisableModelInvocation: ptrBool(true),
				LoadedFrom:             ptrStr("skills"),
			},
			Type: "prompt",
		},
		{
			CommandBase: types.CommandBase{Name: "c", LoadedFrom: ptrStr("skills")},
			Type:        "prompt",
			Source:      ptrStr("builtin"),
		},
		{CommandBase: types.CommandBase{Name: "d"}, Type: "prompt", Source: ptrStr("user")},
		{
			CommandBase: types.CommandBase{Name: "e", LoadedFrom: ptrStr("skills")},
			Type:        "prompt",
			Source:      ptrStr("user"),
		},
		{
			CommandBase: types.CommandBase{Name: "f", LoadedFrom: ptrStr("bundled")},
			Type:        "prompt",
			Source:      ptrStr("plugin"),
		},
		{
			CommandBase: types.CommandBase{Name: "g", LoadedFrom: ptrStr("commands_DEPRECATED")},
			Type:        "prompt",
		},
		{
			CommandBase: types.CommandBase{
				Name:                        "h",
				HasUserSpecifiedDescription: ptrBool(true),
			},
			Type: "prompt",
		},
		{
			CommandBase: types.CommandBase{Name: "i", WhenToUse: ptrStr("  use me  ")},
			Type:        "prompt",
		},
		{
			CommandBase: types.CommandBase{Name: "j", WhenToUse: ptrStr("   ")},
			Type:        "prompt",
		},
	}

	got := SkillToolCommands(all)
	names := make([]string, len(got))
	for i, c := range got {
		names[i] = c.Name
	}
	want := []string{"e", "f", "g", "h", "i"}
	if len(got) != len(want) {
		t.Fatalf("got %d commands %v, want %v", len(got), names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("got names %v, want %v", names, want)
		}
	}
}

func TestSkillToolCommands_disableFalseAllows(t *testing.T) {
	cmd := types.Command{
		CommandBase: types.CommandBase{
			Name:                   "x",
			DisableModelInvocation: ptrBool(false),
			LoadedFrom:             ptrStr("skills"),
		},
		Type: "prompt",
	}
	got := SkillToolCommands([]types.Command{cmd})
	if len(got) != 1 || got[0].Name != "x" {
		t.Fatalf("got %+v", got)
	}
}

func TestSkillToolCommands_nilSourceNotBuiltin(t *testing.T) {
	cmd := types.Command{
		CommandBase: types.CommandBase{Name: "x", LoadedFrom: ptrStr("skills")},
		Type:        "prompt",
	}
	got := SkillToolCommands([]types.Command{cmd})
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
}
