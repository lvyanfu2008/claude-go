package toolpool

import (
	"encoding/json"
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

func TestGoToolWireGeneratesToolSpecs(t *testing.T) {
	setToolWireExportParityEnv(t)
	got := ToolSpecsFromGoWire()
	if len(got) == 0 {
		t.Fatal("ToolSpecsFromGoWire returned no tools")
	}
	
	// Verify we have expected core tools
	toolNames := make(map[string]bool)
	for _, spec := range got {
		toolNames[spec.Name] = true
	}
	
	expectedTools := []string{"Agent", "Read", "Write", "Edit", "Bash"}
	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Fatalf("Expected tool %q not found in output", expected)
		}
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

func TestNativeAgentToolSpecSchemaAndDescription(t *testing.T) {
	t.Setenv("FEATURE_KAIROS", "")
	spec := nativeAgentToolSpec()
	
	// Verify basic fields
	if spec.Name != "Agent" {
		t.Fatalf("expected name 'Agent', got %q", spec.Name)
	}
	
	// Verify description is the full description from AgentToolDescription function
	if spec.Description != AgentToolDescription() {
		t.Fatal("Agent spec description doesn't match AgentToolDescription() function result")
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
	for _, field := range []string{"name", "team_name", "mode", "subagent_type", "model", "isolation"} {
		if _, exists := properties[field]; !exists {
			t.Fatalf("schema missing field: %s", field)
		}
	}
	if _, exists := properties["cwd"]; exists {
		t.Fatal("expected cwd omitted from schema when FEATURE_KAIROS is off (mirrors TS .omit({ cwd: true }))")
	}
	t.Setenv("FEATURE_KAIROS", "1")
	specKairos := nativeAgentToolSpec()
	if err := json.Unmarshal(specKairos.InputJSONSchema, &schema); err != nil {
		t.Fatalf("invalid schema: %v", err)
	}
	properties, ok = schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing properties")
	}
	if _, exists := properties["cwd"]; !exists {
		t.Fatal("expected cwd in schema when FEATURE_KAIROS=1 (mirrors TS fullInputSchema)")
	}
	t.Setenv("FEATURE_KAIROS", "")
	specNoKairos := nativeAgentToolSpec()
	if err := json.Unmarshal(specNoKairos.InputJSONSchema, &schema); err != nil {
		t.Fatalf("invalid schema: %v", err)
	}
	properties, ok = schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing properties")
	}
	if _, exists := properties["cwd"]; exists {
		t.Fatal("expected cwd omitted from schema when FEATURE_KAIROS is off (mirrors TS .omit({ cwd: true }))")
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

	// Intentionally provide no export rows to ensure assembly does not rely on embedded rows for required tools.
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

	denyAllProvider := func(name string) (types.ToolSpec, bool, error) {
		return types.ToolSpec{}, false, nil
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when required tools are missing from native provider")
		}
	}()
	_ = buildGoWireToolSpecsFromExportSpecsWithProvider(nil, denyAllProvider)
}

func TestBuildGoWireToolSpecsFromExportSpecsNativeProviderOverridesExport(t *testing.T) {
	setToolWireExportParityEnv(t)
	
	// Create a fake export spec with incorrect Bash description
	fakeExportSpecs := []types.ToolSpec{
		{Name: "Bash", Description: "fake-export-bash-description"},
	}
	
	out := buildGoWireToolSpecsFromExportSpecsWithProvider(fakeExportSpecs, nativeSpecFromGoProvider)

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

func toolNames(specs []types.ToolSpec) []string {
	out := make([]string, 0, len(specs))
	for _, s := range specs {
		out = append(out, s.Name)
	}
	return out
}