package appstate

import "encoding/json"

// ElicitationWaitingState mirrors elicitationHandler.ElicitationWaitingState.
type ElicitationWaitingState struct {
	ActionLabel string `json:"actionLabel"`
	ShowCancel  *bool  `json:"showCancel,omitempty"`
}

// ElicitationRequestEventSnapshot mirrors elicitationHandler.ElicitationRequestEvent.
// signal, respond, and onWaitingDismiss are omitted.
type ElicitationRequestEventSnapshot struct {
	ServerName   string                   `json:"serverName"`
	RequestID    json.RawMessage          `json:"requestId"`
	Params       json.RawMessage          `json:"params"`
	WaitingState *ElicitationWaitingState `json:"waitingState,omitempty"`
	Completed    *bool                    `json:"completed,omitempty"`
}
