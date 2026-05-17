package slashresolve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecuteStarlarkSkill_basic(t *testing.T) {
	script := `
def resolve(args):
    return "# Hello\n\nThis is a test skill."
`
	result, err := ExecuteStarlarkSkill(script, "", &StarlarkContext{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "# Hello\n\nThis is a test skill." {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_withArgs(t *testing.T) {
	script := `
def resolve(args):
    return "Args: " + args
`
	result, err := ExecuteStarlarkSkill(script, "hello world", &StarlarkContext{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Args: hello world" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_env(t *testing.T) {
	os.Setenv("STARLARK_TEST_KEY", "test_value")
	defer os.Unsetenv("STARLARK_TEST_KEY")

	script := `
def resolve(args):
    return env("STARLARK_TEST_KEY")
`
	result, err := ExecuteStarlarkSkill(script, "", &StarlarkContext{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test_value" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_envDefault(t *testing.T) {
	script := `
def resolve(args):
    return env("NONEXISTENT_VAR_12345", "fallback")
`
	result, err := ExecuteStarlarkSkill(script, "", &StarlarkContext{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "fallback" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_cwd(t *testing.T) {
	script := `
def resolve(args):
    return cwd()
`
	sctx := &StarlarkContext{Cwd: "/test/dir"}
	result, err := ExecuteStarlarkSkill(script, "", sctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/test/dir" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_sessionID(t *testing.T) {
	script := `
def resolve(args):
    return session_id()
`
	sctx := &StarlarkContext{SessionID: "session-123"}
	result, err := ExecuteStarlarkSkill(script, "", sctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "session-123" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_userType(t *testing.T) {
	os.Setenv("USER_TYPE", "ant")
	defer os.Unsetenv("USER_TYPE")

	script := `
def resolve(args):
    return user_type()
`
	result, err := ExecuteStarlarkSkill(script, "", &StarlarkContext{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ant" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_isDemo(t *testing.T) {
	os.Setenv("IS_DEMO", "1")
	defer os.Unsetenv("IS_DEMO")

	script := `
def resolve(args):
    if is_demo():
        return "demo_mode"
    return "normal"
`
	result, err := ExecuteStarlarkSkill(script, "", &StarlarkContext{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "demo_mode" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_featureEnabled(t *testing.T) {
	script := `
def resolve(args):
    if feature_enabled("AGENT_TRIGGERS"):
        return "enabled"
    return "disabled"
`
	result, err := ExecuteStarlarkSkill(script, "", &StarlarkContext{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "enabled" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_fileExists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)

	script := `
def resolve(args):
    if file_exists("test.txt"):
        return "exists"
    return "not_found"
`
	sctx := &StarlarkContext{Cwd: dir}
	result, err := ExecuteStarlarkSkill(script, "", sctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "exists" {
		t.Fatalf("unexpected result: %q", result)
	}

	script2 := `
def resolve(args):
    if file_exists("nonexistent.txt"):
        return "exists"
    return "not_found"
`
	result, err = ExecuteStarlarkSkill(script2, "", sctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "not_found" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_readFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644)

	script := `
def resolve(args):
    return read_file("test.txt")
`
	sctx := &StarlarkContext{Cwd: dir}
	result, err := ExecuteStarlarkSkill(script, "", sctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_readFile_pathTraversal(t *testing.T) {
	script := `
def resolve(args):
    return read_file("../../../etc/passwd")
`
	sctx := &StarlarkContext{Cwd: "/tmp"}
	_, err := ExecuteStarlarkSkill(script, "", sctx, "")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestExecuteStarlarkSkill_strContains(t *testing.T) {
	script := `
def resolve(args):
    if str_contains("hello world", "world"):
        return "yes"
    return "no"
`
	result, err := ExecuteStarlarkSkill(script, "", &StarlarkContext{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "yes" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_strJoin(t *testing.T) {
	script := `
def resolve(args):
    parts = ["a", "b", "c"]
    return str_join(parts, "-")
`
	result, err := ExecuteStarlarkSkill(script, "", &StarlarkContext{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "a-b-c" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_noResolveFunc(t *testing.T) {
	script := `x = 1`
	_, err := ExecuteStarlarkSkill(script, "", &StarlarkContext{}, "")
	if err == nil {
		t.Fatal("expected error for missing resolve function")
	}
}

func TestExecuteStarlarkSkill_syntaxError(t *testing.T) {
	script := `def resolve(args) return "broken"`
	_, err := ExecuteStarlarkSkill(script, "", &StarlarkContext{}, "")
	if err == nil {
		t.Fatal("expected syntax error")
	}
}

func TestExecuteStarlarkSkill_conditional(t *testing.T) {
	os.Setenv("MY_FLAG", "true")
	defer os.Unsetenv("MY_FLAG")

	script := `
def resolve(args):
    prompt = "Header\n\n"
    if env("MY_FLAG") == "true":
        prompt += "Flag is enabled.\n"
    else:
        prompt += "Flag is disabled.\n"
    if args:
        prompt += "\nArgs: " + args + "\n"
    return prompt
`
	result, err := ExecuteStarlarkSkill(script, "test-arg", &StarlarkContext{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Header\n\nFlag is enabled.\n\nArgs: test-arg\n"
	if result != expected {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestExecuteStarlarkSkill_nilContext(t *testing.T) {
	script := `
def resolve(args):
    return cwd()
`
	result, err := ExecuteStarlarkSkill(script, "", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty cwd for nil context, got %q", result)
	}
}

func TestExecuteStarlarkSkill_sessionDebugLogPath(t *testing.T) {
	script := `
def resolve(args):
    return session_debug_log_path()
`
	sctx := &StarlarkContext{SessionID: "test-session"}
	result, err := ExecuteStarlarkSkill(script, "", sctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty debug log path")
	}
}
