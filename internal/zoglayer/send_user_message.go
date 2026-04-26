package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var sendUserMessageAllowedKeys = map[string]struct{}{
	"message":     {},
	"attachments": {},
	"status":      {},
}

type sendUserMessageZogInput struct {
	Message     string   `zog:"message"`
	Attachments []string `zog:"attachments"`
	Status      string   `zog:"status"`
}

func validateSendUserMessageZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("send_user_message: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := sendUserMessageAllowedKeys[k]; !ok {
			return fmt.Errorf("send_user_message: unknown field %q", k)
		}
	}

	var dest sendUserMessageZogInput

	mRaw, ok := raw["message"]
	if !ok {
		return fmt.Errorf("send_user_message: missing required field %q", "message")
	}
	var mVal any
	if err := json.Unmarshal(mRaw, &mVal); err != nil {
		return fmt.Errorf("send_user_message: message: %w", err)
	}
	mStr, ok := mVal.(string)
	if !ok {
		return fmt.Errorf("send_user_message: message must be a string")
	}
	dest.Message = mStr

	sRaw, ok := raw["status"]
	if !ok {
		return fmt.Errorf("send_user_message: missing required field %q", "status")
	}
	var sVal any
	if err := json.Unmarshal(sRaw, &sVal); err != nil {
		return fmt.Errorf("send_user_message: status: %w", err)
	}
	sStr, ok := sVal.(string)
	if !ok {
		return fmt.Errorf("send_user_message: status must be a string")
	}
	if sStr != "normal" && sStr != "proactive" {
		return fmt.Errorf("send_user_message: status must be one of [normal, proactive]")
	}
	dest.Status = sStr

	if ar, ok := raw["attachments"]; ok {
		var arr []any
		if err := json.Unmarshal(ar, &arr); err != nil {
			return fmt.Errorf("send_user_message: attachments must be an array of strings: %w", err)
		}
		for i, v := range arr {
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("send_user_message: attachments[%d] must be a string", i)
			}
			dest.Attachments = append(dest.Attachments, s)
		}
	}

	schema := z.Struct(z.Shape{
		"message": z.String().Required(),
		"status":  z.String().Required(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("send_user_message: zog: %v", issues)
	}
	return nil
}
