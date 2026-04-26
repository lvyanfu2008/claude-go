package zoglayer

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	z "github.com/Oudwins/zog"
)

var toolSearchAllowedKeys = map[string]struct{}{
	"query":       {},
	"max_results": {},
}

type toolSearchZogInput struct {
	Query      string `zog:"query"`
	MaxResults *int   `zog:"max_results"`
}

func validateToolSearchZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("tool_search: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := toolSearchAllowedKeys[k]; !ok {
			return fmt.Errorf("tool_search: unknown field %q", k)
		}
	}

	var dest toolSearchZogInput

	qRaw, ok := raw["query"]
	if !ok {
		return fmt.Errorf("tool_search: missing required field %q", "query")
	}
	var qVal any
	if err := json.Unmarshal(qRaw, &qVal); err != nil {
		return fmt.Errorf("tool_search: query: %w", err)
	}
	qStr, ok := qVal.(string)
	if !ok {
		return fmt.Errorf("tool_search: query must be a string")
	}
	if strings.TrimSpace(qStr) == "" {
		return fmt.Errorf("tool_search: query must be non-empty")
	}
	dest.Query = qStr

	if mr, ok := raw["max_results"]; ok {
		var mv any
		if err := json.Unmarshal(mr, &mv); err != nil {
			return fmt.Errorf("tool_search: max_results: %w", err)
		}
		if mv == nil {
			return fmt.Errorf("tool_search: max_results cannot be null")
		}
		f, ok := mv.(float64)
		if !ok {
			return fmt.Errorf("tool_search: max_results must be a number")
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Errorf("tool_search: max_results must be a finite number")
		}
		i := int(f)
		dest.MaxResults = &i
	}

	schema := z.Struct(z.Shape{
		"query":       z.String().Required(),
		"max_results": z.Int().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("tool_search: zog: %v", issues)
	}
	return nil
}
