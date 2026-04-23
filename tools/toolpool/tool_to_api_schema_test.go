package toolpool

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestToolToAPISchemaMapsCoreFields(t *testing.T) {
	strict := true
	in := types.ToolSpec{
		Name:            "Read",
		Description:     "Read file",
		InputJSONSchema: json.RawMessage(`{"type":"object"}`),
		Strict:          &strict,
	}
	out := ToolToAPISchema(in, ToolToAPISchemaOptions{
		Model:              "claude-sonnet-4-20250514",
		DeferLoading:       true,
		StrictToolsEnabled: true,
		FineGrainedToolStreamingEnabled: true,
		APIProvider: "firstParty",
		IsFirstPartyAnthropicBaseURL: true,
	})
	if out.Name != in.Name {
		t.Fatalf("name mismatch: got=%q want=%q", out.Name, in.Name)
	}
	if out.Description != in.Description {
		t.Fatalf("description mismatch: got=%q want=%q", out.Description, in.Description)
	}
	if string(out.InputSchema) != string(in.InputJSONSchema) {
		t.Fatalf("input schema mismatch: got=%s want=%s", string(out.InputSchema), string(in.InputJSONSchema))
	}
	if out.DeferLoading == nil || *out.DeferLoading != true {
		t.Fatalf("defer_loading mismatch")
	}
	if out.Strict == nil || *out.Strict != strict {
		t.Fatalf("strict mismatch")
	}
	if out.EagerInputStreaming == nil || *out.EagerInputStreaming != true {
		t.Fatalf("eager_input_streaming mismatch")
	}
}

func TestToolToAPISchemaDoesNotEmitStrictWithoutGate(t *testing.T) {
	strict := true
	in := types.ToolSpec{
		Name:            "Read",
		Description:     "Read file",
		InputJSONSchema: json.RawMessage(`{"type":"object"}`),
		Strict:          &strict,
	}
	out := ToolToAPISchema(in, ToolToAPISchemaOptions{
		Model:              "claude-sonnet-4-20250514",
		StrictToolsEnabled: false,
	})
	if out.Strict != nil {
		t.Fatalf("strict should be omitted when strict gate is disabled")
	}
}

func TestToolToAPISchemaDoesNotEmitStrictForNonClaudeModel(t *testing.T) {
	strict := true
	in := types.ToolSpec{
		Name:            "Read",
		Description:     "Read file",
		InputJSONSchema: json.RawMessage(`{"type":"object"}`),
		Strict:          &strict,
	}
	out := ToolToAPISchema(in, ToolToAPISchemaOptions{
		Model:              "gpt-4.1",
		StrictToolsEnabled: true,
	})
	if out.Strict != nil {
		t.Fatalf("strict should be omitted for non-Claude model")
	}
}

func TestToolToAPISchemaStripsExperimentalFieldsWhenDisabled(t *testing.T) {
	strict := true
	in := types.ToolSpec{
		Name:            "Read",
		Description:     "Read file",
		InputJSONSchema: json.RawMessage(`{"type":"object"}`),
		Strict:          &strict,
	}
	out := ToolToAPISchema(in, ToolToAPISchemaOptions{
		Model:                           "claude-sonnet-4-20250514",
		DeferLoading:                    true,
		StrictToolsEnabled:              true,
		FineGrainedToolStreamingEnabled: true,
		DisableExperimentalBetas:        true,
		APIProvider:                     "firstParty",
		IsFirstPartyAnthropicBaseURL:    true,
	})
	if out.DeferLoading != nil || out.Strict != nil || out.EagerInputStreaming != nil {
		t.Fatalf("experimental fields should be stripped when DisableExperimentalBetas is enabled")
	}
}

func TestMarshalToolsAPIDocumentDefinitionsIncludesOptionalFlags(t *testing.T) {
	deferLoading := true
	strict := true
	raw, err := MarshalToolsAPIDocumentDefinitionsWithOptions([]types.ToolSpec{
		{
			Name:            "ToolA",
			Description:     "desc",
			InputJSONSchema: json.RawMessage(`{"type":"object"}`),
			ShouldDefer:     &deferLoading,
			Strict:          &strict,
		},
	}, ToolToAPISchemaOptions{
		Model:              "claude-sonnet-4-20250514",
		StrictToolsEnabled: true,
	})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 tool row, got %d", len(out))
	}
	if v, ok := out[0]["defer_loading"].(bool); !ok || !v {
		t.Fatalf("missing or false defer_loading")
	}
	if v, ok := out[0]["strict"].(bool); !ok || !v {
		t.Fatalf("missing or false strict")
	}
	if v, ok := out[0]["eager_input_streaming"].(bool); ok && v {
		t.Fatalf("eager_input_streaming should not be enabled by default marshal options")
	}
}

func TestToolToAPISchemaEagerStreamingRequiresProviderAndBaseURL(t *testing.T) {
	in := types.ToolSpec{
		Name:            "Read",
		Description:     "Read file",
		InputJSONSchema: json.RawMessage(`{"type":"object"}`),
	}

	noProvider := ToolToAPISchema(in, ToolToAPISchemaOptions{
		FineGrainedToolStreamingEnabled: true,
		APIProvider:                     "bedrock",
		IsFirstPartyAnthropicBaseURL:    true,
	})
	if noProvider.EagerInputStreaming != nil {
		t.Fatalf("eager_input_streaming should be omitted for non-firstParty provider")
	}

	noBaseURL := ToolToAPISchema(in, ToolToAPISchemaOptions{
		FineGrainedToolStreamingEnabled: true,
		APIProvider:                     "firstParty",
		IsFirstPartyAnthropicBaseURL:    false,
	})
	if noBaseURL.EagerInputStreaming != nil {
		t.Fatalf("eager_input_streaming should be omitted when not first-party anthropic base URL")
	}
}

func TestDefaultToolToAPISchemaOptionsFromEnv(t *testing.T) {
	t.Setenv("CCB_ENGINE_MODEL", "claude-sonnet-4")
	t.Setenv("ANTHROPIC_MODEL", "claude-sonnet-4")
	t.Setenv("CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING", "1")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "0")
	t.Setenv("CLAUDE_CODE_USE_OPENAI", "0")
	t.Setenv("CLAUDE_CODE_USE_BEDROCK", "0")
	t.Setenv("CLAUDE_CODE_USE_VERTEX", "0")
	t.Setenv("CLAUDE_CODE_USE_FOUNDRY", "0")
	t.Setenv("ANTHROPIC_BASE_URL", "")

	opts := DefaultToolToAPISchemaOptionsFromEnv()
	if opts.Model != "claude-sonnet-4" {
		t.Fatalf("expected model from env, got %q", opts.Model)
	}
	if !opts.StrictToolsEnabled {
		t.Fatalf("strict tools should be enabled in default options")
	}
	if !opts.FineGrainedToolStreamingEnabled {
		t.Fatalf("fine-grained streaming should be enabled from env")
	}
	if opts.DisableExperimentalBetas {
		t.Fatalf("disable experimental betas should be false")
	}
	if opts.APIProvider != "firstParty" {
		t.Fatalf("expected firstParty provider, got %q", opts.APIProvider)
	}
	if !opts.IsFirstPartyAnthropicBaseURL {
		t.Fatalf("expected first-party anthropic base url when unset")
	}
}

