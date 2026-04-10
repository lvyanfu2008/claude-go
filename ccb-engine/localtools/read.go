package localtools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Read executes the Read tool: text file with optional 1-based line offset and line limit.
// pages (PDF) is not supported in the Go runner.
func Read(filePath string, roots []string, offset, limit int, pages string) (string, bool, error) {
	if strings.TrimSpace(pages) != "" {
		return "", true, fmt.Errorf("PDF pages not supported in Go local tool runner")
	}
	abs, err := ResolveUnderRoots(filePath, roots)
	if err != nil {
		return "", true, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", true, err
	}
	text := string(data)
	if offset <= 0 && limit <= 0 {
		return text, false, nil
	}
	lines := strings.Split(text, "\n")
	start := 0
	if offset > 1 {
		start = offset - 1
	}
	if start > len(lines) {
		return "", false, nil
	}
	end := len(lines)
	if limit > 0 {
		end = start + limit
		if end > len(lines) {
			end = len(lines)
		}
	}
	out := strings.Join(lines[start:end], "\n")
	return out, false, nil
}

// ReadFromJSON parses standard Read tool input and runs [Read].
func ReadFromJSON(raw []byte, roots []string) (string, bool, error) {
	var in struct {
		FilePath string `json:"file_path"`
		Offset   int    `json:"offset"`
		Limit    int    `json:"limit"`
		Pages    string `json:"pages"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	return Read(in.FilePath, roots, in.Offset, in.Limit, in.Pages)
}
