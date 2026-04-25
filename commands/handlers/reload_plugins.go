package handlers

import (
	"encoding/json"
)

// ReloadPluginsResult is the JSON payload returned by /reload-plugins.
type ReloadPluginsResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleReloadPluginsCommand returns a notice about plugin reload for /reload-plugins.
// Mirrors TS src/commands/plugin/reload.ts (local-jsx with interactive plugin reload UI).
// gou-demo does not support plugin reloading.
func HandleReloadPluginsCommand() ([]byte, error) {
	msg := ReloadPluginsResult{
		Type:  "text",
		Value: "Plugin reload is not available in gou-demo. Changes to plugins will take effect on the next session. Use the TS CLI to reload plugins:\n  claude /reload-plugins    — Activate pending plugin changes",
	}
	return json.Marshal(msg)
}
