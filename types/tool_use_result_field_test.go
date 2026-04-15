package types

import (
	"encoding/json"
	"testing"
)

func TestToolUseResultJSONBytes_objectNotDoubleEncoded(t *testing.T) {
	raw := `{"type":"text","file":{"filePath":"/x.go","content":"a","numLines":1,"startLine":1,"totalLines":1}}`
	out := ToolUseResultJSONBytes(raw)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "text" {
		t.Fatalf("expected object, got %s", string(out))
	}
	if out[0] != '{' {
		t.Fatalf("want embedded JSON object, first byte got %q", out[0])
	}
}

func TestToolUseResultJSONBytes_plainErrorIsJSONString(t *testing.T) {
	out := ToolUseResultJSONBytes("permission denied")
	var s string
	if err := json.Unmarshal(out, &s); err != nil {
		t.Fatal(err)
	}
	if s != "permission denied" {
		t.Fatalf("got %q", s)
	}
}
