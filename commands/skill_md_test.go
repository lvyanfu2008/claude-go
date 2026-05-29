package commands

import (
	"testing"
)

func TestCommandFromSkillMarkdown_NoFrontmatter(t *testing.T) {
	// TS parseFrontmatter returns {} when no frontmatter — the file is still valid.
	body := []byte("# Bug Fix Command\n\nSome content here.\n")
	cmd, err := commandFromSkillMarkdown("bug-fix", "", "/tmp/test.md", body, "projectSettings", "commands_DEPRECATED", "Custom command")
	if err != nil {
		t.Fatalf("expected no error for file without frontmatter, got: %v", err)
	}
	if cmd.Name != "bug-fix" {
		t.Errorf("expected name 'bug-fix', got %q", cmd.Name)
	}
	if cmd.Description == "" {
		t.Error("expected description extracted from markdown content")
	}
	if cmd.UserInvocable == nil || !*cmd.UserInvocable {
		t.Error("expected user-invocable to default to true")
	}
}

func TestCommandFromSkillMarkdown_WithFrontmatter(t *testing.T) {
	body := []byte("---\ndescription: test desc\nuser-invocable: false\n---\n# Title\nContent.")
	cmd, err := commandFromSkillMarkdown("test-cmd", "/root", "/tmp/test.md", body, "userSettings", "skills", "Skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Description != "test desc" {
		t.Errorf("expected description 'test desc', got %q", cmd.Description)
	}
	if cmd.UserInvocable == nil || *cmd.UserInvocable {
		t.Error("expected user-invocable false from frontmatter")
	}
}

func TestSplitYAMLFrontmatter_NoDelimiter(t *testing.T) {
	_, _, ok := SplitYAMLFrontmatter([]byte("# Just markdown"))
	if ok {
		t.Error("expected ok=false for no frontmatter delimiters")
	}
}

func TestSplitYAMLFrontmatter_WithDelimiter(t *testing.T) {
	yamlBytes, mdBody, ok := SplitYAMLFrontmatter([]byte("---\nkey: val\n---\n# Body"))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if string(yamlBytes) != "key: val" {
		t.Errorf("unexpected yamlBytes: %q", string(yamlBytes))
	}
	if string(mdBody) != "# Body" {
		t.Errorf("unexpected mdBody: %q", string(mdBody))
	}
}
