package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ResumeResult is the JSON payload for /resume.
type ResumeResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleResumeCommand handles /resume — lists resumable sessions.
func HandleResumeCommand(args string) ([]byte, error) {
	cwd, _ := os.Getwd()
	transcriptDir := filepath.Join(cwd, ".harness", "transcripts")

	entries, err := os.ReadDir(transcriptDir)
	if err != nil {
		return json.Marshal(ResumeResult{
			Type: "text",
			Value: fmt.Sprintf("No saved sessions found.\nTranscript directory: %s\nUse --resume [session-id] to resume a session.", transcriptDir),
		})
	}

	type sessionEntry struct {
		name string
		time string
	}
	var sessions []sessionEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, _ := e.Info()
		sessions = append(sessions, sessionEntry{
			name: strings.TrimSuffix(e.Name(), ".jsonl"),
			time: info.ModTime().Format("2006-01-02 15:04"),
		})
	}

	if len(sessions) == 0 {
		return json.Marshal(ResumeResult{
			Type:  "text",
			Value: "No saved sessions found.\nUse --resume [session-id] to resume a previous session.",
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].time > sessions[j].time
	})

	var lines []string
	lines = append(lines, "Resumable sessions:")
	for _, s := range sessions {
		lines = append(lines, fmt.Sprintf("  %s  %s", s.time, s.name))
	}
	lines = append(lines, "\nUse --resume [session-id] to resume.")
	return json.Marshal(ResumeResult{Type: "text", Value: strings.Join(lines, "\n")})
}
