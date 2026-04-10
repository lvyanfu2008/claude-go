package handwritten

import "goc/types"

// bundledCorePrefix is TS initBundledSkills: update-config, keybindings-help (before verify).
func bundledCorePrefix() []types.Command {
	src := "bundled"
	pm := "running"
	return []types.Command{
		{
			CommandBase: types.CommandBase{
				Name:                        "update-config",
				Description:                 "Use this skill to configure the Claude Code harness via settings.json. Automated behaviors (\"from now on when X\", \"each time X\", \"whenever X\", \"before/after X\") require hooks configured in settings.json - the harness executes these, not Claude, so memory/preferences cannot fulfill them. Also use for: permissions (\"allow X\", \"add permission\", \"move permission to\"), env vars (\"set X=Y\"), hook troubleshooting, or any changes to settings.json/settings.local.json files. Examples: \"allow npm commands\", \"add bq permission to global settings\", \"move permission to user settings\", \"set DEBUG=true\", \"when claude stops show X\". For simple settings like theme/model, use Config tool.",
				HasUserSpecifiedDescription: ptrBool(true),
				IsHidden:                    ptrBool(false),
				LoadedFrom:                  ptrStr(src),
				DisableModelInvocation:      ptrBool(false),
				UserInvocable:               ptrBool(true),
			},
			Type:            "prompt",
			AllowedTools:    strSlice("Read"),
			ContentLength:   ptrInt(0),
			Source:          ptrStr(src),
			ProgressMessage: &pm,
		},
		{
			CommandBase: types.CommandBase{
				Name:                        "keybindings-help",
				Description:                 "Use when the user wants to customize keyboard shortcuts, rebind keys, add chord bindings, or modify ~/.claude/keybindings.json. Examples: \"rebind ctrl+s\", \"add a chord shortcut\", \"change the submit key\", \"customize keybindings\".",
				HasUserSpecifiedDescription: ptrBool(true),
				IsHidden:                    ptrBool(true),
				LoadedFrom:                  ptrStr(src),
				DisableModelInvocation:      ptrBool(false),
				UserInvocable:               ptrBool(false),
			},
			Type:            "prompt",
			AllowedTools:    strSlice("Read"),
			ContentLength:   ptrInt(0),
			Source:          ptrStr(src),
			ProgressMessage: &pm,
		},
	}
}
