package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

var ctxInspectAllowedKeys = map[string]struct{}{
	"query": {},
}

func validateCtxInspectZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := ctxInspectAllowedKeys[k]; !ok {
			return fmt.Errorf("ctx_inspect: unknown field %q", k)
		}
	}
	if br, ok := raw["query"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("ctx_inspect: query: %w", err)
		}
		if v != nil {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("ctx_inspect: query must be a string")
			}
		}
	}
	return nil
}
