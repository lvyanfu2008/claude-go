package gemma

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildVertexPredictRequest_toolsRoleMatchesCanonicalTools(t *testing.T) {
	c := NewClient(DefaultConfig())
	rawTools := `[{"type":"function","function":{"name":"demo_fn","description":"d","parameters":{"type":"object"}}}]`
	want, err := canonicalToolsJSONForVertex(json.RawMessage(rawTools))
	if err != nil {
		t.Fatal(err)
	}
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
	if string(wire.Instances[0].Tools) != string(want) {
		t.Fatalf("inst.tools mismatch\nwant %s\ngot  %s", want, wire.Instances[0].Tools)
	}
	var sawTools bool
	for _, m := range wire.Instances[0].Messages {
		if strings.EqualFold(m.Role, "tools") {
			sawTools = true
			if len(m.Content) != 1 || m.Content[0].Text != string(want) {
				t.Fatalf("tools role text mismatch\nwant %s\ngot  %s", want, m.Content[0].Text)
			}
		}
	}
	if !sawTools {
		t.Fatalf("no tools role in %#v", wire.Instances[0].Messages)
	}
}

func TestBuildVertexPredictRequest_toolsRoleAfterLeadingSystem(t *testing.T) {
	c := NewClient(DefaultConfig())
	rawTools := `[{"type":"function","function":{"name":"x","parameters":{}}}]`
	want, err := canonicalToolsJSONForVertex(json.RawMessage(rawTools))
	if err != nil {
		t.Fatal(err)
	}
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
	if msgs[2].Content[0].Text != string(want) {
		t.Fatalf("tools payload: %s", msgs[2].Content[0].Text)
	}
	if msgs[3].Role != "user" {
		t.Fatalf("want user fourth, got role=%q", msgs[3].Role)
	}
}

func TestCanonicalToolsJSONForVertex_twoFunctionsOpenAIStyle(t *testing.T) {
	raw := `[
	  {"type":"function","function":{"name":"BashZog","description":"run shell","parameters":{"type":"object","properties":{"cmd":{"type":"string"}}}}},
	  {"type":"function","function":{"name":"Edit","description":"edit files","parameters":{"type":"object"}}}
	]`
	out, err := canonicalToolsJSONForVertex(json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	var arr []vertexToolWire
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 {
		t.Fatalf("len=%d %s", len(arr), string(out))
	}
	if arr[0].Type != "function" || arr[0].Function.Name != "BashZog" || arr[0].Function.Parameters == nil {
		t.Fatalf("first tool: %+v", arr[0])
	}
	if arr[1].Function.Name != "Edit" {
		t.Fatalf("second tool: %+v", arr[1])
	}
	if _, ok := arr[0].Function.Parameters["type"]; !ok {
		t.Fatalf("parameters not preserved: %#v", arr[0].Function.Parameters)
	}
}

func TestCanonicalToolsJSONForVertex_flattensTopLevelName(t *testing.T) {
	raw := `[{"name":"FlatTool","parameters":{"x":1}}]`
	out, err := canonicalToolsJSONForVertex(json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	var arr []vertexToolWire
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 1 || arr[0].Function.Name != "FlatTool" || arr[0].Type != "function" {
		t.Fatalf("%s", string(out))
	}
}
