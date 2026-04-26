package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var notebookEditAllowedKeys = map[string]struct{}{
	"notebook_path": {},
	"cell_id":       {},
	"new_source":    {},
	"cell_type":     {},
	"edit_mode":     {},
}

type notebookEditZogInput struct {
	NotebookPath string  `zog:"notebook_path"`
	NewSource    string  `zog:"new_source"`
	CellID       *string `zog:"cell_id"`
	CellType     *string `zog:"cell_type"`
	EditMode     *string `zog:"edit_mode"`
}

func validateNotebookEditZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("notebook_edit: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := notebookEditAllowedKeys[k]; !ok {
			return fmt.Errorf("notebook_edit: unknown field %q", k)
		}
	}

	var dest notebookEditZogInput

	npRaw, ok := raw["notebook_path"]
	if !ok {
		return fmt.Errorf("notebook_edit: missing required field %q", "notebook_path")
	}
	var npVal any
	if err := json.Unmarshal(npRaw, &npVal); err != nil {
		return fmt.Errorf("notebook_edit: notebook_path: %w", err)
	}
	npStr, ok := npVal.(string)
	if !ok {
		return fmt.Errorf("notebook_edit: notebook_path must be a string")
	}
	if strings.TrimSpace(npStr) == "" {
		return fmt.Errorf("notebook_edit: notebook_path must be non-empty")
	}
	dest.NotebookPath = npStr

	nsRaw, ok := raw["new_source"]
	if !ok {
		return fmt.Errorf("notebook_edit: missing required field %q", "new_source")
	}
	var nsVal any
	if err := json.Unmarshal(nsRaw, &nsVal); err != nil {
		return fmt.Errorf("notebook_edit: new_source: %w", err)
	}
	nsStr, ok := nsVal.(string)
	if !ok {
		return fmt.Errorf("notebook_edit: new_source must be a string")
	}
	dest.NewSource = nsStr

	if err := parseZogStringField(raw, "cell_id", &dest.CellID); err != nil {
		return fmt.Errorf("notebook_edit: %w", err)
	}

	if err := parseZogStringField(raw, "cell_type", &dest.CellType); err != nil {
		return fmt.Errorf("notebook_edit: %w", err)
	}
	if dest.CellType != nil {
		v := *dest.CellType
		if v != "code" && v != "markdown" {
			return fmt.Errorf("notebook_edit: cell_type must be one of [code, markdown]")
		}
	}

	if err := parseZogStringField(raw, "edit_mode", &dest.EditMode); err != nil {
		return fmt.Errorf("notebook_edit: %w", err)
	}
	if dest.EditMode != nil {
		v := *dest.EditMode
		if v != "replace" && v != "insert" && v != "delete" {
			return fmt.Errorf("notebook_edit: edit_mode must be one of [replace, insert, delete]")
		}
	}

	schema := z.Struct(z.Shape{
		"notebook_path": z.String().Required(),
		"new_source":    z.String().Required(),
		"cell_id":       z.String().Optional(),
		"cell_type":     z.String().Optional(),
		"edit_mode":     z.String().Optional(),
	})
	if issues := schema.Validate(&dest); len(issues) > 0 {
		return fmt.Errorf("notebook_edit: zog: %v", issues)
	}
	return nil
}
