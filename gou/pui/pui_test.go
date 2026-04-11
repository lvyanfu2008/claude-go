package pui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/gou/conversation"
	"goc/tscontext"
	"goc/types"
)

func clearModelEnvForPuiTest(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"CCB_ENGINE_MODEL",
		"ANTHROPIC_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
	} {
		t.Setenv(k, "")
	}
}

func TestSlashGated(t *testing.T) {
	if !SlashGated(" /foo") {
		t.Fatal("expected gated")
	}
	if SlashGated("hello") {
		t.Fatal("expected not gated")
	}
}

func TestBuildDemoParams_andProcessUserInput_happyPath(t *testing.T) {
	st := &conversation.Store{ConversationID: "t"}
	p, err := BuildDemoParams("hello world", st, DemoConfig{SkipCommands: true})
	if err != nil {
		t.Fatal(err)
	}
	r, err := processuserinput.ProcessUserInput(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if !r.ShouldQuery || len(r.Messages) == 0 {
		t.Fatalf("unexpected result: shouldQuery=%v messages=%d", r.ShouldQuery, len(r.Messages))
	}
	before := len(st.Messages)
	out := ApplyBaseResult(st, r, nil)
	if len(st.Messages) != before+len(r.Messages) {
		t.Fatalf("store len: before=%d after=%d want +%d", before, len(st.Messages), len(r.Messages))
	}
	if !out.EffectiveShouldQuery || out.HadExecutionRequest {
		t.Fatalf("out=%+v", out)
	}
}

func TestApplyBaseResult_executionStub(t *testing.T) {
	st := &conversation.Store{}
	r := &processuserinput.ProcessUserInputBaseResult{
		Execution: &processuserinput.ExecutionRequest{Kind: "slash", CommandName: "compact"},
	}
	out := ApplyBaseResult(st, r, nil)
	if !out.HadExecutionRequest || out.EffectiveShouldQuery {
		t.Fatalf("out=%+v", out)
	}
	if len(st.Messages) != 1 {
		t.Fatalf("messages=%d", len(st.Messages))
	}
}

func TestBuildDemoParams_anthropicModelOverridesTSBridgeMainLoopModel(t *testing.T) {
	clearModelEnvForPuiTest(t)
	t.Setenv("ANTHROPIC_MODEL", "deepseek-chat")
	st := &conversation.Store{ConversationID: "t"}
	p, err := BuildDemoParams("hi", st, DemoConfig{
		SkipCommands:    true,
		TSContextBridge: &tscontext.Snapshot{MainLoopModel: "claude-sonnet-4-20250514"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.RuntimeContext == nil {
		t.Fatal("nil RuntimeContext")
	}
	got := strings.TrimSpace(p.RuntimeContext.ToolUseContext.Options.MainLoopModel)
	if got != "deepseek-chat" {
		t.Fatalf("MainLoopModel=%q want deepseek-chat", got)
	}
}

func TestBuildDemoParams_explicitMainLoopModelBeatsAnthropicEnv(t *testing.T) {
	clearModelEnvForPuiTest(t)
	t.Setenv("ANTHROPIC_MODEL", "deepseek-chat")
	st := &conversation.Store{ConversationID: "t"}
	p, err := BuildDemoParams("hi", st, DemoConfig{
		SkipCommands:    true,
		MainLoopModel:   "custom-from-config",
		TSContextBridge: &tscontext.Snapshot{MainLoopModel: "claude-sonnet-4-20250514"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(p.RuntimeContext.ToolUseContext.Options.MainLoopModel)
	if got != "custom-from-config" {
		t.Fatalf("MainLoopModel=%q want custom-from-config", got)
	}
}

func TestBuildDemoParams_skillListingCommandsSet(t *testing.T) {
	st := &conversation.Store{ConversationID: "t"}
	p, err := BuildDemoParams("hello", st, DemoConfig{SkipCommands: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.SkillListingCommands == nil {
		t.Fatal("expected non-nil slice")
	}
}

func TestBuildDemoParams_mcpCommandsJSONPath(t *testing.T) {
	t.Setenv("FEATURE_MCP_SKILLS", "1")
	tmp := t.TempDir()
	path := filepath.Join(tmp, "mcp.json")
	const data = `[{"type":"prompt","name":"from_json_file","description":"via R1","hasUserSpecifiedDescription":true,"loadedFrom":"mcp"}]`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	st := &conversation.Store{ConversationID: "t"}
	p, err := BuildDemoParams("hello", st, DemoConfig{SkipCommands: true, MCPCommandsJSONPath: path})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range p.Commands {
		if c.Name == "from_json_file" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("commands=%v", p.Commands)
	}
}

func TestBuildDemoParams_mcpCommandsJSONPath_error(t *testing.T) {
	st := &conversation.Store{ConversationID: "t"}
	_, err := BuildDemoParams("hello", st, DemoConfig{SkipCommands: true, MCPCommandsJSONPath: "/nonexistent/gou-demo-mcp.json"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildDemoParams_mcpMergeForCommands(t *testing.T) {
	t.Setenv("FEATURE_MCP_SKILLS", "1")
	st := &conversation.Store{ConversationID: "t"}
	mcp := []types.Command{{
		CommandBase: types.CommandBase{
			Name:        "mcp_skill",
			LoadedFrom:  strPtr("mcp"),
			Description: "d",
		},
		Type: "prompt",
	}}
	p, err := BuildDemoParams("hello", st, DemoConfig{SkipCommands: true, MCPCommands: mcp})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range p.Commands {
		if c.Name == "mcp_skill" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("commands=%v", p.Commands)
	}
}

func strPtr(s string) *string { return &s }

func TestBuildDemoParams_loadsSlashCommandsWhenNotSkipped(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	st := &conversation.Store{ConversationID: "t"}
	p, err := BuildDemoParams("hello", st, DemoConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Commands) < 40 {
		t.Fatalf("expected LoadAndFilterCommands tail, got len=%d", len(p.Commands))
	}
}

func TestApplyProcessUserInputBaseResult_fillsHandoff(t *testing.T) {
	st := &conversation.Store{}
	p, err := BuildDemoParams("hint test", st, DemoConfig{SkipCommands: true})
	if err != nil {
		t.Fatal(err)
	}
	r, err := processuserinput.ProcessUserInput(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	var handoff ProcessUserInputBaseResultHandoff
	ApplyProcessUserInputBaseResult(st, r, &handoff)
	if !handoff.ShouldQuery {
		t.Fatalf("handoff.shouldQuery=false, handoff=%+v", handoff)
	}
	h2 := HandoffFromProcessUserInputBaseResult(r)
	if h2.ShouldQuery != handoff.ShouldQuery || h2.Model != handoff.Model {
		t.Fatalf("HandoffFrom mismatch: handoff=%+v h2=%+v", handoff, h2)
	}
}

func TestBuildDemoParams_useEmbeddedToolsAPI(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	st := &conversation.Store{}
	p, err := BuildDemoParams("hello", st, DemoConfig{SkipCommands: true, UseEmbeddedToolsAPI: true})
	if err != nil {
		t.Fatal(err)
	}
	raw := p.RuntimeContext.ToolUseContext.Options.Tools
	var defs []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &defs); err != nil {
		t.Fatal(err)
	}
	if len(defs) < 5 {
		t.Fatalf("expected several tools, got %d", len(defs))
	}
	seenAgent := false
	for _, d := range defs {
		if d.Name == "Agent" {
			seenAgent = true
			break
		}
	}
	if !seenAgent {
		t.Fatal("expected Agent in embedded tools list")
	}
}

func TestBuildDemoParams_mcpToolsMergedWithEmbedded(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	st := &conversation.Store{}
	path := filepath.Join("..", "..", "mcpcommands", "testdata", "mcp_tools_sample.json")
	p, err := BuildDemoParams("hello", st, DemoConfig{
		SkipCommands:        true,
		UseEmbeddedToolsAPI: true,
		MCPToolsJSONPath:    path,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := p.RuntimeContext.ToolUseContext.Options.Tools
	var defs []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &defs); err != nil {
		t.Fatal(err)
	}
	want := "mcp__fixture__ping"
	found := false
	for _, d := range defs {
		if d.Name == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("want %q in tools, got %d defs", want, len(defs))
	}
}

func TestBuildDemoParams_TSContextBridge_skillListingFromSnapshot(t *testing.T) {
	st := &conversation.Store{}
	skillJSON := `[{"type":"prompt","name":"ts_skill_only","description":"d","loadedFrom":"bundled"}]`
	snap := &tscontext.Snapshot{
		Commands:            json.RawMessage(`[{"type":"prompt","name":"full","description":"x"}]`),
		Tools:               json.RawMessage(`[{"name":"t","description":"d","input_schema":{"type":"object"}}]`),
		SkillToolCommands:   json.RawMessage(skillJSON),
		MainLoopModel:       "m",
	}
	p, err := BuildDemoParams("hi", st, DemoConfig{
		SkipCommands:    true,
		TSContextBridge: snap,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.SkillListingCommands) != 1 || p.SkillListingCommands[0].Name != "ts_skill_only" {
		t.Fatalf("listing: %+v", p.SkillListingCommands)
	}
}

func TestBuildDemoParams_TSContextBridge_toolsAndCommands(t *testing.T) {
	clearModelEnvForPuiTest(t)
	st := &conversation.Store{}
	cmds := `[{"type":"prompt","name":"from_ts","description":"d"}]`
	tools := `[{"name":"echo_stub","description":"x","input_schema":{"type":"object","properties":{}}}]`
	snap := &tscontext.Snapshot{
		Commands:      json.RawMessage(cmds),
		Tools:         json.RawMessage(tools),
		MainLoopModel: "bridge-model",
	}
	p, err := BuildDemoParams("hi", st, DemoConfig{
		SkipCommands:    true,
		TSContextBridge: snap,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.RuntimeContext.ToolUseContext.Options.MainLoopModel != "bridge-model" {
		t.Fatalf("MainLoopModel %q", p.RuntimeContext.ToolUseContext.Options.MainLoopModel)
	}
	var names []string
	for _, c := range p.Commands {
		names = append(names, c.Name)
	}
	if len(names) != 1 || names[0] != "from_ts" {
		t.Fatalf("commands: %#v", names)
	}
	var toolDefs []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(p.RuntimeContext.ToolUseContext.Options.Tools, &toolDefs); err != nil {
		t.Fatal(err)
	}
	if len(toolDefs) != 1 || toolDefs[0].Name != "echo_stub" {
		t.Fatalf("tools: %#v", toolDefs)
	}
}
