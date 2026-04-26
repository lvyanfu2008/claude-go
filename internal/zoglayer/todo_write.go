package zoglayer

import (
	"encoding/json"
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
)

var todoWriteAllowedKeys = map[string]struct{}{
	"todos": {},
}

type todoItemZog struct {
	Content    string  `zog:"content"`
	Status     string  `zog:"status"`
	ActiveForm *string `zog:"activeForm"`
}

type todoWriteZogInput struct {
	Todos []todoItemZog `zog:"todos"`
}

func validateTodoWriteZog(input json.RawMessage) error {
	if len(strings.TrimSpace(string(input))) == 0 {
		return fmt.Errorf("todo_write: empty input")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	for k := range raw {
		if _, ok := todoWriteAllowedKeys[k]; !ok {
			return fmt.Errorf("todo_write: unknown field %q", k)
		}
	}

	tr, ok := raw["todos"]
	if !ok {
		return fmt.Errorf("todo_write: missing required field %q", "todos")
	}
	var todos []json.RawMessage
	if err := json.Unmarshal(tr, &todos); err != nil {
		return fmt.Errorf("todo_write: todos must be an array: %w", err)
	}
	if len(todos) == 0 {
		return fmt.Errorf("todo_write: todos must be non-empty")
	}
	for i, t := range todos {
		var item struct {
			Content    string  `json:"content"`
			Status     string  `json:"status"`
			ActiveForm *string `json:"activeForm"`
		}
		if err := json.Unmarshal(t, &item); err != nil {
			return fmt.Errorf("todo_write: todos[%d]: %w", i, err)
		}
		if strings.TrimSpace(item.Content) == "" {
			return fmt.Errorf("todo_write: todos[%d]: content must be non-empty", i)
		}
		if item.Status != "pending" && item.Status != "in_progress" && item.Status != "completed" {
			return fmt.Errorf("todo_write: todos[%d]: status must be one of [pending, in_progress, completed]", i)
		}
		_ = item.ActiveForm // optional
	}

	itemSchema := z.Struct(z.Shape{
		"content":    z.String().Required(),
		"status":     z.String().Required(),
		"activeForm": z.String().Optional(),
	})
	var sample struct {
		Content    string `zog:"content"`
		Status     string `zog:"status"`
		ActiveForm string `zog:"activeForm"`
	}
	if issues := itemSchema.Validate(&sample); len(issues) > 0 {
		return fmt.Errorf("todo_write: zog: %v", issues)
	}
	return nil
}
