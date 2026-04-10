package commands

import (
	"strings"

	"goc/types"
)

// AppendSkillListingForAPI mirrors getSkillListingAttachments delta behavior for the local command list:
// when hasSkillTool is false or there are no new skills vs sent, returns ("", false).
// On success, mutates sent (by command name) and returns the full API user text (system-reminder wrapped), like TS normalizeAttachmentForAPI skill_listing.
func AppendSkillListingForAPI(
	allSkillToolCommands []types.Command,
	hasSkillTool bool,
	sent map[string]struct{},
	contextWindowTokens *int,
) (apiUserText string, ok bool) {
	if sent == nil {
		return "", false
	}
	if !hasSkillTool {
		return "", false
	}
	var newSkills []types.Command
	for _, cmd := range allSkillToolCommands {
		name := strings.TrimSpace(cmd.Name)
		if name == "" {
			continue
		}
		if _, already := sent[name]; already {
			continue
		}
		newSkills = append(newSkills, cmd)
	}
	if len(newSkills) == 0 {
		return "", false
	}
	for _, cmd := range newSkills {
		sent[strings.TrimSpace(cmd.Name)] = struct{}{}
	}
	formatted := FormatCommandsWithinBudget(newSkills, contextWindowTokens)
	return SkillListingAPIUserText(formatted), true
}
