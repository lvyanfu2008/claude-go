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
	env := buildStdoutEnvelope(out, false)
	if env.Kind != stdoutKindResult {
		t.Fatalf("expected kind result, got %q", env.Kind)
	}
	if env.Result == nil {
		t.Fatalf("expected result payload")
	}
}

func TestBuildStdoutEnvelope_ExecutionRequestWins(t *testing.T) {
	out := &processuserinput.ProcessUserInputBaseResult{
		Execution: &processuserinput.ExecutionRequest{Kind: "bash", Command: "echo hi"},
	}
	env := buildStdoutEnvelope(out, true)
	if env.Kind != stdoutKindExecutionRequest {
		t.Fatalf("expected kind execution_request, got %q", env.Kind)
	}
	if env.Action == nil || env.Action.Kind != "bash" {
		t.Fatalf("expected bash action payload")
	}
}

func TestBuildStdoutEnvelope_ExecutionSequenceWins(t *testing.T) {
	out := &processuserinput.ProcessUserInputBaseResult{
		ExecutionSequence: []processuserinput.ExecutionRequest{
			{Kind: "attachments_plan", Input: "hi @f"},
			{Kind: "hooks_plan", Input: "hi"},
		},
	}
	env := buildStdoutEnvelope(out, true)
	if env.Kind != stdoutKindExecutionRequest {
		t.Fatalf("expected kind execution_request, got %q", env.Kind)
	}
	if len(env.Actions) != 2 || env.Actions[0].Kind != "attachments_plan" || env.Actions[1].Kind != "hooks_plan" {
		t.Fatalf("expected actions sequence, got %#v", env.Actions)
	}
	if env.Action == nil || env.Action.Kind != "attachments_plan" {
		t.Fatalf("expected action mirror first step, got %#v", env.Action)
	}
}

func TestBuildStdoutEnvelope_ExecutionResultWhenEnabled(t *testing.T) {
	out := &processuserinput.ProcessUserInputBaseResult{
		Messages:    nil,
		ShouldQuery: true,
	}
	env := buildStdoutEnvelope(out, true)
	if env.Kind != stdoutKindExecutionResult {
		t.Fatalf("expected kind execution_result, got %q", env.Kind)
	}
	if env.ExecutionResult == nil || !env.ExecutionResult.ShouldQuery {
		t.Fatalf("expected execution_result payload")
	}
}

func TestMaybeConsumeExecutionResultFromStdin(t *testing.T) {
	in := &processuserinput.ProcessUserInputBaseResult{
		Messages:    nil,
		ShouldQuery: true,
	}
	got := maybeConsumeExecutionResultFromStdin(
		stdinEnvelope{ExecutionResult: in},
		true,
	)
	if got == nil {
		t.Fatalf("expected consumed output envelope")
	}
	if got.Kind != stdoutKindExecutionResult {
		t.Fatalf("expected execution_result kind, got %q", got.Kind)
	}
	if got.ExecutionResult == nil || !got.ExecutionResult.ShouldQuery {
		t.Fatalf("expected execution_result payload")
	}

	notConsumed := maybeConsumeExecutionResultFromStdin(
		stdinEnvelope{ExecutionResult: in},
		false,
	)
	if notConsumed != nil {
		t.Fatalf("expected nil when consume flag disabled")
	}
}

func TestBuildResultPayload_IncludesExecutionKind(t *testing.T) {
	p := buildResultPayload("go-cli", &processuserinput.ProcessUserInputBaseResult{
		Messages:    nil,
		ShouldQuery: false,
		Execution: &processuserinput.ExecutionRequest{
			Kind:  "hooks_plan",
			Input: "hello",
		},
	})
	v, ok := p["executionKind"].(string)
	if !ok || v != "hooks_plan" {
		t.Fatalf("expected executionKind hooks_plan, got %#v", p["executionKind"])
	}
}

func TestBuildResultPayload_IncludesExecutionKindsSequence(t *testing.T) {
	p := buildResultPayload("go-cli", &processuserinput.ProcessUserInputBaseResult{
		Messages:    nil,
		ShouldQuery: false,
		ExecutionSequence: []processuserinput.ExecutionRequest{
			{Kind: "attachments_plan", Input: "x"},
			{Kind: "hooks_plan", Input: "x"},
		},
	})
	raw, ok := p["executionKinds"].([]string)
	if !ok || len(raw) != 2 || raw[0] != "attachments_plan" || raw[1] != "hooks_plan" {
		t.Fatalf("expected executionKinds slice, got %#v", p["executionKinds"])
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
	env := buildStdoutEnvelope(out, false)
	env.StatePatchBatch = out.StatePatchBatch
	if env.StatePatchBatch == nil || env.StatePatchBatch.PatchID != "p1" {
		t.Fatalf("expected statePatchBatch to be attached")
	}
}

func TestMaybeConsumeExecutionResultFromStdin_DoesNotDropPatchBatch(t *testing.T) {
	in := &processuserinput.ProcessUserInputBaseResult{
		Messages:    nil,
		ShouldQuery: true,
		StatePatchBatch: &processuserinput.StatePatchBatch{
			PatchID:     "p2",
			BaseVersion: 2,
			Patches: []processuserinput.StatePatch{
				{Op: "register_hook_callbacks"},
			},
		},
	}
	got := maybeConsumeExecutionResultFromStdin(stdinEnvelope{ExecutionResult: in}, true)
	if got == nil || got.ExecutionResult == nil || got.ExecutionResult.StatePatchBatch == nil {
		t.Fatalf("expected executionResult with statePatchBatch")
	}
	if got.ExecutionResult.StatePatchBatch.PatchID != "p2" {
		t.Fatalf("unexpected patch id: %+v", got.ExecutionResult.StatePatchBatch.PatchID)
	}
}

func TestMaybeConsumeExecutionResultFromStdin_HooksReducerForcesNoQuery(t *testing.T) {
	in := &processuserinput.ProcessUserInputBaseResult{
		Messages:    nil,
		ShouldQuery: true,
		HooksReducerInput: &processuserinput.HooksReducerInput{
			PreventContinuation: true,
		},
	}
	got := maybeConsumeExecutionResultFromStdin(
		stdinEnvelope{
			ExecutionResult:     in,
			ExecutionResultKind: "hooks_plan",
		},
		true,
	)
	if got == nil || got.ExecutionResult == nil {
		t.Fatalf("expected execution_result envelope")
	}
	if got.ExecutionResult.ShouldQuery {
		t.Fatalf("expected reducer to force shouldQuery=false")
	}
}
