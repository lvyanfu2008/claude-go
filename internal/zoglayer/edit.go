package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var editAllowedKeys = map[string]struct{}{
	"file_path":   {},
	"old_string":  {},
	"new_string":  {},
	"replace_all": {},
}

type editZogInput struct {
	FilePath   string `zog:"file_path"`
	OldString  string `zog:"old_string"`
	NewString  string `zog:"new_string"`
	ReplaceAll *bool  `zog:"replace_all"`
}

func validateEditZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("edit: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := editAllowedKeys[k]; !ok {
			return fmt.Errorf("edit: unknown field %q", k)
		}
	}

	var dest editZogInput

	fpRaw, ok := raw["file_path"]
	if !ok {
		return fmt.Errorf("edit: missing required field %q", "file_path")
	}
	var fpVal any
	if err := json.Unmarshal(fpRaw, &fpVal); err != nil {
		return fmt.Errorf("edit: file_path: %w", err)
	}
	fpStr, ok := fpVal.(string)
	if !ok {
		return fmt.Errorf("edit: file_path must be a string")
	}
	if strings.TrimSpace(fpStr) == "" {
		return fmt.Errorf("edit: file_path must be non-empty")
	}
	dest.FilePath = fpStr

	osRaw, ok := raw["old_string"]
	if !ok {
		return fmt.Errorf("edit: missing required field %q", "old_string")
	}
	var osVal any
	if err := json.Unmarshal(osRaw, &osVal); err != nil {
		return fmt.Errorf("edit: old_string: %w", err)
	}
	osStr, ok := osVal.(string)
	if !ok {
		return fmt.Errorf("edit: old_string must be a string")
	}
	dest.OldString = osStr

	nsRaw, ok := raw["new_string"]
	if !ok {
		return fmt.Errorf("edit: missing required field %q", "new_string")
	}
	var nsVal any
	if err := json.Unmarshal(nsRaw, &nsVal); err != nil {
		return fmt.Errorf("edit: new_string: %w", err)
	}
	nsStr, ok := nsVal.(string)
	if !ok {
		return fmt.Errorf("edit: new_string must be a string")
	}
	dest.NewString = nsStr

	if br, ok := raw["replace_all"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("edit: replace_all: %w", err)
		}
		if v == nil {
			return fmt.Errorf("edit: replace_all cannot be null")
		}
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("edit: replace_all must be a boolean")
		}
		dest.ReplaceAll = &b
	}

	schema := z.Struct(z.Shape{
		"file_path":  z.String().Required(),
		"old_string": z.String().Required(),
		"new_string": z.String().Required(),
		"replace_all": z.Bool().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("edit: zog: %v", issues)
	}
	return nil
}
