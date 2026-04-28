package memdir

import (
	"os"
	"strings"

	"goc/claudebase"
)

// BuildTeamCombinedMemoryPrompt mirrors teamMemPrompts.ts buildCombinedMemoryPrompt.
// autoDirDisplay and teamDirDisplay should be memoryDirDisplayPath(...) (trailing slash, ToSlash).
func BuildTeamCombinedMemoryPrompt(autoDirDisplay, teamDirDisplay string, skipIndex bool, extraGuidelines []string) string {
	lines := []string{
		"# Memory",
		"",
		"You have a persistent, file-based memory system with two directories: a private directory at `" + autoDirDisplay + "` and a shared team directory at `" + teamDirDisplay + "`. " + dirsExistGuidance,
		"",
		"You should build up this memory system over time so that future conversations can have a complete picture of who the user is, how they'd like to collaborate with you, what behaviors to avoid or repeat, and the context behind the work the user gives you.",
		"",
		"If the user explicitly asks you to remember something, save it immediately as whichever type fits best. If they ask you to forget something, find and remove the relevant entry.",
		"",
		"## Memory scope",
		"",
		"There are two scope levels:",
		"",
		"- private: memories that are private between you and the current user. They persist across conversations with only this specific user and are stored at the root `" + autoDirDisplay + "`.",
		"- team: memories that are shared with and contributed by all of the users who work within this project directory. Team memories are synced at the beginning of every session and they are stored at `" + teamDirDisplay + "`.",
		"",
	}
	lines = append(lines, buildTypesSectionCombined()...)
	lines = append(lines, BuildWhatNotToSaveSection()...)
	lines = append(lines,
		"- You MUST avoid saving sensitive data within shared team memories. For example, never save API keys or user credentials.",
		"",
	)
	lines = append(lines, buildTeamHowToSaveLines(skipIndex)...)
	lines = append(lines, "")
	lines = append(lines, buildWhenToAccessMemoriesTeam()...)
	lines = append(lines, buildTrustingRecallSection()...)
	lines = append(lines,
		"## Memory and other forms of persistence",
		"Memory is one of several persistence mechanisms available to you as you assist the user in a given conversation. The distinction is often that memory can be recalled in future conversations and should not be used for persisting information that is only useful within the scope of the current conversation.",
		"- When to use or update a plan instead of memory: If you are about to start a non-trivial implementation task and would like to reach alignment with the user on your approach you should use a Plan rather than saving this information to memory. Similarly, if you already have a plan within the conversation and you have changed your approach persist that change by updating the plan rather than saving a memory.",
		"- When to use or update tasks instead of memory: When you need to break your work in current conversation into discrete steps or keep track of your progress use tasks instead of saving to memory. Tasks are great for persisting information about the work that needs to be done in the current conversation, but memory should be reserved for information that will be useful in future conversations.",
	)
	if len(extraGuidelines) > 0 {
		for _, g := range extraGuidelines {
			lines = append(lines, g)
		}
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

// IsTeamMemoryPromptActive mirrors the TEAMMEM gate in memory_auto_prompt.go.
func IsTeamMemoryPromptActive() bool {
	return claudebase.Truthy(os.Getenv("FEATURE_TEAMMEM")) && IsAutoMemoryEnabled() && teamMemoryEnabled()
}

func teamMemoryEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_TEAM_MEMORY_ENABLED")); v != "" {
		return claudebase.Truthy(v)
	}
	return true
}
