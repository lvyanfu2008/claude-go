package settingsfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyProjectClaudeEnv_missingFile(t *testing.T) {
	dir := t.TempDir()
	if err := ApplyProjectClaudeEnv(dir); err != nil {
		t.Fatal(err)
	}
}

func TestApplyProjectClaudeEnv_appliesEnvAndSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")
	content := `{
  "env": {
    "CCB_SETTINGSFILE_TEST_A": "from_settings",
    "CCB_SETTINGSFILE_TEST_B": true,
    "CCB_SETTINGSFILE_TEST_C": 42
  }
}`
	if err := os.WriteFile(settingsPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CCB_SETTINGSFILE_TEST_A", "from_shell")
	t.Setenv("CCB_SETTINGSFILE_TEST_B", "")
	t.Setenv("CCB_SETTINGSFILE_TEST_C", "")

	if err := ApplyProjectClaudeEnv(dir); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("CCB_SETTINGSFILE_TEST_A"); got != "from_shell" {
		t.Fatalf("A: want from_shell, got %q", got)
	}
	if got := os.Getenv("CCB_SETTINGSFILE_TEST_B"); got != "1" {
		t.Fatalf("B: want 1, got %q", got)
	}
	if got := os.Getenv("CCB_SETTINGSFILE_TEST_C"); got != "42" {
		t.Fatalf("C: want 42, got %q", got)
	}
}

func TestApplyProjectClaudeEnv_invalidJSON(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyProjectClaudeEnv(dir); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestUserClaudeSettingsPath_respectsClaudeConfigDir(t *testing.T) {
	d := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", d)
	got := UserClaudeSettingsPath()
	want := filepath.Join(d, "settings.json")
	if got != want {
		t.Fatalf("UserClaudeSettingsPath: got %q want %q", got, want)
	}
}

func TestApplyMergedClaudeSettingsEnv_goUserLocalOrder(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	claudeProj := filepath.Join(proj, ".claude")
	if err := os.MkdirAll(claudeProj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{"env":{"CCB_GO_ORDER":"user"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeProj, "settings.go.json"), []byte(`{"env":{"CCB_GO_ORDER":"go","CCB_GO_ONLY":"g"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeProj, "settings.local.json"), []byte(`{"env":{"CCB_GO_ORDER":"local"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// TS settings.json must not affect Go merge (isolation).
	if err := os.WriteFile(filepath.Join(claudeProj, "settings.json"), []byte(`{"env":{"CCB_TS_ONLY":"ts","CCB_GO_ORDER":"from_ts_json"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CCB_GO_ORDER", "")
	t.Setenv("CCB_GO_ONLY", "")
	t.Setenv("CCB_TS_ONLY", "")
	if err := ApplyMergedClaudeSettingsEnv(proj); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("CCB_GO_ORDER"); got != "local" {
		t.Fatalf("local should win: got %q", got)
	}
	if got := os.Getenv("CCB_GO_ONLY"); got != "g" {
		t.Fatalf("CCB_GO_ONLY: got %q", got)
	}
	if got := os.Getenv("CCB_TS_ONLY"); got != "" {
		t.Fatalf("project settings.json must not apply to Go: got %q", got)
	}
}

func TestApplyMergedClaudeSettingsEnv_localOverridesGo(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	claudeProj := filepath.Join(proj, ".claude")
	if err := os.MkdirAll(claudeProj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{"env":{"CCB_LAYER":"user"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeProj, "settings.go.json"), []byte(`{"env":{"CCB_LAYER":"go","CCB_GO_ONLY":"g"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeProj, "settings.local.json"), []byte(`{"env":{"CCB_LAYER":"local"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CCB_LAYER", "")
	t.Setenv("CCB_GO_ONLY", "")
	if err := ApplyMergedClaudeSettingsEnv(proj); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("CCB_LAYER"); got != "local" {
		t.Fatalf("local should win: got %q", got)
	}
	if got := os.Getenv("CCB_GO_ONLY"); got != "g" {
		t.Fatalf("CCB_GO_ONLY: got %q", got)
	}
}

func TestApplyUserAndProjectClaudeEnv_goOverridesUser(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(proj, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{"env":{"CCB_MERGE_KEY":"from_user","CCB_USER_ONLY":"x"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, ".claude", "settings.go.json"), []byte(`{"env":{"CCB_MERGE_KEY":"from_go"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CCB_MERGE_KEY", "")
	t.Setenv("CCB_USER_ONLY", "")
	if err := ApplyUserAndProjectClaudeEnv(home, proj); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("CCB_MERGE_KEY"); got != "from_go" {
		t.Fatalf("go project settings should win over user: got %q", got)
	}
	if got := os.Getenv("CCB_USER_ONLY"); got != "x" {
		t.Fatalf("user-only key: got %q", got)
	}
}

func TestEnsureProjectClaudeEnvOnce_secondCallDoesNotReapply(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goPath := filepath.Join(claudeDir, "settings.go.json")
	if err := os.WriteFile(goPath, []byte(`{"env":{"CCB_ENSURE_ONCE":"from_go"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CCB_ENGINE_PROJECT_ROOT", dir)
	t.Setenv("CCB_ENSURE_ONCE", "")
	if err := EnsureProjectClaudeEnvOnce(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("CCB_ENSURE_ONCE"); got != "from_go" {
		t.Fatalf("first apply: want from_go, got %q", got)
	}
	if err := os.Setenv("CCB_ENSURE_ONCE", ""); err != nil {
		t.Fatal(err)
	}
	if err := EnsureProjectClaudeEnvOnce(); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("CCB_ENSURE_ONCE"); got != "" {
		t.Fatalf("second Ensure should not re-read settings; got %q", got)
	}
}
