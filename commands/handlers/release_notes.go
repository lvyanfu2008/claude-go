package handlers

import (
	"encoding/json"
	"fmt"
)

// ReleaseNotesResult is the JSON payload returned by /release-notes.
type ReleaseNotesResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleReleaseNotesCommand returns a pointer to the changelog for /release-notes.
// The TS version fetches from GitHub with a 500ms timeout. In gou-demo we show
// a static URL since there's no bundled changelog.
func HandleReleaseNotesCommand() ([]byte, error) {
	msg := ReleaseNotesResult{
		Type:  "text",
		Value: fmt.Sprintf("See the changelog at:\nhttps://github.com/anthropics/claude-code/blob/main/CHANGELOG.md"),
	}
	return json.Marshal(msg)
}
