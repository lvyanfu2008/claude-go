package appstate

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/types"
)

func TestAgentDefinitionsResult_emptyArraysJSON(t *testing.T) {
	r := EmptyAgentDefinitionsResult()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"activeAgents":[]`) || !strings.Contains(string(b), `"allAgents":[]`) {
		t.Fatalf("%s", b)
	}
	var back AgentDefinitionsResult
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.ActiveAgents == nil || back.AllAgents == nil {
		t.Fatal("unmarshal should use non-nil slices")
	}
}

func TestNormalizeAgentDefinitionData_slices(t *testing.T) {
	d := AgentDefinitionData{AgentType: "a", Source: "built-in", WhenToUse: "w"}
	NormalizeAgentDefinitionData(&d)
	if d.Tools == nil || d.DisallowedTools == nil || d.Skills == nil || d.McpServers == nil || d.RequiredMcpServers == nil {
		t.Fatalf("%+v", d)
	}
}

func TestNormalizeAppState_agentsAndMcpResources(t *testing.T) {
	a := AppState{
		Mcp: McpState{
			Resources: map[string][]MCPServerResourceSnapshot{
				"k": nil,
			},
		},
		AgentDefinitions: AgentDefinitionsResult{
			ActiveAgents: []AgentDefinitionData{
				{AgentType: "a", Source: "built-in", WhenToUse: "w"},
			},
		},
	}
	NormalizeAppState(&a)
	if a.Mcp.Resources["k"] == nil {
		t.Fatal("mcp resource slice")
	}
	if a.AgentDefinitions.ActiveAgents[0].Tools == nil {
		t.Fatal("agent tools slice")
	}
}

func TestAgentDefinitionsResult_roundTripAgent(t *testing.T) {
	pm := types.PermissionPlan
	r := AgentDefinitionsResult{
		ActiveAgents: []AgentDefinitionData{
			{AgentType: "x", Source: "built-in", WhenToUse: "t", BaseDir: "built-in"},
		},
		AllAgents: []AgentDefinitionData{
			{AgentType: "x", Source: "built-in", WhenToUse: "t", PermissionMode: &pm},
		},
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var back AgentDefinitionsResult
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.AllAgents) != 1 || back.AllAgents[0].PermissionMode == nil {
		t.Fatalf("%+v", back.AllAgents)
	}
}
