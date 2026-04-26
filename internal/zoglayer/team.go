package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

var teamAddMemberAllowedKeys = map[string]struct{}{
	"team_name": {},
	"agent_id":  {},
	"name":      {},
}

func validateTeamAddMemberZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("team_add_member: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := teamAddMemberAllowedKeys[k]; !ok {
			return fmt.Errorf("team_add_member: unknown field %q", k)
		}
	}

	// Required: team_name
	tRaw, ok := raw["team_name"]
	if !ok {
		return fmt.Errorf("team_add_member: missing required field %q", "team_name")
	}
	var tVal any
	if err := json.Unmarshal(tRaw, &tVal); err != nil {
		return fmt.Errorf("team_add_member: team_name: %w", err)
	}
	tStr, ok := tVal.(string)
	if !ok {
		return fmt.Errorf("team_add_member: team_name must be a string")
	}
	if strings.TrimSpace(tStr) == "" {
		return fmt.Errorf("team_add_member: team_name must be non-empty")
	}

	// Required: agent_id
	aRaw, ok := raw["agent_id"]
	if !ok {
		return fmt.Errorf("team_add_member: missing required field %q", "agent_id")
	}
	var aVal any
	if err := json.Unmarshal(aRaw, &aVal); err != nil {
		return fmt.Errorf("team_add_member: agent_id: %w", err)
	}
	aStr, ok := aVal.(string)
	if !ok {
		return fmt.Errorf("team_add_member: agent_id must be a string")
	}
	if strings.TrimSpace(aStr) == "" {
		return fmt.Errorf("team_add_member: agent_id must be non-empty")
	}

	// Optional: name
	if br, ok := raw["name"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("team_add_member: name: %w", err)
		}
		if v != nil {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("team_add_member: name must be a string")
			}
		}
	}

	return nil
}

var teamRemoveMemberAllowedKeys = map[string]struct{}{
	"team_name": {},
	"agent_id":  {},
}

func validateTeamRemoveMemberZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("team_remove_member: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := teamRemoveMemberAllowedKeys[k]; !ok {
			return fmt.Errorf("team_remove_member: unknown field %q", k)
		}
	}

	// Required: team_name
	tRaw, ok := raw["team_name"]
	if !ok {
		return fmt.Errorf("team_remove_member: missing required field %q", "team_name")
	}
	var tVal any
	if err := json.Unmarshal(tRaw, &tVal); err != nil {
		return fmt.Errorf("team_remove_member: team_name: %w", err)
	}
	tStr, ok := tVal.(string)
	if !ok {
		return fmt.Errorf("team_remove_member: team_name must be a string")
	}
	if strings.TrimSpace(tStr) == "" {
		return fmt.Errorf("team_remove_member: team_name must be non-empty")
	}

	// Required: agent_id
	aRaw, ok := raw["agent_id"]
	if !ok {
		return fmt.Errorf("team_remove_member: missing required field %q", "agent_id")
	}
	var aVal any
	if err := json.Unmarshal(aRaw, &aVal); err != nil {
		return fmt.Errorf("team_remove_member: agent_id: %w", err)
	}
	aStr, ok := aVal.(string)
	if !ok {
		return fmt.Errorf("team_remove_member: agent_id must be a string")
	}
	if strings.TrimSpace(aStr) == "" {
		return fmt.Errorf("team_remove_member: agent_id must be non-empty")
	}

	return nil
}
