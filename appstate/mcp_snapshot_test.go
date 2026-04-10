package appstate

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMcpState_emptyJSON(t *testing.T) {
	m := EmptyMcpState()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"clients":[]`) || !strings.Contains(string(b), `"tools":[]`) {
		t.Fatalf("%s", b)
	}
}

func TestMcpState_resourcesRoundTrip(t *testing.T) {
	m := McpState{
		Resources: map[string][]MCPServerResourceSnapshot{
			"u": {{URI: "file:///x", Server: "s", Name: "n"}},
		},
		PluginReconnectKey: 0,
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var back McpState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.Resources["u"]) != 1 || back.Resources["u"][0].URI != "file:///x" {
		t.Fatalf("%+v", back.Resources)
	}
}

func TestMCPServerConnectionSnapshot_roundTrip(t *testing.T) {
	s := MCPServerConnectionSnapshot{Name: "n", Type: MCPConnFailed, Error: "e"}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var back MCPServerConnectionSnapshot
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Name != "n" || back.Type != MCPConnFailed {
		t.Fatalf("%+v", back)
	}
}
