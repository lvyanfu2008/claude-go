package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ExportResult is the JSON payload for /export.
type ExportResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleExportCommand handles /export.
func HandleExportCommand(args string) ([]byte, error) {
	cwd, _ := os.Getwd()
	transcriptPath := filepath.Join(cwd, ".harness", "transcripts")

	return json.Marshal(ExportResult{
		Type: "text",
		Value: fmt.Sprintf("Session transcripts are stored at:\n  %s\n\n"+
			"To export a session, copy the transcript JSON file from that directory.\n"+
			"The transcript path can also be set via CLAUDE_CODE_TRANSCRIPT_DIR.", transcriptPath),
	})
}
