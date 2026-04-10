package anthropic

import (
	"slices"
	"testing"
)

func TestGouParityToolList_namesGolden(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME", "")
	got := GouParityToolNames()
	want := []string{
		"Agent", "TaskOutput", "Bash", "Glob", "Grep", "ExitPlanMode",
		"Read", "Write", "Edit", "NotebookEdit", "WebFetch", "TodoWrite", "WebSearch", "TaskStop",
		"AskUserQuestion", "Skill", "EnterPlanMode", "SendMessage",
		"CronCreate", "CronDelete", "CronList", "SendUserMessage", "Brief",
		"ListMcpResourcesTool", "ReadMcpResourceTool", "echo_stub",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGouParityToolList_discoverOptional(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME", "DiscoverSkills")
	list := GouParityToolList()
	var found bool
	for _, td := range list {
		if td.Name == "DiscoverSkills" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected DiscoverSkills when env set")
	}
}
