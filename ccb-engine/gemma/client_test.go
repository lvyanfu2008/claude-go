package gemma

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildVertexPredictRequest_toolsRoleWithToolsJSON(t *testing.T) {
	c := NewClient(DefaultConfig())
	rawTools := `[{"type":"function","function":{"name":"demo_fn","description":"d","parameters":{"type":"object"}}}]`
	req := ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hi"},
		},
		ToolsJSON: json.RawMessage(rawTools),
	}
	wire, err := c.buildVertexPredictRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(wire.Instances) != 1 {
		t.Fatalf("instances: %d", len(wire.Instances))
	}
	msgs := wire.Instances[0].Messages
	var sawTools bool
	for _, m := range msgs {
		if strings.EqualFold(m.Role, "tools") {
			sawTools = true
			if len(m.Content) != 1 || m.Content[0].Type != "text" {
				t.Fatalf("tools message content: %+v", m.Content)
			}
			if m.Content[0].Text != rawTools {
				t.Fatalf("tools text mismatch:\nwant %s\ngot  %s", rawTools, m.Content[0].Text)
			}
		}
	}
	if !sawTools {
		t.Fatalf("messages: %#v — no role tools", msgs)
	}
	if string(wire.Instances[0].Tools) != rawTools {
		t.Fatalf("inst.tools: %s", wire.Instances[0].Tools)
	}
}

func TestBuildVertexPredictRequest_toolsRoleAfterLeadingSystem(t *testing.T) {
	c := NewClient(DefaultConfig())
	rawTools := `[{"type":"function","function":{"name":"x","parameters":{}}}]`
	req := ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "a"},
			{Role: "system", Content: "b"},
			{Role: "user", Content: "u"},
		},
		ToolsJSON: json.RawMessage(rawTools),
	}
	wire, err := c.buildVertexPredictRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	msgs := wire.Instances[0].Messages
	if len(msgs) < 4 {
		t.Fatalf("len=%d %#v", len(msgs), msgs)
	}
	if !strings.EqualFold(msgs[0].Role, "system") || !strings.EqualFold(msgs[1].Role, "system") {
		t.Fatalf("want two system first: %#v", msgs)
	}
	if msgs[2].Role != "tools" {
		t.Fatalf("want tools third, got role=%q", msgs[2].Role)
	}
	if msgs[3].Role != "user" {
		t.Fatalf("want user fourth, got role=%q", msgs[3].Role)
	}
}
