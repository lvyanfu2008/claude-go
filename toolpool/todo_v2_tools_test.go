package toolpool

import (
	"slices"
	"testing"

	"goc/types"
)

func todoV2GateBaseTools() []types.ToolSpec {
	return []types.ToolSpec{
		{Name: "Read"},
		{Name: "TodoWrite"},
		{Name: "TaskCreate"},
		{Name: "TaskGet"},
		{Name: "TaskList"},
		{Name: "TaskUpdate"},
		{Name: "Bash"},
	}
}

func TestGetTools_todoV2InteractiveShowsTaskToolsNotTodoWrite(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("CLAUDE_REPL_MODE", "")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("CLAUDE_CODE_ENABLE_TASKS", "")
	t.Setenv("GOU_DEMO_NON_INTERACTIVE", "")
	t.Setenv("CLAUDE_CODE_NON_INTERACTIVE", "")

	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, todoV2GateBaseTools())
	names := toolNames(out)
	if slices.Contains(names, "TodoWrite") {
		t.Fatalf("expected TodoWrite hidden when Todo v2 on, got %v", names)
	}
	for _, want := range []string{"TaskCreate", "TaskGet", "TaskList", "TaskUpdate"} {
		if !slices.Contains(names, want) {
			t.Fatalf("missing %s in %v", want, names)
		}
	}
}

func TestGetTools_todoV2NonInteractiveShowsTodoWriteNotTaskTools(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("CLAUDE_REPL_MODE", "")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("CLAUDE_CODE_ENABLE_TASKS", "")
	t.Setenv("GOU_DEMO_NON_INTERACTIVE", "1")
	t.Setenv("CLAUDE_CODE_NON_INTERACTIVE", "")

	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, todoV2GateBaseTools())
	names := toolNames(out)
	if !slices.Contains(names, "TodoWrite") {
		t.Fatalf("expected TodoWrite when Todo v2 off, got %v", names)
	}
	for _, hide := range []string{"TaskCreate", "TaskGet", "TaskList", "TaskUpdate"} {
		if slices.Contains(names, hide) {
			t.Fatalf("did not expect %s when non-interactive without ENABLE_TASKS, got %v", hide, names)
		}
	}
}

func TestGetTools_todoV2NonInteractiveWithEnableTasksShowsTaskTools(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("CLAUDE_REPL_MODE", "")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("CLAUDE_CODE_ENABLE_TASKS", "1")
	t.Setenv("GOU_DEMO_NON_INTERACTIVE", "1")
	t.Setenv("CLAUDE_CODE_NON_INTERACTIVE", "")

	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, todoV2GateBaseTools())
	names := toolNames(out)
	if slices.Contains(names, "TodoWrite") {
		t.Fatalf("expected TodoWrite hidden when ENABLE_TASKS forces v2, got %v", names)
	}
	for _, want := range []string{"TaskCreate", "TaskGet", "TaskList", "TaskUpdate"} {
		if !slices.Contains(names, want) {
			t.Fatalf("missing %s in %v", want, names)
		}
	}
}

func TestGetTools_todoV2NonInteractiveViaClaudeCodeEnv(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SIMPLE", "")
	t.Setenv("CLAUDE_CODE_REPL", "0")
	t.Setenv("CLAUDE_REPL_MODE", "")
	t.Setenv("USER_TYPE", "")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("CLAUDE_CODE_ENABLE_TASKS", "")
	t.Setenv("GOU_DEMO_NON_INTERACTIVE", "")
	t.Setenv("CLAUDE_CODE_NON_INTERACTIVE", "1")

	ctx := types.EmptyToolPermissionContextData()
	out := GetTools(ctx, todoV2GateBaseTools())
	names := toolNames(out)
	if !slices.Contains(names, "TodoWrite") {
		t.Fatalf("expected TodoWrite, got %v", names)
	}
	if slices.Contains(names, "TaskCreate") {
		t.Fatalf("did not expect Task tools, got %v", names)
	}
}
