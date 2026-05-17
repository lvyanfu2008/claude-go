package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ConfigResult is the JSON payload for /config.
type ConfigResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// configKeys lists configurable env vars shown by /config.
var configKeys = []struct {
	Key  string
	Desc string
}{
	{"CLAUDE_CODE_MODEL", "Main model"},
	{"CLAUDE_CODE_PERMISSION_MODE", "Permission mode"},
	{"CLAUDE_CODE_OUTPUT_FORMAT", "Output format"},
	{"CLAUDE_CODE_THEME", "UI theme"},
	{"CLAUDE_CODE_FAST_MODE", "Fast mode"},
	{"CLAUDE_CODE_USE_OPENAI", "Use OpenAI provider"},
	{"CLAUDE_CODE_USE_GROK", "Use Grok provider"},
	{"OPENAI_API_KEY", "OpenAI API key (set: yes/no)"},
	{"GROK_API_KEY", "Grok API key (set: yes/no)"},
	{"CLAUDE_CODE_ADDITIONAL_DIRECTORIES", "Additional dirs"},
}

// HandleConfigCommand handles /config [key [value]].
func HandleConfigCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)

	if args == "" {
		var lines []string
		lines = append(lines, "Current configuration:")
		for _, c := range configKeys {
			val := os.Getenv(c.Key)
			if strings.Contains(strings.ToLower(c.Desc), "(set:") {
				if val != "" {
					val = "yes"
				} else {
					val = "no"
				}
			}
			if val == "" {
				val = "(not set)"
			}
			lines = append(lines, fmt.Sprintf("  %s: %s", c.Key, val))
		}
		lines = append(lines, "\nUse /config [key] [value] to change a setting.")
		return json.Marshal(ConfigResult{Type: "text", Value: strings.Join(lines, "\n")})
	}

	parts := strings.SplitN(args, " ", 2)
	key := strings.TrimSpace(parts[0])
	value := ""
	if len(parts) > 1 {
		value = strings.TrimSpace(parts[1])
	}

	if value == "" {
		cur := os.Getenv(key)
		return json.Marshal(ConfigResult{
			Type: "text",
			Value: fmt.Sprintf("%s = %s", key, cur),
		})
	}

	os.Setenv(key, value)
	return json.Marshal(ConfigResult{
		Type: "text",
		Value: fmt.Sprintf("Set %s = %s", key, value),
	})
}
