package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestActivateConditionalSkillsForPaths_mergesIntoDynamic(t *testing.T) {
	ClearLoadAllCommandsCache()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmp, "cfg"))
	if err := os.MkdirAll(filepath.Join(tmp, "cfg", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(tmp, ".claude", "skills", "condskill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `---
name: condskill
description: conditional
paths: "*.go"
---
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	f := false
	opts := LoadOptions{BareMode: &f}
	auth := DefaultConsoleAPIAuth()
	ctx := context.Background()

	_, err := LoadAndFilterCommands(ctx, tmp, opts, auth)
	if err != nil {
		t.Fatal(err)
	}
	if ConditionalPendingCount() != 1 {
		t.Fatalf("expected 1 pending conditional skill, got %d", ConditionalPendingCount())
	}

	goFile := filepath.Join(tmp, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := LoadAndGetCommandsWithFilePathsDynamic(ctx, tmp, opts, auth, []string{goFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, c := range out {
		if c.Name == "condskill" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected activated conditional skill in merged command list")
	}
	if ConditionalPendingCount() != 0 {
		t.Fatalf("expected pending cleared after activation, got %d", ConditionalPendingCount())
	}
	if !ActivatedConditionalSkillName("condskill") {
		t.Fatal("expected activated name tracked")
	}
}
