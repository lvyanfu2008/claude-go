package query

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"goc/anthropicmessages"
	"goc/tools/toolsearchwire"
	"goc/conversation-runtime/streamingtool"
	"goc/gou/ccbhydrate"
	"goc/messagesapi"
	"goc/modelenv"
	"goc/tools/toolexecution"
	"goc/types"
)

func yieldStreamingParity(ctx context.Context, deps *QueryDeps, qy QueryYield, yield func(QueryYield, error) bool) bool {
	if !yield(qy, nil) {
		return false
	}
	if deps != nil && deps.OnQueryYield != nil {
		_ = deps.OnQueryYield(ctx, qy)
	}
	return true
}

// runStreamingParityModelLoop mirrors query.ts streaming path: Anthropic SSE + [streamingtool.StreamingToolExecutor].
func runStreamingParityModelLoop(
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
	apiKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN"))
	}
	base := strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL"))
	if base == "" {
		base = "https://api.anthropic.com"
	}
	model := strings.TrimSpace(in.ModelID)
	if model == "" {
		model = modelenv.ResolveWithFallback("")
	}

	httpClient := http.DefaultClient
	if deps.HTTPClient != nil {
		httpClient = deps.HTTPClient
	}
	streamPost := deps.StreamPost
	if streamPost == nil {
		streamPost = anthropicmessages.PostStream
	}
	if apiKey == "" && deps.StreamPost != nil {
		apiKey = "test-key"
	}
	if apiKey == "" {
		return fmt.Errorf("query streaming parity: set ANTHROPIC_API_KEY or inject QueryDeps.StreamPost")
	}

	const maxRounds = 200
	cur := append([]types.Message(nil), work...)

	for round := 0; round < maxRounds; round++ {
		msgsJSON, err := ccbhydrate.MessagesJSONNormalized(cur, nil, messagesapi.OptionsFromEnv())
		if err != nil {
			return err
		}
		var msgsWire any
		if err := json.Unmarshal(msgsJSON, &msgsWire); err != nil {
			return fmt.Errorf("messages wire: %w", err)
		}

		toolsForWire := in.Tools
		if len(in.Tools) > 0 {
			if wired, errW := toolsearchwire.WireToolsJSON(in.Tools, model, false, false, msgsJSON); errW == nil {
				toolsForWire = wired
			}
		}

		req := map[string]any{
			"model":      model,
			"max_tokens": 4096,
			"messages":   msgsWire,
			"stream":     true,
		}
		if sys := strings.TrimSpace(strings.Join([]string(in.SystemPrompt), "\n\n")); sys != "" {
			req["system"] = sys
		}
		if len(toolsForWire) > 0 {
			var toolsWire any
			if err := json.Unmarshal(toolsForWire, &toolsWire); err == nil {
				req["tools"] = toolsWire
			}
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

		betas := anthropicmessages.BetasForToolsJSON(toolsForWire)
		if err := streamPost(ctx, anthropicmessages.PostStreamParams{
			BaseURL: base,
			APIKey:  apiKey,
			Body:    body,
			HTTP:    httpClient,
			Beta:    betas,
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
	return fmt.Errorf("streaming parity: max rounds %d exceeded", maxRounds)
}
