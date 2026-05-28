package localtools

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var errNotReviewArtifactOutput = errors.New("not ReviewArtifact structured tool output")

// MapReviewArtifactToolResultToAssistantText mirrors ReviewArtifactTool.mapToolResultToToolResultBlockParam
// (ReviewArtifactTool.ts) for the tool_result block's string content.
func MapReviewArtifactToolResultToAssistantText(toolUseJSON string) (string, error) {
	toolUseJSON = strings.TrimSpace(toolUseJSON)
	if toolUseJSON == "" || toolUseJSON[0] != '{' {
		return "", errNotReviewArtifactOutput
	}

	var wrapper struct {
		Data struct {
			AnnotationCount int    `json:"annotationCount"`
			Summary         string `json:"summary,omitempty"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(toolUseJSON), &wrapper); err != nil {
		return "", err
	}

	suffix := "s"
	if wrapper.Data.AnnotationCount == 1 {
		suffix = ""
	}
	mapped := fmt.Sprintf("Review delivered with %d annotation%s.", wrapper.Data.AnnotationCount, suffix)
	if wrapper.Data.Summary != "" {
		mapped += fmt.Sprintf(" Summary: %s", wrapper.Data.Summary)
	}
	return mapped, nil
}
