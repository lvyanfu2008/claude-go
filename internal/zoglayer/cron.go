package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var cronCreateAllowedKeys = map[string]struct{}{
	"cron":      {},
	"prompt":    {},
	"recurring": {},
	"durable":   {},
}

type cronCreateZogInput struct {
	Cron      string `zog:"cron"`
	Prompt    string `zog:"prompt"`
	Recurring *bool  `zog:"recurring"`
	Durable   *bool  `zog:"durable"`
}

func validateCronCreateZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("cron_create: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := cronCreateAllowedKeys[k]; !ok {
			return fmt.Errorf("cron_create: unknown field %q", k)
		}
	}

	var dest cronCreateZogInput

	cRaw, ok := raw["cron"]
	if !ok {
		return fmt.Errorf("cron_create: missing required field %q", "cron")
	}
	var cVal any
	if err := json.Unmarshal(cRaw, &cVal); err != nil {
		return fmt.Errorf("cron_create: cron: %w", err)
	}
	cStr, ok := cVal.(string)
	if !ok {
		return fmt.Errorf("cron_create: cron must be a string")
	}
	if strings.TrimSpace(cStr) == "" {
		return fmt.Errorf("cron_create: cron must be non-empty")
	}
	dest.Cron = cStr

	pRaw, ok := raw["prompt"]
	if !ok {
		return fmt.Errorf("cron_create: missing required field %q", "prompt")
	}
	var pVal any
	if err := json.Unmarshal(pRaw, &pVal); err != nil {
		return fmt.Errorf("cron_create: prompt: %w", err)
	}
	pStr, ok := pVal.(string)
	if !ok {
		return fmt.Errorf("cron_create: prompt must be a string")
	}
	if strings.TrimSpace(pStr) == "" {
		return fmt.Errorf("cron_create: prompt must be non-empty")
	}
	dest.Prompt = pStr

	if br, ok := raw["recurring"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("cron_create: recurring: %w", err)
		}
		if v == nil {
			return fmt.Errorf("cron_create: recurring cannot be null")
		}
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("cron_create: recurring must be a boolean")
		}
		dest.Recurring = &b
	}

	if br, ok := raw["durable"]; ok {
		var v any
		if err := json.Unmarshal(br, &v); err != nil {
			return fmt.Errorf("cron_create: durable: %w", err)
		}
		if v == nil {
			return fmt.Errorf("cron_create: durable cannot be null")
		}
		b, ok := v.(bool)
		if !ok {
			return fmt.Errorf("cron_create: durable must be a boolean")
		}
		dest.Durable = &b
	}

	schema := z.Struct(z.Shape{
		"cron":      z.String().Required(),
		"prompt":    z.String().Required(),
		"recurring": z.Bool().Optional(),
		"durable":   z.Bool().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("cron_create: zog: %v", issues)
	}
	return nil
}

var cronDeleteAllowedKeys = map[string]struct{}{
	"id": {},
}

func validateCronDeleteZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("cron_delete: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := cronDeleteAllowedKeys[k]; !ok {
			return fmt.Errorf("cron_delete: unknown field %q", k)
		}
	}
	idRaw, ok := raw["id"]
	if !ok {
		return fmt.Errorf("cron_delete: missing required field %q", "id")
	}
	var idVal any
	if err := json.Unmarshal(idRaw, &idVal); err != nil {
		return fmt.Errorf("cron_delete: id: %w", err)
	}
	idStr, ok := idVal.(string)
	if !ok {
		return fmt.Errorf("cron_delete: id must be a string")
	}
	if strings.TrimSpace(idStr) == "" {
		return fmt.Errorf("cron_delete: id must be non-empty")
	}
	return nil
}

var cronListAllowedKeys = map[string]struct{}{}

func validateCronListZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := cronListAllowedKeys[k]; !ok {
			return fmt.Errorf("cron_list: unknown field %q", k)
		}
	}
	return nil
}
