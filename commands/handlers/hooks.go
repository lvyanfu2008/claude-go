package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HooksResult is the JSON payload for /hooks.
type HooksResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleHooksCommand handles /hooks — lists configured hooks.
func HandleHooksCommand(args string) ([]byte, error) {
	cwd, _ := os.Getwd()
	hooksDir := filepath.Join(cwd, ".harness", "hooks")

	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		return json.Marshal(HooksResult{
			Type: "text",
			Value: fmt.Sprintf("No hooks configured.\nHooks directory: %s\n\nAdd executable scripts to %s to register hooks.", hooksDir, hooksDir),
		})
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Hooks directory: %s", hooksDir))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, _ := e.Info()
		exec := ""
		if info.Mode()&0111 != 0 {
			exec = " [executable]"
		}
		lines = append(lines, fmt.Sprintf("  %s%s", e.Name(), exec))
	}
	if len(lines) == 1 {
		return json.Marshal(HooksResult{
			Type: "text",
			Value: fmt.Sprintf("No hooks found in %s", hooksDir),
		})
	}
	return json.Marshal(HooksResult{
		Type: "text", Value: strings.Join(lines, "\n"),
	})
}
