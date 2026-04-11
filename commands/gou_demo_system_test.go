package commands

import (
	"strings"
	"testing"

	"goc/types"
)

func TestBuildGouDemoSystemPrompt_containsSections(t *testing.T) {
	enabled := EnabledToolNames([]string{"Skill", "AskUserQuestion", "Read", "Edit", "Write", "Glob", "Grep", "Bash", "TodoWrite"})
	sk := []types.Command{{CommandBase: types.CommandBase{Name: "s", LoadedFrom: ptrStr("skills")}, Type: "prompt"}}
	s := BuildGouDemoSystemPrompt(GouDemoSystemOpts{
		EnabledToolNames:  enabled,
		SkillToolCommands: sk,
		ModelID:           "m1",
		Cwd:               "/tmp",
		Language:          "Japanese",
	})
	if !strings.Contains(s, "# System") || !strings.Contains(s, "# Doing tasks") || !strings.Contains(s, "# Session-specific guidance") {
		t.Fatal(s[:min(200, len(s))])
	}
	if !strings.Contains(s, "primarily request you to perform software engineering tasks") {
		t.Fatal("missing TS getSimpleDoingTasksSection lead paragraph")
	}
	if !strings.Contains(s, "# Executing actions with care") || !strings.Contains(s, "Carefully consider the reversibility and blast radius of actions") {
		t.Fatal("missing TS getActionsSection")
	}
	if !strings.Contains(s, "# Using your tools") || !strings.Contains(s, "To read files use Read") || !strings.Contains(s, "TodoWrite") {
		t.Fatal("missing TS getUsingYourToolsSection")
	}
	if !strings.Contains(s, "# Tone and style") || !strings.Contains(s, "# Output efficiency") {
		t.Fatal("missing tone/output efficiency sections")
	}
	if !strings.Contains(s, "original tool result may be cleared later") {
		t.Fatal("missing summarize_tool_results section")
	}
	if strings.Index(s, "# Output efficiency") > strings.Index(s, "# Session-specific guidance") {
		t.Fatal("expected # Output efficiency before # Session-specific guidance (TS static prefix order)")
	}
	if !strings.Contains(s, "# Language") || !strings.Contains(s, "Japanese") {
		t.Fatal()
	}
	if !strings.Contains(s, "# Environment") {
		t.Fatal()
	}
	if !strings.Contains(s, "You have been invoked in the following environment") {
		t.Fatal("missing TS computeSimpleEnvInfo framing")
	}
}

func TestBuildGouDemoSystemPrompt_outputStyle(t *testing.T) {
	s := BuildGouDemoSystemPrompt(GouDemoSystemOpts{
		OutputStyleName:   "Concise",
		OutputStylePrompt: "Be short.",
	})
	if !strings.Contains(s, "# Output Style: Concise") || !strings.Contains(s, "Be short.") {
		t.Fatal(s)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
