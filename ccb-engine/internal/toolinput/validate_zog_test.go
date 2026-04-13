package toolinput

import (
	"encoding/json"
	"testing"

	"goc/ccb-engine/internal/anthropic"
	"goc/internal/toolvalidator"
)

func TestValidateAgainstTools_zogBashValid(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	tools := anthropic.GouParityToolList()
	if err := ValidateAgainstTools(tools, "Bash", json.RawMessage(`{"command":"echo hi","description":"x"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAgainstTools_zogBashMissingCommand(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	tools := anthropic.GouParityToolList()
	err := ValidateAgainstTools(tools, "Bash", json.RawMessage(`{"description":"only desc"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}
