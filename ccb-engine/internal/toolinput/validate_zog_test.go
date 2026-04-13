package toolinput

import (
	"encoding/json"
	"testing"

	"goc/ccb-engine/bashzog"
	"goc/ccb-engine/internal/anthropic"
	"goc/internal/toolvalidator"
)

func TestValidateAgainstTools_zogBashValid(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	tools := anthropic.GouParityToolList()
	if err := ValidateAgainstTools(tools, bashzog.ZogToolName, json.RawMessage(`{"command":"echo hi","description":"x"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAgainstTools_zogBashMissingCommand(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	tools := anthropic.GouParityToolList()
	err := ValidateAgainstTools(tools, bashzog.ZogToolName, json.RawMessage(`{"description":"only desc"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateAgainstTools_zogBashSemanticTimeoutAndSed(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	tools := anthropic.GouParityToolList()
	payload := `{"command":"echo","timeout":"120000","_simulatedSedEdit":{"filePath":"/p","newContent":"z"}}`
	if err := ValidateAgainstTools(tools, bashzog.ZogToolName, json.RawMessage(payload)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAgainstTools_zogBashBackgroundTasksDisabled(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	t.Setenv("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS", "1")
	tools := anthropic.GouParityToolList()
	err := ValidateAgainstTools(tools, bashzog.ZogToolName, json.RawMessage(`{"command":"x","run_in_background":true}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateAgainstTools_zogBashNilInputSchemaOnToolRow(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	tools := anthropic.GouParityToolList()
	var zogIdx = -1
	for i := range tools {
		if tools[i].Name == bashzog.ZogToolName {
			zogIdx = i
			tools[i].InputSchema = nil
			break
		}
	}
	if zogIdx < 0 {
		t.Fatal("no BashZog in GouParityToolList")
	}
	if err := ValidateAgainstTools(tools, bashzog.ZogToolName, json.RawMessage(`{"command":"ok"}`)); err != nil {
		t.Fatal(err)
	}
}
