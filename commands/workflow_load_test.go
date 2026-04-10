package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWorkflowCommands_yamlProject(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	wfDir := filepath.Join(tmp, "proj", ".claude", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `name: code-review
description: Run review steps
steps: []
`
	if err := os.WriteFile(filepath.Join(wfDir, "review.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	opts := LoadOptions{BareMode: &f, WorkflowScripts: true}
	out, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "proj"), opts)
	if err != nil {
		t.Fatal(err)
	}
	i := indexOfCommandName(out, "code-review")
	if i < 0 {
		t.Fatalf("expected workflow command code-review in list")
	}
	if out[i].Description != "Run review steps" {
		t.Fatalf("description: got %q", out[i].Description)
	}
	if out[i].LoadedFrom == nil || *out[i].LoadedFrom != "workflow" {
		t.Fatalf("loadedFrom: %#v", out[i].LoadedFrom)
	}
	if out[i].Source == nil || *out[i].Source != "projectSettings" {
		t.Fatalf("source: %#v", out[i].Source)
	}
}

func TestLoadWorkflowCommands_skipsWhenFlagOff(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	wfDir := filepath.Join(tmp, "proj", ".claude", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "x.yaml"), []byte("name: wf-one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	opts := LoadOptions{BareMode: &f, WorkflowScripts: false}
	out, err := LoadAllCommands(context.Background(), filepath.Join(tmp, "proj"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if indexOfCommandName(out, "wf-one") >= 0 {
		t.Fatal("workflow should not load when WorkflowScripts false")
	}
}

func TestSanitizeWorkflowName(t *testing.T) {
	if g := sanitizeWorkflowName("Foo Bar"); g != "foo-bar" {
		t.Fatalf("got %q", g)
	}
	if g := sanitizeWorkflowName(""); g != "" {
		t.Fatalf("got %q", g)
	}
}
