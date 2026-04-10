package commands

import (
	"strings"
	"testing"

	"goc/types"
)

func TestGetCharBudget_env(t *testing.T) {
	t.Setenv("SLASH_COMMAND_TOOL_CHAR_BUDGET", "12345")
	if got := GetCharBudget(nil); got != 12345 {
		t.Fatalf("got %d", got)
	}
}

func TestGetCharBudget_tokens(t *testing.T) {
	t.Setenv("SLASH_COMMAND_TOOL_CHAR_BUDGET", "")
	tw := 100_000
	if got := GetCharBudget(&tw); got != 4000 {
		t.Fatalf("1%% of 100k*4 = 4000, got %d", got)
	}
}

func TestFormatCommandsWithinBudget_empty(t *testing.T) {
	if FormatCommandsWithinBudget(nil, nil) != "" {
		t.Fatal()
	}
}

func TestFormatCommandsWithinBudget_singleFits(t *testing.T) {
	t.Setenv("SLASH_COMMAND_TOOL_CHAR_BUDGET", "")
	cmd := types.Command{
		CommandBase: types.CommandBase{Name: "pdf", Description: "Read PDFs", LoadedFrom: ptrStr("skills")},
		Type:        "prompt",
		Source:      ptrStr("user"),
	}
	out := FormatCommandsWithinBudget([]types.Command{cmd}, nil)
	if out != "- pdf: Read PDFs" {
		t.Fatalf("%q", out)
	}
}

func TestGetCommandDescription_maxLen(t *testing.T) {
	long := strings.Repeat("a", 300)
	cmd := types.Command{
		CommandBase: types.CommandBase{Name: "x", Description: long},
		Type:        "prompt",
	}
	// formatCommandDescription uses getCommandDescription
	out := formatCommandDescription(cmd)
	if !strings.HasSuffix(out, "\u2026") {
		t.Fatalf("expected ellipsis suffix: %q", out)
	}
	if utf8RuneCount(strings.TrimPrefix(out, "- x: ")) > MaxListingDescChars {
		t.Fatalf("too long")
	}
}

func TestSkillListingAPIUserText_shape(t *testing.T) {
	text := SkillListingAPIUserText("- foo: bar")
	if !strings.HasPrefix(text, "<system-reminder>\n") {
		t.Fatal(text)
	}
	if !strings.Contains(text, SkillListingBodyPrefix) {
		t.Fatal(text)
	}
	if !strings.HasSuffix(text, "\n</system-reminder>") {
		t.Fatal(text)
	}
}

func TestSkillToolDescriptionPrompt_hasCommandNameTag(t *testing.T) {
	if !strings.Contains(SkillToolDescriptionPrompt, "<command-name>") {
		t.Fatal()
	}
}
