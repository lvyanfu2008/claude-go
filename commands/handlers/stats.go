package handlers

import (
	"encoding/json"
	"fmt"
)

// StatsResult is the JSON payload for /stats.
type StatsResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleStatsCommand handles /stats.
func HandleStatsCommand(args string) ([]byte, error) {
	return json.Marshal(StatsResult{
		Type: "text",
		Value: fmt.Sprintf("Session statistics:\n"+
			"  Model: check /model\n"+
			"  Permission mode: check /permissions\n"+
			"  Output format: check /output-style\n"+
			"  Theme: check /theme"),
	})
}
