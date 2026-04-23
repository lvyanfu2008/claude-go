package query

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"goc/ccb-engine/gemma"
	"goc/ccb-engine/toolsearchwire"
	"goc/conversation-runtime/streamingtool"
	"goc/gou/ccbhydrate"
	"goc/messagesapi"
	"goc/tools/toolexecution"
	"goc/types"
)

// UseGemmaProvider checks if Gemma model should be used
func UseGemmaProvider() bool {
	if envTruthy("CLAUDE_CODE_USE_GEMMA") {
		return true
	}
	model := strings.TrimSpace(os.Getenv("CCB_ENGINE_MODEL"))
	if strings.Contains(strings.ToLower(model), "gemma") {
		return true
	}
	return false
}

// gemmaWireMessagesFromHydratedJSON turns [ccbhydrate.MessagesJSONNormalized] output
// (role + content string or Anthropic-style block arrays) into Gemma's simple string content.
func gemmaWireMessagesFromHydratedJSON(msgsJSON json.RawMessage) ([]gemma.Message, error) {
	var rows []json.RawMessage
	if err := json.Unmarshal(msgsJSON, &rows); err != nil {
		return nil, fmt.Errorf("messages array: %w", err)
	}
	var out []gemma.Message
	for _, row := range rows {
		var m struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(row, &m); err != nil {
			return nil, fmt.Errorf("message row: %w", err)
		}
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		plain, err := gemmaFlattenAPIContent(m.Role, m.Content)
		if err != nil {
			return nil, err
		}
		out = append(out, gemma.Message{Role: m.Role, Content: plain})
	}
	return out, nil
}

func gemmaFlattenAPIContent(role string, content json.RawMessage) (string, error) {
	if len(content) == 0 || string(content) == "null" {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return s, nil
	}
	var blocks []map[string]any
	if err := json.Unmarshal(content, &blocks); err != nil {
		return "", fmt.Errorf("content: %w", err)
	}
	var parts []string
	for _, b := range blocks {
		typ, _ := b["type"].(string)
		switch typ {
		case "text":
			if tx, ok := b["text"].(string); ok {
				parts = append(parts, tx)
			}
		case "tool_use":
			if role != "assistant" {
				parts = append(parts, fallbackBlockSummary(b))
				continue
			}
			id, _ := b["id"].(string)
			name, _ := b["name"].(string)
			args := "{}"
			if in := b["input"]; in != nil {
				if str, ok := in.(string); ok {
					args = str
				} else {
					raw, err := json.Marshal(in)
					if err == nil && len(raw) > 0 {
						args = string(raw)
					}
				}
			}
			if strings.TrimSpace(args) == "" {
				args = "{}"
			}
			parts = append(parts, fmt.Sprintf("[tool_use id=%s name=%s]\n%s", id, name, args))
		case "tool_result":
			if role != "user" {
				parts = append(parts, fallbackBlockSummary(b))
				continue
			}
			tid, _ := b["tool_use_id"].(string)
			tr := toolResultContentToString(b["content"])
			parts = append(parts, fmt.Sprintf("[tool_result tool_use_id=%s]\n%s", tid, tr))
		default:
			if tx, ok := b["text"].(string); ok && tx != "" {
				parts = append(parts, tx)
			} else if typ != "" {
				parts = append(parts, fallbackBlockSummary(b))
			}
		}
	}
	return strings.Join(parts, "\n"), nil
}

// runGemmaStreamingParityModelLoop runs Gemma via Vertex :rawPredict (full JSON per round), unwraps
// predictions to OpenAI chat.completion, replays through the same accumulator + tool executor as
// OpenAI non-stream parity.
func runGemmaStreamingParityModelLoop(
	ctx context.Context,
	params QueryParams,
	work []types.Message,
	in *CallModelInput,
	deps *QueryDeps,
	yield func(QueryYield, error) bool,
) error {
	if deps == nil {
		return fmt.Errorf("query: nil deps")
	}

	projectID := strings.TrimSpace(os.Getenv("VERTEX_AI_PROJECT_ID"))
	location := strings.TrimSpace(os.Getenv("VERTEX_AI_LOCATION"))
	endpointID := strings.TrimSpace(os.Getenv("VERTEX_AI_ENDPOINT_ID"))
	modelName := strings.TrimSpace(os.Getenv("VERTEX_AI_MODEL_NAME"))

	config := gemma.DefaultConfig()
	if projectID != "" {
		config.ProjectID = projectID
	}
	if location != "" {
		config.Location = location
	}
	if endpointID != "" {
		config.EndpointID = endpointID
	}
	if modelName != "" {
		config.ModelName = modelName
	}

	client := gemma.NewClient(config)

	const maxRounds = 200
	cur := append([]types.Message(nil), work...)

	for round := 0; round < maxRounds; round++ {
		msgsJSON, err := ccbhydrate.MessagesJSONNormalized(cur, nil, messagesapi.OptionsFromEnv())
		if err != nil {
			return err
		}
		msgsWire, err := gemmaWireMessagesFromHydratedJSON(msgsJSON)
		if err != nil {
			return fmt.Errorf("messages wire: %w", err)
		}

		toolsForWire := in.Tools
		if len(in.Tools) > 0 {
			if wired, errW := toolsearchwire.WireToolsJSON(in.Tools, config.ModelName, false, true, msgsJSON); errW == nil {
				toolsForWire = wired
			}
		}

		req := gemma.ChatRequest{
			Model:       config.ModelName,
			Messages:    msgsWire,
			MaxTokens:   100,
			Temperature: 0.7,
		}

		sys := strings.TrimSpace(strings.Join([]string(in.SystemPrompt), "\n\n"))
		if sys != "" {
			req.Messages = append([]gemma.Message{
				{Role: "system", Content: sys},
			}, req.Messages...)
		}

		if len(toolsForWire) > 0 {
			// Same as OpenAI parity: Anthropic-shaped wire (name + input_schema) → OpenAI tools[]
			// with function.parameters (see anthropicToolsWireToOpenAI).
			toolsJSON := toolsForWire
			if oa, errOA := anthropicToolsWireToOpenAI(toolsForWire); errOA == nil && len(oa) > 0 {
				if b, err := json.Marshal(oa); err == nil {
					toolsJSON = b
				}
			}
			req.ToolsJSON = append(json.RawMessage(nil), toolsJSON...)
			var tools []gemma.Tool
			if err := json.Unmarshal(toolsJSON, &tools); err == nil && len(tools) > 0 {
				req.Tools = tools
			}
		}

		raw, err := client.ChatCompletionRaw(ctx, req)
		if err != nil {
			return fmt.Errorf("Gemma call failed: %w", err)
		}
		inner, err := gemma.UnwrapVertexPredictionsOpenAI(raw)
		if err != nil {
			return fmt.Errorf("Gemma response unwrap: %w", err)
		}
		inner = NormalizeOpenAINonStreamChatBodyToolCallsLoose(inner)

		model := strings.TrimSpace(in.ModelID)
		if model == "" {
			model = config.ModelName
		}

		acc := newAssistantStreamAccumulator()
		if err := ReplayOpenAINonStreamChatResponse(inner, model, acc.OnEvent); err != nil {
			return fmt.Errorf("Gemma replay: %w", err)
		}

		toolAbortRoot := streamingtool.NewAbortController()
		go func() {
			<-ctx.Done()
			toolAbortRoot.Abort(ctx.Err())
		}()
		port := newQueryToolUseContextPort(toolAbortRoot)
		depsCopy := deps.ToolexecutionDeps
		if strings.TrimSpace(depsCopy.MainLoopModel) == "" {
			depsCopy.MainLoopModel = strings.TrimSpace(params.ToolUseContext.Options.MainLoopModel)
		}
		if params.ToolPermissionContext != nil {
			pc := *params.ToolPermissionContext
			types.NormalizeToolPermissionContextData(&pc)
			depsCopy.ToolPermission = &pc
		}
		if depsCopy.QueryCanUseTool == nil && params.CanUseTool != nil {
			depsCopy.QueryCanUseTool = params.CanUseTool
		}
		if depsCopy.Registry == nil && len(in.Tools) > 0 {
			if reg, errReg := toolexecution.NewJSONToolRegistry(in.Tools); errReg == nil {
				depsCopy.Registry = reg
			}
		}
		runner := RunToolUseToolRunner{ParentCtx: ctx, Deps: depsCopy}
		var execCanUse any
		if params.CanUseTool != nil {
			execCanUse = toolexecution.QueryCanUseToolFn(params.CanUseTool)
		}
		ex := streamingtool.NewStreamingToolExecutor(makeFindToolBehavior(in.Tools), execCanUse, port, runner)

		notifyStreamingToolUsesSnapshot(ctx, deps, acc)

		asstUUID := randomUUID()
		if deps.NewUUID != nil {
			asstUUID = deps.NewUUID()
		}
		innerWire, err := acc.AssistantWire(asstUUID)
		if err != nil {
			notifyStreamingToolUsesClear(ctx, deps)
			return err
		}
		asst := types.Message{
			Type:    types.MessageTypeAssistant,
			UUID:    asstUUID,
			Message: innerWire,
		}
		types.SyncAssistantMessageID(&asst)
		if !yieldStreamingParity(ctx, deps, QueryYield{Message: &asst}, yield) {
			ex.Discard()
			notifyStreamingToolUsesClear(ctx, deps)
			return context.Canceled
		}

		for _, tb := range acc.ToolUseBlocks() {
			ex.AddTool(tb, asst)
		}

		var toolMsgs []types.Message
		for upd, errUp := range ex.RemainingResults(ctx) {
			if errUp != nil {
				ex.Discard()
				notifyStreamingToolUsesClear(ctx, deps)
				return errUp
			}
			if upd.Message != nil {
				if !yieldStreamingParity(ctx, deps, QueryYield{Message: upd.Message}, yield) {
					ex.Discard()
					notifyStreamingToolUsesClear(ctx, deps)
					return context.Canceled
				}
				toolMsgs = append(toolMsgs, *upd.Message)
			}
		}
		notifyStreamingToolUsesClear(ctx, deps)

		if !acc.HasToolUse() {
			return nil
		}
		cur = append(cur, asst)
		cur = append(cur, toolMsgs...)
	}
	return fmt.Errorf("gemma vertex parity: max rounds %d exceeded", maxRounds)
}
