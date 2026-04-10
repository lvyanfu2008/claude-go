package toolinput

import (
	"encoding/json"
	"testing"

	"goc/ccb-engine/internal/anthropic"
)

func TestValidateAgainstTools_unknownToolSkipped(t *testing.T) {
	tools := anthropic.DefaultStubTools()
	if err := ValidateAgainstTools(tools, "nonexistent", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAgainstTools_echoStubValid(t *testing.T) {
	tools := anthropic.DefaultStubTools()
	if err := ValidateAgainstTools(tools, "echo_stub", json.RawMessage(`{"message":"hi"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAgainstTools_echoStubInvalid(t *testing.T) {
	tools := anthropic.DefaultStubTools()
	err := ValidateAgainstTools(tools, "echo_stub", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected validation error for missing required field")
	}
}

func TestValidateAgainstTools_skipEnv(t *testing.T) {
	t.Setenv("CCB_ENGINE_SKIP_TOOL_INPUT_SCHEMA", "1")
	tools := anthropic.DefaultStubTools()
	if err := ValidateAgainstTools(tools, "echo_stub", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
}
