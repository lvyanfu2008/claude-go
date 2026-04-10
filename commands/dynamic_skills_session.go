package commands

import (
	"strings"
	"sync"

	"goc/types"
)

// Session state mirrors src/skills/loadSkillsDir.ts dynamicSkills Map + dynamicSkillDirs Set.
// Map iteration order: first insertion of each key; set(name) on existing key updates value only.

var (
	dynSessMu     sync.Mutex
	dynSessByName map[string]types.Command
	dynSessOrder  []string

	globalDynDirSeenMu sync.Mutex
	globalDynDirSeen   map[string]struct{}
)

func dynamicSkillsSessionSetLocked(name string, cmd types.Command) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if dynSessByName == nil {
		dynSessByName = make(map[string]types.Command)
	}
	if _, ok := dynSessByName[name]; !ok {
		dynSessOrder = append(dynSessOrder, name)
	}
	dynSessByName[name] = cmd
}

// dynamicSkillsSessionSet merges one skill into the session map (TS dynamicSkills.set).
func dynamicSkillsSessionSet(name string, cmd types.Command) {
	dynSessMu.Lock()
	defer dynSessMu.Unlock()
	dynamicSkillsSessionSetLocked(name, cmd)
}

func clearDynamicSkillsSessionLocked() {
	dynSessByName = nil
	dynSessOrder = nil
}

// GetDynamicSkills mirrors src/skills/loadSkillsDir.ts getDynamicSkills: Array.from(dynamicSkills.values()).
func GetDynamicSkills() []types.Command {
	dynSessMu.Lock()
	defer dynSessMu.Unlock()
	if len(dynSessOrder) == 0 {
		return nil
	}
	out := make([]types.Command, 0, len(dynSessOrder))
	for _, n := range dynSessOrder {
		if c, ok := dynSessByName[n]; ok {
			out = append(out, c)
		}
	}
	return out
}

// AddSkillDirectories mirrors src/skills/loadSkillsDir.ts addSkillDirectories: load each dir (parallel in TS),
// merge shallow→deep so deeper wins, accumulate into session dynamicSkills, then invalidate command memo.
func AddSkillDirectories(dirs []string, opts LoadOptions) error {
	if opts.SkillsPluginOnlyLocked || !opts.isSettingSourceEnabled("projectSettings") {
		return nil
	}
	if len(dirs) == 0 {
		return nil
	}
	loaded := make([][]SkillLoadEntry, len(dirs))
	for i, d := range dirs {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		entries, err := loadSkillsFromDir(d, "projectSettings")
		if err != nil {
			return err
		}
		loaded[i] = entries
	}
	dynSessMu.Lock()
	for i := len(loaded) - 1; i >= 0; i-- {
		for _, e := range loaded[i] {
			if e.Cmd.Type != "prompt" {
				continue
			}
			n := strings.TrimSpace(e.Cmd.Name)
			if n == "" {
				continue
			}
			dynamicSkillsSessionSetLocked(n, e.Cmd)
		}
	}
	dynSessMu.Unlock()
	ClearCommandMemoizationCaches()
	return nil
}

// ProcessFilePathsForDynamicSkills runs activateConditionalSkillsForPaths then discovers new skill dirs and
// AddSkillDirectories (TS file-operation side effects before getCommands). When discoverSeen is nil, uses a
// process-wide seen map (TS dynamicSkillDirs).
func ProcessFilePathsForDynamicSkills(touchedFiles []string, cwd string, opts LoadOptions, discoverSeen map[string]struct{}) error {
	ActivateConditionalSkillsForPaths(touchedFiles, cwd)
	var dirs []string
	if discoverSeen == nil {
		globalDynDirSeenMu.Lock()
		if globalDynDirSeen == nil {
			globalDynDirSeen = make(map[string]struct{})
		}
		dirs = DiscoverSkillDirsForPaths(touchedFiles, cwd, globalDynDirSeen)
		globalDynDirSeenMu.Unlock()
	} else {
		dirs = DiscoverSkillDirsForPaths(touchedFiles, cwd, discoverSeen)
	}
	return AddSkillDirectories(dirs, opts)
}

// ClearDynamicSkills mirrors src/skills/loadSkillsDir.ts clearDynamicSkills (dynamic dirs seen, dynamicSkills,
// conditionalSkills, activatedConditionalSkillNames).
func ClearDynamicSkills() {
	dynSessMu.Lock()
	clearDynamicSkillsSessionLocked()
	dynSessMu.Unlock()

	globalDynDirSeenMu.Lock()
	globalDynDirSeen = nil
	globalDynDirSeenMu.Unlock()

	clearConditionalRuntimeMaps()
}
