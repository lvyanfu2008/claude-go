package pui

import (
	"encoding/json"
	"fmt"
	"time"

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/gou/conversation"
	"goc/types"
)

const executionStubText = "gou-demo: execution_request (bash or slash from slashprepare/bashprepare) is not executed in TUI; inject ProcessBashCommand / ProcessSlashCommand for in-process handling."

// ApplyProcessUserInputBaseResult appends r.Messages to the store, or a stub system line when Execution / ExecutionSequence is set.
// When handoff is non-nil, fills it with [ProcessUserInputBaseResultHandoff] from r, or resets it on execution stub.
func ApplyProcessUserInputBaseResult(
	store *conversation.Store,
	r *processuserinput.ProcessUserInputBaseResult,
	handoff *ProcessUserInputBaseResultHandoff,
) ApplyProcessUserInputBaseResultOutcome {
	out := ApplyProcessUserInputBaseResultOutcome{}
	if r == nil {
		return out
	}
	if r.Execution != nil || len(r.ExecutionSequence) > 0 {
		store.AppendMessage(informationalSystem(executionStubText, "info"))
		if handoff != nil {
			handoff.reset()
		}
		out.HadExecutionRequest = true
		out.EffectiveShouldQuery = false
		return out
	}
	for i := range r.Messages {
		store.AppendMessage(r.Messages[i])
	}
	if r.ResultText != "" && len(r.Messages) == 0 {
		store.AppendMessage(informationalSystem(r.ResultText, "info"))
	}
	if handoff != nil {
		handoff.mergeFromResult(r)
	}
	out.EffectiveShouldQuery = r.ShouldQuery
	out.NextInput = r.NextInput
	out.SubmitNextInput = r.SubmitNextInput
	return out
}

// ApplyBaseResult is an alias for [ApplyProcessUserInputBaseResult] (shorter call sites).
func ApplyBaseResult(
	store *conversation.Store,
	r *processuserinput.ProcessUserInputBaseResult,
	handoff *ProcessUserInputBaseResultHandoff,
) ApplyProcessUserInputBaseResultOutcome {
	return ApplyProcessUserInputBaseResult(store, r, handoff)
}

// SystemNotice is a visible system/informational row for errors and stubs.
func SystemNotice(text string) types.Message {
	return informationalSystem(text, "info")
}

func informationalSystem(text, level string) types.Message {
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	raw, _ := json.Marshal(text)
	sub := "informational"
	return types.Message{
		Type:      types.MessageTypeSystem,
		UUID:      fmt.Sprintf("sys-%d", time.Now().UnixNano()),
		Content:   raw,
		Subtype:   &sub,
		Level:     &level,
		Timestamp: &ts,
	}
}

// SlashSkippedMessage is appended when the user enters a slash command in gou-demo (no TS slash runner).
func SlashSkippedMessage() types.Message {
	return SystemNotice("gou-demo: slash commands are not run here; use plain text or the TS REPL.")
}
