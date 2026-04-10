// ToolUseContext (Go) mirrors src/Tool.ts ToolUseContext: data fields + JSON tags for IPC/tests.
// Function/handle fields (abortController, getAppState, setToolJSX, requestPrompt, …) are not representable
// here — see package tool for execution-facing types. Registration metadata: tool.go (ToolSpec, ToolRunResult).
package types

import "encoding/json"

// QueryChainTracking mirrors src/Tool.ts QueryChainTracking.
type QueryChainTracking struct {
	ChainID string `json:"chainId"`
	Depth   int    `json:"depth"`
}

// FileReadingLimits mirrors tool context fileReadingLimits.
type FileReadingLimits struct {
	MaxTokens    *int `json:"maxTokens,omitempty"`
	MaxSizeBytes *int `json:"maxSizeBytes,omitempty"`
}

// GlobLimits mirrors tool context globLimits.
type GlobLimits struct {
	MaxResults *int `json:"maxResults,omitempty"`
}

// ToolDecision mirrors entries in toolDecisions map (src/Tool.ts).
type ToolDecision struct {
	Source    string `json:"source"`
	Decision  string `json:"decision"` // accept | reject
	Timestamp int64  `json:"timestamp"`
}

// ToolUseContextOptionsData mirrors ToolUseContext.options (src/Tool.ts) — data fields only.
// refreshTools and similar callbacks are omitted.
type ToolUseContextOptionsData struct {
	Commands                     []Command       `json:"commands"`
	Debug                        bool            `json:"debug"`
	MainLoopModel                string          `json:"mainLoopModel"`
	Tools                        json.RawMessage `json:"tools,omitempty"`
	Verbose                      bool            `json:"verbose"`
	ThinkingConfig               ThinkingConfig  `json:"thinkingConfig"`
	MCPClients                   json.RawMessage `json:"mcpClients,omitempty"`
	MCPResources                 json.RawMessage `json:"mcpResources,omitempty"`
	IsNonInteractiveSession      bool            `json:"isNonInteractiveSession"`
	AgentDefinitions             json.RawMessage `json:"agentDefinitions,omitempty"`
	MaxBudgetUsd                 *float64        `json:"maxBudgetUsd,omitempty"`
	CustomSystemPrompt           *string         `json:"customSystemPrompt,omitempty"`
	AppendSystemPrompt           *string         `json:"appendSystemPrompt,omitempty"`
	QuerySource                  QuerySource     `json:"querySource,omitempty"`
	HasUserPromptSubmitHooks     *bool           `json:"hasUserPromptSubmitHooks,omitempty"`
	HooksStateSnapshot           json.RawMessage `json:"hooksStateSnapshot,omitempty"`
	UserPromptSubmitHookCommands *[]string       `json:"userPromptSubmitHookCommands,omitempty"`

	// LocalJSXCommandContext extends options (src/types/command.ts LocalJSXCommandContext).
	DynamicMcpConfig      json.RawMessage `json:"dynamicMcpConfig,omitempty"`
	IdeInstallationStatus *string         `json:"ideInstallationStatus,omitempty"`
	Theme                 *string         `json:"theme,omitempty"`
}

// ToolUseContext is the serializable projection of src/Tool.ts ToolUseContext.
//
// TS fields intentionally not on this struct (callbacks / handles / non-JSON state):
//
//	abortController, readFileState, getAppState, setAppState, setAppStateForTasks,
//	handleElicitation, setToolJSX, addNotification, appendSystemMessage, sendOSNotification,
//	setInProgressToolUseIDs, setHasInterruptibleToolInProgress, setResponseLength,
//	pushApiMetricsEntry, setStreamMode, onCompactProgress, setSDKStatus, openMessageSelector,
//	updateFileHistoryState, updateAttributionState, setConversationId, requestPrompt.
//
// Sets in TS (nestedMemoryAttachmentTriggers, …) are []string for JSON.
type ToolUseContext struct {
	Options ToolUseContextOptionsData `json:"options"`

	Messages []Message `json:"messages"`

	FileReadingLimits *FileReadingLimits `json:"fileReadingLimits,omitempty"`
	GlobLimits        *GlobLimits        `json:"globLimits,omitempty"`

	ToolDecisions map[string]ToolDecision `json:"toolDecisions,omitempty"`

	QueryTracking *QueryChainTracking `json:"queryTracking,omitempty"`

	AgentID   *string `json:"agentId,omitempty"`
	AgentType *string `json:"agentType,omitempty"`
	ToolUseID *string `json:"toolUseId,omitempty"`

	RequireCanUseTool                  *bool   `json:"requireCanUseTool,omitempty"`
	PreserveToolUseResults             *bool   `json:"preserveToolUseResults,omitempty"`
	CriticalSystemReminderExperimental *string `json:"criticalSystemReminder_EXPERIMENTAL,omitempty"`

	// Opaque runtime state blobs (optional snapshots for tests / IPC).
	LocalDenialTracking     json.RawMessage `json:"localDenialTracking,omitempty"`
	ContentReplacementState json.RawMessage `json:"contentReplacementState,omitempty"`
	RenderedSystemPrompt    json.RawMessage `json:"renderedSystemPrompt,omitempty"`

	NestedMemoryAttachmentTriggers []string `json:"nestedMemoryAttachmentTriggers,omitempty"`
	LoadedNestedMemoryPaths        []string `json:"loadedNestedMemoryPaths,omitempty"`
	DynamicSkillDirTriggers        []string `json:"dynamicSkillDirTriggers,omitempty"`
	DiscoveredSkillNames           []string `json:"discoveredSkillNames,omitempty"`
	UserModified                   *bool    `json:"userModified,omitempty"`

	ConversationID *string `json:"conversationId,omitempty"`
}

// ToolUseContextData is an alias for the previous Go name (same type as ToolUseContext).
type ToolUseContextData = ToolUseContext

// ProcessUserInputContextData mirrors ProcessUserInputContext in
// src/conversation-runtime/processUserInput/processUserInput.ts
// (ToolUseContext & LocalJSXCommandContext & { loadBashModeProgress? }).
// Callback fields and loadBashModeProgress are not serialized.
//
// ToolPermissionContext is optional: when present (e.g. IPC from TS), additionalWorkingDirectories
// feed CLAUDE.md discovery together with CLAUDE_CODE_EXTRA_CLAUDE_MD_ROOTS / GOU_DEMO_EXTRA_CLAUDE_MD_ROOTS.
type ProcessUserInputContextData struct {
	ToolUseContext
	ToolPermissionContext *ToolPermissionContextData `json:"toolPermissionContext,omitempty"`
}
