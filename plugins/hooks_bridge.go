package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"goc/ccb-engine/diaglog"
	"goc/ccb-engine/settingsfile"
	"goc/hookexec"
)

// LoadPluginHooksTable reads hooks.json from a plugin directory.
func LoadPluginHooksTable(pluginDir string) (hookexec.HooksTable, error) {
	path := filepath.Join(pluginDir, "hooks", "hooks.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var doc struct {
		Hooks hookexec.HooksTable `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	// Resolve ${CLAUDE_PLUGIN_ROOT} in hook commands
	resolved := make(hookexec.HooksTable)
	for event, matchers := range doc.Hooks {
		var resolvedMatchers []hookexec.MatcherGroup
		for _, mg := range matchers {
			var resolvedHooks []json.RawMessage
			for _, h := range mg.Hooks {
				s := string(h)
				if strings.Contains(s, "${CLAUDE_PLUGIN_ROOT}") {
					s = strings.ReplaceAll(s, "${CLAUDE_PLUGIN_ROOT}", pluginDir)
				}
				resolvedHooks = append(resolvedHooks, json.RawMessage(s))
			}
			resolvedMatchers = append(resolvedMatchers, hookexec.MatcherGroup{
				Matcher: mg.Matcher,
				Hooks:   resolvedHooks,
			})
		}
		resolved[event] = resolvedMatchers
	}

	return resolved, nil
}

// LoadAllPluginHooks scans enabled plugins and merges their hooks tables.
func LoadAllPluginHooks(cwd string) (hookexec.HooksTable, error) {
	merged, err := settingsfile.MergeEnabledPlugins(cwd)
	if err != nil {
		return nil, err
	}

	cacheDir := pluginCacheDir()
	result := make(hookexec.HooksTable)

	for pluginID, enabled := range merged {
		if !enabled {
			continue
		}

		name, marketplace, _ := strings.Cut(pluginID, "@")
		pluginDir := filepath.Join(cacheDir, marketplace, name)
		currentLink := filepath.Join(pluginDir, "current")
		realPath, err := os.Readlink(currentLink)
		if err != nil {
			diaglog.Line("[goc/plugins] LoadAllPluginHooks: skip %s: current symlink: %v", pluginID, err)
			continue
		}
		if !filepath.IsAbs(realPath) {
			realPath = filepath.Join(pluginDir, realPath)
		}

		table, err := LoadPluginHooksTable(realPath)
		if err != nil {
			diaglog.Line("[goc/plugins] LoadAllPluginHooks: skip %s hooks: %v", pluginID, err)
			continue
		}
		if table != nil {
			result = hookexec.MergeHooksTable(result, table)
		}
	}
	return result, nil
}

func init() {
	hookexec.PluginHooksLoader = LoadAllPluginHooks
}
