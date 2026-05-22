package toolpool

import (
	"encoding/json"
	"os"
	"strings"

	"goc/commands"
	"goc/modelenv"
	"goc/tstenv"
	"goc/types"
)

// swarmFieldsByTool mirrors TS SWARM_FIELDS_BY_TOOL in src/utils/api.ts.
// Fields stripped from the input schema when agent swarms are not enabled.
var swarmFieldsByTool = map[string][]string{
	"Agent": {"name", "team_name", "mode"},
}

// APIToolDefinition mirrors model-facing tool rows sent in tools[].
// Shape intentionally tracks TS toolToAPISchema output fields we currently
// support in Go.
type APIToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
	DeferLoading         *bool `json:"defer_loading,omitempty"`
	Strict               *bool `json:"strict,omitempty"`
	EagerInputStreaming  *bool `json:"eager_input_streaming,omitempty"`
}

// ToolToAPISchemaOptions mirrors the per-request overlay behavior in TS
// toolToAPISchema (defer_loading + strict gating by model support).
type ToolToAPISchemaOptions struct {
	Model                           string
	DeferLoading                    bool
	StrictToolsEnabled              bool
	FineGrainedToolStreamingEnabled bool
	DisableExperimentalBetas        bool
	APIProvider                     string
	IsFirstPartyAnthropicBaseURL    bool
}

func envTruthySchema(val string) bool {
	v := strings.ToLower(strings.TrimSpace(val))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// DefaultToolToAPISchemaOptionsFromEnv provides TS-like runtime gating inputs
// for tool schema output decisions.
func DefaultToolToAPISchemaOptionsFromEnv() ToolToAPISchemaOptions {
	return ToolToAPISchemaOptions{
		Model:                           modelenv.FirstNonEmpty(),
		StrictToolsEnabled:              true,
		FineGrainedToolStreamingEnabled: envTruthySchema(os.Getenv("CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING")),
		DisableExperimentalBetas:        envTruthySchema(os.Getenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS")),
		APIProvider:                     string(tstenv.GetAPIProvider()),
		IsFirstPartyAnthropicBaseURL:    tstenv.IsFirstPartyAnthropicBaseUrl(),
	}
}

// ToolToAPISchema converts a ToolSpec into API-facing tool schema.
// This is the Go mirror seam for TS toolToAPISchema, so behavior can evolve
// in one place as parity work continues.
func ToolToAPISchema(spec types.ToolSpec, opts ToolToAPISchemaOptions) APIToolDefinition {
	out := APIToolDefinition{
		Name:        spec.Name,
		Description: spec.Description,
		InputSchema: spec.InputJSONSchema,
	}
	// AskUserQuestion description is dynamic (depends on QuestionPreviewFormat).
	// Re-evaluate here so SetQuestionPreviewFormat takes effect without rebuilding the ToolSpec.
	if spec.Name == "AskUserQuestion" {
		out.Description = getAskUserQuestionDescription()
	}
	// Mirror TS filterSwarmFieldsFromSchema: strip name/team_name/mode from Agent
	// tool input_schema when agent swarms are not enabled (src/utils/api.ts lines 166-168).
	if !commands.AgentSwarmsEnabled() {
		out.InputSchema = filterSwarmFields(out.Name, out.InputSchema)
	}
	if opts.DeferLoading {
		v := true
		out.DeferLoading = &v
	}
	if opts.StrictToolsEnabled && spec.Strict != nil && *spec.Strict && modelSupportsStructuredOutputs(opts.Model) {
		v := true
		out.Strict = &v
	}
	if opts.FineGrainedToolStreamingEnabled &&
		strings.EqualFold(strings.TrimSpace(opts.APIProvider), "firstParty") &&
		opts.IsFirstPartyAnthropicBaseURL {
		v := true
		out.EagerInputStreaming = &v
	}
	if opts.DisableExperimentalBetas {
		out.DeferLoading = nil
		out.Strict = nil
		out.EagerInputStreaming = nil
	}
	return out
}

func modelSupportsStructuredOutputs(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}
	// TS checks model capability; Go mirrors that intent conservatively for
	// Anthropic Claude families where strict structured outputs are supported.
	return strings.Contains(m, "claude")
}

// filterSwarmFields removes swarm-related fields from a tool's input_schema
// properties when agent swarms are not enabled. Mirrors TS filterSwarmFieldsFromSchema
// in src/utils/api.ts.
func filterSwarmFields(name string, schema json.RawMessage) json.RawMessage {
	fieldsToRemove := swarmFieldsByTool[name]
	if len(fieldsToRemove) == 0 {
		return schema
	}
	var m map[string]any
	if err := json.Unmarshal(schema, &m); err != nil {
		return schema
	}
	props, ok := m["properties"].(map[string]any)
	if !ok {
		return schema
	}
	for _, f := range fieldsToRemove {
		delete(props, f)
	}
	m["properties"] = props
	result, err := json.Marshal(m)
	if err != nil {
		return schema
	}
	return json.RawMessage(result)
}

