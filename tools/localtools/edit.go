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

// EditOutput mirrors FileEditTool FileEditOutput (JSON field names).
type EditOutput struct {
	FilePath        string                `json:"filePath"`
	OldString       string                `json:"oldString"`
	NewString       string                `json:"newString"`
	OriginalFile    string                `json:"originalFile"`
	StructuredPatch []StructuredPatchHunk `json:"structuredPatch"`
	UserModified    bool                  `json:"userModified"`
	ReplaceAll      bool                  `json:"replaceAll"`
}

type editInput struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all"`
}

// EditFromJSON mirrors FileEditTool.validateInput + call.
// Gaps vs TS: see [FileEditFeatureStatus] in filetool_parity.go (settings refine, permissions, LSP, …).
func EditFromJSON(raw []byte, roots []string, state *ReadFileState, userModified bool) (string, bool, error) {
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

	// validateInput: UNC — skip filesystem (TS returns { result: true }).
	if isUNCPath(abs) {
		return editCall(in, abs, state, userModified)
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
			return editCall(in, abs, state, userModified)
		}
		cwd, _ := os.Getwd()
		return "", true, fmt.Errorf("File does not exist. %s %s.", fileNotFoundCWDNote, cwd)
	}

	// File exists with empty old_string — only valid if file is empty.
	if in.OldString == "" {
		if strings.TrimSpace(orig) != "" {
			return "", true, fmt.Errorf("Cannot create new file - file already exists.")
		}
		return editCall(in, abs, state, userModified)
	}

	// TS: fullFilePath.endsWith('.ipynb')
	if strings.HasSuffix(abs, ".ipynb") {
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

	return editCall(in, abs, state, userModified)
}

// editCall mirrors FileEditTool.call (mkdir, atomic read, patch, write, readFileState).
func editCall(in editInput, abs string, state *ReadFileState, userModified bool) (string, bool, error) {
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

	var out EditOutput
	out.FilePath = in.FilePath
	out.OldString = actualOld
	out.NewString = in.NewString
	out.OriginalFile = originalFileContents
	out.StructuredPatch = patch
	out.UserModified = userModified
	out.ReplaceAll = in.ReplaceAll
	b, err := json.Marshal(out)
	if err != nil {
		return "", true, err
	}
	return string(b), false, nil
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
