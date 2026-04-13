package bashzog

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
	"goc/commands"
)

type bashSimulatedSedEdit struct {
	FilePath   string `json:"filePath"`
	NewContent string `json:"newContent"`
}

// bashZogCommand holds the only field passed to Zog. Optional scalars use *T in our structs,
// but zog's z.String().Optional() / z.Bool().Optional() expect non-pointer fields (see zog panics).
type bashZogCommand struct {
	Command string `json:"command"`
}

type bashParsed struct {
	Command                   string
	Description               *string
	RunInBackground           *bool
	DangerouslyDisableSandbox *bool
}

// Validate enforces TS BashTool fullInputSchema wire shape: strict object keys, semantic
// number/boolean coercions, timeout bounds (getMaxBashTimeoutMs), optional _simulatedSedEdit,
// and run_in_background forbidden when CLAUDE_CODE_DISABLE_BACKGROUND_TASKS is set.
func Validate(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("bash: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}

	bgDisabled := commands.IsEnvTruthy("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS")
	allowed := allowedKeys(bgDisabled)
	for k := range raw {
		if _, ok := allowed[k]; !ok {
			return fmt.Errorf("bash: unknown field %q", k)
		}
	}

	var dest bashParsed
	cmdRaw, hasCmd := raw["command"]
	if !hasCmd {
		return fmt.Errorf("bash: missing required field %q", "command")
	}
	var cmdVal any
	if err := json.Unmarshal(cmdRaw, &cmdVal); err != nil {
		return fmt.Errorf("bash: command: %w", err)
	}
	cmdStr, ok := cmdVal.(string)
	if !ok {
		return fmt.Errorf("bash: command must be a string")
	}
	dest.Command = strings.TrimSpace(cmdStr)
	if dest.Command == "" {
		return fmt.Errorf("bash: command must be non-empty")
	}

	if err := parseOptionalString(raw, "description", &dest.Description); err != nil {
		return err
	}
	if !bgDisabled {
		if err := parseOptionalBool(raw, "run_in_background", &dest.RunInBackground); err != nil {
			return err
		}
	}
	if err := parseOptionalBool(raw, "dangerouslyDisableSandbox", &dest.DangerouslyDisableSandbox); err != nil {
		return err
	}

	if tr, ok := raw["timeout"]; ok {
		var tv any
		if err := json.Unmarshal(tr, &tv); err != nil {
			return fmt.Errorf("bash: timeout: %w", err)
		}
		if tv == nil {
			return fmt.Errorf("bash: timeout cannot be null")
		}
		tv = normalizeJSONValue("timeout", tv)
		maxMs := float64(maxBashTimeoutMs())
		f, ok, err := parseTimeoutFloat(tv)
		if err != nil || !ok {
			return fmt.Errorf("bash: timeout must be a finite number")
		}
		if f < 0 || f > maxMs {
			return fmt.Errorf("bash: timeout out of range (max %d ms)", int(maxMs))
		}
	}

	if sr, ok := raw["_simulatedSedEdit"]; ok {
		var sedObj bashSimulatedSedEdit
		if err := json.Unmarshal(sr, &sedObj); err != nil {
			return fmt.Errorf("bash: _simulatedSedEdit: %w", err)
		}
		if strings.TrimSpace(sedObj.FilePath) == "" || strings.TrimSpace(sedObj.NewContent) == "" {
			return fmt.Errorf("bash: _simulatedSedEdit requires filePath and newContent")
		}
	}

	zCmd := bashZogCommand{Command: dest.Command}
	schema := z.Struct(z.Shape{
		"Command": z.String().Required(),
	})
	if issues := schema.Validate(&zCmd); len(issues) > 0 {
		return fmt.Errorf("zog: %v", issues)
	}
	return nil
}

func allowedKeys(bgDisabled bool) map[string]struct{} {
	out := map[string]struct{}{
		"command":                   {},
		"timeout":                   {},
		"description":               {},
		"dangerouslyDisableSandbox": {},
		"_simulatedSedEdit":         {},
	}
	if !bgDisabled {
		out["run_in_background"] = struct{}{}
	}
	return out
}

func parseOptionalString(raw map[string]json.RawMessage, key string, out **string) error {
	br, ok := raw[key]
	if !ok {
		return nil
	}
	var v any
	if err := json.Unmarshal(br, &v); err != nil {
		return fmt.Errorf("bash: %s: %w", key, err)
	}
	if v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("bash: %s must be a string", key)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	cp := s
	*out = &cp
	return nil
}

func parseOptionalBool(raw map[string]json.RawMessage, key string, out **bool) error {
	br, ok := raw[key]
	if !ok {
		return nil
	}
	var v any
	if err := json.Unmarshal(br, &v); err != nil {
		return fmt.Errorf("bash: %s: %w", key, err)
	}
	v = normalizeJSONValue(key, v)
	if v == nil {
		return fmt.Errorf("bash: %s cannot be null", key)
	}
	b, ok := v.(bool)
	if !ok {
		return fmt.Errorf("bash: %s must be a boolean", key)
	}
	*out = &b
	return nil
}
