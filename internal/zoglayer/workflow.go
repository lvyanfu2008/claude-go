package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var workflowAllowedKeys = map[string]struct{}{
	"workflow": {},
	"args":     {},
}

type workflowZogInput struct {
	Workflow string  `zog:"workflow"`
	Args     *string `zog:"args"`
}

func validateWorkflowZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("workflow: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := workflowAllowedKeys[k]; !ok {
			return fmt.Errorf("workflow: unknown field %q", k)
		}
	}

	var dest workflowZogInput

	wRaw, ok := raw["workflow"]
	if !ok {
		return fmt.Errorf("workflow: missing required field %q", "workflow")
	}
	var wVal any
	if err := json.Unmarshal(wRaw, &wVal); err != nil {
		return fmt.Errorf("workflow: workflow: %w", err)
	}
	wStr, ok := wVal.(string)
	if !ok {
		return fmt.Errorf("workflow: workflow must be a string")
	}
	if strings.TrimSpace(wStr) == "" {
		return fmt.Errorf("workflow: workflow must be non-empty")
	}
	dest.Workflow = wStr

	if err := parseZogStringField(raw, "args", &dest.Args); err != nil {
		return fmt.Errorf("workflow: %w", err)
	}

	schema := z.Struct(z.Shape{
		"workflow": z.String().Required(),
		"args":     z.String().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("workflow: zog: %v", issues)
	}
	return nil
}
