package handlers

import (
	"encoding/json"
)

// PluginsResult is the JSON payload returned by /plugins.
type PluginsResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandlePluginsCommand returns a notice about plugin management for /plugins.
// Mirrors TS src/commands/plugin/ (local-jsx with full marketplace UI).
// gou-demo does not support plugin management.
func HandlePluginsCommand() ([]byte, error) {
	msg := PluginsResult{
		Type:  "text",
		Value: "Plugin management is not available in gou-demo. Use the TS CLI to manage plugins:\n  claude /plugin         — Browse plugins\n  claude /plugin install <name>  — Install a plugin\n  claude /plugin list    — List installed plugins",
	}
	return json.Marshal(msg)
}
