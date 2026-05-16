package processuserinput

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"goc/tools/localtools"
	"goc/types"
)

// atMentionMaxLines is the default line limit when reading @-mentioned files.
const atMentionMaxLines = 2000

var (
	quotedAtMentionRe  = regexp.MustCompile(`(^|\s)@"([^"]+)"`)
	regularAtMentionRe = regexp.MustCompile(`(^|\s)@([^\s]+)\b`)
	lineRangeRe         = regexp.MustCompile(`^([^#]+)(?:#L(\d+)(?:-(\d+))?)?(?:#[^#]*)?$`)
)

func extractAtMentionedFiles(content string) []string {
	seen := make(map[string]struct{})
	var result []string

	// Quoted mentions: @"path with spaces"
	for _, m := range quotedAtMentionRe.FindAllStringSubmatch(content, -1) {
		if len(m) >= 3 && !strings.HasSuffix(m[2], " (agent)") {
			f := m[2]
			if _, ok := seen[f]; !ok {
				seen[f] = struct{}{}
				result = append(result, f)
			}
		}
	}

	// Regular mentions: @filename
	for _, m := range regularAtMentionRe.FindAllStringSubmatch(content, -1) {
		if len(m) >= 3 {
			f := m[2]
			if strings.HasPrefix(f, `"`) {
				continue
			}
			if _, ok := seen[f]; !ok {
				seen[f] = struct{}{}
				result = append(result, f)
			}
		}
	}

	return result
}

func parseAtMentionedFileLines(mention string) (filename string, lineStart, lineEnd int) {
	m := lineRangeRe.FindStringSubmatch(mention)
	if m == nil {
		return mention, 0, 0
	}
	filename = m[1]
	if filename == "" {
		filename = mention
	}
	if m[2] != "" {
		fmt.Sscanf(m[2], "%d", &lineStart)
	}
	if m[3] != "" {
		fmt.Sscanf(m[3], "%d", &lineEnd)
	} else if lineStart > 0 {
		lineEnd = lineStart
	}
	return
}

func resolveAtMentionedFiles(
	ctx context.Context,
	input string,
	cwd string,
) ([]types.Message, error) {
	files := extractAtMentionedFiles(input)
	if len(files) == 0 {
		return nil, nil
	}

	var msgs []types.Message
	for _, file := range files {
		filename, lineStart, lineEnd := parseAtMentionedFileLines(file)

		abs, err := localtools.ExpandPath(filename, cwd)
		if err != nil {
			continue
		}

		st, err := os.Stat(abs)
		if err != nil {
			continue
		}
		if st.IsDir() {
			// Directories: list entries (max 1000)
			entries, err := os.ReadDir(abs)
			if err != nil {
				continue
			}
			const maxDirEntries = 1000
			truncated := len(entries) > maxDirEntries
			if truncated {
				entries = entries[:maxDirEntries]
			}
			names := make([]string, len(entries))
			for i, e := range entries {
				names[i] = e.Name()
			}
			if truncated {
				names = append(names, fmt.Sprintf("… and %d more entries", len(entries)-maxDirEntries))
			}
			stdout := strings.Join(names, "\n")
			displayPath, _ := filepath.Rel(cwd, abs)
			att := fileAttachmentJSON{
				Type:        "directory",
				Path:        abs,
				Content:     stdout,
				DisplayPath: displayPath,
			}
			raw, _ := json.Marshal(att)
			msgs = append(msgs, types.Message{
				Type:       types.MessageTypeAttachment,
				UUID:       randomUUID(),
				Attachment: raw,
				Timestamp:  timePtr(time.Now().UTC().Format(time.RFC3339)),
			})
			continue
		}

		// Read file content
		content, err := readFileLines(abs, lineStart, lineEnd)
		if err != nil {
			continue
		}

		displayPath, _ := filepath.Rel(cwd, abs)
		att := fileAttachmentJSON{
			Type:        "file",
			Filename:    abs,
			Content:     content,
			DisplayPath: displayPath,
		}
		if lineStart > 0 {
			att.Offset = &lineStart
			if lineEnd > lineStart {
				lim := lineEnd - lineStart + 1
				att.Limit = &lim
			}
		}
		raw, _ := json.Marshal(att)
		msgs = append(msgs, types.Message{
			Type:       types.MessageTypeAttachment,
			UUID:       randomUUID(),
			Attachment: raw,
			Timestamp:  timePtr(time.Now().UTC().Format(time.RFC3339)),
		})
	}

	return msgs, nil
}

type fileAttachmentJSON struct {
	Type        string `json:"type"`
	Filename    string `json:"filename,omitempty"`
	Path        string `json:"path,omitempty"`
	Content     string `json:"content"`
	DisplayPath string `json:"displayPath"`
	Offset      *int   `json:"offset,omitempty"`
	Limit       *int   `json:"limit,omitempty"`
}

// readFileLines reads a file, optionally constrained to a line range.
// lineStart is 1-based; lineEnd is 1-based inclusive.
func readFileLines(abs string, lineStart, lineEnd int) (string, error) {
	raw, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	content := string(raw)

	if lineStart <= 0 {
		// Read from beginning, truncated to atMentionMaxLines
		lines := strings.Split(content, "\n")
		if len(lines) > atMentionMaxLines {
			content = strings.Join(lines[:atMentionMaxLines], "\n")
		}
		return content, nil
	}

	// Line range mode
	lines := strings.Split(content, "\n")
	start := lineStart - 1
	if start >= len(lines) {
		return "", fmt.Errorf("line %d is beyond file end (%d lines)", lineStart, len(lines))
	}
	end := lineEnd
	if end <= 0 || end > len(lines) {
		end = len(lines)
	}
	// Clamp to max lines
	if end-start > atMentionMaxLines {
		end = start + atMentionMaxLines
	}
	content = strings.Join(lines[start:end], "\n")
	return content, nil
}

func timePtr(s string) *string {
	return &s
}

// NewDefaultGetAttachmentMessages returns a GetAttachmentMessages callback that
// resolves @-mentioned files from user input text. It matches the TS behavior of
// extractAtMentionedFiles + resolveAtMentionedFiles.
func NewDefaultGetAttachmentMessages(cwd string) func(
	ctx context.Context,
	inputString string,
	ideSelection *types.IDESelection,
	priorMessages []types.Message,
	querySource types.QuerySource,
) ([]types.Message, error) {
	return func(
		ctx context.Context,
		inputString string,
		ideSelection *types.IDESelection,
		priorMessages []types.Message,
		querySource types.QuerySource,
	) ([]types.Message, error) {
		return resolveAtMentionedFiles(ctx, inputString, cwd)
	}
}
