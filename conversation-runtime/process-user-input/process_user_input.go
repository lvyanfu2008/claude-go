// Package processuserinput mirrors src/conversation-runtime/processUserInput/processUserInput.ts.
// Path rule: src/… in TS ↔ goc/… in Go; module goc (see goc/go.mod).
//
// Entry points: [ProcessUserInput], [ProcessTextPrompt] (process_text_prompt.go).
// Data shapes: [ProcessUserInputParams], [ProcessUserInputBaseResult]; [ProcessUserInputArgs] in args.go.
//
// TS parity highlights vs processUserInput.ts / processTextPrompt.ts:
//   - [ProcessTextPrompt] emits tengu_input_prompt via [ProcessUserInputParams.LogEvent] (is_negative / is_keep_going; [MatchesNegativeKeyword] / [MatchesKeepGoingKeyword]).
//   - CLAUDE_DEBUG_PROCESS_USER_INPUT → stderr lines [processUserInput:IN|AFTER_BASE|OUT] JSON (mirrors logProcessUserInputDebug stages).
//   - [commandsForFind] uses context.options.commands when [ProcessUserInputParams.Commands] is empty (bridge-safe slash lookup).
//   - Non-nil [ProcessBashCommand] / [ProcessSlashCommand] run before bashprepare/slashprepare execution_request paths (mirrors dynamic import of processBashCommand / processSlashCommand).
//   - Prompt path does not emit attachments_plan / hooks_plan / query execution_request (attachments merge into [ProcessTextPrompt]; hooks run in [ProcessUserInput] via [ExecuteUserPromptSubmitHooks] or [ExecuteUserPromptSubmitHooksIter]).
//
// Host query streaming: after [ProcessUserInput] returns [ProcessUserInputBaseResult] with ShouldQuery, callers that
// use [goc/conversation-runtime/query.Query] may call [ApplyQueryHostEnvGates] and [WireToolexecutionFromProcessUserInput]
// so merged settings env (GOU_QUERY_STREAMING_PARITY / GOU_DEMO_STREAMING_TOOL_EXECUTION) and [ProcessUserInputParams.CanUseTool]
// feed [QueryParams.StreamingParity] and [toolexecution.ExecutionDeps] (see gou-demo streaming parity path).
//
// Inject bash/slash/attachment/hook handlers via [ProcessUserInputParams]; nil bash/slash handlers use bashprepare/slashprepare execution_request stubs.
// TS parity: [ProcessBashCommand] / [ProcessSlashCommand] receive *ProcessUserInputParams for context,
// [SetToolJSX], [CanUseTool]; [ExecuteUserPromptSubmitHooks] receives *ProcessUserInputParams for
// [RuntimeContext] and [RequestPrompt] (mirrors toolUseContext + requestPrompt in hooks.ts).
package processuserinput

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	bashprepare "goc/bash-prepare"
	slashprepare "goc/slash-prepare"
	"goc/types"
	"goc/utils"
)

// applyUserPromptSubmitHookItem applies one hook result to result. Returns (early, debugPath, err):
// when early != nil, the caller must return early from ProcessUserInput; debugPath is for CLAUDE_DEBUG logging.
func applyUserPromptSubmitHookItem(p *ProcessUserInputParams, result *ProcessUserInputBaseResult, hookResult types.AggregatedHookResult) (*ProcessUserInputBaseResult, string, error) {
	if len(hookResult.Message) > 0 && messageJSONType(hookResult.Message) == string(types.MessageTypeProgress) {
		return nil, "", nil
	}
	if hookResult.BlockingError != nil {
		blockingText := getUserPromptSubmitHookBlockingMessage(*hookResult.BlockingError)
		orig := originalPromptForHooks(p)
		sys := newSystemInformationalMessage(blockingText+"\n\nOriginal prompt: "+orig, "warning")
		out := &ProcessUserInputBaseResult{
			Messages:     []types.Message{sys},
			ShouldQuery:  false,
			AllowedTools: result.AllowedTools,
		}
		return out, "hook-blocking-error", nil
	}
	if boolVal(hookResult.PreventContinuation) {
		msg := "Operation stopped by hook"
		if hookResult.StopReason != nil && *hookResult.StopReason != "" {
			msg = "Operation stopped by hook: " + *hookResult.StopReason
		}
		um, err := newUserMessage(msg, nil, nil, nil)
		if err != nil {
			return nil, "", err
		}
		result.Messages = append(result.Messages, um)
		result.ShouldQuery = false
		return result, "hook-prevent-continuation", nil
	}
	if len(hookResult.AdditionalContexts) > 0 {
		parts := make([]string, len(hookResult.AdditionalContexts))
		for i, s := range hookResult.AdditionalContexts {
			parts[i] = applyTruncation(s)
		}
		am, err := newHookAdditionalContextAttachment(parts, "hook-"+randomUUID(), "UserPromptSubmit")
		if err != nil {
			return nil, "", err
		}
		result.Messages = append(result.Messages, am)
	}
	if len(hookResult.Message) > 0 {
		if err := mergeHookSuccessMessage(&result.Messages, hookResult.Message); err != nil {
			return nil, "", err
		}
	}
	return nil, "", nil
}

// ExecutionRequest is a TS-executed action produced by Go (Phase 1: parse/decide only; no side effects in Go).
// Exported so the CLI can wrap it in a stdout union envelope.
type ExecutionRequest struct {
	Kind         string `json:"kind"` // e.g. "bash"
	Input        string `json:"input,omitempty"`
	Command      string `json:"command,omitempty"`
	RejectReason string `json:"rejectReason,omitempty"`
	CommandName  string `json:"commandName,omitempty"`
	Args         string `json:"args,omitempty"`
	IsMcp        bool   `json:"isMcp,omitempty"`
	HooksExecutionPlan *HooksExecutionPlan `json:"hooksExecutionPlan,omitempty"`
}

type HooksExecutionPlan struct {
	Strategy  string   `json:"strategy"`
	Commands  []string `json:"commands,omitempty"`
	Matchers  []string `json:"matchers,omitempty"`
	HookCount int      `json:"hookCount,omitempty"`
	TimeoutMs int      `json:"timeoutMs,omitempty"`
	PredictedReducerInput *HooksReducerInput `json:"predictedReducerInput,omitempty"`
}

// HookExecResult is one UserPromptSubmit hook subprocess outcome for the TS bridge.
// Stdout/stderr are truncated like analytics so execution_request JSON stays bounded.
type HookExecResult struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exitCode,omitempty"`
	Error    string `json:"error,omitempty"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
}

type HooksReducerInput struct {
	Blocked                 bool             `json:"blocked,omitempty"`
	PreventContinuation     bool             `json:"preventContinuation,omitempty"`
	AdditionalContextsCount int              `json:"additionalContextsCount,omitempty"`
	MessageCount            int              `json:"messageCount,omitempty"`
	HookExecResults         []HookExecResult `json:"hookExecResults,omitempty"`
}

// ProcessUserInputBaseResult mirrors processUserInput.ts ProcessUserInputBaseResult.
type ProcessUserInputBaseResult struct {
	// Messages matches TS: UserMessage | AssistantMessage | AttachmentMessage | SystemMessage | ProgressMessage.
	Messages []types.Message `json:"messages"`
	// ShouldQuery when false skips the model query turn (commands, hooks, errors).
	ShouldQuery bool `json:"shouldQuery"`
	// Execution when set requests TS to execute a side-effecting action (e.g. bash) exactly once.
	Execution *ExecutionRequest `json:"execution,omitempty"`
	// ExecutionSequence when non-empty is an ordered list of TS-executed steps (stdout uses "actions").
	// When set, takes precedence over Execution for CLI emission; leave Execution nil for multi-step.
	ExecutionSequence []ExecutionRequest `json:"executionSequence,omitempty"`
	AllowedTools []string `json:"allowedTools,omitempty"`
	Model        string   `json:"model,omitempty"`
	Effort       *utils.EffortValue `json:"effort,omitempty"`
	// ResultText is output for non-interactive / -p mode when set instead of an empty string.
	ResultText string `json:"resultText,omitempty"`
	// NextInput prefills or chains input after a command completes (e.g. /discover).
	NextInput string `json:"nextInput,omitempty"`
	// SubmitNextInput when true submits NextInput automatically after the command completes.
	SubmitNextInput bool `json:"submitNextInput,omitempty"`
	StatePatchBatch *StatePatchBatch `json:"statePatchBatch,omitempty"`
	HooksReducerInput *HooksReducerInput `json:"hooksReducerInput,omitempty"`
}

const userImageRejectMessage = "当前版本不支持在消息中加入图片（粘贴或图片块）。请去掉图片后重试。"

// commandsForFind mirrors TS findCommand(..., context.options.commands); prefers [ProcessUserInputParams.Commands], else RuntimeContext.Options.Commands.
func commandsForFind(p *ProcessUserInputParams) []types.Command {
	if p == nil {
		return nil
	}
	if len(p.Commands) > 0 {
		return p.Commands
	}
	if p.RuntimeContext != nil {
		return p.RuntimeContext.Options.Commands
	}
	return nil
}

// precedingBlocksForNormalized mirrors processUserInputBase precedingInputBlocks vs last text block (processUserInput.ts).
func precedingBlocksForNormalized(isString bool, normalizedBlocks []types.ContentBlockParam) []types.ContentBlockParam {
	if isString || len(normalizedBlocks) == 0 {
		return nil
	}
	last := normalizedBlocks[len(normalizedBlocks)-1]
	if last.Type == "text" {
		if len(normalizedBlocks) <= 1 {
			return nil
		}
		return normalizedBlocks[:len(normalizedBlocks)-1]
	}
	return normalizedBlocks
}

func textPromptLog(p *ProcessUserInputParams) func(string, map[string]any) {
	if p == nil {
		return nil
	}
	return func(n string, pl map[string]any) { p.logEvent(n, pl) }
}

// originalPromptForHooks mirrors TS blocking message `${input}` for string JSON input;
// for non-string JSON (blocks), returns raw JSON text (closer than "[object Object]").
func originalPromptForHooks(p *ProcessUserInputParams) string {
	if p == nil {
		return ""
	}
	raw := json.RawMessage(strings.TrimSpace(string(p.Input)))
	if len(raw) > 0 && raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return s
		}
	}
	return string(raw)
}

func boolVal(p *bool) bool {
	return p != nil && *p
}

func envTruthy(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// ProcessUserInput mirrors src/conversation-runtime/processUserInput/processUserInput.ts processUserInput.
func ProcessUserInput(ctx context.Context, p *ProcessUserInputParams) (*ProcessUserInputBaseResult, error) {
	if p == nil {
		return nil, errors.New("processuserinput: nil ProcessUserInputParams")
	}
	text, blocks, isStr, err := parseInput(p.Input)
	if err != nil {
		return nil, err
	}

	debugProcessUserInput("IN", buildProcessUserInputDebugInPayload(p, isStr, len(blocks), text))

	if p.Mode == types.PromptInputModePrompt && isStr && !boolVal(p.IsMeta) && p.SetUserInputOnProcessing != nil {
		p.SetUserInputOnProcessing(text)
	}

	p.checkpoint("query_process_user_input_base_start")
	result, err := processUserInputBase(ctx, p, text, blocks, isStr)
	if err != nil {
		return nil, err
	}
	p.checkpoint("query_process_user_input_base_end")

	debugProcessUserInput("AFTER_BASE", buildProcessUserInputDebugResultPayload(result))

	if !result.ShouldQuery {
		debugProcessUserInput("OUT", withDebugPath("no-query", buildProcessUserInputDebugResultPayload(result)))
		return result, nil
	}

	p.checkpoint("query_hooks_start")
	inputMessage := ""
	if ct := getContentText(text, blocks); ct != nil {
		inputMessage = *ct
	}

	if p.ExecuteUserPromptSubmitHooksIter != nil {
		for hookResult, err := range p.ExecuteUserPromptSubmitHooksIter(ctx, p, inputMessage) {
			if err != nil {
				return nil, err
			}
			early, path, err := applyUserPromptSubmitHookItem(p, result, hookResult)
			if err != nil {
				return nil, err
			}
			if early != nil {
				p.checkpoint("query_hooks_end")
				debugProcessUserInput("OUT", withDebugPath(path, buildProcessUserInputDebugResultPayload(early)))
				return early, nil
			}
		}
	} else if p.ExecuteUserPromptSubmitHooks != nil {
		hookResults, err := p.ExecuteUserPromptSubmitHooks(ctx, p, inputMessage)
		if err != nil {
			return nil, err
		}
		for _, hookResult := range hookResults {
			early, path, err := applyUserPromptSubmitHookItem(p, result, hookResult)
			if err != nil {
				return nil, err
			}
			if early != nil {
				p.checkpoint("query_hooks_end")
				debugProcessUserInput("OUT", withDebugPath(path, buildProcessUserInputDebugResultPayload(early)))
				return early, nil
			}
		}
	}
	p.checkpoint("query_hooks_end")
	debugProcessUserInput("OUT", withDebugPath("after-hooks", buildProcessUserInputDebugResultPayload(result)))
	return result, nil
}

func processUserInputBase(
	ctx context.Context,
	p *ProcessUserInputParams,
	text string,
	blocks []types.ContentBlockParam,
	isString bool,
) (*ProcessUserInputBaseResult, error) {
	var err error
	if blocksHaveImage(blocks) {
		p.logEvent("tengu_multimodal_images_disabled", map[string]any{"reason": "user_images_rejected", "source": "array_input"})
		return rejectImageResult(), nil
	}
	if pastedHasImagePaste(p.PastedContents) {
		p.logEvent("tengu_multimodal_images_disabled", map[string]any{"reason": "user_images_rejected", "source": "paste"})
		return rejectImageResult(), nil
	}

	var inputString *string
	var normalizedStr string
	var normalizedBlocks []types.ContentBlockParam

	if isString {
		inputString = &text
		normalizedStr = text
	} else if len(blocks) > 0 {
		last := blocks[len(blocks)-1]
		normalizedBlocks = blocks
		if last.Type == "text" {
			s := last.Text
			inputString = &s
		}
	}

	if inputString == nil && p.Mode != types.PromptInputModePrompt {
		return nil, fmt.Errorf("processUserInput: mode %q requires a string input", p.Mode)
	}

	imageMetadataTexts := []string{}

	skipSlash := boolVal(p.SkipSlashCommands)
	bridge := boolVal(p.BridgeOrigin)
	effectiveSkipSlash := skipSlash

	var inputStr string
	if inputString != nil {
		inputStr = *inputString
	}

	if bridge && inputString != nil && strings.HasPrefix(inputStr, "/") {
		parsed := ParseSlashCommand(inputStr)
		if parsed != nil {
			cmd := FindCommand(parsed.CommandName, commandsForFind(p))
			if cmd != nil {
				if IsBridgeSafeCommand(*cmd) {
					effectiveSkipSlash = false
				} else {
					msg := fmt.Sprintf("/%s isn't available over Remote Control.", GetCommandName(*cmd))
					u1, err := newUserMessage(inputStr, p.UUID, nil, nil)
					if err != nil {
						return nil, err
					}
					u2 := newCommandInputMessage("<local-command-stdout>" + msg + "</local-command-stdout>")
					return &ProcessUserInputBaseResult{
						Messages:    []types.Message{u1, u2},
						ShouldQuery: false,
						ResultText:  msg,
					}, nil
				}
			}
		}
	}

	skipAtt := boolVal(p.SkipAttachments)
	shouldExtract := !skipAtt && inputString != nil &&
		(p.Mode != types.PromptInputModePrompt || effectiveSkipSlash || !strings.HasPrefix(*inputString, "/"))

	var attachmentMessages []types.Message
	if shouldExtract {
		p.checkpoint("query_attachment_loading_start")
		if p.BridgeAttachmentMessages != nil {
			attachmentMessages = *p.BridgeAttachmentMessages
		} else if p.GetAttachmentMessages != nil {
			attachmentMessages, err = p.GetAttachmentMessages(ctx, *inputString, p.IdeSelection, p.Messages, p.QuerySource)
			if err != nil {
				return nil, err
			}
		}
		p.checkpoint("query_attachment_loading_end")
	}

	if inputString != nil && p.Mode == types.PromptInputModeBash {
		if p.ProcessBashCommand != nil {
			prec := precedingBlocksForNormalized(isString, normalizedBlocks)
			r, err := p.ProcessBashCommand(ctx, *inputString, prec, attachmentMessages, p)
			if err != nil {
				return nil, err
			}
			return addImageMetadataMessage(r, imageMetadataTexts), nil
		}
		// Phase 1: do not execute shell in Go. Instead, parse/validate and request TS to execute.
		prep := bashprepare.Prepare(bashprepare.StdinRequest{
			Input: *inputString,
			Shell: "bash",
		})
		if prep.Reject != nil {
			// Return a TS-UI-friendly reject and let TS show it as a bash stderr message.
			return &ProcessUserInputBaseResult{
				Messages:    []types.Message{},
				ShouldQuery: false,
				Execution: &ExecutionRequest{
					Kind:         "bash",
					RejectReason: prep.Reject.Reason,
				},
			}, nil
		}
		return &ProcessUserInputBaseResult{
			Messages:    []types.Message{},
			ShouldQuery: false,
			Execution: &ExecutionRequest{
				Kind:    "bash",
				Command: prep.Command,
			},
		}, nil
	}

	if inputString != nil && !effectiveSkipSlash && strings.HasPrefix(inputStr, "/") {
		if p.ProcessSlashCommand != nil {
			prec := precedingBlocksForNormalized(isString, normalizedBlocks)
			var imageBlocks []types.ContentBlockParam
			r, err := p.ProcessSlashCommand(ctx, inputStr, prec, imageBlocks, attachmentMessages, p.UUID, p.IsAlreadyProcessing, p)
			if err != nil {
				return nil, err
			}
			return addImageMetadataMessage(r, imageMetadataTexts), nil
		}
		// Phase B: Go parses slash intent only; TS executes processSlashCommand.
		prep := slashprepare.Prepare(slashprepare.StdinRequest{
			Input: inputStr,
		})
		if prep.Reject != nil {
			return &ProcessUserInputBaseResult{
				Messages:    []types.Message{},
				ShouldQuery: false,
				Execution: &ExecutionRequest{
					Kind:         "slash",
					RejectReason: prep.Reject.Reason,
				},
			}, nil
		}
		return &ProcessUserInputBaseResult{
			Messages:    []types.Message{},
			ShouldQuery: false,
			Execution: &ExecutionRequest{
				Kind:        "slash",
				CommandName: prep.CommandName,
				Args:        prep.Args,
				IsMcp:       prep.IsMcp,
			},
		}, nil
	}

	if inputString != nil && p.Mode == types.PromptInputModePrompt {
		trimmed := strings.TrimSpace(inputStr)
		if agentType := findAgentMention(attachmentMessages); agentType != nil {
			mention := "@agent-" + *agentType
			isSubagentOnly := trimmed == mention
			isPrefix := strings.HasPrefix(trimmed, mention) && !isSubagentOnly
			p.logEvent("tengu_subagent_at_mention", map[string]any{
				"is_subagent_only": isSubagentOnly,
				"is_prefix":        isPrefix,
			})
		}
	}

	// Regular user prompt (TS processUserInputBase: processTextPrompt for string input).
	// UserPromptSubmit hooks run in [ProcessUserInput] after base, not via execution_request.
	if isString {
		tp, err := ProcessTextPrompt(normalizedStr, nil, nil, nil, attachmentMessages, p.UUID, permModePtr(p), p.IsMeta, textPromptLog(p))
		if err != nil {
			return nil, err
		}
		base := &ProcessUserInputBaseResult{Messages: tp.Messages, ShouldQuery: tp.ShouldQuery}
		return addImageMetadataMessage(base, imageMetadataTexts), nil
	}

	// Non-string fallback keeps previous behavior (not expected on current TS bridge gate).
	tp, err := ProcessTextPrompt("", normalizedBlocks, nil, nil, attachmentMessages, p.UUID, permModePtr(p), p.IsMeta, textPromptLog(p))
	if err != nil {
		return nil, err
	}
	base := &ProcessUserInputBaseResult{Messages: tp.Messages, ShouldQuery: tp.ShouldQuery}
	return addImageMetadataMessage(base, imageMetadataTexts), nil
}

func permModePtr(p *ProcessUserInputParams) *types.PermissionMode {
	if p.PermissionMode == "" {
		return nil
	}
	pm := p.PermissionMode
	return &pm
}

func rejectImageResult() *ProcessUserInputBaseResult {
	return &ProcessUserInputBaseResult{
		Messages:    []types.Message{newSystemInformationalMessage(userImageRejectMessage, "warning")},
		ShouldQuery: false,
		ResultText:  userImageRejectMessage,
	}
}

func findAgentMention(msgs []types.Message) *string {
	for _, m := range msgs {
		if m.Type != types.MessageTypeAttachment || len(m.Attachment) == 0 {
			continue
		}
		var att struct {
			Type      string `json:"type"`
			AgentType string `json:"agentType"`
		}
		if json.Unmarshal(m.Attachment, &att) != nil {
			continue
		}
		if att.Type == "agent_mention" && att.AgentType != "" {
			s := att.AgentType
			return &s
		}
	}
	return nil
}

func addImageMetadataMessage(result *ProcessUserInputBaseResult, imageMetadataTexts []string) *ProcessUserInputBaseResult {
	if len(imageMetadataTexts) == 0 {
		return result
	}
	blocks := make([]any, 0, len(imageMetadataTexts))
	for _, t := range imageMetadataTexts {
		blocks = append(blocks, map[string]any{"type": "text", "text": t})
	}
	meta := true
	um, err := newUserMessage(blocks, nil, &meta, nil)
	if err != nil {
		return result
	}
	result.Messages = append(result.Messages, um)
	return result
}
