package transcript

import (
	"testing"

	"goc/types"
)

func TestDecodeJSON_UIArray(t *testing.T) {
	data := []byte(`[
		{"type":"user","uuid":"u1","content":[{"type":"text","text":"hi"}]},
		{"type":"assistant","uuid":"a1","content":[{"type":"text","text":"yo"}]}
	]`)
	msgs, err := DecodeJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("len=%d", len(msgs))
	}
	if msgs[0].Type != types.MessageTypeUser || msgs[0].UUID != "u1" {
		t.Fatalf("first: %+v", msgs[0])
	}
}

func TestDecodeJSON_APIArray(t *testing.T) {
	data := []byte(`[
		{"role":"user","content":"plain"},
		{"role":"assistant","content":[{"type":"text","text":"ok"}]}
	]`)
	msgs, err := DecodeJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("len=%d", len(msgs))
	}
	if msgs[0].Type != types.MessageTypeUser {
		t.Fatalf("first type %v", msgs[0].Type)
	}
	if len(msgs[0].Message) == 0 {
		t.Fatal("expected Message JSON for API row")
	}
	if msgs[1].Type != types.MessageTypeAssistant {
		t.Fatalf("second type %v", msgs[1].Type)
	}
}
