package settingsfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// partialEnabledPluginsJSON is the minimal shape for enabledPlugins (TS SettingsJson.enabledPlugins).
type partialEnabledPluginsJSON struct {
	EnabledPlugins map[string]bool `json:"enabledPlugins"`
}

// MergeEnabledPlugins loads user → project → local settings files (same paths as TS when cwd is the
// project root for project/local) and merges enabledPlugins with later files winning per key.
// Omits policy/flag/managed sources (subset aligned with go-slash-resolve.md).
func MergeEnabledPlugins(cwd string) (map[string]bool, error) {
	merged := map[string]bool{}

	userPath := userClaudeSettingsPath()
	if err := mergeEnabledPluginsFile(userPath, merged); err != nil {
		return nil, err
	}

	projRoot, err := FindClaudeProjectRootAny(cwd)
	if err != nil {
		return nil, err
	}
	cl := filepath.Join(projRoot, ".claude")
	if err := mergeEnabledPluginsFile(filepath.Join(cl, "settings.json"), merged); err != nil {
		return nil, err
	}
	if err := mergeEnabledPluginsFile(filepath.Join(cl, "settings.local.json"), merged); err != nil {
		return nil, err
	}

	return merged, nil
}

func userClaudeSettingsPath() string {
	if d := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); d != "" {
		return filepath.Join(d, "settings.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".claude", "settings.json")
	}
	return filepath.Join(home, ".claude", "settings.json")
}

func mergeEnabledPluginsFile(path string, merged map[string]bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var p partialEnabledPluginsJSON
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil
	}
	for k, v := range p.EnabledPlugins {
		merged[k] = v
	}
	return nil
}

// PluginEnabled reports whether pluginID is enabled after [MergeEnabledPlugins]; missing key => false.
func PluginEnabled(merged map[string]bool, pluginID string) bool {
	if merged == nil {
		return false
	}
	return merged[pluginID]
}
