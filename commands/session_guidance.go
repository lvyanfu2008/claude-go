package commands

import (
	"strings"

	"goc/types"
)

const skillToolName = "Skill"

// SessionSpecificGuidance mirrors the skills-related branch of getSessionSpecificGuidanceSection in src/constants/prompts.ts
// when hasSkills is true (skillToolCommands non-empty and Skill in enabledTools).
// Format matches TS: "# Session-specific guidance" + prependBullets single item (" - " prefix).
func SessionSpecificGuidance(enabledToolNames map[string]struct{}, skillToolCommands []types.Command) string {
	b := skillSlashGuidanceBullet(enabledToolNames, skillToolCommands)
	if b == "" {
		return ""
	}
	return "# Session-specific guidance\n - " + b
}

func skillSlashGuidanceBullet(enabledToolNames map[string]struct{}, skillToolCommands []types.Command) string {
	if len(skillToolCommands) == 0 {
		return ""
	}
	if enabledToolNames == nil {
		return ""
	}
	if _, ok := enabledToolNames[skillToolName]; !ok {
		return ""
	}
	return `/<skill-name> (e.g., /commit) is shorthand for users to invoke a user-invocable skill. When executed, the skill gets expanded to a full prompt. Use the ` + skillToolName + ` tool to execute them. IMPORTANT: Only use ` + skillToolName + ` for skills listed in its user-invocable skills section - do not guess or use built-in CLI commands.`
}

const askUserQuestionToolName = "AskUserQuestion"

// SessionSpecificGuidanceFull extends TS getSessionSpecificGuidanceSection bullets we can mirror without feature bundles (AskUserQuestion, ! shell, skills, optional DiscoverSkills).
func SessionSpecificGuidanceFull(enabledToolNames map[string]struct{}, skillToolCommands []types.Command, discoverSkillsToolName string, nonInteractive bool) string {
	if enabledToolNames == nil {
		enabledToolNames = map[string]struct{}{}
	}
	var bullets []string
	if _, ok := enabledToolNames[askUserQuestionToolName]; ok {
		bullets = append(bullets, `If you do not understand why the user has denied a tool call, use the AskUserQuestion tool to ask them.`)
	}
	if !nonInteractive {
		bullets = append(bullets, `If you need the user to run a shell command themselves (e.g., an interactive login like `+"`gcloud auth login`"+`), suggest they type `+"`! <command>`"+` in the prompt — the `+"`!`"+` prefix runs the command in this session so its output lands directly in the conversation.`)
	}
	if b := skillSlashGuidanceBullet(enabledToolNames, skillToolCommands); b != "" {
		bullets = append(bullets, b)
	}
	ds := strings.TrimSpace(discoverSkillsToolName)
	if ds != "" && len(skillToolCommands) > 0 {
		if _, ok := enabledToolNames[ds]; ok {
			bullets = append(bullets, discoverSkillsGuidanceText(ds))
		}
	}
	if len(bullets) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("# Session-specific guidance\n")
	for _, bl := range bullets {
		b.WriteString(" - ")
		b.WriteString(bl)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func discoverSkillsGuidanceText(toolName string) string {
	return `Relevant skills are automatically surfaced each turn as "Skills relevant to your task:" reminders. If you're about to do something those don't cover — a mid-task pivot, an unusual workflow, a multi-step plan — call ` + toolName + ` with a specific description of what you're doing. Skills already visible or loaded are filtered automatically. Skip this if the surfaced skills already cover your next action.`
}

// EnabledToolNames builds a set from tool definition names (TS Set(tools.map(t => t.name))).
func EnabledToolNames(toolNames []string) map[string]struct{} {
	m := make(map[string]struct{}, len(toolNames))
	for _, n := range toolNames {
		n = strings.TrimSpace(n)
		if n != "" {
			m[n] = struct{}{}
		}
	}
	return m
}
