package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"goc/commands"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	"goc/gou/ccbhydrate"
	"goc/gou/conversation"
	"goc/tools/toolexecution"
	"goc/gou/pui"
	"goc/messagesapi"
	"goc/modelenv"
	"goc/querycontext"
	"goc/sessiontranscript"
	"goc/types"

	"goc/engine"
)

// agentdSubmitFn 返回 agentd 版本的 SubmitFunc，完整实现 ProcessUserInput → ApplyBaseResult → Query 管线。
func agentdSubmitFn(cwd, sessionID string, permBridge engine.PermissionBridge) engine.SubmitFunc {
	return func(ctx context.Context, text string, store *conversation.Store, events engine.EventHandler, _ engine.PermissionBridge) error {
		permMode := types.PermissionDefault
		demoCfg := pui.DemoConfig{
			SessionID:      sessionID,
			PermissionMode: &permMode,
			Language:       strings.TrimSpace(os.Getenv("CLAUDE_CODE_LANGUAGE")),
		}
		if mcpPath := os.Getenv("GOU_DEMO_MCP_COMMANDS_JSON"); mcpPath != "" {
			demoCfg.MCPCommandsJSONPath = mcpPath
		}
		if mcpToolPath := os.Getenv("GOU_DEMO_MCP_TOOLS_JSON"); mcpToolPath != "" {
			demoCfg.MCPToolsJSONPath = mcpToolPath
		}

		params, err := pui.BuildDemoParams(text, store, demoCfg)
		if err != nil {
			events.OnErrorMessage(fmt.Sprintf("build params: %v", err))
			return err
		}

		r, err := processuserinput.ProcessUserInput(ctx, params)
		if err != nil {
			events.OnErrorMessage(fmt.Sprintf("processUserInput: %v", err))
			return err
		}

		var handoff pui.ProcessUserInputBaseResultHandoff
		out := pui.ApplyBaseResult(store, r, &handoff)

		if len(r.Messages) > 0 || out.EffectiveShouldQuery {
			events.OnStateSnapshot(store.Messages, engine.StateMetadata{SessionID: sessionID})
		}

		if out.EffectiveShouldQuery && !out.HadExecutionRequest {
			if err := runAgentdQuery(ctx, store, events, permBridge, params); err != nil {
				events.OnErrorMessage(err.Error())
				return err
			}
		}

		recordAgentdTranscript(store, sessionID, cwd)
		events.OnTurnDone("completed")
		return nil
	}
}

// runAgentdQuery 构建 QueryParams 并迭代 Query。
func runAgentdQuery(ctx context.Context, store *conversation.Store, events engine.EventHandler, permBridge engine.PermissionBridge, params *processuserinput.ProcessUserInputParams) error {
	cwd, _ := os.Getwd()
	mainLoopModel := modelenv.EffectiveMainLoopModel()

	var normToolsJSON json.RawMessage
	if params.RuntimeContext != nil {
		normToolsJSON = params.RuntimeContext.ToolUseContext.Options.Tools
	}
	var normToolDefs []struct {
		Name string `json:"name"`
	}
	json.Unmarshal(normToolsJSON, &normToolDefs)
	toolSpecs := make([]messagesapi.ToolSpec, 0, len(normToolDefs))
	for _, t := range normToolDefs {
		toolSpecs = append(toolSpecs, messagesapi.ToolSpec{Name: t.Name})
	}

	normOpts := messagesapi.OptionsFromEnv()
	baseMsgs, err := ccbhydrate.MessagesJSONNormalized(store.Messages, toolSpecs, normOpts)
	if err != nil {
		return fmt.Errorf("messages JSON: %w", err)
	}
	if len(bytes.TrimSpace(baseMsgs)) < 3 || bytes.Equal(bytes.TrimSpace(baseMsgs), []byte("[]")) {
		return fmt.Errorf("empty chat transcript")
	}

	gouOpts := commands.GouDemoSystemOpts{
		ModelID:               mainLoopModel,
		Cwd:                   cwd,
		NonInteractiveSession: true,
	}
	commands.ApplyGouDemoRuntimeEnv(&gouOpts)

	var customSys string
	if params.RuntimeContext != nil {
		if p := params.RuntimeContext.ToolUseContext.Options.CustomSystemPrompt; p != nil {
			customSys = strings.TrimSpace(*p)
		}
	}
	extraRoots := querycontext.ExtraClaudeMdRootsForFetch(params.RuntimeContext)
	fetchOpts := querycontext.FetchOpts{
		CustomSystemPrompt: customSys,
		Gou:                gouOpts,
		ExtraClaudeMdRoots: extraRoots,
		SessionStartSource: "startup",
		HooksSessionID:     store.ConversationID,
	}

	fetchResult, err := querycontext.FetchSystemPromptParts(ctx, fetchOpts)
	if err != nil {
		return fmt.Errorf("fetch system prompt: %w", err)
	}

	qp := query.QueryParams{
		Messages:        store.Messages,
		SystemPrompt:    query.SystemPrompt(fetchResult.DefaultSystemPrompt),
		CanUseTool:       canUseToolFn(permBridge),
		UserContext:     fetchResult.UserContext,
		SystemContext:   fetchResult.SystemContext,
		StreamingParity: true,
	}

	for y, err := range query.Query(ctx, qp) {
		if err != nil {
			return fmt.Errorf("query: %w", err)
		}
		if y.Message != nil {
			handleQueryYieldMessage(store, events, *y.Message)
		}
		if y.Terminal != nil {
			if y.Terminal.Error != nil {
				return y.Terminal.Error
			}
			return nil
		}
	}
	return nil
}

func handleQueryYieldMessage(store *conversation.Store, events engine.EventHandler, msg types.Message) {
	switch msg.Type {
	case types.MessageTypeAssistant:
		var blocks []struct {
			Type    string          `json:"type"`
			Text    string          `json:"text,omitempty"`
			Name    string          `json:"name,omitempty"`
			ID      string          `json:"id,omitempty"`
			Input   json.RawMessage `json:"input,omitempty"`
			Content json.RawMessage `json:"content,omitempty"`
		}
		if err := json.Unmarshal([]byte(msg.Content), &blocks); err != nil {
			events.OnAssistantMessage(msg)
			return
		}
		for _, block := range blocks {
			switch block.Type {
			case "text":
				events.OnStreamDelta(block.Text)
			case "tool_use":
				events.OnToolUseStart(block.Name, block.ID, block.Input)
			case "tool_result":
				events.OnToolResult(block.ID, block.Content, false)
			case "thinking":
				events.OnStreamThinkingDelta(block.Text)
			}
		}
		store.AppendMessage(msg)
		events.OnAssistantMessage(msg)

	case types.MessageTypeUser:
		store.AppendMessage(msg)
		events.OnStateSnapshot(store.Messages, engine.StateMetadata{})
	}
}


func canUseToolFn(b engine.PermissionBridge) toolexecution.QueryCanUseToolFn {
	if b == nil {
		return nil
	}
	return func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (toolexecution.PermissionDecision, error) {
		pd, err := b.AskPermission(ctx, toolName, input)
		if err != nil {
			return toolexecution.DenyDecision(err.Error()), err
		}
		if pd.Allow {
			return toolexecution.AllowDecision(), nil
		}
		return toolexecution.DenyDecision(pd.Reason), nil
	}
}

func askResolverFn(b engine.PermissionBridge) func(ctx context.Context, toolName, toolUseID string, input json.RawMessage, prompt string) (toolexecution.PermissionDecision, error) {
	if b == nil {
		return nil
	}
	return func(ctx context.Context, toolName string, _ string, input json.RawMessage, _ string) (toolexecution.PermissionDecision, error) {
		pd, err := b.AskPermission(ctx, toolName, input)
		if err != nil {
			return toolexecution.DenyDecision(err.Error()), err
		}
		if pd.Allow {
			return toolexecution.AllowDecision(), nil
		}
		return toolexecution.DenyDecision(pd.Reason), nil
	}
}

func recordAgentdTranscript(store *conversation.Store, sessionID, cwd string) {
	tr := &sessiontranscript.Store{
		SessionID:   sessionID,
		OriginalCwd: cwd,
		Cwd:         cwd,
	}
	msgs := make([]types.Message, len(store.Messages))
	copy(msgs, store.Messages)
	_, _ = tr.RecordTranscript(context.Background(), msgs, sessiontranscript.RecordOpts{AllMessages: msgs})
}
