package toolrefine

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var (
	regexpHtmlDocumentWrapper = regexp.MustCompile(`(?i)<\s*(html|body|!doctype)\b`)
	regexpScriptOrStyle       = regexp.MustCompile(`(?i)<\s*(script|style)\b`)
	regexpHtmlTag             = regexp.MustCompile(`(?i)<[a-z][^>]*>`)
)

// ValidateAskUserQuestionUniqueness mirrors AskUserQuestionTool.tsx UNIQUENESS_REFINE
// (Zod .refine) — JSON Schema from toolToAPISchema does not encode duplicate checks.
func ValidateAskUserQuestionUniqueness(input json.RawMessage) error {
	var p struct {
		Questions []struct {
			Question string `json:"question"`
			Options  []struct {
				Label string `json:"label"`
			} `json:"options"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(input, &p); err != nil {
		return nil
	}
	qs := make([]string, 0, len(p.Questions))
	for _, q := range p.Questions {
		qs = append(qs, q.Question)
	}
	if len(qs) != uniqueStringCount(qs) {
		return fmt.Errorf("AskUserQuestion: question texts must be unique")
	}
	for _, q := range p.Questions {
		labels := make([]string, 0, len(q.Options))
		for _, o := range q.Options {
			labels = append(labels, o.Label)
		}
		if len(labels) != uniqueStringCount(labels) {
			return fmt.Errorf("AskUserQuestion: option labels must be unique within each question")
		}
	}
	return nil
}

func uniqueStringCount(ss []string) int {
	seen := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		seen[s] = struct{}{}
	}
	return len(seen)
}

// ValidateAskUserQuestionHtmlPreview mirrors validateHtmlPreview in
// AskUserQuestionTool.tsx. Lightweight HTML fragment check — checks model intent
// (did it emit HTML?) and catches the specific things the model was told not to do.
// Returns an error message or empty string if valid.
func ValidateAskUserQuestionHtmlPreview(preview string) string {
	if preview == "" {
		return ""
	}
	if containsHtmlDocumentWrapper(preview) {
		return "preview must be an HTML fragment, not a full document (no <html>, <body>, or <!DOCTYPE>)"
	}
	if containsScriptOrStyle(preview) {
		return "preview must not contain <script> or <style> tags. Use inline styles via the style attribute if needed."
	}
	if !containsHtmlTag(preview) {
		return "preview must contain HTML (previewFormat is set to \"html\"). Wrap content in a tag like <div> or <pre>."
	}
	return ""
}

func containsHtmlDocumentWrapper(s string) bool {
	return regexpHtmlDocumentWrapper.MatchString(s)
}

func containsScriptOrStyle(s string) bool {
	return regexpScriptOrStyle.MatchString(s)
}

func containsHtmlTag(s string) bool {
	return regexpHtmlTag.MatchString(s)
}
