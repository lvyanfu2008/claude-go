// Package types: this file mirrors data shapes from src/Tool.ts (Tool, ToolResult, ToolPermissionContext,
// ValidationResult, CompactProgressEvent, ToolInputJSONSchema). React/Ink-only surfaces (render*,
// mapToolResultToToolResultBlockParam, setToolJSX, etc.) are intentionally omitted — use TS for UI.
package types

import (
	"encoding/json"
)

// ToolInputJSONSchema mirrors src/Tool.ts ToolInputJSONSchema (JSON Schema object for MCP / passthrough tools).
type ToolInputJSONSchema struct {
	Type       string                     `json:"type"` // "object"
	Properties map[string]json.RawMessage `json:"properties,omitempty"`
}

// RawToolInputJSONSchema is a type alias for a full JSON Schema blob when the structured ToolInputJSONSchema is not used.
type RawToolInputJSONSchema = json.RawMessage

// ValidationResult mirrors src/Tool.ts ValidationResult.
type ValidationResult struct {
	Result    bool   `json:"result"`
	Message   string `json:"message,omitempty"`
	ErrorCode int    `json:"errorCode,omitempty"`
}

// ValidationOK is the successful branch `{ result: true }`.
func ValidationOK() ValidationResult {
	return ValidationResult{Result: true}
}

// ValidationFail is the failure branch with message + errorCode.
func ValidationFail(message string, errorCode int) ValidationResult {
	return ValidationResult{Result: false, Message: message, ErrorCode: errorCode}
}

// CompactProgressEvent mirrors src/Tool.ts CompactProgressEvent (compact / hooks progress).
type CompactProgressEvent struct {
	Type     string  `json:"type"`               // hooks_start | compact_start | compact_end
	HookType *string `json:"hookType,omitempty"` // pre_compact | post_compact | session_start
}

// ToolPermissionContextData mirrors serializable fields of src/Tool.ts ToolPermissionContext.
// Map-valued rules match ToolPermissionRulesBySource (opaque JSON objects from settings).
type ToolPermissionContextData struct {
	Mode PermissionMode `json:"mode"`

	AdditionalWorkingDirectories json.RawMessage `json:"additionalWorkingDirectories,omitempty"`
	AlwaysAllowRules             json.RawMessage `json:"alwaysAllowRules,omitempty"`
	AlwaysDenyRules              json.RawMessage `json:"alwaysDenyRules,omitempty"`
	AlwaysAskRules               json.RawMessage `json:"alwaysAskRules,omitempty"`

	IsBypassPermissionsModeAvailable bool            `json:"isBypassPermissionsModeAvailable"`
	IsAutoModeAvailable              *bool           `json:"isAutoModeAvailable,omitempty"`
	StrippedDangerousRules           json.RawMessage `json:"strippedDangerousRules,omitempty"`

	ShouldAvoidPermissionPrompts     *bool           `json:"shouldAvoidPermissionPrompts,omitempty"`
	AwaitAutomatedChecksBeforeDialog *bool           `json:"awaitAutomatedChecksBeforeDialog,omitempty"`
	PrePlanMode                      *PermissionMode `json:"prePlanMode,omitempty"`
}

// EmptyToolPermissionContextData matches src/Tool.ts getEmptyToolPermissionContext().
func EmptyToolPermissionContextData() ToolPermissionContextData {
	empty := json.RawMessage(`{}`)
	return ToolPermissionContextData{
		Mode:                             PermissionDefault,
		AdditionalWorkingDirectories:     empty,
		AlwaysAllowRules:                 empty,
		AlwaysDenyRules:                  empty,
		AlwaysAskRules:                   empty,
		IsBypassPermissionsModeAvailable: false,
	}
}

// jsonEmptyObject is the JSON `{}` blob for normalized Map-shaped permission fields.
var jsonEmptyObject = json.RawMessage(`{}`)

// NormalizeToolPermissionContextData fills omitted Map-shaped fields after json.Unmarshal
// (TS getEmptyToolPermissionContext uses empty Maps for these keys).
func NormalizeToolPermissionContextData(d *ToolPermissionContextData) {
	if d == nil {
		return
	}
	if len(d.AdditionalWorkingDirectories) == 0 {
		d.AdditionalWorkingDirectories = jsonEmptyObject
	}
	if len(d.AlwaysAllowRules) == 0 {
		d.AlwaysAllowRules = jsonEmptyObject
	}
	if len(d.AlwaysDenyRules) == 0 {
		d.AlwaysDenyRules = jsonEmptyObject
	}
	if len(d.AlwaysAskRules) == 0 {
		d.AlwaysAskRules = jsonEmptyObject
	}
}

// InterruptBehavior mirrors Tool.interruptBehavior() in src/Tool.ts.
type InterruptBehavior string

const (
	InterruptCancel InterruptBehavior = "cancel"
	InterruptBlock  InterruptBehavior = "block"
)

// SearchOrReadCollapse mirrors return type of Tool.isSearchOrReadCommand? (src/Tool.ts).
type SearchOrReadCollapse struct {
	IsSearch bool `json:"isSearch"`
	IsRead   bool `json:"isRead"`
	IsList   bool `json:"isList,omitempty"`
}

// MCPInfo mirrors Tool.mcpInfo (src/Tool.ts).
type MCPInfo struct {
	ServerName string `json:"serverName"`
	ToolName   string `json:"toolName"`
}

// ToolMCPMeta mirrors ToolResult.mcpMeta (structuredContent / _meta for MCP passthrough).
type ToolMCPMeta struct {
	Meta              json.RawMessage `json:"_meta,omitempty"`
	StructuredContent json.RawMessage `json:"structuredContent,omitempty"`
}

// ToolRunResult mirrors ToolResult<T> from src/Tool.ts (serializable subset; contextModifier omitted).
type ToolRunResult struct {
	Data        json.RawMessage `json:"data"`
	NewMessages []Message       `json:"newMessages,omitempty"`
	MCPMeta     *ToolMCPMeta    `json:"mcpMeta,omitempty"`
}

// ToolSpec holds static registration metadata for a tool — the data counterpart of Tool<> without
// generics, Zod, or React. Per-tool JSON schemas are carried as RawMessage (API / buildTool output).
type ToolSpec struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases,omitempty"`
	// Description: model-facing tool description (Anthropic API / tools_api.json export).
	Description string `json:"description,omitempty"`
	// SearchHint: 3–10 words for ToolSearch keyword routing (src/Tool.ts).
	SearchHint string `json:"searchHint,omitempty"`

	InputSchema     json.RawMessage `json:"inputSchema,omitempty"`
	InputJSONSchema json.RawMessage `json:"inputJSONSchema,omitempty"`
	OutputSchema    json.RawMessage `json:"outputSchema,omitempty"`

	MaxResultSizeChars int64 `json:"maxResultSizeChars"`

	Strict      *bool    `json:"strict,omitempty"`
	ShouldDefer *bool    `json:"shouldDefer,omitempty"`
	AlwaysLoad  *bool    `json:"alwaysLoad,omitempty"`
	IsMcp       *bool    `json:"isMcp,omitempty"`
	IsLsp       *bool    `json:"isLsp,omitempty"`
	MCPInfo     *MCPInfo `json:"mcpInfo,omitempty"`

	InterruptBehavior *InterruptBehavior `json:"interruptBehavior,omitempty"`
}

// ToolDefaults matches TOOL_DEFAULTS in src/Tool.ts (buildTool fill-ins; fail-closed where noted).
type ToolDefaults struct {
	ConcurrencySafe bool // default false
	ReadOnly        bool // default false
	Destructive     bool // default false
	UserFacingName  string
}

// DefaultToolDefaults returns the same defaults as src/Tool.ts TOOL_DEFAULTS (userFacingName filled from name at merge).
func DefaultToolDefaults() ToolDefaults {
	return ToolDefaults{
		ConcurrencySafe: false,
		ReadOnly:        false,
		Destructive:     false,
		UserFacingName:  "",
	}
}
