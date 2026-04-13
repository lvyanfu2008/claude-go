package toolpool

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestParseToolsAPIDocumentJSON_minimal(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"tools":[{"name":"Alpha","description":"d","input_schema":{"type":"object"}}]}`)
	got, err := ParseToolsAPIDocumentJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Alpha" || got[0].Description != "d" {
		t.Fatalf("%+v", got)
	}
}

func TestGetTools_removesSpecialIfPresent(t *testing.T) {
	base := []types.ToolSpec{
		{Name: "Read"},
		{Name: ListMcpResourcesToolName},
		{Name: "Bash"},
	}
	ctx := types.EmptyToolPermissionContextData()
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("CLAUDE_REPL_MODE", "")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	out := GetTools(ctx, base)
	if len(out) != 2 {
		t.Fatalf("got %#v", out)
	}
}

func TestGetTools_kairosChannelsStripsAskUserQuestion(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("CLAUDE_REPL_MODE", "")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("FEATURE_KAIROS", "1")
	t.Setenv("CLAUDE_CODE_GO_ALLOWED_CHANNELS", "discord")
	base := []types.ToolSpec{
		{Name: "Read"},
		{Name: "AskUserQuestion"},
		{Name: "Bash"},
	}
	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, base)
	for _, spec := range out {
		if spec.Name == "AskUserQuestion" {
			t.Fatalf("expected AskUserQuestion filtered under KAIROS + channels, got %#v", out)
		}
	}
	if len(out) != 2 {
		t.Fatalf("want 2 tools got %#v", out)
	}
	t.Setenv("FEATURE_KAIROS", "")
	t.Setenv("FEATURE_KAIROS_CHANNELS", "")
	t.Setenv("CLAUDE_CODE_GO_ALLOWED_CHANNELS", "")
	out2 := GetTools(ctx, base)
	if len(out2) != 3 {
		t.Fatalf("want 3 tools got %#v", out2)
	}
}

func TestGetTools_simpleMode(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "1")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	base := []types.ToolSpec{
		{Name: "Bash"}, {Name: "Read"}, {Name: "Edit"}, {Name: "Agent"},
	}
	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, base)
	if len(out) != 3 {
		t.Fatalf("want 3 tools got %#v", out)
	}
}

// TS getTools (tools.ts): CLAUDE_CODE_SIMPLE + isReplModeEnabled returns [REPL] (+ coordinator tools when active).
func TestGetTools_simpleReplMode_returnsREPLOnly(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "1")
	t.Setenv("CLAUDE_CODE_REPL", "")
	t.Setenv("CLAUDE_REPL_MODE", "1")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("FEATURE_COORDINATOR_MODE", "")
	t.Setenv("CLAUDE_CODE_COORDINATOR_MODE", "")
	base := []types.ToolSpec{
		{Name: "REPL"}, {Name: "Bash"}, {Name: "Read"}, {Name: "Edit"},
	}
	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, base)
	if len(out) != 1 || out[0].Name != "REPL" {
		t.Fatalf("want [REPL] got %#v", out)
	}
}

func TestGetTools_fullReplMode_hidesReplOnlyPrimitives(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "")
	t.Setenv("CLAUDE_REPL_MODE", "1")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	base := []types.ToolSpec{
		{Name: "REPL"}, {Name: "Read"}, {Name: "Bash"}, {Name: "Skill"},
	}
	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, base)
	names := map[string]struct{}{}
	for _, s := range out {
		names[s.Name] = struct{}{}
	}
	if _, ok := names["Read"]; ok {
		t.Fatalf("Read should be hidden when REPL present, got %#v", out)
	}
	if _, ok := names["REPL"]; !ok {
		t.Fatalf("want REPL in pool, got %#v", out)
	}
	if _, ok := names["Skill"]; !ok {
		t.Fatalf("want non-REPL_ONLY tool Skill, got %#v", out)
	}
}

func TestGetTools_fullReplMode_noHideWithoutREPLTool(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "")
	t.Setenv("CLAUDE_REPL_MODE", "1")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	base := []types.ToolSpec{{Name: "Read"}, {Name: "Bash"}}
	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, base)
	if len(out) != 2 {
		t.Fatalf("without REPL tool primitives stay visible, got %#v", out)
	}
}

func TestMarshalToolsAPIDocumentDefinitions(t *testing.T) {
	t.Parallel()
	raw, err := MarshalToolsAPIDocumentDefinitions([]types.ToolSpec{
		{Name: "Bash", Description: "x", InputJSONSchema: json.RawMessage(`{}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	var a []map[string]any
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 || a[0]["name"] != "Bash" {
		t.Fatalf("%s", raw)
	}
}

func TestToolSpecsFromEmbeddedToolsAPIJSON(t *testing.T) {
	t.Parallel()
	specs, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) < 5 {
		t.Fatalf("embedded tools_api.json too small: %d", len(specs))
	}
}

func TestGetTools_stripsGlobGrepWhenEmbeddedSearch(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("CLAUDE_REPL_MODE", "")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "cli")
	t.Setenv("EMBEDDED_SEARCH_TOOLS", "1")
	base := []types.ToolSpec{
		{Name: "Read"}, {Name: "Glob"}, {Name: "Grep"}, {Name: "Bash"},
	}
	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, base)
	for _, s := range out {
		if s.Name == "Glob" || s.Name == "Grep" {
			t.Fatalf("expected Glob/Grep removed when EMBEDDED_SEARCH_TOOLS, got %#v", out)
		}
	}
	if len(out) != 2 {
		t.Fatalf("want Read+Bash got %#v", out)
	}
}

func TestGetTools_cronDisabledEnvStripsCronTools(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("CLAUDE_CODE_DISABLE_CRON", "1")
	base := []types.ToolSpec{{Name: "CronCreate"}, {Name: "Read"}}
	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, base)
	for _, s := range out {
		if s.Name == "CronCreate" {
			t.Fatalf("expected CronCreate off when CLAUDE_CODE_DISABLE_CRON, got %#v", out)
		}
	}
}

func TestGetTools_planModeDisabledUnderKairosChannels(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("FEATURE_KAIROS_CHANNELS", "1")
	t.Setenv("CLAUDE_CODE_GO_ALLOWED_CHANNELS", "discord")
	base := []types.ToolSpec{{Name: "EnterPlanMode"}, {Name: "ExitPlanMode"}, {Name: "Read"}}
	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, base)
	for _, s := range out {
		if s.Name == "EnterPlanMode" || s.Name == "ExitPlanMode" {
			t.Fatalf("expected plan tools stripped, got %#v", out)
		}
	}
	if len(out) != 1 || out[0].Name != "Read" {
		t.Fatalf("got %#v", out)
	}
}

func TestGetTools_taskOutputHiddenForAnt(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("USER_TYPE", "ant")
	base := []types.ToolSpec{{Name: "TaskOutput"}, {Name: "Read"}}
	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, base)
	for _, s := range out {
		if s.Name == "TaskOutput" {
			t.Fatalf("expected TaskOutput hidden for ant, got %#v", out)
		}
	}
}

func TestGetTools_toolSearchDisabledInStandardMode(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "1")
	base := []types.ToolSpec{{Name: "ToolSearch"}, {Name: "Read"}}
	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, base)
	for _, s := range out {
		if s.Name == "ToolSearch" {
			t.Fatalf("expected ToolSearch off in standard tst mode, got %#v", out)
		}
	}
}
