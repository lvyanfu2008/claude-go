package anthropic

import (
	"slices"
	"testing"

	"goc/internal/toolvalidator"
)

func TestGouParityToolList_namesGolden(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME", "")
	t.Setenv(toolvalidator.EnvToolInputValidator, "")
	got := GouParityToolNames()
	want := []string{
		"Agent", "AskUserQuestion", "TaskOutput", "Bash", "Glob", "Grep", "ExitPlanMode",
		"Read", "Write", "Edit", "NotebookEdit", "WebFetch", "TodoWrite", "WebSearch", "TaskStop",
		"Skill", "EnterPlanMode", "SendMessage",
		"CronCreate", "CronDelete", "CronList", "SendUserMessage", "Brief",
		"ListMcpResourcesTool", "ReadMcpResourceTool", "echo_stub",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGouParityToolList_includesBashZogWhenValidatorZog(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME", "")
	t.Setenv(toolvalidator.EnvToolInputValidator, "zog")
	names := GouParityToolNames()
	var seenBash, seenBashZog bool
	for _, n := range names {
		if n == "Bash" {
			seenBash = true
		}
		if n == "BashZog" {
			seenBashZog = true
		}
	}
	if !seenBash || !seenBashZog {
		t.Fatalf("expected Bash and BashZog in list, got %v", names)
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
