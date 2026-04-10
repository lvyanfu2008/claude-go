package handwritten

import (
	"goc/commands/featuregates"
	"goc/types"
)

func skillVerify() types.Command {
	src := "bundled"
	pm := "running"
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "verify",
			Description:                 "Verify a code change does what it should by running the app.",
			HasUserSpecifiedDescription: ptrBool(true),
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			UserInvocable:               ptrBool(true),
		},
		Type:            "prompt",
		AllowedTools:    []string{},
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

func skillLoremIpsum() types.Command {
	src := "bundled"
	pm := "running"
	ah := "[token_count]"
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "lorem-ipsum",
			Description:                 "Generate filler text for long context testing. Specify token count as argument (e.g., /lorem-ipsum 50000). Outputs approximately the requested number of tokens. Ant-only.",
			HasUserSpecifiedDescription: ptrBool(true),
			ArgumentHint:                &ah,
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			UserInvocable:               ptrBool(true),
		},
		Type:            "prompt",
		AllowedTools:    []string{},
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

func skillSkillify() types.Command {
	src := "bundled"
	pm := "running"
	ah := "[description of the process you want to capture]"
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "skillify",
			Description:                 "Capture this session's repeatable process into a skill. Call at end of the process you want to capture with an optional description.",
			HasUserSpecifiedDescription: ptrBool(true),
			ArgumentHint:                &ah,
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			UserInvocable:               ptrBool(true),
			DisableModelInvocation:      ptrBool(true),
		},
		Type:            "prompt",
		AllowedTools:    strSlice("Read", "Write", "Edit", "Glob", "Grep", "AskUserQuestion", "Bash(mkdir:*)"),
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

func skillRemember() types.Command {
	src := "bundled"
	pm := "running"
	w := "Use when the user wants to review, organize, or promote their auto-memory entries. Also useful for cleaning up outdated or conflicting entries across CLAUDE.md, CLAUDE.local.md, and auto-memory."
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "remember",
			Description:                 "Review auto-memory entries and propose promotions to CLAUDE.md, CLAUDE.local.md, or shared memory. Also detects outdated, conflicting, and duplicate entries across memory layers.",
			HasUserSpecifiedDescription: ptrBool(true),
			WhenToUse:                   &w,
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			UserInvocable:               ptrBool(true),
		},
		Type:            "prompt",
		AllowedTools:    []string{},
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

func skillStuck() types.Command {
	src := "bundled"
	pm := "running"
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "stuck",
			Description:                 "[ANT-ONLY] Investigate frozen/stuck/slow Claude Code sessions on this machine and post a diagnostic report to #claude-code-feedback.",
			HasUserSpecifiedDescription: ptrBool(true),
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			UserInvocable:               ptrBool(true),
		},
		Type:            "prompt",
		AllowedTools:    []string{},
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

func bundledAntBlock1() []types.Command {
	if !featuregates.UserTypeAnt() {
		return nil
	}
	return []types.Command{skillVerify()}
}

func bundledAntBlock2() []types.Command {
	if !featuregates.UserTypeAnt() {
		return nil
	}
	return []types.Command{
		skillLoremIpsum(),
		skillSkillify(),
		skillRemember(),
	}
}

func bundledAntStuck() []types.Command {
	if !featuregates.UserTypeAnt() {
		return nil
	}
	return []types.Command{skillStuck()}
}
