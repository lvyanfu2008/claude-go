package commands

import (
	"strings"
	"testing"

	"goc/types"
)

func TestBuildGouDemoSystemPrompt_containsSections(t *testing.T) {
	enabled := EnabledToolNames([]string{"Skill", "AskUserQuestion"})
	sk := []types.Command{{CommandBase: types.CommandBase{Name: "s", LoadedFrom: ptrStr("skills")}, Type: "prompt"}}
	s := BuildGouDemoSystemPrompt(GouDemoSystemOpts{
		EnabledToolNames:  enabled,
		SkillToolCommands: sk,
		ModelID:           "m1",
		Cwd:               "/tmp",
		Language:          "Japanese",
	})
	if !strings.Contains(s, "# System") || !strings.Contains(s, "# Session-specific guidance") {
		t.Fatal(s[:min(200, len(s))])
	}
	if !strings.Contains(s, "# Language") || !strings.Contains(s, "Japanese") {
		t.Fatal()
	}
	if !strings.Contains(s, "# Environment information") {
		t.Fatal()
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
