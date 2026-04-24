package toolexecution

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func testToolPerm(rules ...string) *types.ToolPermissionContextData {
	b, err := json.Marshal(map[string][]string{"userSettings": rules})
	if err != nil {
		panic(err)
	}
	ctx := types.ToolPermissionContextData{
		Mode:            types.PermissionDefault,
		AlwaysDenyRules: b,
		AlwaysAllowRules: json.RawMessage(`{}`),
		AlwaysAskRules:   json.RawMessage(`{}`),
		AdditionalWorkingDirectories: json.RawMessage(`{}`),
	}
	types.NormalizeToolPermissionContextData(&ctx)
	return &ctx
}

func TestWholeToolAlwaysDenyAskAgentSubagent(t *testing.T) {
	t.Parallel()
	perm := testToolPerm("Agent(Explore)")
	in := []byte(`{"description":"d","prompt":"p","subagent_type":"Explore"}`)
	d := wholeToolAlwaysDenyAsk("Agent", in, perm, nil)
	if d == nil || d.Behavior != PermissionDeny {
		t.Fatalf("expected subagent deny, got %#v", d)
	}
}

func TestWholeToolAlwaysDenyAskAgentSkipsOnResume(t *testing.T) {
	t.Parallel()
	perm := testToolPerm("Agent(Explore)")
	in := []byte(`{"description":"d","prompt":"p","subagent_type":"Explore","resume":"agent-1"}`)
	d := wholeToolAlwaysDenyAsk("Agent", in, perm, nil)
	if d != nil {
		t.Fatalf("expected resume to skip per-agent deny, got %#v", d)
	}
}

func TestWholeToolAlwaysDenyAskAgentDefaultGeneralPurpose(t *testing.T) {
	t.Parallel()
	perm := testToolPerm("Agent(general-purpose)")
	in := []byte(`{"description":"d","prompt":"p"}`)
	d := wholeToolAlwaysDenyAsk("Agent", in, perm, nil)
	if d == nil || d.Behavior != PermissionDeny {
		t.Fatalf("expected deny for default subagent, got %#v", d)
	}
}
