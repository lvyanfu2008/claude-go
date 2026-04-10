package commands

import (
	"context"
	"strings"

	"goc/types"
)

// LoadSkillsFromSkillDirectories loads prompt skills from each directory (TS loadSkillsFromSkillsDir).
// dirs should be ordered **deepest first** (output of [DiscoverSkillDirsForPaths]).
// Same skill name in multiple dirs: **deeper path wins** (matches TS addSkillDirectories reverse merge).
// Output order matches TS [getDynamicSkills]: [Array.from(dynamicSkills.values())] — insertion order is
// first time each name appears when merging **shallow → deep** (i = len-1 … 0), same as Map set order.
func LoadSkillsFromSkillDirectories(dirs []string) ([]types.Command, error) {
	if len(dirs) == 0 {
		return nil, nil
	}
	byName := make(map[string]types.Command)
	order := make([]string, 0)
	// TS: for i = loadedSkills.length-1 down to 0 — shallow dirs first, deep dirs last → map overwrite deeper wins.
	for i := len(dirs) - 1; i >= 0; i-- {
		d := strings.TrimSpace(dirs[i])
		if d == "" {
			continue
		}
		entries, err := loadSkillsFromDir(d, "projectSettings")
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.Cmd.Type != "prompt" {
				continue
			}
			n := e.Cmd.Name
			if _, ok := byName[n]; !ok {
				order = append(order, n)
			}
			byName[n] = e.Cmd
		}
	}
	out := make([]types.Command, 0, len(order))
	for _, n := range order {
		out = append(out, byName[n])
	}
	return out, nil
}

// LoadDynamicSkillCommandsForPaths runs [DiscoverSkillDirsForPaths] then [LoadSkillsFromSkillDirectories].
// Respects [LoadOptions]: skips when projectSettings disabled or [LoadOptions.SkillsPluginOnlyLocked] (TS gates).
// Applies [filterUnconditionalSkills] when opts does not set [LoadOptions.IncludeConditionalSkills].
func LoadDynamicSkillCommandsForPaths(filePaths []string, cwd string, opts LoadOptions, discoverSeen map[string]struct{}) ([]types.Command, error) {
	if opts.SkillsPluginOnlyLocked || !opts.isSettingSourceEnabled("projectSettings") {
		return nil, nil
	}
	dirs := DiscoverSkillDirsForPaths(filePaths, cwd, discoverSeen)
	cmds, err := LoadSkillsFromSkillDirectories(dirs)
	if err != nil {
		return nil, err
	}
	return filterUnconditionalSkills(cmds, opts), nil
}

// LoadAndGetCommandsWithFilePathsDynamic primes conditional state, applies file-path dynamic side effects
// (TS activateConditionalSkillsForPaths + addSkillDirectories), then [GetCommands] (load + filter + getDynamicSkills merge).
func LoadAndGetCommandsWithFilePathsDynamic(ctx context.Context, cwd string, opts LoadOptions, auth GetCommandsAuth, touchedFiles []string, discoverSeen map[string]struct{}) ([]types.Command, error) {
	if _, err := LoadAndFilterCommands(ctx, cwd, opts, auth); err != nil {
		return nil, err
	}
	if err := ProcessFilePathsForDynamicSkills(touchedFiles, cwd, opts, discoverSeen); err != nil {
		return nil, err
	}
	return GetCommands(ctx, cwd, opts, auth)
}
