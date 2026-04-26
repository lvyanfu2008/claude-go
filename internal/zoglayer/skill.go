package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var skillAllowedKeys = map[string]struct{}{
	"skill": {},
	"args":  {},
}

type skillZogInput struct {
	Skill string  `zog:"skill"`
	Args  *string `zog:"args"`
}

func validateSkillZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("skill: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := skillAllowedKeys[k]; !ok {
			return fmt.Errorf("skill: unknown field %q", k)
		}
	}

	var dest skillZogInput

	sRaw, ok := raw["skill"]
	if !ok {
		return fmt.Errorf("skill: missing required field %q", "skill")
	}
	var sVal any
	if err := json.Unmarshal(sRaw, &sVal); err != nil {
		return fmt.Errorf("skill: skill: %w", err)
	}
	sStr, ok := sVal.(string)
	if !ok {
		return fmt.Errorf("skill: skill must be a string")
	}
	if strings.TrimSpace(sStr) == "" {
		return fmt.Errorf("skill: skill must be non-empty")
	}
	dest.Skill = sStr

	if err := parseZogStringField(raw, "args", &dest.Args); err != nil {
		return fmt.Errorf("skill: %w", err)
	}

	schema := z.Struct(z.Shape{
		"skill": z.String().Required(),
		"args":  z.String().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("skill: zog: %v", issues)
	}
	return nil
}
