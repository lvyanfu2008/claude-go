// Package plugins adapts claude-go-plugins (core plugin system) to claude-go types
// and hooks. It is the integration layer between the independent plugin module
// and the claude-go runtime.
package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	plugins "claude-go-plugins"

	"goc/ccb-engine/diaglog"
	"goc/ccb-engine/settingsfile"
	goccommands "goc/commands"
	"goc/types"
)

func init() {
	goccommands.RegisterPluginSkillsLoader(LoadAllPluginSkills)
}

// LoadSkillsFromPluginDir loads all skill commands from a plugin directory
// by resolving the manifest first, then loading each skill.
func LoadSkillsFromPluginDir(pluginDir, pluginName, sourceName, pluginPath string) ([]types.Command, error) {
	manifest, err := plugins.LoadManifest(pluginDir)
	if err != nil {
		return nil, fmt.Errorf("load manifest for %s: %w", pluginName, err)
	}
	return LoadSkillsFromPlugin(pluginDir, manifest, pluginName, sourceName)
}

// LoadSkillsFromPlugin loads all skill commands from an already-parsed manifest.
func LoadSkillsFromPlugin(pluginDir string, manifest *plugins.PluginManifest, pluginName, sourceName string) ([]types.Command, error) {
	var all []types.Command
	loadedPaths := make(map[string]struct{})

	for _, skillRel := range manifest.Skills {
		skillPath := filepath.Join(pluginDir, skillRel)
		cmds, err := goccommands.LoadSkillsFromDirectory(
			context.Background(),
			skillPath,
			pluginName,
			sourceName,
			pluginDir,
			manifestToRaw(manifest),
			loadedPaths,
		)
		if err != nil {
			diaglog.Line("[goc/plugins] LoadSkillsFromPlugin: skip %s: %v", skillRel, err)
			continue
		}
		all = append(all, cmds...)
	}
	return all, nil
}

func manifestToRaw(m *plugins.PluginManifest) json.RawMessage {
	b, _ := json.Marshal(m)
	return json.RawMessage(b)
}

// LoadAllPluginSkills scans enabled plugins and returns their skill commands.
// This replaces the stub in commands.loadPluginSkills().
func LoadAllPluginSkills(cwd string) ([]types.Command, error) {
	merged, err := settingsfile.MergeEnabledPlugins(cwd)
	if err != nil {
		return nil, err
	}

	cacheDir := pluginCacheDir()
	var all []types.Command

	for pluginID, enabled := range merged {
		if !enabled {
			continue
		}

		name, marketplace, _ := strings.Cut(pluginID, "@")
		pluginDir := filepath.Join(cacheDir, marketplace, name)
		currentLink := filepath.Join(pluginDir, "current")
		realPath, err := os.Readlink(currentLink)
		if err != nil {
			diaglog.Line("[goc/plugins] LoadAllPluginSkills: skip %s: current symlink: %v", pluginID, err)
			continue
		}
		if !filepath.IsAbs(realPath) {
			realPath = filepath.Join(pluginDir, realPath)
		}

		cmds, err := LoadSkillsFromPluginDir(realPath, name, pluginID, realPath)
		if err != nil {
			diaglog.Line("[goc/plugins] LoadAllPluginSkills: skip %s: %v", pluginID, err)
			continue
		}
		all = append(all, cmds...)
	}
	return all, nil
}

// pluginCacheDir returns the directory where plugins are cached on disk.
func pluginCacheDir() string {
	if d := os.Getenv("CLAUDE_PLUGIN_CACHE_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "plugins", "cache")
}
