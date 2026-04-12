package query

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"goc/commands"
)

// UseOpenAIChatProvider mirrors TS getAPIProvider() === 'openai':
// CLAUDE_CODE_USE_OPENAI, or settings.json modelType "openai" (user + project, TS-shaped paths).
func UseOpenAIChatProvider() bool {
	if envTruthy("CLAUDE_CODE_USE_OPENAI") {
		return true
	}
	home := filepath.Join(commands.ClaudeConfigHome(), "settings.json")
	if modelTypeOpenAI(home) {
		return true
	}
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	proj := filepath.Join(cwd, ".claude", "settings.json")
	return modelTypeOpenAI(proj)
}

func modelTypeOpenAI(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		return false
	}
	var m struct {
		ModelType string `json:"modelType"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(m.ModelType), "openai")
}
