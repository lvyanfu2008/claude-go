package localtools

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

var errNotGrepStructuredOutput = errors.New("not grep structured tool output")

// MapGrepToolOutputToToolResultContent mirrors GrepTool.mapToolResultToToolResultBlockParam
// (GrepTool.ts) for the tool_result block's string content. toolUseJSON is the marshalled
// structured grep output (same object TS stores on userMessage.toolUseResult).
func MapGrepToolOutputToToolResultContent(toolUseJSON string) (string, error) {
	toolUseJSON = strings.TrimSpace(toolUseJSON)
	if toolUseJSON == "" || toolUseJSON[0] != '{' {
		return "", errNotGrepStructuredOutput
	}
	var o struct {
		Mode          string   `json:"mode"`
		NumFiles      int      `json:"numFiles"`
		Filenames     []string `json:"filenames"`
		Content       string   `json:"content"`
		NumMatches    int      `json:"numMatches"`
		AppliedLimit  *int     `json:"appliedLimit"`
		AppliedOffset int      `json:"appliedOffset"`
	}
	if err := json.Unmarshal([]byte(toolUseJSON), &o); err != nil {
		return "", err
	}
	mode := o.Mode
	if mode == "" {
		mode = "files_with_matches"
	}
	limitInfo := GrepFormatLimitInfo(o.AppliedLimit, o.AppliedOffset)

	switch mode {
	case "content":
		resultContent := o.Content
		if resultContent == "" {
			resultContent = "No matches found"
		}
		if limitInfo != "" {
			return resultContent + "\n\n[Showing results with pagination = " + limitInfo + "]", nil
		}
		return resultContent, nil
	case "count":
		rawContent := o.Content
		if rawContent == "" {
			rawContent = "No matches found"
		}
		matches := o.NumMatches
		files := o.NumFiles
		summary := "\n\nFound " + strconv.Itoa(matches) + " total "
		if matches == 1 {
			summary += "occurrence"
		} else {
			summary += "occurrences"
		}
		summary += " across " + strconv.Itoa(files) + " "
		summary += grepPlural(files, "file", "files")
		if limitInfo != "" {
			summary += " with pagination = " + limitInfo
		}
		summary += "."
		return rawContent + summary, nil
	default: // files_with_matches
		if o.NumFiles == 0 {
			return "No files found", nil
		}
		suffix := ""
		if limitInfo != "" {
			suffix = " " + limitInfo
		}
		return "Found " + strconv.Itoa(o.NumFiles) + " " + grepPlural(o.NumFiles, "file", "files") + suffix + "\n" + strings.Join(o.Filenames, "\n"), nil
	}
}
