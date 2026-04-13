package toolpool

import (
	"strings"
	"testing"

	"goc/ccb-engine/bashzog"
	"goc/internal/toolvalidator"
	"goc/types"
)

func TestReplaceBashToolSpecIfZogMode_appendsBashZogLeavesBash(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	base := []types.ToolSpec{
		{Name: "Read", Description: "r", InputJSONSchema: []byte(`{}`)},
		{Name: "Bash", Description: "short", InputJSONSchema: []byte(`{"type":"object"}`), MaxResultSizeChars: 42},
	}
	out, err := ReplaceBashToolSpecIfZogMode(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(base)+1 {
		t.Fatalf("len got %d want %d", len(out), len(base)+1)
	}
	if out[0].Description != "r" {
		t.Fatal("unexpected Read change")
	}
	if out[1].Name != "Bash" || out[1].Description != "short" || string(out[1].InputJSONSchema) != `{"type":"object"}` {
		t.Fatalf("Bash row must be unchanged, got %#v", out[1])
	}
	z := out[2]
	if z.Name != bashzog.ZogToolName {
		t.Fatalf("name %q", z.Name)
	}
	if z.Description == "short" || !strings.Contains(z.Description, "bash command") {
		t.Fatalf("expected bashzog snapshot description, got prefix %q", truncate(z.Description, 60))
	}
	if len(z.InputJSONSchema) < 100 {
		t.Fatalf("expected full input_schema from snapshot, got %d bytes", len(z.InputJSONSchema))
	}
	if z.MaxResultSizeChars != 42 {
		t.Fatalf("expected MaxResultSizeChars copied from Bash, got %d", z.MaxResultSizeChars)
	}
}

func TestReplaceBashToolSpecIfZogMode_idempotent(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	base := []types.ToolSpec{{Name: "Bash", MaxResultSizeChars: 7}}
	once, err := ReplaceBashToolSpecIfZogMode(base)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := ReplaceBashToolSpecIfZogMode(once)
	if err != nil {
		t.Fatal(err)
	}
	if len(once) != 2 || len(twice) != 2 {
		t.Fatalf("once %d twice %d", len(once), len(twice))
	}
}

func TestReplaceBashToolSpecIfZogMode_noopWhenJsonschema(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "")
	base := []types.ToolSpec{{Name: "Bash", Description: "keep", InputJSONSchema: []byte(`{"x":1}`)}}
	out, err := ReplaceBashToolSpecIfZogMode(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Description != "keep" || string(out[0].InputJSONSchema) != `{"x":1}` {
		t.Fatalf("got %#v", out[0])
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
