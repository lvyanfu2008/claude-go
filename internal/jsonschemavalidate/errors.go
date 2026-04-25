package jsonschemavalidate

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// FormatInputValidationError formats a JSON Schema validation error into a structured,
// human-readable message that mirrors formatZodValidationError in src/utils/toolErrors.ts.
//
// The output categorizes issues as:
//   - The required parameter `X` is missing
//   - An unexpected parameter `X` was provided
//   - The parameter `X` type is expected as `Y` but provided as `Z`
//
// When categorization is not possible, the raw error text is used as a fallback.
func FormatInputValidationError(toolName string, err error) string {
	var valErr *jsonschema.ValidationError
	if !asValidationError(err, &valErr) {
		return fmt.Sprintf("%s failed due to the following issue(s):\n%s", toolName, err.Error())
	}

	issues := collectIssues(valErr, nil)
	if len(issues) == 0 {
		return fmt.Sprintf("%s failed due to the following issue(s):\n%s", toolName, err.Error())
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s failed due to the following issue(s):", toolName))
	for _, iss := range issues {
		sb.WriteString("\n")
		sb.WriteString(iss.text)
	}
	return sb.String()
}

// validationIssue is a categorized validation error line.
type validationIssue struct {
	text      string
	category  string // "missing", "unexpected", "type", "other"
	fieldPath string // for deduplication
}

func collectIssues(valErr *jsonschema.ValidationError, pathPrefix []string) []validationIssue {
	var issues []validationIssue
	effectivePath := append(pathPrefix, valErr.InstanceLocation...)

	switch k := valErr.ErrorKind.(type) {
	case *kind.Group:
		// Group is a container — recurse into causes.
		for _, cause := range valErr.Causes {
			issues = append(issues, collectIssues(cause, pathPrefix)...)
		}

	case *kind.Required:
		for _, m := range k.Missing {
			issues = append(issues, validationIssue{
				text:      fmt.Sprintf("The required parameter `%s` is missing", m),
				category:  "missing",
				fieldPath: joinPath(append(effectivePath, m)),
			})
		}

	case *kind.AdditionalProperties:
		for _, p := range k.Properties {
			issues = append(issues, validationIssue{
				text:      fmt.Sprintf("An unexpected parameter `%s` was provided", p),
				category:  "unexpected",
				fieldPath: joinPath(append(effectivePath, p)),
			})
		}

	case *kind.Type:
		path := joinPath(effectivePath)
		if path == "" {
			path = "input"
		}
		want := strings.Join(k.Want, " or ")
		issues = append(issues, validationIssue{
			text:      fmt.Sprintf("The parameter `%s` type is expected as `%s` but provided as `%s`", path, want, k.Got),
			category:  "type",
			fieldPath: joinPath(effectivePath),
		})

	default:
		// Recurse into causes first (deeper specific errors are more useful).
		if len(valErr.Causes) > 0 {
			for _, cause := range valErr.Causes {
				issues = append(issues, collectIssues(cause, effectivePath)...)
			}
		} else {
			raw := valErr.Error()
			if raw == "" {
				raw = fmt.Sprintf("validation failed at %s", joinPath(effectivePath))
			}
			issues = append(issues, validationIssue{
				text:      raw,
				category:  "other",
				fieldPath: joinPath(effectivePath),
			})
		}
	}

	return issues
}

// joinPath converts a slice of JSON path segments into a human-readable string.
// Examples: [] → ""; ["count"] → "count"; ["todos", "0", "name"] → "todos[0].name"
func joinPath(segments []string) string {
	if len(segments) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, s := range segments {
		if i == 0 {
			sb.WriteString(s)
		} else if isArrayIndex(s) {
			sb.WriteString("[" + s + "]")
		} else {
			sb.WriteString("." + s)
		}
	}
	return sb.String()
}

func isArrayIndex(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.Atoi(s)
	return err == nil
}

// FormatValidationError wraps FormatInputValidationError but returns the raw issue
// text without a tool name prefix, suitable for embedding as a hint string.
func FormatValidationError(err error) string {
	return formatValidationErrorRaw(err)
}

func formatValidationErrorRaw(err error) string {
	var valErr *jsonschema.ValidationError
	if !asValidationError(err, &valErr) {
		return err.Error()
	}
	issues := collectIssues(valErr, nil)
	if len(issues) == 0 {
		return err.Error()
	}
	lines := make([]string, 0, len(issues))
	for _, iss := range issues {
		lines = append(lines, iss.text)
	}
	return strings.Join(lines, "\n")
}

// asValidationError extracts a *jsonschema.ValidationError from err,
// unwrapping fmt.Errorf-wrapped errors.
func asValidationError(err error, target **jsonschema.ValidationError) bool {
	if err == nil {
		return false
	}
	return errors.As(err, target)
}
