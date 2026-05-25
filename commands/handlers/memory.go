package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MemoryResult is the JSON payload for /memory.
type MemoryResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleMemoryCommand handles /memory.
func HandleMemoryCommand(args string) ([]byte, error) {
	home, _ := os.UserHomeDir()
	memDir := filepath.Join(home, ".harness", "memory")
	if envDir := os.Getenv("CLAUDE_MEMORY_DIR"); envDir != "" {
		memDir = envDir
	}

	entries, err := os.ReadDir(memDir)
	if err != nil {
		return json.Marshal(MemoryResult{
			Type: "text",
			Value: fmt.Sprintf("No memory files found.\nMemory directory: %s", memDir),
		})
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Memory files (%s):", memDir))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			info, _ := entry.Info()
			lines = append(lines, fmt.Sprintf("  - %s (%d bytes)", entry.Name(), info.Size()))
		}
	}
	if len(lines) == 1 {
		return json.Marshal(MemoryResult{
			Type: "text", Value: fmt.Sprintf("No memory files found.\nMemory directory: %s", memDir),
		})
	}
	return json.Marshal(MemoryResult{
		Type: "text", Value: strings.Join(lines, "\n"),
	})
}
