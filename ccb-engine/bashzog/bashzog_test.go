package bashzog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLoadAPIData_hasCommandProperty(t *testing.T) {
	d, err := LoadAPIData()
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "Bash" {
		t.Fatalf("name %q", d.Name)
	}
	if !strings.Contains(d.Description, "bash command") {
		t.Fatalf("description should be TS-export style, got prefix %q", truncate(d.Description, 80))
	}
	props, ok := d.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties: %#v", d.InputSchema["properties"])
	}
	cmd, ok := props["command"].(map[string]any)
	if !ok || cmd["type"] != "string" {
		t.Fatalf("command schema: %#v", props["command"])
	}
}

func TestBashToolSpec_matchesLoadAPIDataSchema(t *testing.T) {
	d, err := LoadAPIData()
	if err != nil {
		t.Fatal(err)
	}
	spec, err := BashToolSpec()
	if err != nil {
		t.Fatal(err)
	}
	if string(spec.InputJSONSchema) != string(d.InputSchemaRaw) {
		t.Fatal("InputJSONSchema mismatch vs LoadAPIData raw")
	}
}

func TestValidate_minimal(t *testing.T) {
	if err := Validate(json.RawMessage(`{"command":"x"}`)); err != nil {
		t.Fatal(err)
	}
	if err := Validate(json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadAPIData_embeddedJSONNonTrivial(t *testing.T) {
	d, err := LoadAPIData()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.InputSchemaRaw) < 50 {
		t.Fatal("embedded bash tool json too small")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
