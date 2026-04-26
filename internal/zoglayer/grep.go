package zoglayer

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	z "github.com/Oudwins/zog"
)

var grepAllowedKeys = map[string]struct{}{
	"pattern":      {},
	"path":         {},
	"glob":         {},
	"output_mode":  {},
	"-B":           {},
	"-A":           {},
	"-C":           {},
	"context":      {},
	"-n":           {},
	"-i":           {},
	"type":         {},
	"head_limit":   {},
	"offset":       {},
	"multiline":    {},
}

type grepZogInput struct {
	Pattern    string  `zog:"pattern"`
	OutputMode *string `zog:"output_mode"`
	TypeFilter *string `zog:"type"`
	Path       *string `zog:"path"`
	Glob       *string `zog:"glob"`
}

// validateGrepZog enforces the same shape as TS GrepTool inputSchema
// (z.strictObject with semantic number/boolean coercions for some fields).
func validateGrepZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("grep: empty input")
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}

	// Strict keys: reject unknown fields (mirrors z.strictObject)
	for k := range raw {
		if _, ok := grepAllowedKeys[k]; !ok {
			return fmt.Errorf("grep: unknown field %q", k)
		}
	}

	// Required: pattern
	patRaw, hasPat := raw["pattern"]
	if !hasPat {
		return fmt.Errorf("grep: missing required field %q", "pattern")
	}
	var patVal any
	if err := json.Unmarshal(patRaw, &patVal); err != nil {
		return fmt.Errorf("grep: pattern: %w", err)
	}
	patStr, ok := patVal.(string)
	if !ok {
		return fmt.Errorf("grep: pattern must be a string")
	}
	if strings.TrimSpace(patStr) == "" {
		return fmt.Errorf("grep: pattern must be non-empty")
	}
	dest := grepZogInput{Pattern: patStr}

	// output_mode: optional enum
	if err := parseGrepOptionalEnum(raw, "output_mode", &dest.OutputMode, "content", "files_with_matches", "count"); err != nil {
		return err
	}

	// type: optional string
	if err := parseGrepOptionalString(raw, "type", &dest.TypeFilter); err != nil {
		return err
	}
	// path: optional string
	if err := parseGrepOptionalString(raw, "path", &dest.Path); err != nil {
		return err
	}
	// glob: optional string
	if err := parseGrepOptionalString(raw, "glob", &dest.Glob); err != nil {
		return err
	}

	// -B: optional number (semantic)
	if err := parseGrepOptionalNumber(raw, "-B"); err != nil {
		return err
	}
	// -A: optional number (semantic)
	if err := parseGrepOptionalNumber(raw, "-A"); err != nil {
		return err
	}
	// -C: optional number (semantic)
	if err := parseGrepOptionalNumber(raw, "-C"); err != nil {
		return err
	}
	// context: optional number (semantic)
	if err := parseGrepOptionalNumber(raw, "context"); err != nil {
		return err
	}
	// head_limit: optional number (semantic)
	if err := parseGrepOptionalNumber(raw, "head_limit"); err != nil {
		return err
	}
	// offset: optional number (semantic)
	if err := parseGrepOptionalNumber(raw, "offset"); err != nil {
		return err
	}

	// -n: optional boolean (semantic)
	if err := parseGrepOptionalBool(raw, "-n"); err != nil {
		return err
	}
	// -i: optional boolean (semantic)
	if err := parseGrepOptionalBool(raw, "-i"); err != nil {
		return err
	}
	// multiline: optional boolean (semantic)
	if err := parseGrepOptionalBool(raw, "multiline"); err != nil {
		return err
	}

	// Run zog validation for declared fields
	schema := z.Struct(z.Shape{
		"pattern":    z.String().Required(),
		"output_mode": z.String().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("grep: zog: %v", issues)
	}

	return nil
}

func parseGrepOptionalString(raw map[string]json.RawMessage, key string, out **string) error {
	br, ok := raw[key]
	if !ok {
		return nil
	}
	var v any
	if err := json.Unmarshal(br, &v); err != nil {
		return fmt.Errorf("grep: %s: %w", key, err)
	}
	if v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("grep: %s must be a string", key)
	}
	cp := strings.TrimSpace(s)
	*out = &cp
	return nil
}

func parseGrepOptionalEnum(raw map[string]json.RawMessage, key string, out **string, allowed ...string) error {
	br, ok := raw[key]
	if !ok {
		return nil
	}
	var v any
	if err := json.Unmarshal(br, &v); err != nil {
		return fmt.Errorf("grep: %s: %w", key, err)
	}
	if v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("grep: %s must be a string", key)
	}
	for _, a := range allowed {
		if s == a {
			cp := s
			*out = &cp
			return nil
		}
	}
	return fmt.Errorf("grep: %s must be one of [%s]", key, strings.Join(allowed, ", "))
}

func parseGrepOptionalNumber(raw map[string]json.RawMessage, key string) error {
	br, ok := raw[key]
	if !ok {
		return nil
	}
	var v any
	if err := json.Unmarshal(br, &v); err != nil {
		return fmt.Errorf("grep: %s: %w", key, err)
	}
	if v == nil {
		return nil
	}
	v = grepSemanticNumber(v)
	switch t := v.(type) {
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return fmt.Errorf("grep: %s must be a finite number", key)
		}
		return nil
	case int:
		return nil
	case int64:
		return nil
	default:
		return fmt.Errorf("grep: %s must be a number", key)
	}
}

func parseGrepOptionalBool(raw map[string]json.RawMessage, key string) error {
	br, ok := raw[key]
	if !ok {
		return nil
	}
	var v any
	if err := json.Unmarshal(br, &v); err != nil {
		return fmt.Errorf("grep: %s: %w", key, err)
	}
	v = grepSemanticBool(v)
	if v == nil {
		return fmt.Errorf("grep: %s cannot be null", key)
	}
	if _, ok := v.(bool); !ok {
		return fmt.Errorf("grep: %s must be a boolean", key)
	}
	return nil
}

var grepSemNumRe = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

// grepSemanticNumber mirrors TS semanticNumber for Grep input fields.
func grepSemanticNumber(v any) any {
	if s, ok := v.(string); ok && grepSemNumRe.MatchString(strings.TrimSpace(s)) {
		// json.Unmarshal into any always produces float64 for numbers
		return nil // signal: let json decoder handle it, this is just a coercion check
	}
	return v
}

// grepSemanticBool mirrors TS semanticBoolean for Grep input fields.
func grepSemanticBool(v any) any {
	if s, ok := v.(string); ok {
		switch strings.TrimSpace(strings.ToLower(s)) {
		case "true":
			return true
		case "false":
			return false
		}
	}
	return v
}
