package commands

import (
	"path/filepath"
	"strings"

	"goc/ccb-engine/diaglog"
)

// loadAllCounts is per-source lengths right before concat in [LoadAllCommands].
type loadAllCounts struct {
	Bundled       int
	BuiltinPlugin int
	SkillDir      int
	Workflow      int
	PluginCmd     int
	PluginSkills  int
	Builtins      int
}

func logLoadAllCommands(cwd string, cacheHit bool, c loadAllCounts, total int) {
	cwdShow := strings.TrimSpace(cwd)
	if cwdShow == "" {
		cwdShow = "."
	}
	if abs, err := filepath.Abs(cwdShow); err == nil {
		cwdShow = abs
	}
	if cacheHit {
		diaglog.Line("[goc/commands] LoadAllCommands cache=hit cwd=%q total=%d", cwdShow, total)
		return
	}
	diaglog.Line("[goc/commands] LoadAllCommands cache=miss cwd=%q bundled=%d builtin_plugin=%d skill_dir=%d workflow=%d plugin_cmd=%d plugin_skills=%d builtins=%d total=%d",
		cwdShow, c.Bundled, c.BuiltinPlugin, c.SkillDir, c.Workflow, c.PluginCmd, c.PluginSkills, c.Builtins, total)
}

func logLoadAndFilterCommands(loaded, filtered int) {
	diaglog.Line("[goc/commands] LoadAndFilterCommands loaded=%d after_availability_filter=%d", loaded, filtered)
}
