package toolpool

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func boolPtr(b bool) *bool { return &b }

func toolNames(tools []types.ToolSpec) []string {
	s := make([]string, len(tools))
	for i, t := range tools {
		s[i] = t.Name
	}
	return s
}

func TestAssembleToolPool_orderAndDedup(t *testing.T) {
	t.Parallel()
	ctx := types.EmptyToolPermissionContextData()
	builtIn := []types.ToolSpec{{Name: "Zebra"}, {Name: "Apple"}}
	mcp := []types.ToolSpec{
		{Name: "Mango", IsMcp: boolPtr(true)},
		{Name: "Apple", IsMcp: boolPtr(true)},
	}
	out := AssembleToolPool(ctx, builtIn, mcp)
	got := toolNames(out)
	want := []string{"Apple", "Zebra", "Mango"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestAssembleToolPool_denyMcp(t *testing.T) {
	t.Parallel()
	deny, _ := json.Marshal(map[string][]string{"localSettings": {"mcp__srv__t"}})
	ctx := types.ToolPermissionContextData{AlwaysDenyRules: deny, Mode: types.PermissionDefault}
	builtIn := []types.ToolSpec{{Name: "Read"}}
	mcp := []types.ToolSpec{
		{Name: "mcp__srv__t", MCPInfo: &types.MCPInfo{ServerName: "srv", ToolName: "t"}},
	}
	out := AssembleToolPool(ctx, builtIn, mcp)
	if len(out) != 1 || out[0].Name != "Read" {
		t.Fatalf("got %#v", out)
	}
}

func TestGetMergedTools(t *testing.T) {
	t.Parallel()
	ctx := types.EmptyToolPermissionContextData()
	a := []types.ToolSpec{{Name: "A"}}
	b := []types.ToolSpec{{Name: "B"}}
	out := GetMergedTools(ctx, a, b)
	if len(out) != 2 || out[0].Name != "A" || out[1].Name != "B" {
		t.Fatalf("got %#v", out)
	}
}

func TestMergeAndFilterTools_partitionSort(t *testing.T) {
	t.Setenv("FEATURE_COORDINATOR_MODE", "")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "")
	initial := []types.ToolSpec{{Name: "Zeta"}}
	assembled := []types.ToolSpec{
		{Name: "mcp__g__z", IsMcp: boolPtr(true)},
		{Name: "Alpha"},
	}
	out := MergeAndFilterTools(initial, assembled, types.PermissionDefault)
	// uniq: Zeta, mcp__g__z, Alpha — partition: builtIn Zeta Alpha, mcp mcp__g__z — sort: Alpha Zeta, mcp
	got := toolNames(out)
	want := []string{"Alpha", "Zeta", "mcp__g__z"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestMergeAndFilterTools_initialWinsDedup(t *testing.T) {
	t.Setenv("FEATURE_COORDINATOR_MODE", "")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "")
	initial := []types.ToolSpec{{Name: "Bash", MaxResultSizeChars: 99}}
	assembled := []types.ToolSpec{{Name: "Bash", MaxResultSizeChars: 1}}
	out := MergeAndFilterTools(initial, assembled, types.PermissionDefault)
	if len(out) != 1 || out[0].MaxResultSizeChars != 99 {
		t.Fatalf("initial should win uniqBy: %#v", out)
	}
}

func TestApplyCoordinatorToolFilter(t *testing.T) {
	t.Parallel()
	in := []types.ToolSpec{
		{Name: "Agent"},
		{Name: "Bash"},
		{Name: "x_subscribe_pr_activity"},
	}
	out := ApplyCoordinatorToolFilter(in)
	if len(out) != 2 {
		t.Fatalf("got %#v", out)
	}
}

func TestCoordinatorMergeFilterActive_env(t *testing.T) {
	t.Setenv("FEATURE_COORDINATOR_MODE", "")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "")
	if CoordinatorMergeFilterActive() {
		t.Fatal("expected inactive")
	}
	t.Setenv("FEATURE_COORDINATOR_MODE", "1")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "1")
	if !CoordinatorMergeFilterActive() {
		t.Fatal("expected active")
	}
}

func TestIsMcpTool(t *testing.T) {
	t.Parallel()
	if !IsMcpTool(types.ToolSpec{Name: "mcp__a__b"}) {
		t.Fatal("prefix")
	}
	if !IsMcpTool(types.ToolSpec{Name: "Write", IsMcp: boolPtr(true)}) {
		t.Fatal("flag")
	}
	if IsMcpTool(types.ToolSpec{Name: "Write"}) {
		t.Fatal("builtin Write")
	}
}
