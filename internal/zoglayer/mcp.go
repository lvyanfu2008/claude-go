package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

var listMcpResourcesAllowedKeys = map[string]struct{}{
	"server": {},
}

func validateListMcpResourcesZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := listMcpResourcesAllowedKeys[k]; !ok {
			return fmt.Errorf("list_mcp_resources: unknown field %q", k)
		}
	}
	if br, ok := raw["server"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("list_mcp_resources: server: %w", err)
		}
		if v != nil {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("list_mcp_resources: server must be a string")
			}
		}
	}
	return nil
}

var readMcpResourceAllowedKeys = map[string]struct{}{
	"server": {},
	"uri":    {},
}

func validateReadMcpResourceZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("read_mcp_resource: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := readMcpResourceAllowedKeys[k]; !ok {
			return fmt.Errorf("read_mcp_resource: unknown field %q", k)
		}
	}

	sRaw, ok := raw["server"]
	if !ok {
		return fmt.Errorf("read_mcp_resource: missing required field %q", "server")
	}
	var sVal any
	if err := json.Unmarshal(sRaw, &sVal); err != nil {
		return fmt.Errorf("read_mcp_resource: server: %w", err)
	}
	sStr, ok := sVal.(string)
	if !ok {
		return fmt.Errorf("read_mcp_resource: server must be a string")
	}
	if strings.TrimSpace(sStr) == "" {
		return fmt.Errorf("read_mcp_resource: server must be non-empty")
	}

	uRaw, ok := raw["uri"]
	if !ok {
		return fmt.Errorf("read_mcp_resource: missing required field %q", "uri")
	}
	var uVal any
	if err := json.Unmarshal(uRaw, &uVal); err != nil {
		return fmt.Errorf("read_mcp_resource: uri: %w", err)
	}
	uStr, ok := uVal.(string)
	if !ok {
		return fmt.Errorf("read_mcp_resource: uri must be a string")
	}
	if strings.TrimSpace(uStr) == "" {
		return fmt.Errorf("read_mcp_resource: uri must be non-empty")
	}

	return nil
}
