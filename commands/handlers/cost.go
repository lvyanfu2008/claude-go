package handlers

import (
	"encoding/json"
)

// CostResult is the JSON payload returned by /cost.
type CostResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleCostCommand returns session token usage for the /cost local command.
// In gou-demo this is a stub — the actual session store is not reachable from
// the handler registry. The gou-demo TUI wires a store-backed version directly.
func HandleCostCommand() ([]byte, error) {
	msg := CostResult{
		Type:  "text",
		Value: "Cost tracking is session-scoped in the TUI. Use /cost from the TS CLI for full cost breakdown.",
	}
	return json.Marshal(msg)
}
