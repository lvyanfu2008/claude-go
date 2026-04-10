package commands

import (
	"testing"

	"goc/types"
)

func TestMergeCommandsUniqByName_dedupWithinFirst(t *testing.T) {
	a := []types.Command{
		{CommandBase: types.CommandBase{Name: "x", Description: "1"}, Type: "prompt"},
		{CommandBase: types.CommandBase{Name: "x", Description: "2"}, Type: "prompt"},
	}
	out := MergeCommandsUniqByName(a, nil)
	if len(out) != 1 || out[0].Description != "1" {
		t.Fatalf("%+v", out)
	}
}

func TestMergeCommandsUniqByName_firstWins(t *testing.T) {
	a := []types.Command{{CommandBase: types.CommandBase{Name: "x", Description: "a"}, Type: "prompt"}}
	b := []types.Command{{CommandBase: types.CommandBase{Name: "x", Description: "b"}, Type: "prompt"}}
	out := MergeCommandsUniqByName(a, b)
	if len(out) != 1 || out[0].Description != "a" {
		t.Fatalf("%+v", out)
	}
}

func TestSkillListingCommandsForAPI_mcpAdds(t *testing.T) {
	local := []types.Command{{CommandBase: types.CommandBase{Name: "a", LoadedFrom: ptrStr("skills")}, Type: "prompt"}}
	mcp := []types.Command{{CommandBase: types.CommandBase{Name: "mcp1", LoadedFrom: ptrStr("mcp")}, Type: "prompt"}}
	out := SkillListingCommandsForAPI(local, mcp, true)
	if len(out) != 2 {
		t.Fatalf("got %d", len(out))
	}
}

func TestSkillListingCommandsForAPI_featureOffIgnoresMcp(t *testing.T) {
	local := []types.Command{{CommandBase: types.CommandBase{Name: "a", LoadedFrom: ptrStr("skills")}, Type: "prompt"}}
	mcp := []types.Command{{CommandBase: types.CommandBase{Name: "mcp1", LoadedFrom: ptrStr("mcp")}, Type: "prompt"}}
	out := SkillListingCommandsForAPI(local, mcp, false)
	if len(out) != 1 || out[0].Name != "a" {
		t.Fatal()
	}
}

func TestSkillListingFromTSPresliced_mcpAdds(t *testing.T) {
	tsSlice := []types.Command{{CommandBase: types.CommandBase{Name: "from_ts", LoadedFrom: ptrStr("bundled")}, Type: "prompt"}}
	mcp := []types.Command{{CommandBase: types.CommandBase{Name: "mcp1", LoadedFrom: ptrStr("mcp")}, Type: "prompt"}}
	out := SkillListingFromTSPresliced(tsSlice, mcp, true)
	if len(out) != 2 {
		t.Fatalf("got %d %+v", len(out), out)
	}
}

func TestSkillListingFromTSPresliced_noMcpCopy(t *testing.T) {
	tsSlice := []types.Command{{CommandBase: types.CommandBase{Name: "only", LoadedFrom: ptrStr("skills")}, Type: "prompt"}}
	out := SkillListingFromTSPresliced(tsSlice, nil, true)
	if len(out) != 1 || out[0].Name != "only" {
		t.Fatalf("%+v", out)
	}
}
