package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var pushNotificationAllowedKeys = map[string]struct{}{
	"title":    {},
	"body":     {},
	"priority": {},
}

type pushNotificationZogInput struct {
	Title    string  `zog:"title"`
	Body     string  `zog:"body"`
	Priority *string `zog:"priority"`
}

func validatePushNotificationZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("push_notification: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := pushNotificationAllowedKeys[k]; !ok {
			return fmt.Errorf("push_notification: unknown field %q", k)
		}
	}

	var dest pushNotificationZogInput

	tRaw, ok := raw["title"]
	if !ok {
		return fmt.Errorf("push_notification: missing required field %q", "title")
	}
	var tVal any
	if err := json.Unmarshal(tRaw, &tVal); err != nil {
		return fmt.Errorf("push_notification: title: %w", err)
	}
	tStr, ok := tVal.(string)
	if !ok {
		return fmt.Errorf("push_notification: title must be a string")
	}
	if strings.TrimSpace(tStr) == "" {
		return fmt.Errorf("push_notification: title must be non-empty")
	}
	dest.Title = tStr

	bRaw, ok := raw["body"]
	if !ok {
		return fmt.Errorf("push_notification: missing required field %q", "body")
	}
	var bVal any
	if err := json.Unmarshal(bRaw, &bVal); err != nil {
		return fmt.Errorf("push_notification: body: %w", err)
	}
	bStr, ok := bVal.(string)
	if !ok {
		return fmt.Errorf("push_notification: body must be a string")
	}
	dest.Body = bStr

	if pr, ok := raw["priority"]; ok {
		var pv any
		if err := json.Unmarshal(pr, &pv); err != nil {
			return fmt.Errorf("push_notification: priority: %w", err)
		}
		if pv == nil {
			return fmt.Errorf("push_notification: priority cannot be null")
		}
		pStr, ok := pv.(string)
		if !ok {
			return fmt.Errorf("push_notification: priority must be a string")
		}
		if pStr != "normal" && pStr != "high" {
			return fmt.Errorf("push_notification: priority must be one of [normal, high]")
		}
		dest.Priority = &pStr
	}

	schema := z.Struct(z.Shape{
		"title":    z.String().Required(),
		"body":     z.String().Required(),
		"priority": z.String().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("push_notification: zog: %v", issues)
	}
	return nil
}
