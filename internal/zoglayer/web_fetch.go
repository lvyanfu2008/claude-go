package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var webFetchAllowedKeys = map[string]struct{}{
	"url":    {},
	"prompt": {},
}

type webFetchZogInput struct {
	URL    string `zog:"url"`
	Prompt string `zog:"prompt"`
}

func validateWebFetchZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("web_fetch: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := webFetchAllowedKeys[k]; !ok {
			return fmt.Errorf("web_fetch: unknown field %q", k)
		}
	}

	var dest webFetchZogInput

	uRaw, ok := raw["url"]
	if !ok {
		return fmt.Errorf("web_fetch: missing required field %q", "url")
	}
	var uVal any
	if err := json.Unmarshal(uRaw, &uVal); err != nil {
		return fmt.Errorf("web_fetch: url: %w", err)
	}
	uStr, ok := uVal.(string)
	if !ok {
		return fmt.Errorf("web_fetch: url must be a string")
	}
	if strings.TrimSpace(uStr) == "" {
		return fmt.Errorf("web_fetch: url must be non-empty")
	}
	dest.URL = uStr

	pRaw, ok := raw["prompt"]
	if !ok {
		return fmt.Errorf("web_fetch: missing required field %q", "prompt")
	}
	var pVal any
	if err := json.Unmarshal(pRaw, &pVal); err != nil {
		return fmt.Errorf("web_fetch: prompt: %w", err)
	}
	pStr, ok := pVal.(string)
	if !ok {
		return fmt.Errorf("web_fetch: prompt must be a string")
	}
	dest.Prompt = pStr

	schema := z.Struct(z.Shape{
		"url":    z.String().Required(),
		"prompt": z.String().Required(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("web_fetch: zog: %v", issues)
	}
	return nil
}
