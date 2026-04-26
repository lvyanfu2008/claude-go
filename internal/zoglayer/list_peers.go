package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

var listPeersAllowedKeys = map[string]struct{}{
	"include_self": {},
}

func validateListPeersZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := listPeersAllowedKeys[k]; !ok {
			return fmt.Errorf("list_peers: unknown field %q", k)
		}
	}
	if br, ok := raw["include_self"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("list_peers: include_self: %w", err)
		}
		if v == nil {
			return fmt.Errorf("list_peers: include_self cannot be null")
		}
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("list_peers: include_self must be a boolean")
		}
	}
	return nil
}
