package localtools

import (
	"encoding/json"
	"fmt"
	"os"
	pathpkg "path/filepath"
	"strings"
	"unicode/utf16"
)

const fileUnexpectedlyModified = "File has been unexpectedly modified. Read it again before attempting to write it."

// isUNCPath mirrors FileWriteTool / FileEditTool validateInput: skip stat-based checks (NTLM safety).
func isUNCPath(abs string) bool {
	return strings.HasPrefix(abs, `\\`) || strings.HasPrefix(abs, "//")
}

// WriteOutput mirrors FileWriteTool Output (JSON).
type WriteOutput struct {
	Type            string                `json:"type"` // create | update
	FilePath        string                `json:"filePath"`
	Content         string                `json:"content"`
	StructuredPatch []StructuredPatchHunk `json:"structuredPatch"`
	OriginalFile    *string               `json:"originalFile"`
}

// WriteFromJSON mirrors FileWriteTool.validateInput + call.
// Gaps vs TS: see [FileWriteFeatureStatus] in filetool_parity.go (permissions, team memory, LSP, …).
func WriteFromJSON(raw []byte, roots []string, state *ReadFileState) (string, bool, error) {
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

	// validateInput: UNC — skip filesystem (TS returns { result: true }).
	if isUNCPath(abs) {
		return writeCall(in.FilePath, abs, in.Content, state)
	}

	st, statErr := os.Stat(abs)
	if statErr != nil && !os.IsNotExist(statErr) {
		return "", true, statErr
	}
	if os.IsNotExist(statErr) {
		return writeCall(in.FilePath, abs, in.Content, state)
	}
	if st.IsDir() {
		return "", true, fmt.Errorf("path is a directory: %s", abs)
	}

	// validateInput: existing file must have been read; stale mtime is strict (no content fallback).
	if state == nil {
		return "", true, fmt.Errorf("File has not been read yet. Read it first before writing to it.")
	}
	prev := state.Get(abs)
	if prev == nil || prev.IsPartialView {
		return "", true, fmt.Errorf("File has not been read yet. Read it first before writing to it.")
	}
	lastWrite := st.ModTime().UnixMilli()
	if lastWrite > prev.Timestamp {
		return "", true, fmt.Errorf("File has been modified since read, either by the user or by a linter. Read it again before attempting to write it.")
	}

	return writeCall(in.FilePath, abs, in.Content, state)
}

// writeCall mirrors FileWriteTool.call (mkdir, atomic read, write, readFileState update).
func writeCall(filePath, abs, content string, state *ReadFileState) (string, bool, error) {
	if err := os.MkdirAll(pathpkg.Dir(abs), 0o755); err != nil {
		return "", true, err
	}

	meta, err := readRawFileMeta(abs)
	if err != nil && !os.IsNotExist(err) {
		return "", true, err
	}
	exists := err == nil && !meta.isDir
	if exists {
		if state == nil {
			return "", true, fmt.Errorf(fileUnexpectedlyModified)
		}
		prev := state.Get(abs)
		if prev == nil {
			return "", true, fmt.Errorf(fileUnexpectedlyModified)
		}
		st2, e2 := os.Stat(abs)
		if e2 != nil {
			return "", true, e2
		}
		if st2.ModTime().UnixMilli() > prev.Timestamp {
			isFull := IsFullReadEntry(prev)
			if !isFull || meta.normalized != prev.Content {
				return "", true, fmt.Errorf(fileUnexpectedlyModified)
			}
		}
	}

	oldNorm := ""
	var origPtr *string
	if exists {
		oldNorm = meta.normalized
		origPtr = &oldNorm
	}

	toWrite := strings.ReplaceAll(content, "\r\n", "\n")
	toWrite = strings.ReplaceAll(toWrite, "\r", "\n")
	if err := os.WriteFile(abs, []byte(toWrite), 0o644); err != nil {
		return "", true, err
	}

	st3, err := os.Stat(abs)
	if err != nil {
		return "", true, err
	}
	if state != nil {
		state.Set(abs, &ReadFileEntry{
			Content:   toWrite,
			Timestamp: st3.ModTime().UnixMilli(),
			Offset:    nil,
			Limit:     nil,
		})
	}

	var out WriteOutput
	out.Content = content
	out.FilePath = filePath
	if exists {
		out.Type = "update"
		out.StructuredPatch = GetPatchFromContents(filePath, oldNorm, toWrite)
		out.OriginalFile = origPtr
	} else {
		out.Type = "create"
		out.StructuredPatch = []StructuredPatchHunk{}
		out.OriginalFile = nil
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", true, err
	}
	return string(b), false, nil
}

type rawFileMeta struct {
	normalized string
	isDir      bool
}

func readRawFileMeta(abs string) (rawFileMeta, error) {
	st, err := os.Stat(abs)
	if err != nil {
		return rawFileMeta{}, err
	}
	if st.IsDir() {
		return rawFileMeta{isDir: true}, nil
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return rawFileMeta{}, err
	}
	s := decodeTextBytes(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return rawFileMeta{normalized: s}, nil
}

func decodeTextBytes(b []byte) string {
	if len(b) >= 2 && b[0] == 0xff && b[1] == 0xfe {
		u16 := make([]uint16, 0, (len(b)-2)/2)
		for i := 2; i+1 < len(b); i += 2 {
			u16 = append(u16, uint16(b[i])|uint16(b[i+1])<<8)
		}
		return string(utf16ToRunes(u16))
	}
	return string(b)
}

func utf16ToRunes(u []uint16) []rune {
	return utf16.Decode(u)
}
