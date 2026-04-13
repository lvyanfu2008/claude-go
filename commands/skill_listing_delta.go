package commands

import (
	"strings"

	"goc/types"
)

// AppendSkillListingForAPI mirrors getSkillListingAttachments delta behavior for the local command list:
// when hasSkillTool is false or there are no new skills vs sent, returns ("", 0, false, false).
// On success, mutates sent (by command name) and returns the full API user text (system-reminder wrapped), like TS normalizeAttachmentForAPI skill_listing.
// isInitial matches TS sent.size === 0 before marking new skills (AttachmentMessage hides the first batch in UI).
func AppendSkillListingForAPI(
	allSkillToolCommands []types.Command,
	hasSkillTool bool,
	sent map[string]struct{},
	contextWindowTokens *int,
) (apiUserText string, skillCount int, isInitial bool, ok bool) {
	if sent == nil {
		return "", 0, false, false
	}
	if !hasSkillTool {
		return "", 0, false, false
	}
	isInitial = len(sent) == 0
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
		return "", 0, false, false
	}
	for _, cmd := range newSkills {
		sent[strings.TrimSpace(cmd.Name)] = struct{}{}
	}
	formatted := FormatCommandsWithinBudget(newSkills, contextWindowTokens)
	return SkillListingAPIUserText(formatted), len(newSkills), isInitial, true
}
