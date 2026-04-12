package toolsearch

import (
	"encoding/json"
	"testing"
)

func TestWireToolsJSON_openAICompat_dynamicFirstTurn(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "")
	t.Setenv("ENABLE_TOOL_SEARCH", "true")
	raw := []byte(`[
	  {"name":"Agent","description":"a","input_schema":{"type":"object"}},
	  {"name":"ToolSearch","description":"s","input_schema":{"type":"object"}},
	  {"name":"CronCreate","description":"c","input_schema":{"type":"object"}},
	  {"name":"Read","description":"r","input_schema":{"type":"object"}}
	]`)
	msgs := json.RawMessage(`[{"role":"user","content":"hi"}]`)
	out, err := WireToolsJSON(raw, "deepseek-chat", false, true, msgs)
	if err != nil {
		t.Fatal(err)
	}
	var got []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"Agent": true, "ToolSearch": true, "Read": true}
	if len(got) != len(want) {
		t.Fatalf("got %d tools %+v", len(got), got)
	}
	for _, g := range got {
		if !want[g.Name] {
			t.Errorf("unexpected %q", g.Name)
		}
	}
}

func TestWireToolsJSON_emptyPassthrough(t *testing.T) {
	out, err := WireToolsJSON(nil, "m", false, false, nil)
	if err != nil || out != nil {
		t.Fatalf("out=%v err=%v", out, err)
	}
}
