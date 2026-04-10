package pui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/gou/conversation"
	"goc/types"
)

func TestNewSlashResolveProcessSlashCommand_diskSkill(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "demo-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	md := `---
name: demo-skill
description: x
---
Hello from skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	root := skillDir
	cmd := types.Command{
		CommandBase: types.CommandBase{Name: "demo-skill", Description: "x"},
		Type:        "prompt",
		SkillRoot:   &root,
	}

	h := NewSlashResolveProcessSlashCommand(SlashResolveHandlerOptions{SessionID: "sess-e2e"})
	p := &processuserinput.ProcessUserInputParams{
		Commands: []types.Command{cmd},
		Mode:     types.PromptInputModePrompt,
	}
	r, err := h(context.Background(), "/demo-skill", nil, nil, nil, nil, nil, p)
	if err != nil {
		t.Fatal(err)
	}
	if r == nil || !r.ShouldQuery || len(r.Messages) == 0 {
		t.Fatalf("result: %+v", r)
	}
}

func TestProcessUserInput_withSlashResolveHandler(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "z-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	md := `---
name: z-skill
description: y
---
Body
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	root := skillDir
	st := &conversation.Store{ConversationID: "t"}
	p, err := BuildDemoParams("/z-skill", st, DemoConfig{SkipCommands: true})
	if err != nil {
		t.Fatal(err)
	}
	p.Commands = []types.Command{{
		CommandBase: types.CommandBase{Name: "z-skill", Description: "y"},
		Type:        "prompt",
		SkillRoot:   &root,
	}}
	p.ProcessSlashCommand = NewSlashResolveProcessSlashCommand(SlashResolveHandlerOptions{SessionID: "t"})

	r, err := processuserinput.ProcessUserInput(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if !r.ShouldQuery || len(r.Messages) == 0 {
		t.Fatalf("shouldQuery=%v msgs=%d", r.ShouldQuery, len(r.Messages))
	}
}
