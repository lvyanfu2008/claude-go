package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var snipAllowedKeys = map[string]struct{}{
	"message_ids": {},
	"reason":      {},
}

type snipZogInput struct {
	MessageIDs []string `zog:"message_ids"`
	Reason     *string  `zog:"reason"`
}

func validateSnipZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("snip: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := snipAllowedKeys[k]; !ok {
			return fmt.Errorf("snip: unknown field %q", k)
		}
	}

	var dest snipZogInput

	mr, ok := raw["message_ids"]
	if !ok {
		return fmt.Errorf("snip: missing required field %q", "message_ids")
	}
	var arr []any
	if err := json.Unmarshal(mr, &arr); err != nil {
		return fmt.Errorf("snip: message_ids must be an array of strings: %w", err)
	}
	if len(arr) == 0 {
		return fmt.Errorf("snip: message_ids must be non-empty")
	}
	for i, v := range arr {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("snip: message_ids[%d] must be a string", i)
		}
		dest.MessageIDs = append(dest.MessageIDs, s)
	}

	if err := parseZogStringField(raw, "reason", &dest.Reason); err != nil {
		return fmt.Errorf("snip: %w", err)
	}

	schema := z.Struct(z.Shape{
		"message_ids": z.Slice(z.String()).Required(),
		"reason":      z.String().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("snip: zog: %v", issues)
	}
	return nil
}
