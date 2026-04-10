package commands

import (
	"errors"
	"testing"

	"goc/types"
)

func TestMeetsAvailabilityRequirement(t *testing.T) {
	open := types.Command{
		CommandBase: types.CommandBase{Name: "x"},
		Type:        "prompt",
	}
	if !MeetsAvailabilityRequirement(open, false, false, false) {
		t.Fatal("no availability should pass")
	}
	claudeOnly := types.Command{
		CommandBase: types.CommandBase{
			Name:         "y",
			Availability: []types.CommandAvailability{types.CommandAvailabilityClaudeAI},
		},
		Type: "prompt",
	}
	if MeetsAvailabilityRequirement(claudeOnly, false, false, false) {
		t.Fatal("claude subscriber required")
	}
	if !MeetsAvailabilityRequirement(claudeOnly, true, false, false) {
		t.Fatal("subscriber should pass")
	}
}

func TestFindCommand_GetCommand(t *testing.T) {
	cmds := []types.Command{
		{CommandBase: types.CommandBase{Name: "foo", Aliases: []string{"f"}}, Type: "prompt"},
		{CommandBase: types.CommandBase{Name: "bar"}, Type: "local"},
	}
	if p := FindCommand("f", cmds); p == nil || p.Name != "foo" {
		t.Fatal()
	}
	_, err := GetCommand("missing", cmds)
	if !errors.Is(err, ErrCommandNotFound) {
		t.Fatalf("err=%v", err)
	}
	got, err := GetCommand("bar", cmds)
	if err != nil || got.Name != "bar" {
		t.Fatal()
	}
}

func TestIsBridgeSafeCommand(t *testing.T) {
	if !IsBridgeSafeCommand(types.Command{CommandBase: types.CommandBase{Name: "x"}, Type: "prompt"}) {
		t.Fatal("prompt safe")
	}
	if IsBridgeSafeCommand(types.Command{CommandBase: types.CommandBase{Name: "x"}, Type: "local-jsx"}) {
		t.Fatal("jsx blocked")
	}
	if !IsBridgeSafeCommand(types.Command{CommandBase: types.CommandBase{Name: "compact"}, Type: "local"}) {
		t.Fatal("compact local safe")
	}
}

func TestFormatDescriptionWithSource(t *testing.T) {
	w := "workflow"
	s := "bundled"
	got := FormatDescriptionWithSource(types.Command{
		CommandBase: types.CommandBase{Description: "D", Kind: &w},
		Type:        "prompt",
	})
	if got != "D (workflow)" {
		t.Fatalf("%q", got)
	}
	got2 := FormatDescriptionWithSource(types.Command{
		CommandBase: types.CommandBase{Description: "B"},
		Type:        "prompt",
		Source:      &s,
	})
	if got2 != "B (bundled)" {
		t.Fatalf("%q", got2)
	}
}
