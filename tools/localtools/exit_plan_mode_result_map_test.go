package localtools

import (
	"strings"
	"testing"
)

func TestMapExitPlanModeToolResultToAssistantText(t *testing.T) {
	t.Run("normal plan with filePath", func(t *testing.T) {
		json := `{"data":{"plan":"# Plan\n- do X\n- do Y","isAgent":false,"filePath":"/work/plan.md"}}`
		result, err := MapExitPlanModeToolResultToAssistantText(json)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Approved Plan:") {
			t.Error("result should contain 'Approved Plan:'")
		}
		if !strings.Contains(result, "# Plan") {
			t.Error("result should contain plan content")
		}
		if !strings.Contains(result, "/work/plan.md") {
			t.Error("result should contain filePath")
		}
	})

	t.Run("empty plan", func(t *testing.T) {
		json := `{"data":{"plan":"","isAgent":false,"filePath":"/work/plan.md"}}`
		result, err := MapExitPlanModeToolResultToAssistantText(json)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "You can now proceed") {
			t.Errorf("expected proceed message, got: %s", result)
		}
	})

	t.Run("agent context", func(t *testing.T) {
		json := `{"data":{"plan":"# Plan\n- do X","isAgent":true,"filePath":"/work/plan.md"}}`
		result, err := MapExitPlanModeToolResultToAssistantText(json)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "ok") {
			t.Errorf("expected agent response, got: %s", result)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		_, err := MapExitPlanModeToolResultToAssistantText("")
		if err == nil {
			t.Error("should return error for empty input")
		}
	})
}
