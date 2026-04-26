package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var sendMessageAllowedKeys = map[string]struct{}{
	"to":      {},
	"message": {},
}

type sendMessageZogInput struct {
	To      string `zog:"to"`
	Message string `zog:"message"`
}

func validateSendMessageZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("send_message: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := sendMessageAllowedKeys[k]; !ok {
			return fmt.Errorf("send_message: unknown field %q", k)
		}
	}

	var dest sendMessageZogInput

	toRaw, ok := raw["to"]
	if !ok {
		return fmt.Errorf("send_message: missing required field %q", "to")
	}
	var toVal any
	if err := json.Unmarshal(toRaw, &toVal); err != nil {
		return fmt.Errorf("send_message: to: %w", err)
	}
	toStr, ok := toVal.(string)
	if !ok {
		return fmt.Errorf("send_message: to must be a string")
	}
	if strings.TrimSpace(toStr) == "" {
		return fmt.Errorf("send_message: to must be non-empty")
	}
	dest.To = toStr

	mRaw, ok := raw["message"]
	if !ok {
		return fmt.Errorf("send_message: missing required field %q", "message")
	}
	var mVal any
	if err := json.Unmarshal(mRaw, &mVal); err != nil {
		return fmt.Errorf("send_message: message: %w", err)
	}
	mStr, ok := mVal.(string)
	if !ok {
		return fmt.Errorf("send_message: message must be a string")
	}
	dest.Message = mStr

	schema := z.Struct(z.Shape{
		"to":      z.String().Required(),
		"message": z.String().Required(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("send_message: zog: %v", issues)
	}
	return nil
}
