package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var globAllowedKeys = map[string]struct{}{
	"pattern": {},
	"path":    {},
}

type globZogInput struct {
	Pattern string  `zog:"pattern"`
	Path    *string `zog:"path"`
}

func validateGlobZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("glob: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := globAllowedKeys[k]; !ok {
			return fmt.Errorf("glob: unknown field %q", k)
		}
	}

	var dest globZogInput

	patRaw, ok := raw["pattern"]
	if !ok {
		return fmt.Errorf("glob: missing required field %q", "pattern")
	}
	var patVal any
	if err := json.Unmarshal(patRaw, &patVal); err != nil {
		return fmt.Errorf("glob: pattern: %w", err)
	}
	patStr, ok := patVal.(string)
	if !ok {
		return fmt.Errorf("glob: pattern must be a string")
	}
	if strings.TrimSpace(patStr) == "" {
		return fmt.Errorf("glob: pattern must be non-empty")
	}
	dest.Pattern = patStr

	if err := parseZogStringField(raw, "path", &dest.Path); err != nil {
		return err
	}

	schema := z.Struct(z.Shape{
		"pattern": z.String().Required(),
		"path":    z.String().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("glob: zog: %v", issues)
	}
	return nil
}

func parseZogStringField(raw map[string]json.RawMessage, key string, out **string) error {
	br, ok := raw[key]
	if !ok {
		return nil
	}
	var v any
	if err := json.Unmarshal(br, &v); err != nil {
		return fmt.Errorf("%s: %s: %w", "field", key, err)
	}
	if v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("%s: %s must be a string", "field", key)
	}
	cp := strings.TrimSpace(s)
	*out = &cp
	return nil
}
