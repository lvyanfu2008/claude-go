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

func TestValidateAgainstTools_zogBashNilInputSchemaOnToolRow(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	tools := anthropic.GouParityToolList()
	var bashIdx = -1
	for i := range tools {
		if tools[i].Name == "Bash" {
			bashIdx = i
			tools[i].InputSchema = nil
			break
		}
	}
	if bashIdx < 0 {
		t.Fatal("no Bash in GouParityToolList")
	}
	if err := ValidateAgainstTools(tools, "Bash", json.RawMessage(`{"command":"ok"}`)); err != nil {
		t.Fatal(err)
	}
}
