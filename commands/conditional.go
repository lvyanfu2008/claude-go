package commands

import (
	"goc/types"
)

// PathsAreOnlyDoubleStar matches TS parseSkillPaths: every pattern is exactly ** (no path filter).
func PathsAreOnlyDoubleStar(paths []string) bool {
	if len(paths) == 0 {
		return false
	}
	for _, p := range paths {
		if p != "**" {
			return false
		}
	}
	return true
}

// filterUnconditionalSkills drops path-filtered conditional skills (TS getSkillDirCommands returns only unconditionalSkills).
// When IncludeConditionalSkills is true, no filtering (for hosts that track activation separately).
// Skills activated via [ActivateConditionalSkillsForPaths] are included (TS activatedConditionalSkillNames branch).
func filterUnconditionalSkills(cmds []types.Command, opts LoadOptions) []types.Command {
	if opts.IncludeConditionalSkills {
		return cmds
	}
	out := make([]types.Command, 0, len(cmds))
	for _, c := range cmds {
		if c.Type != "prompt" {
			out = append(out, c)
			continue
		}
		if len(c.Paths) == 0 {
			out = append(out, c)
			continue
		}
		if PathsAreOnlyDoubleStar(c.Paths) {
			out = append(out, c)
			continue
		}
		if ActivatedConditionalSkillName(c.Name) {
			out = append(out, c)
			continue
		}
	}
	return out
}
