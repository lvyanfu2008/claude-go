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
				// When dynamic tool loading is active, prepend <available-deferred-tools>
				// to messages so the model knows which tools to discover via ToolSearch.
				if prep := toolsearchwire.PrepareMessagesForWire(msgsJSON, toolsForWire, model, false, false); len(prep) > 0 {
					var prepWire any
					if err := json.Unmarshal(prep, &prepWire); err == nil {
						msgsWire = prepWire
					}
				}
			}
		}

		req := map[string]any{
			"model":      model,
			"max_tokens": 4096,
			"messages":   msgsWire,
			"stream":     true,
		}
		// Build system prompt with attribution header, CLI prefix, and cache_control blocks (TS splitSysPromptPrefix).
		var sysArr SystemPrompt
		if h := GetAttributionHeader(); h != "" {
			sysArr = append(sysArr, h)
		}
		sysArr = append(sysArr, GetCLISyspromptPrefix())
		sysArr = append(sysArr, in.SystemPrompt...)
		if HasAdvisorModel() {
			sysArr = append(sysArr, ADVISOR_TOOL_INSTRUCTIONS)
		}
		if HasChromeTools(in.Tools) {
			sysArr = append(sysArr, CHROME_TOOL_SEARCH_INSTRUCTIONS)
		}
		blocks := SplitSysPromptPrefix(sysArr)
		if len(blocks) > 0 {
			var sysBlocks []map[string]any
			for _, b := range blocks {
				m := map[string]any{"type": "text", "text": b.Text}
				if b.CacheScope != nil {
					cc := map[string]any{"type": "ephemeral"}
					if *b.CacheScope == CacheScopeGlobal {
						cc["scope"] = "global"
					}
					m["cache_control"] = cc
				}
				sysBlocks = append(sysBlocks, m)
			}
			req["system"] = sysBlocks
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
				// Yield stream_event to UI for incremental display (mirrors TS claude.ts
				// which yields { type: 'stream_event', event: part } for every SSE event).
				if ev.Type == "content_block_delta" {
					if !yieldStreamingParity(ctx, deps, QueryYield{StreamEvent: ev.Raw}, yield) {
						return context.Canceled
					}
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
		var contentExtract struct{Content json.RawMessage `json:"content"`}; json.Unmarshal(inner, &contentExtract)
		asst := types.Message{
			Type:    types.MessageTypeAssistant,
			UUID:    asstUUID,
			Message: inner,
			Content: contentExtract.Content,
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

		// Consume memory prefetch after tool results (mirrors TS query.ts
		// collect point after toolResults.push). Attachment messages are
		// converted to <system-reminder>-wrapped user messages by
		// normalizeAttachmentForAPI in the next API round.
		if in.MemoryPrefetch != nil {
			for _, mm := range in.MemoryPrefetch.Poll() {
				mmCopy := mm
				if !yieldStreamingParity(ctx, deps, QueryYield{Message: &mmCopy}, yield) {
					ex.Discard()
					return context.Canceled
				}
				cur = append(cur, mmCopy)
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
