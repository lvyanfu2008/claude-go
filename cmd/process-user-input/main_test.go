package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"goc/commands"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/types"
)

func TestApplyGoCommandsLoad_loadsFromCwd(t *testing.T) {
	commands.ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := &processuserinput.ProcessUserInputParams{
		RuntimeContext: &types.ProcessUserInputContextData{
			ToolUseContext: types.ToolUseContext{
				Options: types.ToolUseContextOptionsData{},
			},
		},
	}
	load := &goCommandsLoad{Cwd: tmp}
	if err := applyGoCommandsLoad(context.Background(), p, load); err != nil {
		t.Fatal(err)
	}
	if len(p.Commands) < 40 {
		t.Fatalf("expected filled builtins tail, got len=%d", len(p.Commands))
	}
	if len(p.RuntimeContext.Options.Commands) != len(p.Commands) {
		t.Fatalf("runtime options.commands out of sync")
	}
}

func TestApplyGoCommandsLoad_replacesStaleCommands(t *testing.T) {
	commands.ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := &processuserinput.ProcessUserInputParams{
		Commands: []types.Command{{CommandBase: types.CommandBase{Name: "x", Description: "y"}, Type: "prompt"}},
		RuntimeContext: &types.ProcessUserInputContextData{
			ToolUseContext: types.ToolUseContext{
				Options: types.ToolUseContextOptionsData{
					Commands: []types.Command{{CommandBase: types.CommandBase{Name: "x", Description: "y"}, Type: "prompt"}},
				},
			},
		},
	}
	load := &goCommandsLoad{Cwd: tmp}
	if err := applyGoCommandsLoad(context.Background(), p, load); err != nil {
		t.Fatal(err)
	}
	if len(p.Commands) < 40 {
		t.Fatalf("expected Go-loaded builtins, got len=%d", len(p.Commands))
	}
	for _, c := range p.Commands {
		if c.Name == "x" {
			t.Fatalf("expected stale placeholder command replaced by Go load")
		}
	}
}

func TestResolveCwdForGoCommands_prefersLoadOverEnv(t *testing.T) {
	t.Setenv(envProcessUserInputCwd, "/from-env")
	got, err := resolveCwdForGoCommands(&goCommandsLoad{Cwd: "/from-load"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/from-load" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveCwdForGoCommands_usesEnvWhenLoadEmpty(t *testing.T) {
	t.Setenv(envProcessUserInputCwd, "/env-only")
	got, err := resolveCwdForGoCommands(&goCommandsLoad{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/env-only" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyGoCommandsLoad_nilLoadUsesEnvCwd(t *testing.T) {
	commands.ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	t.Setenv(envProcessUserInputCwd, tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := &processuserinput.ProcessUserInputParams{
		RuntimeContext: &types.ProcessUserInputContextData{
			ToolUseContext: types.ToolUseContext{
				Options: types.ToolUseContextOptionsData{},
			},
		},
	}
	if err := applyGoCommandsLoad(context.Background(), p, nil); err != nil {
		t.Fatal(err)
	}
	if len(p.Commands) < 40 {
		t.Fatalf("expected Go load via %s, got len=%d", envProcessUserInputCwd, len(p.Commands))
	}
}

func TestBuildStdoutEnvelope_Result(t *testing.T) {
	out := &processuserinput.ProcessUserInputBaseResult{
		Messages:    nil,
		ShouldQuery: false,
	}
	env := buildStdoutEnvelope(out)
	if env.Kind != "result" {
		t.Fatalf("expected kind result, got %q", env.Kind)
	}
	if env.Result == nil {
		t.Fatalf("expected result payload")
	}
}

func TestBuildStdoutEnvelope_IncludesExecutionInsideResult(t *testing.T) {
	out := &processuserinput.ProcessUserInputBaseResult{
		Execution: &processuserinput.ExecutionRequest{Kind: "bash", Command: "echo hi"},
	}
	env := buildStdoutEnvelope(out)
	if env.Kind != "result" {
		t.Fatalf("expected kind result, got %q", env.Kind)
	}
	if env.Result == nil || env.Result.Execution == nil || env.Result.Execution.Kind != "bash" {
		t.Fatalf("expected bash execution inside result, got %#v", env.Result)
	}
}

func TestBuildStdoutEnvelope_QueryStillUsesResultKind(t *testing.T) {
	out := &processuserinput.ProcessUserInputBaseResult{
		Messages:    nil,
		ShouldQuery: true,
	}
	env := buildStdoutEnvelope(out)
	if env.Kind != "result" {
		t.Fatalf("expected kind result, got %q", env.Kind)
	}
	if env.Result == nil || !env.Result.ShouldQuery {
		t.Fatalf("expected result payload with shouldQuery")
	}
}

func TestBuildResultPayload_IncludesExecutionKind(t *testing.T) {
	p := buildResultPayload("go-cli", &processuserinput.ProcessUserInputBaseResult{
		Messages:    nil,
		ShouldQuery: false,
		Execution: &processuserinput.ExecutionRequest{
			Kind:    "bash",
			Command: "echo hello",
		},
	})
	v, ok := p["executionKind"].(string)
	if !ok || v != "bash" {
		t.Fatalf("expected executionKind bash, got %#v", p["executionKind"])
	}
}

func TestStdoutEnvelope_CarriesStatePatchBatch(t *testing.T) {
	out := &processuserinput.ProcessUserInputBaseResult{
		Messages:    nil,
		ShouldQuery: false,
		StatePatchBatch: &processuserinput.StatePatchBatch{
			PatchID:     "p1",
			BaseVersion: 1,
			Patches: []processuserinput.StatePatch{
				{Op: "clear_session_hooks"},
			},
		},
	}
	env := buildStdoutEnvelope(out)
	env.StatePatchBatch = out.StatePatchBatch
	if env.StatePatchBatch == nil || env.StatePatchBatch.PatchID != "p1" {
		t.Fatalf("expected statePatchBatch to be attached")
	}
}

