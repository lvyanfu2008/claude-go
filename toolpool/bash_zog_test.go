package toolpool

import (
	"strings"
	"testing"

	"goc/internal/toolvalidator"
	"goc/types"
)

func TestReplaceBashToolSpecIfZogMode_swapsDescriptionAndSchema(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	base := []types.ToolSpec{
		{Name: "Read", Description: "r", InputJSONSchema: []byte(`{}`)},
		{Name: "Bash", Description: "short", InputJSONSchema: []byte(`{"type":"object"}`)},
	}
	out, err := ReplaceBashToolSpecIfZogMode(base)
	if err != nil {
		t.Fatal(err)
	}
	if out[0].Description != "r" {
		t.Fatal("unexpected Read change")
	}
	if out[1].Name != "Bash" {
		t.Fatal("bash name")
	}
	if out[1].Description == "short" {
		t.Fatal("expected Bash description replaced from bashzog snapshot")
	}
	if !strings.Contains(out[1].Description, "bash command") {
		t.Fatalf("bash description prefix: %q", truncate(out[1].Description, 60))
	}
	if len(out[1].InputJSONSchema) < 100 {
		t.Fatalf("expected full input_schema from snapshot, got %d bytes", len(out[1].InputJSONSchema))
	}
}

func TestReplaceBashToolSpecIfZogMode_noopWhenJsonschema(t *testing.T) {
	t.Setenv(toolvalidator.EnvToolInputValidator, "")
	base := []types.ToolSpec{{Name: "Bash", Description: "keep", InputJSONSchema: []byte(`{"x":1}`)}}
	out, err := ReplaceBashToolSpecIfZogMode(base)
	if err != nil {
		t.Fatal(err)
	}
	if out[0].Description != "keep" || string(out[0].InputJSONSchema) != `{"x":1}` {
		t.Fatalf("got %#v", out[0])
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
