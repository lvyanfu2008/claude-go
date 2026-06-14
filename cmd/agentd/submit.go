package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"goc/appstate"
	"goc/commands"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	"goc/gou/ccbhydrate"
	"goc/gou/commandqueue"
	"goc/gou/conversation"
	"goc/gou/pui"
	"goc/growthbook"
	"goc/messagesapi"
	"goc/modelenv"
	"goc/querycontext"
	"goc/compactservice"
	"goc/services/autodream"
	"goc/services/extractmemories"
	"goc/services/sessionmemory"
	"goc/sessiontranscript"
	"goc/tools/skilltools"
	"goc/tools/toolexecution"
	"goc/tools/toolresultpersist"
	"goc/types"

	"goc/engine"
)

var (
	agentdSessionStarted    bool
	agentdExtractMemState   = extractmemories.NewState()
	agentdAutoDreamState    = autodream.NewState()
	agentdSessionMemState   = sessionmemory.NewState()
	agentdAppStateStore     = appstate.NewStore(appstate.DefaultAppState())
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

		// Send commands list to ink-gateway for UI slash autocomplete.
		events.OnCommandsList(params.Commands)

		// Inject slash command handler so user-typed /commands are resolved in-process
		// instead of returning a stub ExecutionRequest.
		params.ProcessSlashCommand = pui.NewSlashResolveProcessSlashCommand(pui.SlashResolveHandlerOptions{
			SessionID: sessionID,
			Store:     store,
			Cwd:       cwd,
		})

		r, err := processuserinput.ProcessUserInput(ctx, params)
		if err != nil {
			events.OnErrorMessage(fmt.Sprintf("processUserInput: %v", err))
			return err
		}

		var handoff pui.ProcessUserInputBaseResultHandoff
		out := pui.ApplyBaseResult(store, r, &handoff)

		if len(r.Messages) > 0 || out.EffectiveShouldQuery {
			// Filter out attachment messages before sending to UI — they are
			// internal (CLAUDE.md, hooks) and should not be displayed to the user.
			uiMsgs := make([]types.Message, 0, len(store.Messages))
			for _, m := range store.Messages {
				if m.Type != types.MessageTypeAttachment {
					uiMsgs = append(uiMsgs, m)
				}
			}
			events.OnStateSnapshot(uiMsgs, engine.StateMetadata{SessionID: sessionID})
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

	// Build skill listing from params (mirrors gou/app/view.go skill listing injection)
	skillListing := params.SkillListingCommands
	if len(skillListing) == 0 {
		skillListing = commands.SkillToolCommands(params.Commands)
	}
	hasSkillTool := false
	for _, t := range toolSpecs {
		if t.Name == skilltools.SkillToolName() {
			hasSkillTool = true
			break
		}
	}

	normOpts := messagesapi.OptionsFromEnv()
	var baseMsgs json.RawMessage
	var err error
	var skillListingText string
	if len(skillListing) > 0 && hasSkillTool {
		listingSent := make(map[string]struct{})
		if s, _, _, ok := commands.AppendSkillListingForAPI(skillListing, hasSkillTool, listingSent, nil); ok {
			skillListingText = s
		}
	}
	if skillListingText != "" {
		baseMsgs, err = ccbhydrate.MessagesJSONWithSkillListing(store.Messages, skillListingText, toolSpecs, normOpts)
	} else {
		baseMsgs, err = ccbhydrate.MessagesJSONNormalized(store.Messages, toolSpecs, normOpts)
	}
	if err != nil {
		return fmt.Errorf("messages JSON: %w", err)
	}
	if len(bytes.TrimSpace(baseMsgs)) < 3 || bytes.Equal(bytes.TrimSpace(baseMsgs), []byte("[]")) {
		return fmt.Errorf("empty chat transcript")
	}

	gouOpts := commands.GouDemoSystemOpts{
		ModelID:               mainLoopModel,
		Cwd:                   cwd,
		NonInteractiveSession: false,
		SkillToolCommands:     skillListing,
	}
	commands.ApplyGouDemoRuntimeEnv(&gouOpts)

	var customSys string
	if params.RuntimeContext != nil {
		if p := params.RuntimeContext.ToolUseContext.Options.CustomSystemPrompt; p != nil {
			customSys = strings.TrimSpace(*p)
		}
	}
	extraRoots := querycontext.ExtraClaudeMdRootsForFetch(params.RuntimeContext)
	ssSource := ""
	if !agentdSessionStarted {
		agentdSessionStarted = true
		ssSource = "startup"
	}
	fetchOpts := querycontext.FetchOpts{
		CustomSystemPrompt: customSys,
		Gou:                gouOpts,
		ExtraClaudeMdRoots: extraRoots,
		SessionStartSource: ssSource,
		HooksSessionID:     store.ConversationID,
	}

	fetchResult, err := querycontext.FetchSystemPromptParts(ctx, fetchOpts)
	if err != nil {
		return fmt.Errorf("fetch system prompt: %w", err)
	}

	// Inject SessionStart hook messages into conversation (mirrors gou/app/view.go)
	for _, m := range fetchResult.SessionStartHookMessages {
		store.AppendMessage(m)
	}

	// Build full ParityToolRunner covering all tools: Agent, Task, Skill,
	// Bash, Read, Write, Edit, Glob, Grep, etc.
	absWorkDir, _ := filepath.Abs(cwd)
	var cmds []types.Command
	if params.RuntimeContext != nil {
		cmds = params.RuntimeContext.Options.Commands
	}
	runner := &skilltools.ParityToolRunner{
		DemoToolRunner: skilltools.DemoToolRunner{
			Commands:  cmds,
			SessionID: store.ConversationID,
		},
		WorkDir:          absWorkDir,
		ProjectRoot:      absWorkDir,
		LocalBashDefault: true,
		AskAutoFirst:     true,
		MainLoopModel:    mainLoopModel,
		Messages:         store.Messages,
		MessagesFunc:     func() []types.Message { return store.Messages },
		AppStateStore:    agentdAppStateStore,
		ProgressCallback: func(msg *types.Message) {
			if msg == nil || msg.Type != types.MessageTypeProgress || len(msg.Data) == 0 {
				return
			}
			var data struct {
				Type        string `json:"type"`
				AgentID     string `json:"agentId"`
				AgentType   string `json:"agentType"`
				Name        string `json:"name"`
				Description string `json:"description"`
				Summary     string `json:"summary"`
				Status      string `json:"status"`
				Message     string `json:"message"`
			}
			if json.Unmarshal(msg.Data, &data) == nil {
				switch data.Type {
				case "agent_registered":
					label := data.Description
					if label == "" {
						label = data.Name
					}
					if label == "" {
						label = data.AgentType
					}
					events.OnAgentProgress(data.AgentID, "running", label)
				case "agent_summary":
					events.OnAgentProgress(data.AgentID, "running", data.Summary)
				case "agent_completed":
					events.OnAgentProgress(data.AgentID, data.Status, data.Message)
				}
			}
		},
		NotificationCallback: func(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64) {
			events.OnAgentProgress(agentID, status, fmt.Sprintf("%s (%d tool uses, %d tokens)", summary, toolUseCount, tokenCount))
			// Push to command queue so DrainCommandQueue injects the result
			// as a user message into the next API round (TS parity).
			commandqueue.EnqueueAgentNotification(agentID, toolUseID, outputFile, status, summary, output, tokenCount, toolUseCount, durationMs)
		},
	}

	// Build production deps with auto-compact, snip compact, and session memory compact.
	smState := agentdSessionMemState
	var trySMCompact compactservice.TrySessionMemoryCompactFn
	if smState != nil {
		trySMCompact = func(ctx context.Context, messages []types.Message, agentID string, autoCompactThreshold *int) (*compactservice.CompactionResult, error) {
			return sessionmemory.TrySessionMemoryCompaction(
				ctx, smState, store.ConversationID, cwd, messages,
				"", autoCompactThreshold,
				nil, // sessionStartHookRunner
				agentID, mainLoopModel, nil,
			)
		}
	}
	qdeps := query.ProductionDeps(trySMCompact, nil)
	// P0-2: complete tool execution deps
	te := toolexecution.ExecutionDeps{
		InvokeTool:              runner.Run,
		MainLoopModel:           mainLoopModel,
		ReadToolRoots:           runner.ToolReadMappingRoots(),
		ReadToolMemCWD:          runner.ToolReadMappingMemCWD(),
		MultiMessageToolHandler: skilltools.NewSkillMultiMessageHandler(cmds, store.ConversationID, nil),
		QueryCanUseTool: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (toolexecution.PermissionDecision, error) {
			if toolName == "AskUserQuestion" {
				return toolexecution.AskDecision("Answer questions?"), nil
			}
			return toolexecution.AskDecision("Allow " + toolName + "?"), nil
		},
		SandboxingEnabled:                      true,
		AutoAllowBashWholeToolAskWhenSandboxed: true,
	}
	te.AskResolver = func(ctx context.Context, toolName, toolUseID string, input json.RawMessage, prompt string) (toolexecution.PermissionDecision, error) {
		return canUseToolFn(permBridge, events)(ctx, toolName, toolUseID, input)
	}
	qdeps.ToolexecutionDeps = te

	// Prepend SessionStart hook messages to query messages (ephemeral, not persisted to store).
	// Mirrors gou/app/view.go msgsForQ prepend.
	msgsForQ := store.Messages
	if len(fetchResult.SessionStartHookMessages) > 0 {
		msgsForQ = append(slices.Clone(fetchResult.SessionStartHookMessages), msgsForQ...)
	}

	qp := query.QueryParams{
		Messages:        msgsForQ,
		SystemPrompt:    query.SystemPrompt(fetchResult.DefaultSystemPrompt),
		CanUseTool: nil, // te.QueryCanUseTool handles per-tool permissions (AskDecision for AskUserQuestion)
		UserContext:     fetchResult.UserContext,
		SystemContext:   fetchResult.SystemContext,
		StreamingParity: true,
		Deps:            &qdeps,
	}
	if params.RuntimeContext != nil {
		qp.ToolUseContext = params.RuntimeContext.ToolUseContext
	}
	// OnQueryComplete: post-turn memory extraction and auto-dream (P1-2)
	// Call Hook once outside the closure so the sequential-gate sync.Mutex
	// is shared across all OnQueryComplete invocations.
	smHook := sessionmemory.Hook(agentdSessionMemState, store.ConversationID, cwd)
	qdeps.OnQueryComplete = func(ctx context.Context, qcp query.QueryCompleteParams) {
		extractmemories.Execute(ctx, agentdExtractMemState, extractmemories.ExtractionParams{
			Messages:       qcp.Messages,
			ToolUseContext: qcp.ToolUseContext,
			SystemPrompt:   qcp.SystemPrompt,
			UserContext:    qcp.UserContext,
			SystemContext:  qcp.SystemContext,
			Cwd:            qcp.Cwd,
			QuerySource:    qcp.QuerySource,
			NewUUID:        query.RandomUUID,
			SkipIndex:      growthbook.IsTenguMothCopse(),
			AppendSystemMessage: func(msg types.Message) {
				store.AppendMessage(msg)
				events.OnStateSnapshot(store.Messages, engine.StateMetadata{SessionID: store.ConversationID})
			},
		})
		autoDreamPaths, _ := autodream.Execute(ctx, agentdAutoDreamState, qcp.ToolUseContext, qcp.SystemPrompt,
			qcp.UserContext, qcp.SystemContext, qcp.QuerySource, query.RandomUUID,
			commands.ClaudeConfigHome(), qcp.Cwd, "", store.ConversationID)
		if len(autoDreamPaths) > 0 {
			msg := extractmemories.CreateMemorySavedMessage(autoDreamPaths, query.RandomUUID)
			store.AppendMessage(msg)
			events.OnStateSnapshot(store.Messages, engine.StateMetadata{SessionID: store.ConversationID})
		}
		// Session memory compaction (TS sessionMemory post-turn hook).
		smHook(ctx, qcp)
	}

	// ApplyToolResultBudget: enforce tool result size budget
	qdeps.ApplyToolResultBudget = func(ctx context.Context, in *query.ToolResultBudgetInput) ([]types.Message, error) {
		return toolresultpersist.ApplyToolResultBudget(
			in.Messages,
			nil, // contentReplacementState (nil = no persistent trunction state)
			toolresultpersist.SessionInfo{SessionID: store.ConversationID, Cwd: cwd},
			0,   // use default MaxToolResultsPerMessageChars
			nil, // skipToolNames
		), nil
	}

	// DrainCommandQueue: forward background agent notifications
	qdeps.DrainCommandQueue = func() []string {
		var result []string
		for _, cmd := range commandqueue.DrainCommandQueue() {
			result = append(result, cmd.Value)
		}
		return result
	}

	processuserinput.ApplyQueryHostEnvGates(&qp)
	processuserinput.WireToolexecutionFromProcessUserInput(&qp, params)

	// Run the query, then drain any background agent notifications that
	// arrived after the loop finished. Start another turn if needed so
	// the model sees the agent's output. Mirrors TS print.ts do-while.
	for {
		for y, err := range query.Query(ctx, qp) {
			if err != nil {
				return fmt.Errorf("query: %w", err)
			}
			if len(y.StreamEvent) > 0 {
				handleStreamEvent(events, y.StreamEvent)
			}
			if y.Message != nil {
				handleQueryYieldMessage(store, events, *y.Message)
			}
			if y.Terminal != nil {
				if y.Terminal.Error != nil {
					return y.Terminal.Error
				}
				break
			}
		}

		// If background agents are still running, wait for them to
		// complete. The NotificationCallback will enqueue a command
		// and signal the notify channel.
		drained := commandqueue.DrainCommandQueue()
		if len(drained) == 0 && commandqueue.HasPendingBgAgents() {
			select {
			case <-commandqueue.NotifyChan():
				drained = commandqueue.DrainCommandQueue()
			case <-time.After(120 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if len(drained) == 0 {
			return nil
		}

		// Build user messages from queued notifications.
		var notifyTexts []string
		for _, cmd := range drained {
			if cmd.Value != "" {
				notifyTexts = append(notifyTexts, "A background agent completed a task:\n"+cmd.Value)
			}
		}
		if len(notifyTexts) == 0 {
			return nil
		}

		content, _ := json.Marshal([]map[string]any{{"type": "text", "text": strings.Join(notifyTexts, "\n\n")}})
		msg := types.Message{
			Type:    types.MessageTypeUser,
			Content: content,
		}
		store.AppendMessage(msg)
		events.OnStateSnapshot(store.Messages, engine.StateMetadata{})

		// Start another turn with the notification injected.
		qp.Messages = append(append([]types.Message{}, store.Messages...), msg)
	}
}

// handleStreamEvent parses a content_block_delta SSE event and forwards text/thinking
// deltas to the ink-gateway in real time for token-level streaming.
func handleStreamEvent(events engine.EventHandler, raw json.RawMessage) {
	var ev struct {
		Delta json.RawMessage `json:"delta"`
	}
	if json.Unmarshal(raw, &ev) == nil {
		var d struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			Thinking string `json:"thinking"`
		}
		if json.Unmarshal(ev.Delta, &d) == nil {
			switch d.Type {
			case "text_delta":
				if d.Text != "" {
					events.OnStreamDelta(d.Text)
				}
			case "thinking_delta":
				if d.Thinking != "" {
					events.OnStreamThinkingDelta(d.Thinking)
				}
			}
		}
	}
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
			case "tool_use":
				events.OnToolUseStart(block.Name, block.ID, block.Input)
				// Track background agents: if the LLM requests a
				// background agent, increment the pending counter.
				if block.Name == "Agent" || block.Name == "Task" {
					var in struct {
						RunInBackground bool `json:"run_in_background"`
					}
					if json.Unmarshal(block.Input, &in) == nil && in.RunInBackground {
						commandqueue.AddPendingBgAgent()
					}
				}
			case "tool_result":
				events.OnToolResult(block.ID, block.Content, false)
			}
		}
		store.AppendMessage(msg)
		events.OnAssistantMessage(msg)

	case types.MessageTypeUser:
		store.AppendMessage(msg)
		uiMsgs := make([]types.Message, 0, len(store.Messages))
		for _, m := range store.Messages {
			if m.Type != types.MessageTypeAttachment {
				uiMsgs = append(uiMsgs, m)
			}
		}
		events.OnStateSnapshot(uiMsgs, engine.StateMetadata{})
	}
}

func canUseToolFn(b engine.PermissionBridge, events engine.EventHandler) toolexecution.QueryCanUseToolFn {
	if b == nil {
		return nil
	}
	return func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (toolexecution.PermissionDecision, error) {
		events.OnPermissionAsk(toolName, toolUseID, input)
		pd, err := b.AskPermission(ctx, toolName, input)
		if err != nil {
			return toolexecution.DenyDecision(err.Error()), err
		}
		if pd.Allow {
			d := toolexecution.AllowDecision()
			if len(pd.UpdatedInput) > 0 {
				d.UpdatedInput = pd.UpdatedInput
			}
			return d, nil
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
			d := toolexecution.AllowDecision()
			if len(pd.UpdatedInput) > 0 {
				d.UpdatedInput = pd.UpdatedInput
			}
			return d, nil
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
