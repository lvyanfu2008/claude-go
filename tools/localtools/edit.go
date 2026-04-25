package localtools

import (
	"encoding/json"
	"fmt"
	"os"
	pathpkg "path/filepath"
	"strings"
)

const maxEditFileSize = 1024 * 1024 * 1024 // 1 GiB (TS FileEditTool)

// fileNotFoundCWDNote mirrors TS utils/file.ts FILE_NOT_FOUND_CWD_NOTE.
const fileNotFoundCWDNote = "Note: your current working directory is"

// EditDeps holds optional callbacks for Edit tool parity features.
// Each field maps to a TS FileEditTool feature; nil means "not available / skip".
type EditDeps struct {
	// CheckEditPermission mirrors TS checkWritePermissionForTool (validateInput).
	// Called after path resolution, before stat. Returned non-nil DenyResult denies the edit.
	CheckEditPermission func(filePath string) *DenyResult

	// CheckDenyRule mirrors TS matchingRuleForInput (validateInput).
	// Called after CheckEditPermission. Returned non-nil DenyResult denies the edit.
	CheckDenyRule func(filePath string) *DenyResult

	// CheckSecrets mirrors TS checkTeamMemSecrets (validateInput).
	// Called after computing the edit, before writing to disk. Returned error string denies the edit.
	CheckSecrets func(filePath, content string) string

	// CheckEditSettings mirrors TS validateSettingsJsonOnEdit (call).
	// Called before writing to disk; validates settings.json edits.
	// Returned error string denies the edit.
	CheckEditSettings func(filePath, oldContent, newContent string) string

	// GitDiffFn mirrors TS fetchSingleFileGitDiff (call).
	// Called after edit succeeds, produces a git diff for telemetry.
	GitDiffFn func(absPath string) (any, error)

	// OnFileChange mirrors TS lspManager.changeFile (call).
	// Called after edit succeeds; signals to LSP that content changed.
	OnFileChange func(absPath, content string)

	// OnFileSave mirrors TS lspManager.saveFile (call).
	// Called after OnFileChange; signals to LSP that file was saved.
	OnFileSave func(absPath string)

	// OnNotebookEdit handles Edit calls targeting .ipynb files.
	// When set, Edit redirects .ipynb edits to this callback instead of returning an error.
	// The callback receives the edit parameters and should perform a notebook edit.
	// When nil, the Edit tool returns an error telling the user to use NotebookEdit.
	OnNotebookEdit func(absPath, oldString, newString string, replaceAll bool, roots []string, state *ReadFileState, userModified bool) (string, bool, error)
}

// EditOutput mirrors FileEditTool FileEditOutput (JSON field names).
type EditOutput struct {
	FilePath        string                `json:"filePath"`
	OldString       string                `json:"oldString"`
	NewString       string                `json:"newString"`
	OriginalFile    string                `json:"originalFile"`
	StructuredPatch []StructuredPatchHunk `json:"structuredPatch"`
	UserModified    bool                  `json:"userModified"`
	ReplaceAll      bool                  `json:"replaceAll"`
	GitDiff         any                   `json:"gitDiff,omitempty"`
}

type editInput struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all"`
}

// EditFromJSON mirrors FileEditTool.validateInput + call (backward-compatible, no deps).
func EditFromJSON(raw []byte, roots []string, state *ReadFileState, userModified bool) (string, bool, error) {
	return EditFromJSONDeps(raw, roots, state, userModified, nil)
}

// EditFromJSONDeps mirrors FileEditTool.validateInput + call with optional dependency callbacks.
// When deps is nil or a callback is nil, the corresponding TS feature is skipped.
// See [FileEditFeatureStatus] in filetool_parity.go for the parity matrix.
func EditFromJSONDeps(raw []byte, roots []string, state *ReadFileState, userModified bool, deps *EditDeps) (string, bool, error) {
	var in editInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", true, err
	}
	if in.OldString == in.NewString {
		return "", true, fmt.Errorf("No changes to make: old_string and new_string are exactly the same.")
	}

	abs, err := ResolveUnderRoots(in.FilePath, roots)
	if err != nil {
		return "", true, err
	}

	// pre-edit: check edit permission (TS validateInput checkPermissions → checkWritePermissionForTool)
	if deps != nil && deps.CheckEditPermission != nil {
		if dr := deps.CheckEditPermission(abs); dr != nil {
			return "", true, fmt.Errorf("Edit permission denied: %s (code %d)", dr.Message, dr.Code)
		}
	}

	// pre-edit: check deny rule (TS validateInput matchingRuleForInput)
	if deps != nil && deps.CheckDenyRule != nil {
		if dr := deps.CheckDenyRule(abs); dr != nil {
			return "", true, fmt.Errorf("File is in a directory that is denied by your permission settings: %s (code %d)", dr.Message, dr.Code)
		}
	}

	// validateInput: UNC — skip filesystem (TS returns { result: true }).
	if isUNCPath(abs) {
		return editCallDeps(in, abs, state, userModified, deps)
	}

	// Prevent OOM on multi-GB files (TS stat before read).
	if st, err := os.Stat(abs); err == nil && !st.IsDir() && st.Size() > maxEditFileSize {
		return "", true, fmt.Errorf("File is too large to edit (%d bytes). Maximum editable file size is %d bytes.", st.Size(), maxEditFileSize)
	} else if err != nil && !os.IsNotExist(err) {
		return "", true, err
	}

	orig, exists, _, err := readFileForEdit(abs)
	if err != nil {
		return "", true, err
	}

	// File doesn't exist (TS: fileContent === null).
	if !exists {
		if in.OldString == "" {
			return editCallDeps(in, abs, state, userModified, deps)
		}
		cwd, _ := os.Getwd()
		return "", true, fmt.Errorf("File does not exist. %s %s.", fileNotFoundCWDNote, cwd)
	}

	// File exists with empty old_string — only valid if file is empty.
	if in.OldString == "" {
		if strings.TrimSpace(orig) != "" {
			return "", true, fmt.Errorf("Cannot create new file - file already exists.")
		}
		return editCallDeps(in, abs, state, userModified, deps)
	}

	// TS: fullFilePath.endsWith('.ipynb') → redirect to NotebookEdit
	if strings.HasSuffix(abs, ".ipynb") {
		if deps != nil && deps.OnNotebookEdit != nil {
			return deps.OnNotebookEdit(abs, in.OldString, in.NewString, in.ReplaceAll, roots, state, userModified)
		}
		return "", true, fmt.Errorf("File is a Jupyter Notebook. Use the NotebookEdit tool to edit this file.")
	}

	if state == nil {
		return "", true, fmt.Errorf("File has not been read yet. Read it first before writing to it.")
	}
	prev := state.Get(abs)
	if prev == nil || prev.IsPartialView {
		return "", true, fmt.Errorf("File has not been read yet. Read it first before writing to it.")
	}

	st2, e2 := os.Stat(abs)
	if e2 != nil {
		return "", true, e2
	}
	if st2.ModTime().UnixMilli() > prev.Timestamp {
		isFull := IsFullReadEntry(prev)
		if !(isFull && orig == prev.Content) {
			return "", true, fmt.Errorf("File has been modified since read, either by the user or by a linter. Read it again before attempting to write it.")
		}
	}

	actual := FindActualString(orig, in.OldString)
	if actual == "" {
		return "", true, fmt.Errorf("String to replace not found in file.\nString: %s", in.OldString)
	}
	matches := strings.Count(orig, actual)
	if matches > 1 && !in.ReplaceAll {
		return "", true, fmt.Errorf("Found %d matches of the string to replace, but replace_all is false. To replace all occurrences, set replace_all to true. To replace only one occurrence, please provide more context to uniquely identify the instance.\nString: %s", matches, in.OldString)
	}

	return editCallDeps(in, abs, state, userModified, deps)
}

// editCall mirrors FileEditTool.call (mkdir, atomic read, patch, write, readFileState, LSP, git diff).
func editCall(in editInput, abs string, state *ReadFileState, userModified bool) (string, bool, error) {
	return editCallDeps(in, abs, state, userModified, nil)
}

// editCallDeps mirrors FileEditTool.call with optional parity callbacks.
func editCallDeps(in editInput, abs string, state *ReadFileState, userModified bool, deps *EditDeps) (string, bool, error) {
	if err := os.MkdirAll(pathpkg.Dir(abs), 0o755); err != nil {
		return "", true, err
	}

	originalFileContents, fileExists, hadCRLF, err := readFileForEdit(abs)
	if err != nil {
		return "", true, err
	}

	if fileExists {
		st3, e3 := os.Stat(abs)
		if e3 != nil {
			return "", true, e3
		}
		lastWrite := st3.ModTime().UnixMilli()
		var prev *ReadFileEntry
		if state != nil {
			prev = state.Get(abs)
		}
		if prev == nil || lastWrite > prev.Timestamp {
			isFull := IsFullReadEntry(prev)
			contentUnchanged := isFull && originalFileContents == prev.Content
			if !contentUnchanged {
				return "", true, fmt.Errorf(fileUnexpectedlyModified)
			}
		}
	}

	actualOld := FindActualString(originalFileContents, in.OldString)
	if actualOld == "" {
		actualOld = in.OldString
	}
	newS := PreserveQuoteStyle(in.OldString, actualOld, in.NewString)

	patch, updated := GetPatchForEdit(in.FilePath, originalFileContents, actualOld, newS, in.ReplaceAll)
	if in.OldString == "" && !fileExists {
		updated = newS
		patch = GetPatchFromContents(in.FilePath, "", updated)
	}
	if updated == originalFileContents && fileExists {
		return "", true, fmt.Errorf("String not found in file. Failed to apply edit.")
	}

	// Pre-write: check team memory secrets (TS validateInput checkTeamMemSecrets)
	if deps != nil && deps.CheckSecrets != nil {
		if msg := deps.CheckSecrets(abs, updated); msg != "" {
			return "", true, fmt.Errorf("Team memory secret check failed: %s", msg)
		}
	}

	// Pre-write: validate settings file edits (TS validateSettingsJsonOnEdit)
	if deps != nil && deps.CheckEditSettings != nil {
		if msg := deps.CheckEditSettings(abs, originalFileContents, updated); msg != "" {
			return "", true, fmt.Errorf("Settings validation failed: %s", msg)
		}
	}

	outBytes := []byte(updated)
	if hadCRLF && fileExists {
		outBytes = []byte(strings.ReplaceAll(updated, "\n", "\r\n"))
	}
	if err := os.WriteFile(abs, outBytes, 0o644); err != nil {
		return "", true, err
	}

	st4, err := os.Stat(abs)
	if err != nil {
		return "", true, err
	}
	normalizedAfter := strings.ReplaceAll(string(outBytes), "\r\n", "\n")
	normalizedAfter = strings.ReplaceAll(normalizedAfter, "\r", "\n")
	if state != nil {
		state.Set(abs, &ReadFileEntry{
			Content:   normalizedAfter,
			Timestamp: st4.ModTime().UnixMilli(),
			Offset:    nil,
			Limit:     nil,
		})
	}

	// Post-write: LSP change notification (TS call lspManager.changeFile)
	if deps != nil && deps.OnFileChange != nil {
		deps.OnFileChange(abs, normalizedAfter)
	}

	// Post-write: LSP save notification (TS call lspManager.saveFile)
	if deps != nil && deps.OnFileSave != nil {
		deps.OnFileSave(abs)
	}

	var out EditOutput
	out.FilePath = in.FilePath
	out.OldString = actualOld
	out.NewString = in.NewString
	out.OriginalFile = originalFileContents
	out.StructuredPatch = patch
	out.UserModified = userModified
	out.ReplaceAll = in.ReplaceAll

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

// ValidateEditSettingsJSON validates whether edits to settings JSON files produce valid JSON.
// This is a default helper for the EditDeps.CheckEditSettings callback.
// It checks if the file path matches .claude/settings*.json patterns and validates the JSON.
// Returns an empty string if valid, or an error message if invalid.
func ValidateEditSettingsJSON(filePath, oldContent, newContent string) string {
	base := pathpkg.Base(filePath)
	if !strings.HasPrefix(base, "settings") || !strings.HasSuffix(base, ".json") {
		return ""
	}
	dir := pathpkg.Base(pathpkg.Dir(filePath))
	if dir != ".claude" {
		return ""
	}
	var v any
	if err := json.Unmarshal([]byte(newContent), &v); err != nil {
		return fmt.Sprintf("Invalid JSON in settings file %s: %v", filePath, err)
	}
	return ""
}

func readFileForEdit(abs string) (normalized string, exists bool, hadCRLF bool, err error) {
	st, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, false, nil
		}
		return "", false, false, err
	}
	if st.IsDir() {
		return "", false, false, fmt.Errorf("path is a directory: %s", abs)
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", false, false, err
	}
	raw := string(b)
	hadCRLF = strings.Contains(raw, "\r\n")
	s := decodeTextBytes(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s, true, hadCRLF, nil
}
