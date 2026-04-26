package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var writeAllowedKeys = map[string]struct{}{
	"file_path": {},
	"content":   {},
}

type writeZogInput struct {
	FilePath string `zog:"file_path"`
	Content  string `zog:"content"`
}

func validateWriteZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("write: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := writeAllowedKeys[k]; !ok {
			return fmt.Errorf("write: unknown field %q", k)
		}
	}

	fpRaw, ok := raw["file_path"]
	if !ok {
		return fmt.Errorf("write: missing required field %q", "file_path")
	}
	var fpVal any
	if err := json.Unmarshal(fpRaw, &fpVal); err != nil {
		return fmt.Errorf("write: file_path: %w", err)
	}
	fpStr, ok := fpVal.(string)
	if !ok {
		return fmt.Errorf("write: file_path must be a string")
	}
	if strings.TrimSpace(fpStr) == "" {
		return fmt.Errorf("write: file_path must be non-empty")
	}

	cRaw, ok := raw["content"]
	if !ok {
		return fmt.Errorf("write: missing required field %q", "content")
	}
	var cVal any
	if err := json.Unmarshal(cRaw, &cVal); err != nil {
		return fmt.Errorf("write: content: %w", err)
	}
	cStr, ok := cVal.(string)
	if !ok {
		return fmt.Errorf("write: content must be a string")
	}

	dest := writeZogInput{FilePath: fpStr, Content: cStr}
	schema := z.Struct(z.Shape{
		"file_path": z.String().Required(),
		"content":   z.String().Required(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("write: zog: %v", issues)
	}
	return nil
}
