package paritytools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ReviewArtifactFromJSON implements the ReviewArtifactTool from TypeScript.
// Tool name: "ReviewArtifact"
// Description: "Review an artifact (code snippet, document, or other content) with inline annotations and feedback."
//
// TypeScript input schema:
// z.strictObject({
//   artifact: z.string().describe('The content of the artifact to review (code snippet, document text, etc.).'),
//   title: z.string().optional().describe('Optional title or file path for the artifact being reviewed.'),
//   annotations: z.array(
//     z.object({
//       line: z.number().optional().describe('Line number for the annotation (1-based).'),
//       message: z.string().describe('The annotation or feedback message.'),
//       severity: z.enum(['info', 'warning', 'error', 'suggestion']).optional().describe('Severity level of the annotation.'),
//     })
//   ).describe('List of annotations/comments on the artifact.'),
//   summary: z.string().optional().describe('An overall summary of the review.'),
// })
//
// TypeScript output schema:
// z.object({
//   artifact: z.string().describe('The reviewed artifact content.'),
//   title: z.string().optional().describe('Title of the reviewed artifact.'),
//   annotationCount: z.number().describe('Number of annotations applied.'),
//   summary: z.string().optional().describe('Summary of the review.'),
// })
func ReviewArtifactFromJSON(raw []byte) (string, bool, error) {
	var input struct {
		Artifact    string                     `json:"artifact"`
		Title       string                     `json:"title,omitempty"`
		Annotations []ReviewArtifactAnnotation `json:"annotations"`
		Summary     string                     `json:"summary,omitempty"`
	}

	if err := json.Unmarshal(raw, &input); err != nil {
		return "", true, fmt.Errorf("invalid input: %v", err)
	}

	// Validate required fields (matching TypeScript strictObject validation)
	if strings.TrimSpace(input.Artifact) == "" {
		return "", true, fmt.Errorf("artifact is required")
	}

	// Validate annotations if present
	for i, annotation := range input.Annotations {
		if strings.TrimSpace(annotation.Message) == "" {
			return "", true, fmt.Errorf("annotation[%d].message is required", i)
		}
		if annotation.Severity != "" {
			validSeverities := map[string]bool{
				"info":       true,
				"warning":    true,
				"error":      true,
				"suggestion": true,
			}
			if !validSeverities[annotation.Severity] {
				return "", true, fmt.Errorf("annotation[%d].severity must be one of: info, warning, error, suggestion", i)
			}
		}
		if annotation.Line != nil && *annotation.Line < 1 {
			return "", true, fmt.Errorf("annotation[%d].line must be >= 1 if provided", i)
		}
	}

	// Prepare output matching TypeScript output schema
	output := map[string]interface{}{
		"artifact":        input.Artifact,
		"annotationCount": len(input.Annotations),
	}

	// Add optional fields if present
	if input.Title != "" {
		output["title"] = input.Title
	}
	if input.Summary != "" {
		output["summary"] = input.Summary
	}

	// Wrap in "data" field to match TypeScript tool.call() return format
	// TypeScript returns: { data: output }
	result := map[string]interface{}{
		"data": output,
	}

	b, err := json.Marshal(result)
	if err != nil {
		return "", true, fmt.Errorf("failed to marshal output: %v", err)
	}

	return string(b), false, nil
}

// ReviewArtifactAnnotation represents a single annotation in the review.
// Mirrors TypeScript: { line?: number; message: string; severity?: 'info' | 'warning' | 'error' | 'suggestion' }
type ReviewArtifactAnnotation struct {
	Line     *int   `json:"line,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity,omitempty"`
}