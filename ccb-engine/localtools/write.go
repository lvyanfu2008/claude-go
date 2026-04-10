package localtools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteFromJSON writes file_path with content (creates parent directories).
func WriteFromJSON(raw []byte, roots []string) (string, bool, error) {
	var in struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	abs, err := ResolveUnderRoots(in.FilePath, roots)
	if err != nil {
		return "", true, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", true, err
	}
	if err := os.WriteFile(abs, []byte(in.Content), 0o644); err != nil {
		return "", true, err
	}
	return fmt.Sprintf("Wrote %d bytes to %s", len(in.Content), abs), false, nil
}
