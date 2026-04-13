package toolvalidator

import (
	"encoding/json"
	"testing"

	"goc/internal/zoglayer"
)

func TestValidateInput_zogBashNilSchemaStillValidates(t *testing.T) {
	t.Setenv(EnvToolInputValidator, "zog")
	if !zoglayer.Has("Bash") {
		t.Fatal("expected zoglayer to register Bash")
	}
	err := ValidateInput("Bash", nil, json.RawMessage(`{"command":"echo hi"}`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateInput_zogBashNilSchemaMissingCommand(t *testing.T) {
	t.Setenv(EnvToolInputValidator, "zog")
	err := ValidateInput("Bash", nil, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected zog validation error")
	}
}
