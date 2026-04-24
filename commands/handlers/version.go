package handlers

import (
	"encoding/json"
	"runtime/debug"
)

// VersionResult is the JSON payload returned by /version.
type VersionResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleVersionCommand returns the gou-demo version string for /version.
func HandleVersionCommand() ([]byte, error) {
	value := "gou-demo (dev)"
	if bi, ok := debug.ReadBuildInfo(); ok {
		value = bi.Main.Version
		if value == "" || value == "(devel)" {
			value = "gou-demo (dev)"
		}
	}
	msg := VersionResult{
		Type:  "text",
		Value: value,
	}
	return json.Marshal(msg)
}
