package commands

import (
	"path/filepath"

	"goc/types"
)

func finalizeSkillDirCommands(entries []SkillLoadEntry, opts LoadOptions) ([]types.Command, error) {
	deduped := dedupeSkillEntries(entries)
	syncConditionalSkillsFromLoaded(deduped)
	return filterUnconditionalSkills(deduped, opts), nil
}

// loadSkillDirCommands mirrors src/skills/loadSkillsDir.ts getSkillDirCommands (P2: managed policy skills, order, legacy /commands, dedup).
func loadSkillDirCommands(cwd string, opts LoadOptions) ([]types.Command, error) {
	policyOff := opts.disablePolicySkillsEffective()
	projectSettingsEnabled := opts.isSettingSourceEnabled("projectSettings")

	if opts.effectiveBare() {
		if len(opts.AddSkillDirs) == 0 || !projectSettingsEnabled {
			return nil, nil
		}
		var entries []SkillLoadEntry
		for _, root := range opts.AddSkillDirs {
			abs, err := filepath.Abs(root)
			if err != nil {
				return nil, err
			}
			skillsDir := filepath.Join(abs, ".claude", "skills")
			part, err := loadSkillsFromDir(skillsDir, "projectSettings")
			if err != nil {
				return nil, err
			}
			entries = append(entries, part...)
		}
		return finalizeSkillDirCommands(entries, opts)
	}

	locked := opts.SkillsPluginOnlyLocked
	var entries []SkillLoadEntry

	if !policyOff {
		managedSkillsDir := filepath.Join(ManagedFilePath(), ".claude", "skills")
		managed, err := loadSkillsFromDir(managedSkillsDir, "policySettings")
		if err != nil {
			return nil, err
		}
		entries = append(entries, managed...)
	}

	if locked {
		return finalizeSkillDirCommands(entries, opts)
	}

	if cfgHome := ClaudeConfigHome(); cfgHome != "" && opts.isSettingSourceEnabled("userSettings") {
		userDir := filepath.Join(cfgHome, "skills")
		userSkills, err := loadSkillsFromDir(userDir, "userSettings")
		if err != nil {
			return nil, err
		}
		entries = append(entries, userSkills...)
	}

	if projectSettingsEnabled {
		projDirs, err := projectSkillDirs(cwd, opts.sessionRootForBoundary(cwd))
		if err != nil {
			return nil, err
		}
		for _, d := range projDirs {
			part, err := loadSkillsFromDir(d, "projectSettings")
			if err != nil {
				return nil, err
			}
			entries = append(entries, part...)
		}

		for _, root := range opts.AddSkillDirs {
			abs, err := filepath.Abs(root)
			if err != nil {
				return nil, err
			}
			skillsDir := filepath.Join(abs, ".claude", "skills")
			part, err := loadSkillsFromDir(skillsDir, "projectSettings")
			if err != nil {
				return nil, err
			}
			entries = append(entries, part...)
		}
	}

	legacy, err := loadLegacyCommandEntries(cwd, opts)
	if err != nil {
		return nil, err
	}
	entries = append(entries, legacy...)

	return finalizeSkillDirCommands(entries, opts)
}
