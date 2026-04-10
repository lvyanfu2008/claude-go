package localtools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// EditFromJSON replaces old_string with new_string in file_path (optionally all occurrences).
func EditFromJSON(raw []byte, roots []string) (string, bool, error) {
	var in struct {
		FilePath   string `json:"file_path"`
		OldString  string `json:"old_string"`
		NewString  string `json:"new_string"`
		ReplaceAll bool   `json:"replace_all"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	abs, err := ResolveUnderRoots(in.FilePath, roots)
	if err != nil {
		return "", true, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", true, err
	}
	s := string(data)
	if !strings.Contains(s, in.OldString) {
		return "", true, fmt.Errorf("old_string not found in file")
	}
	var out string
	if in.ReplaceAll {
		out = strings.ReplaceAll(s, in.OldString, in.NewString)
	} else {
		i := strings.Index(s, in.OldString)
		out = s[:i] + in.NewString + s[i+len(in.OldString):]
	}
	if err := os.WriteFile(abs, []byte(out), 0o644); err != nil {
		return "", true, err
	}
	return fmt.Sprintf("Updated %s (%d -> %d bytes)", abs, len(s), len(out)), false, nil
}
