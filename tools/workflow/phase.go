package workflow

import (
	"encoding/json"

	"goc/types"
)

// bindPhase returns a JS-callable function for phase(title).
func bindPhase(state *RunState, progressFn func(agentID, status, message string)) func(title string) {
	return func(title string) {
		state.CurrentPhase = title
		if state.ProgressCallback != nil {
			state.ProgressCallback(&types.Message{
				Type:    types.MessageTypeProgress,
				Content: json.RawMessage(`"phase: ` + title + `"`),
			})
		}
		if progressFn != nil {
			progressFn(state.RunID, "running", "Phase: "+title)
		}
	}
}

// bindLog returns a JS-callable function for log(message).
func bindLog(state *RunState, progressFn func(agentID, status, message string)) func(message string) {
	return func(message string) {
		if state.ProgressCallback != nil {
			state.ProgressCallback(&types.Message{
				Type:    types.MessageTypeProgress,
				Content: json.RawMessage(`"` + message + `"`),
			})
		}
		if progressFn != nil {
			progressFn(state.RunID, "running", message)
		}
	}
}
