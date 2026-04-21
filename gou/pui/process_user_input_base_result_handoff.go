package pui

import (
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/utils"
)

// ProcessUserInputBaseResultHandoff holds the non-messages fields of ProcessUserInputBaseResult after Apply,
// for the next engine / query step. Messages stay in conversation.Store.
//
// JSON field names match TS [ProcessUserInputBaseResult] in
// src/conversation-runtime/processUserInput/processUserInput.ts (camelCase).
//
// TS type (reference, same names in processUserInput.ts):
//
//	messages, shouldQuery, allowedTools?, model?, effort?, resultText?, nextInput?, submitNextInput?
type ProcessUserInputBaseResultHandoff struct {
	ShouldQuery     bool               `json:"shouldQuery"`
	AllowedTools    []string           `json:"allowedTools,omitempty"`
	Model           string             `json:"model,omitempty"`
	Effort          *utils.EffortValue `json:"effort,omitempty"`
	ResultText      string             `json:"resultText,omitempty"`
	NextInput       string             `json:"nextInput,omitempty"`
	SubmitNextInput bool               `json:"submitNextInput"`
}

// HandoffFromProcessUserInputBaseResult copies scalar fields from r into a new handoff value.
func HandoffFromProcessUserInputBaseResult(r *processuserinput.ProcessUserInputBaseResult) ProcessUserInputBaseResultHandoff {
	if r == nil {
		return ProcessUserInputBaseResultHandoff{}
	}
	h := ProcessUserInputBaseResultHandoff{
		ShouldQuery:     r.ShouldQuery,
		Model:           r.Model,
		ResultText:      r.ResultText,
		NextInput:       r.NextInput,
		SubmitNextInput: r.SubmitNextInput,
		Effort:          r.Effort,
	}
	if len(r.AllowedTools) > 0 {
		h.AllowedTools = append([]string(nil), r.AllowedTools...)
	}
	return h
}

func (h *ProcessUserInputBaseResultHandoff) reset() {
	if h == nil {
		return
	}
	*h = ProcessUserInputBaseResultHandoff{}
}

func (h *ProcessUserInputBaseResultHandoff) mergeFromResult(r *processuserinput.ProcessUserInputBaseResult) {
	if h == nil || r == nil {
		return
	}
	*h = HandoffFromProcessUserInputBaseResult(r)
}

// ApplyProcessUserInputBaseResultOutcome is the return shape of ApplyProcessUserInputBaseResult (Go-only;
// TS has no single struct—see executeUserInput materialization in handlePromptSubmit.ts).
//
// nextInput / submitNextInput mirror ProcessUserInputBaseResult when present.
type ApplyProcessUserInputBaseResultOutcome struct {
	// EffectiveShouldQuery is false when deferred execution was stubbed or r.ShouldQuery was false.
	EffectiveShouldQuery bool `json:"effectiveShouldQuery"`
	// HadExecutionRequest is true when r.Execution or r.ExecutionSequence was set (host must run bash/slash outside this apply path).
	HadExecutionRequest bool `json:"hadExecutionRequest"`
	NextInput           string `json:"nextInput,omitempty"`
	SubmitNextInput     bool   `json:"submitNextInput"`
}
