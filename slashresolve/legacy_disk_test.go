package slashresolve

import (
	"os"
	"path/filepath"
	"testing"

	"goc/types"
)

func TestResolveLegacyMarkdownCommand_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-cmd.md")
	content := "# Test Command\n\nThis is a test command body.\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := types.Command{
		CommandBase: types.CommandBase{
			Name: "test-cmd",
		},
		Type:                "prompt",
		LegacyMarkdownPath:  &path,
	}

	res, err := ResolveLegacyMarkdownCommand(cmd, "", "test-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.UserText != content {
		t.Errorf("expected body %q, got %q", content, res.UserText)
	}
	if res.Source != types.SlashResolveDisk {
		t.Errorf("expected source disk, got %q", res.Source)
	}
}

func TestResolveLegacyMarkdownCommand_WithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-cmd.md")
	content := "---\ndescription: test\n---\n# Body\n\nActual body content.\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := types.Command{
		CommandBase: types.CommandBase{
			Name: "test-cmd",
		},
		Type:                "prompt",
		LegacyMarkdownPath:  &path,
	}

	res, err := ResolveLegacyMarkdownCommand(cmd, "", "test-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "# Body\n\nActual body content.\n"
	if res.UserText != expected {
		t.Errorf("expected body %q, got %q", expected, res.UserText)
	}
}

func TestResolveLegacyMarkdownCommand_ArgSubstitution(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-cmd.md")
	content := "# Command\n\nArgs: $0 $1\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := types.Command{
		CommandBase: types.CommandBase{
			Name: "test-cmd",
		},
		Type:                "prompt",
		LegacyMarkdownPath:  &path,
		ArgNames:            []string{"first", "second"},
	}

	res, err := ResolveLegacyMarkdownCommand(cmd, "hello world", "test-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "# Command\n\nArgs: hello world\n"
	if res.UserText != expected {
		t.Errorf("expected %q, got %q", expected, res.UserText)
	}
}

func TestResolveLegacyMarkdownCommand_NoPath(t *testing.T) {
	cmd := types.Command{
		CommandBase: types.CommandBase{
			Name: "test-cmd",
		},
		Type: "prompt",
	}
	_, err := ResolveLegacyMarkdownCommand(cmd, "", "test-session")
	if err == nil {
		t.Fatal("expected error for missing LegacyMarkdownPath")
	}
}

func TestResolveLegacyMarkdownCommand_FileNotFound(t *testing.T) {
	path := "/nonexistent/path/file.md"
	cmd := types.Command{
		CommandBase: types.CommandBase{
			Name: "test-cmd",
		},
		Type:                "prompt",
		LegacyMarkdownPath:  &path,
	}
	_, err := ResolveLegacyMarkdownCommand(cmd, "", "test-session")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestResolveLegacyMarkdownCommand_SessionIDSubstitution(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-cmd.md")
	content := "# Command\n\nSession: ${CLAUDE_SESSION_ID}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := types.Command{
		CommandBase: types.CommandBase{
			Name: "test-cmd",
		},
		Type:                "prompt",
		LegacyMarkdownPath:  &path,
	}

	res, err := ResolveLegacyMarkdownCommand(cmd, "", "my-session-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "# Command\n\nSession: my-session-123\n"
	if res.UserText != expected {
		t.Errorf("expected %q, got %q", expected, res.UserText)
	}
}
