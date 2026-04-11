package anthropicmessages

import (
	"testing"
)

func TestBetasForToolsJSON_empty(t *testing.T) {
	if b := BetasForToolsJSON(nil); len(b) != 0 {
		t.Fatalf("got %v", b)
	}
	if b := BetasForToolsJSON([]byte(`[]`)); len(b) != 0 {
		t.Fatalf("got %v", b)
	}
}

func TestBetasForToolsJSON_toolSearch(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ANTHROPIC_BASE_URL", "")
	raw := []byte(`[{"name":"Read"},{"name":"ToolSearch","input_schema":{"type":"object"}}]`)
	b := BetasForToolsJSON(raw)
	if len(b) != 1 || b[0] != toolSearchBeta1P {
		t.Fatalf("got %v want [%s]", b, toolSearchBeta1P)
	}
}

func TestBetasForToolsJSON_toolSearch_vertexBeta(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ANTHROPIC_BASE_URL", "https://vertex.example/v1")
	raw := []byte(`[{"name":"ToolSearch"}]`)
	b := BetasForToolsJSON(raw)
	if len(b) != 1 || b[0] != toolSearchBeta3P {
		t.Fatalf("got %v want [%s]", b, toolSearchBeta3P)
	}
}

func TestBetasForToolsJSON_killSwitch(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "1")
	raw := []byte(`[{"name":"ToolSearch"}]`)
	if b := BetasForToolsJSON(raw); len(b) != 0 {
		t.Fatalf("got %v", b)
	}
}
