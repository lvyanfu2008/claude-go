package toolpool

import (
	"encoding/json"
	"os"
	"strings"

	"goc/modelenv"
	"goc/tstenv"
	"goc/types"
)

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

