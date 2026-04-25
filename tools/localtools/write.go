package localtools

import (
	"encoding/json"
	"fmt"
	"os"
	pathpkg "path/filepath"
	"strings"
	"unicode/utf16"
)

// WriteDeps holds optional callbacks for Write tool parity features.
// Each field maps to a TS FileWriteTool feature; nil means "not available / skip".
type WriteDeps struct {
	// CheckWritePermission mirrors TS checkWritePermissionForTool (validateInput).
	// Called after path resolution, before stat. Returned non-nil DenyResult denies the write.
	CheckWritePermission func(filePath string) *DenyResult

	// CheckDenyRule mirrors TS matchingRuleForInput (validateInput).
	// Called after CheckWritePermission. Returned non-nil DenyResult denies the write.
	CheckDenyRule func(filePath string) *DenyResult

	// CheckSecrets mirrors TS checkTeamMemSecrets (validateInput).
	// Called after CheckDenyRule. Returned error string denies the write.
	CheckSecrets func(filePath, content string) string

	// GitDiffFn mirrors TS fetchSingleFileGitDiff (call).
	// Called after write succeeds, produces a git diff for telemetry.
	// Returned value is serialized into WriteOutput.GitDiff.
	GitDiffFn func(absPath string) (any, error)

	// OnFileChange mirrors TS lspManager.changeFile (call).
	// Called after write succeeds; signals to LSP that content changed.
	OnFileChange func(absPath, content string)

	// OnFileSave mirrors TS lspManager.saveFile (call).
	// Called after OnFileChange; signals to LSP that file was saved.
	OnFileSave func(absPath string)

	// OnFileUpdated mirrors TS notifyVscodeFileUpdated (call).
	// Called after LSP notifications; notifies VSCode of the file change.
	OnFileUpdated func(absPath, oldContent, newContent string)
}

// DenyResult carries the reason a write was denied (permissions / deny-rule).
type DenyResult struct {
	Message string
	Code    int
}

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
	GitDiff         any                   `json:"gitDiff,omitempty"` // TS ToolUseDiff; omitted in Go (no remote git diff infrastructure)
}

// WriteFromJSON mirrors FileWriteTool.validateInput + call (backward-compatible, no deps).
func WriteFromJSON(raw []byte, roots []string, state *ReadFileState) (string, bool, error) {
	return WriteFromJSONDeps(raw, roots, state, nil)
}

// WriteFromJSONDeps mirrors FileWriteTool.validateInput + call with optional dependency callbacks.
// When deps is nil or a callback is nil, the corresponding TS feature is skipped.
// See [FileWriteFeatureStatus] in filetool_parity.go for the parity matrix.
func WriteFromJSONDeps(raw []byte, roots []string, state *ReadFileState, deps *WriteDeps) (string, bool, error) {
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

	// pre-write: check write permission (TS validateInput checkPermissions → checkWritePermissionForTool)
	if deps != nil && deps.CheckWritePermission != nil {
		if dr := deps.CheckWritePermission(abs); dr != nil {
			return "", true, fmt.Errorf("Write permission denied: %s (code %d)", dr.Message, dr.Code)
		}
	}

	// pre-write: check deny rule (TS validateInput matchingRuleForInput)
	if deps != nil && deps.CheckDenyRule != nil {
		if dr := deps.CheckDenyRule(abs); dr != nil {
			return "", true, fmt.Errorf("File is in a directory that is denied by your permission settings: %s (code %d)", dr.Message, dr.Code)
		}
	}

	// pre-write: check team memory secrets (TS validateInput checkTeamMemSecrets)
	if deps != nil && deps.CheckSecrets != nil {
		if msg := deps.CheckSecrets(abs, in.Content); msg != "" {
			return "", true, fmt.Errorf("Team memory secret check failed: %s", msg)
		}
	}

	// validateInput: UNC — skip filesystem (TS returns { result: true }).
	if isUNCPath(abs) {
		return writeCall(in.FilePath, abs, in.Content, state, deps)
	}

	st, statErr := os.Stat(abs)
	if statErr != nil && !os.IsNotExist(statErr) {
		return "", true, statErr
	}
	if os.IsNotExist(statErr) {
		return writeCall(in.FilePath, abs, in.Content, state, deps)
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

	return writeCall(in.FilePath, abs, in.Content, state, deps)
}

// writeCall mirrors FileWriteTool.call (mkdir, atomic read, write, readFileState update).
func writeCall(filePath, abs, content string, state *ReadFileState, deps *WriteDeps) (string, bool, error) {
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

	// Post-write: LSP change notification (TS call lspManager.changeFile)
	if deps != nil && deps.OnFileChange != nil {
		deps.OnFileChange(abs, toWrite)
	}

	// Post-write: LSP save notification (TS call lspManager.saveFile)
	if deps != nil && deps.OnFileSave != nil {
		deps.OnFileSave(abs)
	}

	// Post-write: VSCode update notification (TS call notifyVscodeFileUpdated)
	if deps != nil && deps.OnFileUpdated != nil {
		deps.OnFileUpdated(abs, oldNorm, toWrite)
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

	// Post-write: git diff for telemetry (TS call fetchSingleFileGitDiff)
	if deps != nil && deps.GitDiffFn != nil {
		diff, dErr := deps.GitDiffFn(abs)
		if dErr == nil && diff != nil {
			out.GitDiff = diff
		}
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

// MapWriteToolResultToAssistantText mirrors FileWriteTool.mapToolResultToToolResultBlockParam.
// Converts a WriteOutput JSON to a short assistant-facing text summary.
func MapWriteToolResultToAssistantText(dataJSON string) (string, error) {
	var out WriteOutput
	if err := json.Unmarshal([]byte(dataJSON), &out); err != nil {
		return dataJSON, nil
	}
	switch out.Type {
	case "create":
		return fmt.Sprintf("File created successfully at: %s", out.FilePath), nil
	case "update":
		return fmt.Sprintf("The file %s has been updated successfully.", out.FilePath), nil
	default:
		return dataJSON, nil
	}
}
