package toolresultpersist

import (
		"encoding/json"
		"errors"
		"fmt"
		"os"
		"strings"

		"goc/ccb-engine/diaglog"
)

// PersistedToolResult mirrors TS PersistedToolResult.
type PersistedToolResult struct {
	FilePath     string `json:"filepath"`
	OriginalSize int    `json:"originalSize"`
	IsJSON       bool   `json:"isJson"`
	Preview      string `json:"preview"`
	HasMore      bool   `json:"hasMore"`
}

// PersistToolResultError mirrors TS PersistToolResultError.
type PersistToolResultError struct {
	Error string `json:"error"`
}

// IsPersistError type guard (TS isPersistError).
func IsPersistError(r any) bool {
	_, ok := r.(*PersistToolResultError)
	return ok
}

// PersistToolResult mirrors TS persistToolResult(content, toolUseId).
// Writes content to {sessionDir}/tool-results/{toolUseId}.{json|txt} with 'exclusive create'
// semantics (os.O_EXCL) — identical content from microcompact replay is skipped.
func PersistToolResult(info SessionInfo, content any, toolUseID string) (*PersistedToolResult, *PersistToolResultError) {
	isJSON := false
	var contentStr string

	switch v := content.(type) {
	case string:
		contentStr = v
	case []any:
		isJSON = true
		// Check for non-text blocks (TS: hasNonTextContent guard)
		for _, block := range v {
			bm, ok := block.(map[string]any)
			if !ok {
				return nil, &PersistToolResultError{Error: "cannot persist tool results containing non-text content"}
			}
			typ, _ := bm["type"].(string)
			if typ != "text" {
				return nil, &PersistToolResultError{Error: "cannot persist tool results containing non-text content"}
			}
		}
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, &PersistToolResultError{Error: fmt.Sprintf("failed to marshal content: %v", err)}
		}
		contentStr = string(b)
	default:
		return nil, &PersistToolResultError{Error: fmt.Sprintf("unsupported content type: %T", content)}
	}

	if _, err := EnsureToolResultsDir(info); err != nil {
		diaglog.Line("[toolresultpersist] ensure dir: %v", err)
	}

	filePath := GetToolResultPath(info, toolUseID, isJSON)

	// Exclusive create: skip if already persisted (microcompact replay).
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			// EEXIST: already persisted on a prior turn, fall through to preview.
			diaglog.Line("[toolresultpersist] already persisted: %s", filePath)
		} else {
			return nil, &PersistToolResultError{Error: fileSystemErrorMessage(err)}
		}
	} else {
		if _, werr := f.WriteString(contentStr); werr != nil {
			f.Close()
			return nil, &PersistToolResultError{Error: fileSystemErrorMessage(werr)}
		}
		f.Close()
		diaglog.Line("[toolresultpersist] persisted: %s (%s)", filePath, formatFileSize(len(contentStr)))
	}

	preview, hasMore := GeneratePreview(contentStr, PreviewSizeBytes)

	return &PersistedToolResult{
		FilePath:     filePath,
		OriginalSize: len(contentStr),
		IsJSON:       isJSON,
		Preview:      preview,
		HasMore:      hasMore,
	}, nil
}

// BuildLargeToolResultMessage mirrors TS buildLargeToolResultMessage.
func BuildLargeToolResultMessage(result *PersistedToolResult) string {
	var b strings.Builder
	b.WriteString(PersistedOutputTag)
	b.WriteString("\n")
	fmt.Fprintf(&b, "Output too large (%s). Full output saved to: %s\n\n", formatFileSize(result.OriginalSize), result.FilePath)
	fmt.Fprintf(&b, "Preview (first %s):\n", formatFileSize(PreviewSizeBytes))
	b.WriteString(result.Preview)
	if result.HasMore {
		b.WriteString("\n...\n")
	} else {
		b.WriteString("\n")
	}
	b.WriteString(PersistedOutputClosingTag)
	return b.String()
}

// GeneratePreview mirrors TS generatePreview(content, maxBytes).
// Returns the first maxBytes of content, truncating at a newline boundary when possible.
func GeneratePreview(content string, maxBytes int) (preview string, hasMore bool) {
	if len(content) <= maxBytes {
		return content, false
	}
	truncated := content[:maxBytes]
	lastNewline := strings.LastIndex(truncated, "\n")
	// Use newline boundary if reasonably close to limit (within 50%)
	cutPoint := maxBytes
	if lastNewline > maxBytes/2 {
		cutPoint = lastNewline + 1 // include the newline
	}
	return content[:cutPoint], true
}

func formatFileSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
}

func fileSystemErrorMessage(err error) string {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		switch {
		case errors.Is(err, os.ErrNotExist):
			return fmt.Sprintf("Directory not found: %s", pathErr.Path)
		case errors.Is(err, os.ErrPermission):
			return fmt.Sprintf("Permission denied: %s", pathErr.Path)
		case errors.Is(err, os.ErrExist):
			return fmt.Sprintf("File already exists: %s", pathErr.Path)
		}
	}
	return err.Error()
}
