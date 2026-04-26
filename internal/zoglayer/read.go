package zoglayer

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	z "github.com/Oudwins/zog"
)

var readAllowedKeys = map[string]struct{}{
	"file_path": {},
	"offset":    {},
	"limit":     {},
	"pages":     {},
}

type readZogInput struct {
	FilePath string  `zog:"file_path"`
	Offset   *int    `zog:"offset"`
	Limit    *int    `zog:"limit"`
	Pages    *string `zog:"pages"`
}

func validateReadZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("read: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := readAllowedKeys[k]; !ok {
			return fmt.Errorf("read: unknown field %q", k)
		}
	}
	var dest readZogInput

	fpRaw, ok := raw["file_path"]
	if !ok {
		return fmt.Errorf("read: missing required field %q", "file_path")
	}
	var fpVal any
	if err := json.Unmarshal(fpRaw, &fpVal); err != nil {
		return fmt.Errorf("read: file_path: %w", err)
	}
	fpStr, ok := fpVal.(string)
	if !ok {
		return fmt.Errorf("read: file_path must be a string")
	}
	dest.FilePath = strings.TrimSpace(fpStr)
	if dest.FilePath == "" {
		return fmt.Errorf("read: file_path must be non-empty")
	}

	if err := parseZogOptionalInt(raw, "offset", &dest.Offset); err != nil {
		return err
	}
	if dest.Offset != nil && *dest.Offset < 0 {
		return fmt.Errorf("read: offset must be non-negative")
	}

	if err := parseZogOptionalInt(raw, "limit", &dest.Limit); err != nil {
		return err
	}
	if dest.Limit != nil && *dest.Limit <= 0 {
		return fmt.Errorf("read: limit must be positive")
	}

	if err := parseZogOptionalString(raw, "pages", &dest.Pages); err != nil {
		return err
	}

	schema := z.Struct(z.Shape{
		"file_path": z.String().Required(),
		"offset":    z.Int().Optional(),
		"limit":     z.Int().Optional(),
		"pages":     z.String().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("read: zog: %v", issues)
	}
	return nil
}

func parseZogOptionalString(raw map[string]json.RawMessage, key string, out **string) error {
	br, ok := raw[key]
	if !ok {
		return nil
	}
	var v any
	if err := json.Unmarshal(br, &v); err != nil {
		return fmt.Errorf("%s: %s: %w", "read", key, err)
	}
	if v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("%s: %s must be a string", "read", key)
	}
	cp := strings.TrimSpace(s)
	*out = &cp
	return nil
}

func parseZogOptionalInt(raw map[string]json.RawMessage, key string, out **int) error {
	br, ok := raw[key]
	if !ok {
		return nil
	}
	var v any
	if err := json.Unmarshal(br, &v); err != nil {
		return fmt.Errorf("%s: %s: %w", "read", key, err)
	}
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return fmt.Errorf("%s: %s must be a finite number", "read", key)
		}
		i := int(t)
		*out = &i
		return nil
	default:
		return fmt.Errorf("%s: %s must be a number", "read", key)
	}
}
