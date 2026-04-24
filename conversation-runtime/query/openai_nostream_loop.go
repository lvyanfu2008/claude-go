package query

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"goc/anthropicmessages"
	"goc/ccb-engine/apilog"
	"goc/ccb-engine/toolsearchwire"
	"goc/conversation-runtime/streamingtool"
	"goc/gou/ccbhydrate"
	"goc/messagesapi"
	"goc/tools/toolexecution"
	"goc/types"
)

// runOpenAINonStreamingParityModelLoop is like [runOpenAIStreamingParityModelLoop] but uses POST
// /v1/chat/completions with stream omitted (non-streaming JSON). Enable with GOU_QUERY_OPENAI_CHAT_NO_STREAM=1.
func runOpenAINonStreamingParityModelLoop(
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
	if apiKey == "" {
		return fmt.Errorf("query openai non-stream: set OPENAI_API_KEY")
	}
	base := strings.TrimSpace(openAIBaseURLFromEnv())
	model := ResolveOpenAIModel(strings.TrimSpace(in.ModelID))
	url := strings.TrimSuffix(base, "/") + "/chat/completions"

	httpClient := http.DefaultClient
	if deps.HTTPClient != nil {
		httpClient = deps.HTTPClient
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
			"model":       model,
			"messages":    openaiMsgs,
			"max_tokens":  maxTok,
		}
		mergeOpenAIThinkingBodyFields(req, model)
		if len(toolsOA) > 0 {
			req["tools"] = toolsOA
		}
		body, err := anthropicmessages.MarshalJSONNoEscapeHTML(req)
		if err != nil {
			return err
		}

		if apilog.ApiBodyLoggingEnabled() {
			apilog.PrepareIfEnabled()
		}
		apilog.LogRequestBody("POST "+url+" (no-stream)", body)

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		httpReq.Header.Set("authorization", "Bearer "+apiKey)
		httpReq.Header.Set("content-type", "application/json")

		resp, err := httpClient.Do(httpReq)
		if err != nil {
			return err
		}
		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return err
		}
		apilog.LogResponseBody("POST "+url+" (no-stream "+resp.Status+")", respBody)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("openai chat non-stream %s: %s", resp.Status, truncateOpenAIErr(string(respBody), 800))
		}

		acc := newAssistantStreamAccumulator()
		if err := ReplayOpenAINonStreamChatResponse(respBody, model, func(ev anthropicmessages.MessageStreamEvent) error {
			return acc.OnEvent(ev)
		}); err != nil {
			return err
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

		notifyStreamingToolUsesSnapshot(ctx, deps, acc)

		asstUUID := randomUUID()
		if deps.NewUUID != nil {
			asstUUID = deps.NewUUID()
		}
		inner, err := acc.AssistantWire(asstUUID)
		if err != nil {
			notifyStreamingToolUsesClear(ctx, deps)
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
			notifyStreamingToolUsesClear(ctx, deps)
			return context.Canceled
		}

		for _, tb := range acc.ToolUseBlocks() {
			ex.AddTool(tb, asst)
		}

		var toolMsgs []types.Message
		for upd, err := range ex.RemainingResults(ctx) {
			if err != nil {
				ex.Discard()
				notifyStreamingToolUsesClear(ctx, deps)
				return err
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
	return fmt.Errorf("openai non-stream parity: max rounds %d exceeded", maxRounds)
}
