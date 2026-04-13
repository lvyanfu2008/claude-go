package bashzog

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strings"
	"testing"
)

//go:embed bash_zog_tool_export.json
var bashZogExportGolden []byte

func TestExportBashZogToolJSON_matchesGoldenFile(t *testing.T) {
	got, err := ExportBashZogToolJSON()
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.TrimSpace(bashZogExportGolden)
	got = bytes.TrimSpace(got)
	if !bytes.Equal(got, want) {
		t.Fatal("export drift: from claude-go run: go run ./cmd/export-bashzog-json")
	}
}

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

func TestBashZogToolSpec_nameAndSchema(t *testing.T) {
	spec, err := BashZogToolSpec()
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name != ZogToolName {
		t.Fatalf("name %q", spec.Name)
	}
	if len(spec.InputJSONSchema) < 50 {
		t.Fatal("expected embedded input_schema bytes")
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

func TestValidate_timeoutSemanticString(t *testing.T) {
	payload := `{"command":"x","timeout":"30"}`
	if err := Validate(json.RawMessage(payload)); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_boolSemanticStrings(t *testing.T) {
	if err := Validate(json.RawMessage(`{"command":"x","run_in_background":"false","dangerouslyDisableSandbox":"true"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_simulatedSedEdit_ok(t *testing.T) {
	p := `{"command":"x","_simulatedSedEdit":{"filePath":"/tmp/a","newContent":"hello"}}`
	if err := Validate(json.RawMessage(p)); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_simulatedSedEdit_missingContent(t *testing.T) {
	p := `{"command":"x","_simulatedSedEdit":{"filePath":"/tmp/a"}}`
	if err := Validate(json.RawMessage(p)); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_unknownTopLevelKey(t *testing.T) {
	if err := Validate(json.RawMessage(`{"command":"x","extra":1}`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_timeoutOverMax(t *testing.T) {
	p := `{"command":"x","timeout":600001}`
	if err := Validate(json.RawMessage(p)); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_backgroundDisabledEnv_rejectsRunInBackground(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS", "1")
	err := Validate(json.RawMessage(`{"command":"x","run_in_background":false}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_backgroundDisabledEnv_allowsWithoutRunInBackground(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS", "1")
	if err := Validate(json.RawMessage(`{"command":"x"}`)); err != nil {
		t.Fatal(err)
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
