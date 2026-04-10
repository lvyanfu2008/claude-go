package processuserinput

import (
	"context"
	"encoding/json"
	"iter"

	"goc/types"
	"goc/utils"
)

// SetToolJSXOptions mirrors the non-null argument shape of SetToolJSXFn in src/Tool.ts.
// Pass nil to SetToolJSX to mean "clear" (TS passes null).
// JSX is an opaque payload (e.g. serialized UI tree) when crossing from TS.
type SetToolJSXOptions struct {
	JSX                     json.RawMessage `json:"jsx,omitempty"`
	ShouldHidePromptInput   bool            `json:"shouldHidePromptInput"`
	ShouldContinueAnimation *bool           `json:"shouldContinueAnimation,omitempty"`
	ShowSpinner             *bool           `json:"showSpinner,omitempty"`
	IsLocalJSXCommand       *bool           `json:"isLocalJSXCommand,omitempty"`
	IsImmediate             *bool           `json:"isImmediate,omitempty"`
	ClearLocalJSX           *bool           `json:"clearLocalJSX,omitempty"`
}

// ProcessUserInputParams mirrors the arguments of processUserInput() in processUserInput.ts.
// Optional callbacks implement attachment loading, bash/slash commands, hooks, and checkpoints.
type ProcessUserInputParams struct {
	Input json.RawMessage `json:"input"`

	PreExpansionInput *string `json:"preExpansionInput,omitempty"`
	Mode                types.PromptInputMode `json:"mode"`

	PastedContents map[string]utils.PastedContent `json:"pastedContents,omitempty"`
	IdeSelection   *types.IDESelection            `json:"ideSelection,omitempty"`
	Messages       []types.Message              `json:"messages,omitempty"`

	SetUserInputOnProcessing func(prompt string) `json:"-"`

	UUID *string `json:"uuid,omitempty"`

	IsAlreadyProcessing *bool `json:"isAlreadyProcessing,omitempty"`
	QuerySource         types.QuerySource `json:"querySource,omitempty"`

	SkipSlashCommands *bool `json:"skipSlashCommands,omitempty"`
	BridgeOrigin        *bool `json:"bridgeOrigin,omitempty"`
	IsMeta              *bool `json:"isMeta,omitempty"`
	SkipAttachments     *bool `json:"skipAttachments,omitempty"`

	Commands       []types.Command     `json:"commands,omitempty"`
	// SkillListingCommands when set matches TS getSkillListingAttachments merged list (getSkillToolCommands + getMcpSkillCommands); gou-demo uses for API listing. When nil, callers may derive from Commands.
	SkillListingCommands []types.Command `json:"-"`
	PermissionMode types.PermissionMode `json:"permissionMode,omitempty"`

	// RuntimeContext mirrors ProcessUserInputContext serializable slice (types.ToolUseContext etc.).
	// Passed to injected bash/slash/hook implementations for parity with TS context.* / getAppState.
	RuntimeContext *types.ProcessUserInputContextData `json:"runtimeContext,omitempty"`
	// StatePatchAck carries TS apply feedback for previous Go-emitted statePatchBatch.
	StatePatchAck *StatePatchAck `json:"statePatchAck,omitempty"`

	// SetToolJSX mirrors setToolJSX from processUserInput (TS). Nil opts clears (TS null).
	SetToolJSX func(opts *SetToolJSXOptions) `json:"-"`

	// CanUseTool mirrors CanUseToolFn for processSlashCommand (TS). Optional; slash injectors may ignore.
	CanUseTool func(ctx context.Context, toolName string, toolUseID string, input json.RawMessage) (allowed bool, err error) `json:"-"`

	// RequestPrompt mirrors toolUseContext.requestPrompt for executeUserPromptSubmitHooks (TS).
	RequestPrompt func(ctx context.Context, sourceName string, toolInputSummary *string, request json.RawMessage) (response json.RawMessage, err error) `json:"-"`

	// QueryCheckpoint mirrors src/utils/queryProfiler.ts queryCheckpoint (optional no-op).
	QueryCheckpoint func(label string) `json:"-"`

	// BridgeAttachmentMessages is copied from [ProcessUserInputArgs.BridgeAttachmentMessages] when the host preloads @-attachments.
	// When non-nil, [processUserInputBase] uses *BridgeAttachmentMessages instead of calling [GetAttachmentMessages].
	BridgeAttachmentMessages *[]types.Message `json:"-"`

	// GetAttachmentMessages mirrors getAttachmentMessages (async generator in TS); return [] for none.
	GetAttachmentMessages func(ctx context.Context, inputString string, ideSelection *types.IDESelection, priorMessages []types.Message, querySource types.QuerySource) ([]types.Message, error) `json:"-"`

	// ProcessBashCommand receives full params as last arg (mirrors context + setToolJSX on TS processBashCommand).
	ProcessBashCommand func(ctx context.Context, inputString string, precedingBlocks []types.ContentBlockParam, attachmentMessages []types.Message, p *ProcessUserInputParams) (*ProcessUserInputBaseResult, error) `json:"-"`

	// ProcessSlashCommand receives full params as last arg (mirrors context + setToolJSX + canUseTool on TS processSlashCommand).
	ProcessSlashCommand func(ctx context.Context, inputString string, precedingBlocks []types.ContentBlockParam, imageContentBlocks []types.ContentBlockParam, attachmentMessages []types.Message, uuid *string, isAlreadyProcessing *bool, p *ProcessUserInputParams) (*ProcessUserInputBaseResult, error) `json:"-"`

	// ExecuteUserPromptSubmitHooks mirrors executeUserPromptSubmitHooks; receives full params like TS (toolUseContext + requestPrompt via p).
	// Implementations should read p.PermissionMode, p.RuntimeContext, p.RequestPrompt as needed.
	ExecuteUserPromptSubmitHooks func(ctx context.Context, p *ProcessUserInputParams, inputMessage string) ([]types.AggregatedHookResult, error) `json:"-"`

	// ExecuteUserPromptSubmitHooksIter when non-nil is used instead of ExecuteUserPromptSubmitHooks (both set → Iter wins).
	// Each hook result is applied before the next is pulled (TS for-await order on the async generator).
	ExecuteUserPromptSubmitHooksIter func(ctx context.Context, p *ProcessUserInputParams, inputMessage string) iter.Seq2[types.AggregatedHookResult, error] `json:"-"`

	LogEvent func(name string, payload map[string]any) `json:"-"`
}

func (p *ProcessUserInputParams) checkpoint(label string) {
	if p != nil && p.QueryCheckpoint != nil {
		p.QueryCheckpoint(label)
	}
}

func (p *ProcessUserInputParams) logEvent(name string, payload map[string]any) {
	if p != nil && p.LogEvent != nil {
		p.LogEvent(name, payload)
	}
}
