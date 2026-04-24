package handlers

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"goc/tools/localtools"
)

// FilesResult is the JSON payload returned by /files.
type FilesResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleFilesCommand returns files currently tracked in ReadFileState,
// relativized against cwd. Mirrors TS src/commands/files/files.ts call().
// readFileState may be nil (no files tracked).
func HandleFilesCommand(rfs *localtools.ReadFileState, cwd string) ([]byte, error) {
	var keys []string
	if rfs != nil {
		keys = rfs.Keys()
	}
	if len(keys) == 0 {
		return json.Marshal(FilesResult{Type: "text", Value: "No files in context."})
	}
	sort.Strings(keys)
	var lines []string
	for _, k := range keys {
		rel, err := filepath.Rel(cwd, k)
		if err == nil && !strings.HasPrefix(rel, "..") {
			lines = append(lines, rel)
		} else {
			lines = append(lines, k)
		}
	}
	out := "Files in context:\n" + strings.Join(lines, "\n")
	return json.Marshal(FilesResult{Type: "text", Value: out})
}
