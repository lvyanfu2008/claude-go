package slashresolve

import (
	"goc/ccb-engine/settingsfile"
)

// MergeEnabledPlugins delegates to [settingsfile.MergeEnabledPlugins] (shared with commands builtin plugins).
func MergeEnabledPlugins(cwd string) (map[string]bool, error) {
	return settingsfile.MergeEnabledPlugins(cwd)
}

// PluginEnabled delegates to [settingsfile.PluginEnabled].
func PluginEnabled(merged map[string]bool, pluginID string) bool {
	return settingsfile.PluginEnabled(merged, pluginID)
}
