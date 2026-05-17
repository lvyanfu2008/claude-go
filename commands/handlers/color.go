package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ColorResult is the JSON payload for /color.
type ColorResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleColorCommand handles /color [on|off].
func HandleColorCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)
	noColor := os.Getenv("NO_COLOR")
	forceColor := os.Getenv("FORCE_COLOR")
	enabled := noColor == "" || forceColor == "1"

	if args == "" {
		status := "enabled"
		if !enabled {
			status = "disabled"
		}
		return json.Marshal(ColorResult{
			Type:  "text",
			Value: fmt.Sprintf("Color output is %s.\nUse /color on or /color off to toggle.", status),
		})
	}

	switch strings.ToLower(args) {
	case "on", "enable", "true", "1":
		os.Unsetenv("NO_COLOR")
		os.Setenv("FORCE_COLOR", "1")
		return json.Marshal(ColorResult{
			Type: "text", Value: "Color output enabled.",
		})
	case "off", "disable", "false", "0":
		os.Setenv("NO_COLOR", "1")
		os.Unsetenv("FORCE_COLOR")
		return json.Marshal(ColorResult{
			Type: "text", Value: "Color output disabled.",
		})
	default:
		return json.Marshal(ColorResult{
			Type: "text", Value: "Usage: /color [on|off]",
		})
	}
}
