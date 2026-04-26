package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var webSearchAllowedKeys = map[string]struct{}{
	"query":           {},
	"allowed_domains": {},
	"blocked_domains": {},
}

type webSearchZogInput struct {
	Query          string   `zog:"query"`
	AllowedDomains []string `zog:"allowed_domains"`
	BlockedDomains []string `zog:"blocked_domains"`
}

func validateWebSearchZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("web_search: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := webSearchAllowedKeys[k]; !ok {
			return fmt.Errorf("web_search: unknown field %q", k)
		}
	}

	var dest webSearchZogInput

	qRaw, ok := raw["query"]
	if !ok {
		return fmt.Errorf("web_search: missing required field %q", "query")
	}
	var qVal any
	if err := json.Unmarshal(qRaw, &qVal); err != nil {
		return fmt.Errorf("web_search: query: %w", err)
	}
	qStr, ok := qVal.(string)
	if !ok {
		return fmt.Errorf("web_search: query must be a string")
	}
	if strings.TrimSpace(qStr) == "" {
		return fmt.Errorf("web_search: query must be non-empty")
	}
	dest.Query = qStr

	if err := parseZogStringArray(raw, "allowed_domains", &dest.AllowedDomains); err != nil {
		return fmt.Errorf("web_search: %w", err)
	}
	if err := parseZogStringArray(raw, "blocked_domains", &dest.BlockedDomains); err != nil {
		return fmt.Errorf("web_search: %w", err)
	}

	schema := z.Struct(z.Shape{
		"query": z.String().Required(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("web_search: zog: %v", issues)
	}
	return nil
}

func parseZogStringArray(raw map[string]json.RawMessage, key string, out *[]string) error {
	br, ok := raw[key]
	if !ok {
		return nil
	}
	var arr []any
	if err := json.Unmarshal(br, &arr); err != nil {
		return fmt.Errorf("%s must be an array of strings", key)
	}
	for i, v := range arr {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s[%d] must be a string", key, i)
		}
		*out = append(*out, s)
	}
	return nil
}
