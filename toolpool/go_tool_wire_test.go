package toolpool

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"goc/ccb-engine/bashzog"
	"goc/types"
)

func setToolWireExportParityEnv(t *testing.T) {
	t.Helper()
	t.Setenv("EMBEDDED_SEARCH_TOOLS", "0")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "1")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "0")
}

func TestGoToolWireParsesEmbeddedExport(t *testing.T) {
	setToolWireExportParityEnv(t)
	got := ToolSpecsFromGoWire()
	if len(got) == 0 {
		t.Fatal("ToolSpecsFromGoWire returned no tools")
	}
}

func TestGoToolWireAgentToolHasFullDescription(t *testing.T) {
	setToolWireExportParityEnv(t)
	got := ToolSpecsFromGoWire()
	
	var agentTool *types.ToolSpec
	for i := range got {
		if got[i].Name == "Agent" {
			agentTool = &got[i]
			break
		}
	}
	
	if agentTool == nil {
		t.Fatal("Agent tool not found in ToolSpecsFromGoWire output")
	}
	
	// Verify that Agent tool has the full description, not the short fallback
	if agentTool.Description == "Launch a new agent to handle complex tasks." {
		t.Fatal("Agent tool is still using fallback description instead of full description")
	}
	
	// Verify the description contains key sections that should be in the full description
	expectedSections := []string{
		"Launch a new agent to handle complex, multi-step tasks autonomously",
		"Usage notes:",
		"Writing the prompt",
		"Example usage:",
	}
	
	for _, section := range expectedSections {
		if !strings.Contains(agentTool.Description, section) {
			t.Fatalf("Agent description missing expected section: %q", section)
		}
	}
	
	// Verify the description is substantial (not just the short fallback)
	if len(agentTool.Description) < 1000 {
		t.Fatalf("Agent description too short (%d chars), expected full description", len(agentTool.Description))
	}
}

func TestToolSpecsFromGoWireEmbeddedSearchOmitsGlobGrep(t *testing.T) {
	t.Setenv("EMBEDDED_SEARCH_TOOLS", "1")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "1")

	got := toolNames(ToolSpecsFromGoWire())
	for _, name := range got {
		if name == "Glob" || name == "Grep" {
			t.Fatalf("expected %s to be omitted when EMBEDDED_SEARCH_TOOLS=1", name)
		}
	}
}

func TestToolSpecsFromGoWireToolSearchGate(t *testing.T) {
	t.Setenv("EMBEDDED_SEARCH_TOOLS", "0")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "0")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "0")

	got := toolNames(ToolSpecsFromGoWire())
	for _, name := range got {
		if name == "ToolSearch" {
			t.Fatalf("ToolSearch should be omitted when tool-search gate is off")
		}
	}
}

func TestToolSpecsFromGoWireOptionalFeatureToolsNativeProviderCanIncludeWithoutExportRows(t *testing.T) {
	t.Setenv("EMBEDDED_SEARCH_TOOLS", "0")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "1")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "0")
	t.Setenv("FEATURE_MONITOR_TOOL", "1")
	t.Setenv("FEATURE_WORKFLOW_SCRIPTS", "1")
	t.Setenv("FEATURE_HISTORY_SNIP", "1")
	t.Setenv("FEATURE_CONTEXT_COLLAPSE", "1")
	t.Setenv("FEATURE_UDS_INBOX", "1")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ToolSpecsFromGoWire should not panic for missing optional tool schemas, panic=%v", r)
		}
	}()

	names := toolNames(ToolSpecsFromGoWire())
	for _, optional := range []string{"Monitor", "workflow", "Snip", "CtxInspect", "ListPeers"} {
		found := false
		for _, got := range names {
			if got == optional {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected optional tool %q to be present when feature gate is enabled", optional)
		}
	}
}

func TestBuildGoWireToolSpecsFromExportSpecsDoesNotUseInjectedOptionalExportRows(t *testing.T) {
	t.Setenv("FEATURE_MONITOR_TOOL", "1")
	t.Setenv("FEATURE_WORKFLOW_SCRIPTS", "1")
	t.Setenv("FEATURE_HISTORY_SNIP", "1")
	t.Setenv("FEATURE_CONTEXT_COLLAPSE", "1")
	t.Setenv("FEATURE_UDS_INBOX", "1")

	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	in := append(base, []types.ToolSpec{
		{Name: "Monitor"},
		{Name: "workflow"},
		{Name: "Snip"},
		{Name: "CtxInspect"},
		{Name: "ListPeers"},
	}...)
	denyProvider := func(name string) (types.ToolSpec, bool, error) {
		switch name {
		case "Monitor", "workflow", "Snip", "CtxInspect", "ListPeers":
			return types.ToolSpec{}, false, nil
		default:
			return nativeSpecFromGoProvider(name)
		}
	}
	out := buildGoWireToolSpecsFromExportSpecsWithProvider(in, denyProvider)
	names := toolNames(out)
	for _, blocked := range []string{"Monitor", "workflow", "Snip", "CtxInspect", "ListPeers"} {
		for _, got := range names {
			if got == blocked {
				t.Fatalf("optional tool %q should not be injected from export rows", blocked)
			}
		}
	}
}

func TestBuildGoWireToolSpecsFromExportSpecsNativeProviderOverridesExport(t *testing.T) {
	setToolWireExportParityEnv(t)
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	for i := range base {
		if base[i].Name == "Bash" {
			base[i].Description = "fake-export-bash-description"
			break
		}
	}
	out := buildGoWireToolSpecsFromExportSpecsWithProvider(base, nativeSpecFromGoProvider)

	var got *types.ToolSpec
	for i := range out {
		if out[i].Name == "Bash" {
			got = &out[i]
			break
		}
	}
	if got == nil {
		t.Fatal("Bash not found in output")
	}
	if got.Description == "fake-export-bash-description" {
		t.Fatal("expected native provider to override export Bash description")
	}
	native, err := bashzog.BashToolSpec()
	if err != nil {
		t.Fatalf("load native Bash spec: %v", err)
	}
	if got.Description != native.Description {
		t.Fatalf("bash description mismatch: got=%q want(native)=%q", got.Description, native.Description)
	}
}

func TestBuildGoWireToolSpecsFromExportSpecsProviderOverridesRead(t *testing.T) {
	setToolWireExportParityEnv(t)
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	for i := range base {
		if base[i].Name == "Read" {
			base[i].Description = "fake-export-read-description"
			break
		}
	}
	out := buildGoWireToolSpecsFromExportSpecsWithProvider(base, nativeSpecFromGoProvider)

	var got *types.ToolSpec
	for i := range out {
		if out[i].Name == "Read" {
			got = &out[i]
			break
		}
	}
	if got == nil {
		t.Fatal("Read not found in output")
	}
	if got.Description == "fake-export-read-description" {
		t.Fatal("expected provider path to override export Read description")
	}
}

func TestNativeReadToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	var embedded *types.ToolSpec
	for i := range base {
		if base[i].Name == "Read" {
			embedded = &base[i]
			break
		}
	}
	if embedded == nil {
		t.Fatal("embedded Read tool not found")
	}
	native := nativeReadToolSpec()
	var gotSchema any
	if err := json.Unmarshal(native.InputJSONSchema, &gotSchema); err != nil {
		t.Fatalf("unmarshal native Read schema: %v", err)
	}
	var wantSchema any
	if err := json.Unmarshal(embedded.InputJSONSchema, &wantSchema); err != nil {
		t.Fatalf("unmarshal embedded Read schema: %v", err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Fatal("native Read schema does not match embedded export")
	}
}

func TestNativeWriteToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	var embedded *types.ToolSpec
	for i := range base {
		if base[i].Name == "Write" {
			embedded = &base[i]
			break
		}
	}
	if embedded == nil {
		t.Fatal("embedded Write tool not found")
	}
	native := nativeWriteToolSpec()
	var gotSchema any
	if err := json.Unmarshal(native.InputJSONSchema, &gotSchema); err != nil {
		t.Fatalf("unmarshal native Write schema: %v", err)
	}
	var wantSchema any
	if err := json.Unmarshal(embedded.InputJSONSchema, &wantSchema); err != nil {
		t.Fatalf("unmarshal embedded Write schema: %v", err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Fatal("native Write schema does not match embedded export")
	}
}

func TestNativeEditToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	var embedded *types.ToolSpec
	for i := range base {
		if base[i].Name == "Edit" {
			embedded = &base[i]
			break
		}
	}
	if embedded == nil {
		t.Fatal("embedded Edit tool not found")
	}
	native := nativeEditToolSpec()
	var gotSchema any
	if err := json.Unmarshal(native.InputJSONSchema, &gotSchema); err != nil {
		t.Fatalf("unmarshal native Edit schema: %v", err)
	}
	var wantSchema any
	if err := json.Unmarshal(embedded.InputJSONSchema, &wantSchema); err != nil {
		t.Fatalf("unmarshal embedded Edit schema: %v", err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Fatal("native Edit schema does not match embedded export")
	}
}

func TestNativeGlobToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	var embedded *types.ToolSpec
	for i := range base {
		if base[i].Name == "Glob" {
			embedded = &base[i]
			break
		}
	}
	if embedded == nil {
		t.Fatal("embedded Glob tool not found")
	}
	native := nativeGlobToolSpec()
	var gotSchema any
	if err := json.Unmarshal(native.InputJSONSchema, &gotSchema); err != nil {
		t.Fatalf("unmarshal native Glob schema: %v", err)
	}
	var wantSchema any
	if err := json.Unmarshal(embedded.InputJSONSchema, &wantSchema); err != nil {
		t.Fatalf("unmarshal embedded Glob schema: %v", err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Fatal("native Glob schema does not match embedded export")
	}
}

func TestNativeGrepToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	var embedded *types.ToolSpec
	for i := range base {
		if base[i].Name == "Grep" {
			embedded = &base[i]
			break
		}
	}
	if embedded == nil {
		t.Fatal("embedded Grep tool not found")
	}
	native := nativeGrepToolSpec()
	var gotSchema any
	if err := json.Unmarshal(native.InputJSONSchema, &gotSchema); err != nil {
		t.Fatalf("unmarshal native Grep schema: %v", err)
	}
	var wantSchema any
	if err := json.Unmarshal(embedded.InputJSONSchema, &wantSchema); err != nil {
		t.Fatalf("unmarshal embedded Grep schema: %v", err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Fatal("native Grep schema does not match embedded export")
	}
}

func TestNativeNotebookEditToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	var embedded *types.ToolSpec
	for i := range base {
		if base[i].Name == "NotebookEdit" {
			embedded = &base[i]
			break
		}
	}
	if embedded == nil {
		t.Fatal("embedded NotebookEdit tool not found")
	}
	native := nativeNotebookEditToolSpec()
	var gotSchema any
	if err := json.Unmarshal(native.InputJSONSchema, &gotSchema); err != nil {
		t.Fatalf("unmarshal native NotebookEdit schema: %v", err)
	}
	var wantSchema any
	if err := json.Unmarshal(embedded.InputJSONSchema, &wantSchema); err != nil {
		t.Fatalf("unmarshal embedded NotebookEdit schema: %v", err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Fatal("native NotebookEdit schema does not match embedded export")
	}
}

func TestNativeTaskStopToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	var embedded *types.ToolSpec
	for i := range base {
		if base[i].Name == "TaskStop" {
			embedded = &base[i]
			break
		}
	}
	if embedded == nil {
		t.Fatal("embedded TaskStop tool not found")
	}
	native := nativeTaskStopToolSpec()
	var gotSchema any
	if err := json.Unmarshal(native.InputJSONSchema, &gotSchema); err != nil {
		t.Fatalf("unmarshal native TaskStop schema: %v", err)
	}
	var wantSchema any
	if err := json.Unmarshal(embedded.InputJSONSchema, &wantSchema); err != nil {
		t.Fatalf("unmarshal embedded TaskStop schema: %v", err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Fatal("native TaskStop schema does not match embedded export")
	}
}

func TestNativeTodoWriteToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	var embedded *types.ToolSpec
	for i := range base {
		if base[i].Name == "TodoWrite" {
			embedded = &base[i]
			break
		}
	}
	if embedded == nil {
		t.Fatal("embedded TodoWrite tool not found")
	}
	native := nativeTodoWriteToolSpec()
	var gotSchema any
	if err := json.Unmarshal(native.InputJSONSchema, &gotSchema); err != nil {
		t.Fatalf("unmarshal native TodoWrite schema: %v", err)
	}
	var wantSchema any
	if err := json.Unmarshal(embedded.InputJSONSchema, &wantSchema); err != nil {
		t.Fatalf("unmarshal embedded TodoWrite schema: %v", err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Fatal("native TodoWrite schema does not match embedded export")
	}
}

func TestNativeWebFetchToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	var embedded *types.ToolSpec
	for i := range base {
		if base[i].Name == "WebFetch" {
			embedded = &base[i]
			break
		}
	}
	if embedded == nil {
		t.Fatal("embedded WebFetch tool not found")
	}
	native := nativeWebFetchToolSpec()
	var gotSchema any
	if err := json.Unmarshal(native.InputJSONSchema, &gotSchema); err != nil {
		t.Fatalf("unmarshal native WebFetch schema: %v", err)
	}
	var wantSchema any
	if err := json.Unmarshal(embedded.InputJSONSchema, &wantSchema); err != nil {
		t.Fatalf("unmarshal embedded WebFetch schema: %v", err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Fatal("native WebFetch schema does not match embedded export")
	}
}

func TestNativeWebSearchToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	var embedded *types.ToolSpec
	for i := range base {
		if base[i].Name == "WebSearch" {
			embedded = &base[i]
			break
		}
	}
	if embedded == nil {
		t.Fatal("embedded WebSearch tool not found")
	}
	native := nativeWebSearchToolSpec()
	var gotSchema any
	if err := json.Unmarshal(native.InputJSONSchema, &gotSchema); err != nil {
		t.Fatalf("unmarshal native WebSearch schema: %v", err)
	}
	var wantSchema any
	if err := json.Unmarshal(embedded.InputJSONSchema, &wantSchema); err != nil {
		t.Fatalf("unmarshal embedded WebSearch schema: %v", err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Fatal("native WebSearch schema does not match embedded export")
	}
}

func TestNativeEnterPlanModeToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	assertNativeSchemaMatchesEmbedded(t, "EnterPlanMode", nativeEnterPlanModeToolSpec)
}

func TestNativeExitPlanModeToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	assertNativeSchemaMatchesEmbedded(t, "ExitPlanMode", nativeExitPlanModeToolSpec)
}

func TestNativeCronCreateToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	assertNativeSchemaMatchesEmbedded(t, "CronCreate", nativeCronCreateToolSpec)
}

func TestNativeCronDeleteToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	assertNativeSchemaMatchesEmbedded(t, "CronDelete", nativeCronDeleteToolSpec)
}

func TestNativeCronListToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	assertNativeSchemaMatchesEmbedded(t, "CronList", nativeCronListToolSpec)
}

func TestNativeAskUserQuestionToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	assertNativeSchemaMatchesEmbedded(t, "AskUserQuestion", nativeAskUserQuestionToolSpec)
}

func TestNativeEnterWorktreeToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	assertNativeSchemaMatchesEmbedded(t, "EnterWorktree", nativeEnterWorktreeToolSpec)
}

func TestNativeExitWorktreeToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	assertNativeSchemaMatchesEmbedded(t, "ExitWorktree", nativeExitWorktreeToolSpec)
}

func TestNativeSkillToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	assertNativeSchemaMatchesEmbedded(t, "Skill", nativeSkillToolSpec)
}

func TestNativeTaskOutputToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	assertNativeSchemaMatchesEmbedded(t, "TaskOutput", nativeTaskOutputToolSpec)
}

func TestNativeToolSearchToolSpecSchemaMatchesEmbeddedExport(t *testing.T) {
	assertNativeSchemaMatchesEmbedded(t, "ToolSearch", nativeToolSearchToolSpec)
}

func TestNativeAgentToolSpecSchemaAndDescription(t *testing.T) {
	spec := nativeAgentToolSpec()
	
	// Verify basic fields
	if spec.Name != "Agent" {
		t.Fatalf("expected name 'Agent', got %q", spec.Name)
	}
	
	// Verify description is the full description from agentToolDescription
	if spec.Description != agentToolDescription {
		t.Fatal("Agent spec description doesn't match agentToolDescription constant")
	}
	
	// Verify schema has required fields
	var schema map[string]any
	if err := json.Unmarshal(spec.InputJSONSchema, &schema); err != nil {
		t.Fatalf("invalid schema: %v", err)
	}
	
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing properties")
	}
	
	requiredFields := []string{"description", "prompt"}
	for _, field := range requiredFields {
		if _, exists := properties[field]; !exists {
			t.Fatalf("schema missing required field: %s", field)
		}
	}
}

func TestRequiredGoWireToolsCoveredByNativeProvider(t *testing.T) {
	for _, e := range goWireBaseTools {
		if !e.Required {
			continue
		}
		spec, ok, err := nativeSpecFromGoProvider(e.Name)
		if err != nil {
			t.Fatalf("required tool %q native provider returned error: %v", e.Name, err)
		}
		if !ok {
			t.Fatalf("required tool %q is not covered by native provider", e.Name)
		}
		if spec.Name != e.Name {
			t.Fatalf("required tool %q provider returned mismatched name %q", e.Name, spec.Name)
		}
		if len(spec.InputJSONSchema) == 0 {
			t.Fatalf("required tool %q provider returned empty input schema", e.Name)
		}
	}
}

func TestBuildGoWireToolSpecsNativeProviderCoversRequiredWithoutExportRows(t *testing.T) {
	t.Setenv("EMBEDDED_SEARCH_TOOLS", "0")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "1")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "0")

	// Intentionally provide no required rows from export to ensure assembly does
	// not rely on embedded rows for required tools.
	out := buildGoWireToolSpecsFromExportSpecsWithProvider(nil, nativeSpecFromGoProvider)
	if len(out) == 0 {
		t.Fatal("expected non-empty tool list from native provider coverage")
	}

	names := toolNames(out)
	required := map[string]struct{}{}
	for _, e := range goWireBaseTools {
		if e.Required && (e.Enabled == nil || e.Enabled()) {
			required[e.Name] = struct{}{}
		}
	}
	for _, n := range names {
		delete(required, n)
	}
	if len(required) != 0 {
		t.Fatalf("missing required tools from native assembly: %v", required)
	}
}

func TestBuildGoWireToolSpecsRequiredDoesNotFallbackToExport(t *testing.T) {
	t.Setenv("EMBEDDED_SEARCH_TOOLS", "0")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "1")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "0")

	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	denyAllProvider := func(name string) (types.ToolSpec, bool, error) {
		return types.ToolSpec{}, false, nil
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when required tools are missing from native provider")
		}
	}()
	_ = buildGoWireToolSpecsFromExportSpecsWithProvider(base, denyAllProvider)
}

func TestBuildGoWireToolSpecsOptionalDoesNotFallbackToExport(t *testing.T) {
	t.Setenv("EMBEDDED_SEARCH_TOOLS", "0")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "1")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "0")
	t.Setenv("FEATURE_MONITOR_TOOL", "1")

	denyMonitorProvider := func(name string) (types.ToolSpec, bool, error) {
		if name == "Monitor" {
			return types.ToolSpec{}, false, nil
		}
		return nativeSpecFromGoProvider(name)
	}
	out := buildGoWireToolSpecsFromExportSpecsWithProvider([]types.ToolSpec{
		{Name: "Monitor", InputJSONSchema: []byte(`{"type":"object"}`)},
	}, denyMonitorProvider)
	found := false
	for _, s := range out {
		if s.Name == "Monitor" {
			found = true
			break
		}
	}
	if found {
		t.Fatal("optional tool Monitor should not fallback from export when provider misses it")
	}
}

func assertNativeSchemaMatchesEmbedded(t *testing.T, toolName string, build func() types.ToolSpec) {
	t.Helper()
	base, err := ToolSpecsFromEmbeddedToolsAPIJSON()
	if err != nil {
		t.Fatalf("parse embedded export: %v", err)
	}
	var embedded *types.ToolSpec
	for i := range base {
		if base[i].Name == toolName {
			embedded = &base[i]
			break
		}
	}
	if embedded == nil {
		t.Fatalf("embedded %s tool not found", toolName)
	}
	native := build()
	var gotSchema any
	if err := json.Unmarshal(native.InputJSONSchema, &gotSchema); err != nil {
		t.Fatalf("unmarshal native %s schema: %v", toolName, err)
	}
	var wantSchema any
	if err := json.Unmarshal(embedded.InputJSONSchema, &wantSchema); err != nil {
		t.Fatalf("unmarshal embedded %s schema: %v", toolName, err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Fatalf("native %s schema does not match embedded export", toolName)
	}
}

func toolNames(specs []types.ToolSpec) []string {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Name)
	}
	return out
}

