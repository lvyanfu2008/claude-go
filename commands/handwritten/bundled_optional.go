package handwritten

import (
	"goc/commands/featuregates"
	"goc/types"
)

// initBundledSkills optional registrations (src/skills/bundled/index.ts).
func bundledOptionalSkills() []types.Command {
	var out []types.Command
	if featuregates.Feature("REVIEW_ARTIFACT") {
		out = append(out, skillHunter())
	}
	if featuregates.Feature("AGENT_TRIGGERS_REMOTE") {
		out = append(out, skillSchedule())
	}
	if featuregates.Feature("BUILDING_CLAUDE_APPS") {
		out = append(out, skillClaudeAPI())
	}
	if featuregates.BundledChromeSkillEnabled() {
		out = append(out, skillClaudeInChrome())
	}
	if featuregates.Feature("RUN_SKILL_GENERATOR") {
		out = append(out, skillRunSkillGenerator())
	}
	return out
}

func skillHunter() types.Command {
	src := "bundled"
	pm := "running"
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "hunter",
			Description:                 "Artifact / review hunter skill (REVIEW_ARTIFACT; metadata only in Go listing)",
			HasUserSpecifiedDescription: ptrBool(true),
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
		},
		Type:            "prompt",
		AllowedTools:    []string{},
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

func skillSchedule() types.Command {
	src := "bundled"
	pm := "running"
	w := "When the user wants to schedule a recurring remote agent, set up automated tasks, create a cron job for Claude Code, or manage their scheduled agents/triggers."
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "schedule",
			Description:                 "Create, update, list, or run scheduled remote agents (triggers) that execute on a cron schedule.",
			HasUserSpecifiedDescription: ptrBool(true),
			WhenToUse:                   &w,
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			UserInvocable:               ptrBool(true),
		},
		Type:            "prompt",
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

func skillClaudeAPI() types.Command {
	src := "bundled"
	pm := "running"
	d := "Build apps with the Claude API or Anthropic SDK.\n" +
		"TRIGGER when: code imports `anthropic`/`@anthropic-ai/sdk`/`claude_agent_sdk`, or user asks to use Claude API, Anthropic SDKs, or Agent SDK.\n" +
		"DO NOT TRIGGER when: code imports `openai`/other AI SDK, general programming, or ML/data-science tasks."
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "claude-api",
			Description:                 d,
			HasUserSpecifiedDescription: ptrBool(true),
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			UserInvocable:               ptrBool(true),
		},
		Type:            "prompt",
		AllowedTools:    strSlice("Read", "Grep", "Glob", "WebFetch"),
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

func skillClaudeInChrome() types.Command {
	src := "bundled"
	pm := "running"
	d := "Automates your Chrome browser to interact with web pages - clicking elements, filling forms, capturing screenshots, reading console logs, and navigating sites. Opens pages in new tabs within your existing Chrome session. Requires site-level permissions before executing (configured in the extension)."
	w := "When the user wants to interact with web pages, automate browser tasks, capture screenshots, read console logs, or perform any browser-based actions. Always invoke BEFORE attempting to use any mcp__claude-in-chrome__* tools."
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "claude-in-chrome",
			Description:                 d,
			HasUserSpecifiedDescription: ptrBool(true),
			WhenToUse:                   &w,
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			UserInvocable:               ptrBool(true),
		},
		Type:            "prompt",
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

func skillRunSkillGenerator() types.Command {
	src := "bundled"
	pm := "running"
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "run-skill-generator",
			Description:                 "Run skill generator (RUN_SKILL_GENERATOR; metadata only in Go listing)",
			HasUserSpecifiedDescription: ptrBool(true),
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
		},
		Type:            "prompt",
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

// AssembleBundledSkills matches getBundledSkills() registration order: unconditional block then feature-gated skills.
func AssembleBundledSkills() []types.Command {
	out := make([]types.Command, 0, 16)
	out = append(out, bundledCoreSkills()...)
	out = append(out, bundledOptionalSkills()...)
	return out
}
