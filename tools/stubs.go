package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var imageExtensionRegex = regexp.MustCompile(`(?i)\.(png|jpe?g|gif|webp)$`)

type resolvedAttachment struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsImage bool   `json:"isImage"`
}

// BriefFromJSON records a user-visible message path with resolved attachment metadata.
func BriefFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		Message     string   `json:"message"`
		Attachments []string `json:"attachments"`
		Status      string   `json:"status"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if strings.TrimSpace(in.Message) == "" {
		return "", true, fmt.Errorf("message is required")
	}
	st := strings.TrimSpace(in.Status)
	if st != "normal" && st != "proactive" {
		return "", true, fmt.Errorf("status must be normal or proactive")
	}

	// validate attachment paths
	for _, rawPath := range in.Attachments {
		fullPath, err := filepath.Abs(rawPath)
		if err != nil {
			return "", true, fmt.Errorf("Attachment %q: cannot resolve path: %w", rawPath, err)
		}
		fi, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				cwd, _ := os.Getwd()
				return "", true, fmt.Errorf("Attachment %q does not exist. Current working directory: %s.", rawPath, cwd)
			}
			if os.IsPermission(err) {
				return "", true, fmt.Errorf("Attachment %q is not accessible (permission denied).", rawPath)
			}
			return "", true, fmt.Errorf("Attachment %q: %w", rawPath, err)
		}
		if !fi.Mode().IsRegular() {
			return "", true, fmt.Errorf("Attachment %q is not a regular file.", rawPath)
		}
	}

	sentAt := time.Now().UTC().Format(time.RFC3339Nano)
	var data map[string]any
	if len(in.Attachments) == 0 {
		data = map[string]any{"message": in.Message, "sentAt": sentAt}
	} else {
		resolved := make([]resolvedAttachment, len(in.Attachments))
		for i, rawPath := range in.Attachments {
			fullPath, _ := filepath.Abs(rawPath)
			fi, _ := os.Stat(fullPath)
			resolved[i] = resolvedAttachment{
				Path:    fullPath,
				Size:    fi.Size(),
				IsImage: imageExtensionRegex.MatchString(fullPath),
			}
		}
		data = map[string]any{"message": in.Message, "attachments": resolved, "sentAt": sentAt}
	}
	out := map[string]any{"data": data}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// ListMcpResourcesFromJSON mirrors ListMcpResourcesTool.call with zero MCP clients (TS returns {data: []}).
// If server is set, TS throws when no client matches — same error text as TS.
func ListMcpResourcesFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		Server string `json:"server"`
	}
	_ = json.Unmarshal(raw, &in)
	target := strings.TrimSpace(in.Server)
	if target != "" {
		return "", true, fmt.Errorf(`Server "%s" not found. Available servers: `, target)
	}
	out := map[string]any{"data": []any{}}
	b, _ := json.Marshal(out)
	return string(b), false, nil
}

// ReadMcpResourceFromJSON mirrors ReadMcpResourceTool when no MCP client exists for server (TS throws).
func ReadMcpResourceFromJSON(raw []byte) (string, bool, error) {
	var in struct {
		Server string `json:"server"`
		URI    string `json:"uri"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	srv := strings.TrimSpace(in.Server)
	if srv == "" {
		return "", true, fmt.Errorf("server is required")
	}
	if strings.TrimSpace(in.URI) == "" {
		return "", true, fmt.Errorf("uri is required")
	}
	return "", true, fmt.Errorf(`Server "%s" not found. Available servers: `, srv)
}
