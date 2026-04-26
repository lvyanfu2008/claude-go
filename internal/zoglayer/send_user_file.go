package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var sendUserFileAllowedKeys = map[string]struct{}{
	"file_path":   {},
	"description": {},
}

type sendUserFileZogInput struct {
	FilePath    string  `zog:"file_path"`
	Description *string `zog:"description"`
}

func validateSendUserFileZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("send_user_file: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := sendUserFileAllowedKeys[k]; !ok {
			return fmt.Errorf("send_user_file: unknown field %q", k)
		}
	}

	var dest sendUserFileZogInput

	fpRaw, ok := raw["file_path"]
	if !ok {
		return fmt.Errorf("send_user_file: missing required field %q", "file_path")
	}
	var fpVal any
	if err := json.Unmarshal(fpRaw, &fpVal); err != nil {
		return fmt.Errorf("send_user_file: file_path: %w", err)
	}
	fpStr, ok := fpVal.(string)
	if !ok {
		return fmt.Errorf("send_user_file: file_path must be a string")
	}
	if strings.TrimSpace(fpStr) == "" {
		return fmt.Errorf("send_user_file: file_path must be non-empty")
	}
	dest.FilePath = fpStr

	if err := parseZogStringField(raw, "description", &dest.Description); err != nil {
		return fmt.Errorf("send_user_file: %w", err)
	}

	schema := z.Struct(z.Shape{
		"file_path":   z.String().Required(),
		"description": z.String().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("send_user_file: zog: %v", issues)
	}
	return nil
}
