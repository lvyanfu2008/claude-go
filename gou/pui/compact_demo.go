package pui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"goc/compactservice"
	"goc/conversation-runtime/query"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/gou/conversation"
	"goc/modelenv"
	"goc/types"
)

// handleCompactCommand compacts the conversation using the Anthropic API.
// It calls compactservice.CompactConversation and replaces store messages
// with the compaction result (boundary, summary, attachments, hooks).
func handleCompactCommand(store *conversation.Store) (*processuserinput.ProcessUserInputBaseResult, error) {
	if store == nil || len(store.Messages) < 2 {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{SystemNotice("Not enough messages to compact. Continue the conversation first.")},
			ShouldQuery: false,
		}, nil
	}

	model := strings.TrimSpace(os.Getenv("CLAUDE_MODEL"))
	if model == "" {
		model = modelenv.EffectiveMainLoopModel()
	}
	if model == "" {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{SystemNotice("Cannot compact: no model configured. Set CLAUDE_MODEL.")},
			ShouldQuery: false,
		}, nil
	}

	summarizer := query.SummarizeAutocompact

	deps := compactservice.Deps{
		Summarize: summarizer,
	}

	ctx := context.Background()
	result, err := compactservice.CompactConversation(ctx, store.Messages, deps, compactservice.CompactOptions{
		Model: model,
	})
	if err != nil {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{SystemNotice(fmt.Sprintf("Compaction failed: %v", err))},
			ShouldQuery: false,
		}, nil
	}

	postMsgs := compactservice.BuildPostCompactMessages(result)
	store.Messages = postMsgs
	store.ClearStreaming()
	store.StreamingToolUses = nil

	display := "Conversation compacted."
	if result.UserDisplayMessage != "" {
		display += " " + result.UserDisplayMessage
	}

	return &processuserinput.ProcessUserInputBaseResult{
		Messages:    []types.Message{SystemNotice(display)},
		ShouldQuery: false,
	}, nil
}

