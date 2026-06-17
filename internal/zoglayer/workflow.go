package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var workflowAllowedKeys = map[string]struct{}{
	"workflow":        {},
	"args":            {},
	"script":          {},
	"name":            {},
	"description":     {},
	"title":           {},
	"scriptPath":      {},
	"resumeFromRunId": {},
}

type workflowZogInput struct {
	Workflow        string  `zog:"workflow"`
	Args            *string `zog:"args"`
	Script          *string `zog:"script"`
	Name            *string `zog:"name"`
	Description     *string `zog:"description"`
	Title           *string `zog:"title"`
	ScriptPath      *string `zog:"scriptPath"`
	ResumeFromRunID *string `zog:"resumeFromRunId"`
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

	// At least one of: workflow, script, scriptPath, name must be present.
	hasWorkflow := false
	hasScript := false
	hasName := false

	wRaw, ok := raw["workflow"]
	if ok {
		var wVal any
		if err := json.Unmarshal(wRaw, &wVal); err != nil {
			return fmt.Errorf("workflow: workflow: %w", err)
		}
		wStr, ok := wVal.(string)
		if !ok {
			return fmt.Errorf("workflow: workflow must be a string")
		}
		if strings.TrimSpace(wStr) != "" {
			hasWorkflow = true
			dest.Workflow = wStr
		}
	}

	if err := parseZogStringField(raw, "script", &dest.Script); err != nil {
		return fmt.Errorf("workflow: %w", err)
	}
	if dest.Script != nil && strings.TrimSpace(*dest.Script) != "" {
		hasScript = true
	}
	if err := parseZogStringField(raw, "scriptPath", &dest.ScriptPath); err != nil {
		return fmt.Errorf("workflow: %w", err)
	}
	if dest.ScriptPath != nil && strings.TrimSpace(*dest.ScriptPath) != "" {
		hasScript = true
	}
	if err := parseZogStringField(raw, "name", &dest.Name); err != nil {
		return fmt.Errorf("workflow: %w", err)
	}
	if dest.Name != nil && strings.TrimSpace(*dest.Name) != "" {
		hasName = true
	}

	if !hasWorkflow && !hasScript && !hasName {
		return fmt.Errorf("workflow: at least one of 'workflow', 'script', 'scriptPath', or 'name' is required")
	}

	if err := parseZogStringField(raw, "args", &dest.Args); err != nil {
		return fmt.Errorf("workflow: %w", err)
	}
	if err := parseZogStringField(raw, "description", &dest.Description); err != nil {
		return fmt.Errorf("workflow: %w", err)
	}
	if err := parseZogStringField(raw, "title", &dest.Title); err != nil {
		return fmt.Errorf("workflow: %w", err)
	}
	if err := parseZogStringField(raw, "resumeFromRunId", &dest.ResumeFromRunID); err != nil {
		return fmt.Errorf("workflow: %w", err)
	}

	schema := z.Struct(z.Shape{
		"workflow":        z.String().Optional(),
		"args":            z.String().Optional(),
		"script":          z.String().Optional(),
		"name":            z.String().Optional(),
		"description":     z.String().Optional(),
		"title":           z.String().Optional(),
		"scriptPath":      z.String().Optional(),
		"resumeFromRunId": z.String().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("workflow: zog: %v", issues)
	}
	return nil
}
