package toolsearch

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/ccb-engine/internal/anthropic"
)

func TestExecToolSearchForRunner_selectDeferred(t *testing.T) {
	// TodoWrite deferred + Read non-deferred (TS still resolves select:Read from full registry).
	in := []byte(`{"query":"select:TodoWrite,Read","max_results":10}`)
	s, isErr, err := ExecToolSearchForRunner(in, sampleToolsAPIStyle(), false, nil)
	if err != nil || isErr {
		t.Fatalf("err=%v isErr=%v", err, isErr)
	}
	var refs []struct {
		Type     string `json:"type"`
		ToolName string `json:"tool_name"`
	}
	if err := json.Unmarshal([]byte(s), &refs); err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("got %#v", refs)
	}
}

func TestExecToolSearchForRunner_selectNonDeferredLikeTS(t *testing.T) {
	// TS: findToolByName(deferred) ?? findToolByName(tools) — Read is not deferred but select:Read still resolves.
	in := []byte(`{"query":"select:Read","max_results":5}`)
	s, isErr, err := ExecToolSearchForRunner(in, sampleToolsAPIStyle(), false, nil)
	if err != nil || isErr {
		t.Fatalf("err=%v isErr=%v", err, isErr)
	}
	var refs []struct {
		ToolName string `json:"tool_name"`
	}
	if err := json.Unmarshal([]byte(s), &refs); err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].ToolName != "Read" {
		t.Fatalf("got %#v", refs)
	}
}

func TestExecToolSearchForRunner_noMatchesPlainText(t *testing.T) {
	in := []byte(`{"query":"select:DefinitelyNotATool","max_results":5}`)
	s, isErr, err := ExecToolSearchForRunner(in, sampleToolsAPIStyle(), false, nil)
	if err != nil || isErr {
		t.Fatalf("err=%v isErr=%v", err, isErr)
	}
	if s != "No matching deferred tools found" {
		t.Fatalf("got %q", s)
	}
}

func TestExecToolSearchForRunner_builtinOnlyEmptyWithPendingMCP(t *testing.T) {
	in := []byte(`{"query":"select:DefinitelyNotATool","max_results":5}`)
	s, isErr, err := ExecToolSearchForRunner(in, nil, true, nil)
	if err != nil || isErr {
		t.Fatalf("err=%v isErr=%v", err, isErr)
	}
	if !strings.Contains(s, "still connecting") {
		t.Fatalf("got %q", s)
	}
}

func TestExecToolSearchForRunner_noMatchesWithPendingMCP(t *testing.T) {
	in := []byte(`{"query":"select:DefinitelyNotATool","max_results":5}`)
	s, isErr, err := ExecToolSearchForRunner(in, sampleToolsAPIStyle(), true, []string{"slack", "github"})
	if err != nil || isErr {
		t.Fatalf("err=%v isErr=%v", err, isErr)
	}
	wantSub := "Some MCP servers are still connecting: slack, github"
	if !strings.Contains(s, wantSub) {
		t.Fatalf("got %q", s)
	}
}

func TestExtractDiscovered_toolResultStringToolReferenceArray(t *testing.T) {
	body := `[{"type":"tool_reference","tool_name":"TodoWrite"}]`
	msgs := []anthropic.Message{
		{Role: "user", Content: []anthropic.ContentBlock{
			{Type: "tool_result", ToolUseID: "1", Content: body},
		}},
	}
	out := ExtractDiscoveredToolNames(msgs)
	if _, ok := out["TodoWrite"]; !ok {
		t.Fatalf("got %#v", out)
	}
}

func TestExtractDiscovered_toolResultStringWithDiscoveryWrapper(t *testing.T) {
	body := `{"discovery":[{"type":"tool_reference","tool_name":"TodoWrite"}],"note":"x"}`
	msgs := []anthropic.Message{
		{Role: "user", Content: []anthropic.ContentBlock{
			{Type: "tool_result", ToolUseID: "1", Content: body},
		}},
	}
	out := ExtractDiscoveredToolNames(msgs)
	if _, ok := out["TodoWrite"]; !ok {
		t.Fatalf("got %#v", out)
	}
}
