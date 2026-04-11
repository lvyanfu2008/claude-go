package anthropic

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCanonicalizeMessages_arrayContentStaysArrayOnRemarshal(t *testing.T) {
	const in = `{"role":"user","content":[{"type":"text","text":"<a>"},{"type":"text","text":"b"}]}`
	var m Message
	if err := json.Unmarshal([]byte(in), &m); err != nil {
		t.Fatal(err)
	}
	out := CanonicalizeMessages([]Message{m})
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	blocks, ok := out[0].Content.([]ContentBlock)
	if !ok || len(blocks) != 2 {
		t.Fatalf("want []ContentBlock len 2, got %#v", out[0].Content)
	}
	req := CreateMessageRequest{Model: "m", MaxTokens: 1, Messages: out}
	raw, err := marshalJSONNoEscapeHTML(req)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `\u003c`) {
		t.Fatalf("want literal < not unicode escape, got %s", raw)
	}
	if !strings.Contains(string(raw), `"content":[`) {
		t.Fatalf("want array content in wire JSON, got %s", raw)
	}
}
