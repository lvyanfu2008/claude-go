package query

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"goc/anthropicmessages"
	"goc/ccb-engine/toolsearchwire"
	"goc/conversation-runtime/streamingtool"
	"goc/gou/ccbhydrate"
	"goc/messagesapi"
	"goc/toolexecution"
	"goc/types"
)

func openAIBaseURLFromEnv() string {
	b := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if b == "" {
		return "https://api.openai.com/v1"
	}
	return strings.TrimSuffix(b, "/")
}

// runOpenAIStreamingParityModelLoop mirrors TS [queryModelOpenAI] + [adaptOpenAIStreamToAnthropic]:
// POST /v1/chat/completions with stream:true, system as first role:system message, same tool runner as Anthropic parity.
func runOpenAIStreamingParityModelLoop(
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
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" && deps.OpenAIPostStream != nil {
		apiKey = "test-key"
	}
	if apiKey == "" {
		return fmt.Errorf("query openai streaming: set OPENAI_API_KEY or inject QueryDeps.OpenAIPostStream")
	}
	base := openAIBaseURLFromEnv()
	model := ResolveOpenAIModel(strings.TrimSpace(in.ModelID))

	httpClient := http.DefaultClient
	if deps.HTTPClient != nil {
		httpClient = deps.HTTPClient
	}
	openAIPost := deps.OpenAIPostStream
	if openAIPost == nil {
		openAIPost = PostOpenAIChatStream
	}

	const maxRounds = 200
	cur := append([]types.Message(nil), work...)

	for round := 0; round < maxRounds; round++ {
		msgsJSON, err := ccbhydrate.MessagesJSONNormalized(cur, nil, messagesapi.OptionsFromEnv())
		if err != nil {
			return err
		}
		openaiMsgs, err := anthropicWireMessagesToOpenAI(msgsJSON, []string(in.SystemPrompt))
		if err != nil {
			return err
		}
		toolsForWire := in.Tools
		if len(in.Tools) > 0 {
			if wired, errW := toolsearchwire.WireToolsJSON(in.Tools, model, false, true, msgsJSON); errW == nil {
				toolsForWire = wired
			}
		}
		toolsOA, err := anthropicToolsWireToOpenAI(toolsForWire)
		if err != nil {
			return err
		}

		maxTok := openAIMaxTokensForChatCompletion(params, in.ModelID)
		req := map[string]any{
			"model":    model,
			"messages": openaiMsgs,
			"stream":   true,
			"stream_options": map[string]any{
				"include_usage": true,
			},
			"max_tokens": maxTok,
		}
		if len(toolsOA) > 0 {
			req["tools"] = toolsOA
		}
		body, err := anthropicmessages.MarshalJSONNoEscapeHTML(req)
		if err != nil {
			return err
		}

		acc := newAssistantStreamAccumulator()
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
			if reg, err := toolexecution.NewJSONToolRegistry(in.Tools); err == nil {
				depsCopy.Registry = reg
			}
		}
		runner := RunToolUseToolRunner{ParentCtx: ctx, Deps: depsCopy}
		var execCanUse any
		if params.CanUseTool != nil {
			execCanUse = toolexecution.QueryCanUseToolFn(params.CanUseTool)
		}
		ex := streamingtool.NewStreamingToolExecutor(makeFindToolBehavior(in.Tools), execCanUse, port, runner)

		if err := openAIPost(ctx, OpenAIPostStreamParams{
			BaseURL: base,
			APIKey:  apiKey,
			Body:    body,
			HTTP:    httpClient,
			Emit: func(ev anthropicmessages.MessageStreamEvent) error {
				if err := acc.OnEvent(ev); err != nil {
					return err
				}
				switch ev.Type {
				case "content_block_start", "content_block_delta", "content_block_stop":
					notifyStreamingToolUsesSnapshot(ctx, deps, acc)
				case "message_stop":
					notifyStreamingToolUsesClear(ctx, deps)
				}
				return nil
			},
		}); err != nil {
			return err
		}

		asstUUID := randomUUID()
		if deps.NewUUID != nil {
			asstUUID = deps.NewUUID()
		}
		inner, err := acc.AssistantWire(asstUUID)
		if err != nil {
			return err
		}
		asst := types.Message{
			Type:    types.MessageTypeAssistant,
			UUID:    asstUUID,
			Message: inner,
		}
		types.SyncAssistantMessageID(&asst)
		if !yieldStreamingParity(ctx, deps, QueryYield{Message: &asst}, yield) {
			ex.Discard()
			return context.Canceled
		}

		for _, tb := range acc.ToolUseBlocks() {
			ex.AddTool(tb, asst)
		}

		var toolMsgs []types.Message
		for upd, err := range ex.RemainingResults(ctx) {
			if err != nil {
				ex.Discard()
				return err
			}
			if upd.Message != nil {
				if !yieldStreamingParity(ctx, deps, QueryYield{Message: upd.Message}, yield) {
					ex.Discard()
					return context.Canceled
				}
				toolMsgs = append(toolMsgs, *upd.Message)
			}
		}

		if !acc.HasToolUse() {
			return nil
		}
		cur = append(cur, asst)
		cur = append(cur, toolMsgs...)
	}
	return fmt.Errorf("openai streaming parity: max rounds %d exceeded", maxRounds)
}
