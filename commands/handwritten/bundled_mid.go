package handwritten

import (
	"goc/types"
	"goc/utils"
)

// kairosCronEnabled matches toolpool/tool_enabled.go kairosCronEnabled function
func kairosCronEnabled() bool {
	return !utils.IsEnvTruthy("CLAUDE_CODE_DISABLE_CRON")
}

func skillDebug() types.Command {
	src := "bundled"
	pm := "running"
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "debug",
			Description:                 "Enable debug logging for this session and help diagnose issues",
			HasUserSpecifiedDescription: ptrBool(true),
			ArgumentHint:                ptrStr("[issue description]"),
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			DisableModelInvocation:      ptrBool(true),
			UserInvocable:               ptrBool(true),
		},
		Type:            "prompt",
		AllowedTools:    strSlice("Read", "Grep", "Glob"),
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

func bundledSimplifyBatch() []types.Command {
	src := "bundled"
	pm := "running"
	return []types.Command{
		{
			CommandBase: types.CommandBase{
				Name:                        "simplify",
				Description:                 "Review changed code for reuse, quality, and efficiency, then fix any issues found.",
				HasUserSpecifiedDescription: ptrBool(true),
				IsHidden:                    ptrBool(false),
				LoadedFrom:                  ptrStr(src),
				DisableModelInvocation:      ptrBool(false),
				UserInvocable:               ptrBool(true),
			},
			Type:            "prompt",
			AllowedTools:    []string{},
			ContentLength:   ptrInt(0),
			Source:          ptrStr(src),
			ProgressMessage: &pm,
		},
		{
			CommandBase: types.CommandBase{
				Name:                        "batch",
				Description:                 "Research and plan a large-scale change, then execute it in parallel across 5–30 isolated worktree agents that each open a PR.",
				HasUserSpecifiedDescription: ptrBool(true),
				ArgumentHint:                ptrStr("<instruction>"),
				WhenToUse:                   ptrStr("Use when the user wants to make a sweeping, mechanical change across many files (migrations, refactors, bulk renames) that can be decomposed into independent parallel units."),
				IsHidden:                    ptrBool(false),
				LoadedFrom:                  ptrStr(src),
				DisableModelInvocation:      ptrBool(true),
				UserInvocable:               ptrBool(true),
			},
			Type:            "prompt",
			AllowedTools:    []string{},
			ContentLength:   ptrInt(0),
			Source:          ptrStr(src),
			ProgressMessage: &pm,
		},
	}
}

func bundledLoopDream() []types.Command {
	src := "bundled"
	pm := "running"
	skills := []types.Command{
		{
			CommandBase: types.CommandBase{
				Name:                        "loop",
				Description:                 "Run a prompt or slash command on a recurring interval (e.g. /loop 5m /foo, defaults to 10m)",
				HasUserSpecifiedDescription: ptrBool(true),
				ArgumentHint:                ptrStr("[interval] <prompt>"),
				WhenToUse:                   ptrStr("When the user wants to set up a recurring task, poll for status, or run something repeatedly on an interval (e.g. \"check the deploy every 5 minutes\", \"keep running /babysit-prs\"). Do NOT invoke for one-off tasks."),
				IsHidden:                    ptrBool(false),
				LoadedFrom:                  ptrStr(src),
				DisableModelInvocation:      ptrBool(false),
				UserInvocable:               ptrBool(true),
			},
			Type:            "prompt",
			AllowedTools:    []string{},
			ContentLength:   ptrInt(0),
			Source:          ptrStr(src),
			ProgressMessage: &pm,
		},
	}

	// Cron management skills - mirrors src/skills/bundled/cronManage.ts
	// These are conditionally added based on kairosCronEnabled() like TypeScript's isEnabled check
	if kairosCronEnabled() {
		skills = append(skills, skillCronList(), skillCronDelete())
	}

	skills = append(skills, types.Command{
		CommandBase: types.CommandBase{
			Name:                        "dream",
			Description:                 "Manually trigger memory consolidation — review, organize, and prune your auto-memory files.",
			HasUserSpecifiedDescription: ptrBool(true),
			WhenToUse:                   ptrStr("Use when the user says /dream or wants to manually consolidate memories, organize memory files, or clean up stale entries."),
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			DisableModelInvocation:      ptrBool(false),
			UserInvocable:               ptrBool(true),
		},
		Type:            "prompt",
		AllowedTools:    []string{},
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	})

	return skills
}

// skillCronList mirrors registerCronListSkill in src/skills/bundled/cronManage.ts
func skillCronList() types.Command {
	src := "bundled"
	pm := "running"
	w := "When the user wants to see their scheduled/recurring tasks, check active cron jobs, or review what is currently looping."
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "cron-list",
			Description:                 "List all scheduled cron jobs in this session",
			HasUserSpecifiedDescription: ptrBool(true),
			WhenToUse:                   &w,
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			UserInvocable:               ptrBool(true),
		},
		Type:            "prompt",
		AllowedTools:    strSlice("CronList"),
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}

// skillCronDelete mirrors registerCronDeleteSkill in src/skills/bundled/cronManage.ts
func skillCronDelete() types.Command {
	src := "bundled"
	pm := "running"
	w := "When the user wants to cancel, stop, or remove a scheduled/recurring task or cron job."
	hint := "<job-id>"
	return types.Command{
		CommandBase: types.CommandBase{
			Name:                        "cron-delete",
			Description:                 "Cancel a scheduled cron job by ID",
			HasUserSpecifiedDescription: ptrBool(true),
			ArgumentHint:                &hint,
			WhenToUse:                   &w,
			IsHidden:                    ptrBool(false),
			LoadedFrom:                  ptrStr(src),
			UserInvocable:               ptrBool(true),
		},
		Type:            "prompt",
		AllowedTools:    strSlice("CronDelete"),
		ContentLength:   ptrInt(0),
		Source:          ptrStr(src),
		ProgressMessage: &pm,
	}
}
