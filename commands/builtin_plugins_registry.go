package commands

import (
	"sort"
	"strings"
	"sync"

	"goc/ccb-engine/settingsfile"
	"goc/types"
)

// BuiltinMarketplaceName matches TS BUILTIN_MARKETPLACE_NAME ("builtin").
const BuiltinMarketplaceName = "builtin"

// BuiltinPluginDefinition mirrors TS BuiltinPluginDefinition (listing / toggle metadata).
type BuiltinPluginDefinition struct {
	Name            string
	Description     string
	Version         string
	Skills          []types.Command
	DefaultEnabled  *bool
	IsAvailable     func() bool // nil => always available
	HooksConfig     []byte      // TS HooksSettings JSON; optional
	MCPServers      []byte      // optional JSON object
}

// PluginManifestLite is the subset of TS PluginManifest used for built-in listing.
type PluginManifestLite struct {
	Name        string
	Description string
	Version     string
}

// LoadedPlugin mirrors TS LoadedPlugin fields used for built-in plugins.
type LoadedPlugin struct {
	Name        string
	Manifest    PluginManifestLite
	Path        string
	Source      string
	Repository  string
	Enabled     bool
	IsBuiltin   bool
	HooksConfig []byte
	MCPServers  []byte
}

var (
	builtinPluginsMu       sync.RWMutex
	builtinPluginsRegistry = map[string]BuiltinPluginDefinition{}
)

// RegisterBuiltinPlugin registers a built-in plugin (call from initBuiltinPlugins). Mirrors registerBuiltinPlugin.
func RegisterBuiltinPlugin(def BuiltinPluginDefinition) {
	if strings.TrimSpace(def.Name) == "" {
		return
	}
	builtinPluginsMu.Lock()
	defer builtinPluginsMu.Unlock()
	builtinPluginsRegistry[def.Name] = def
}

// ClearBuiltinPluginsRegistry clears the registry (tests only). Mirrors clearBuiltinPlugins.
func ClearBuiltinPluginsRegistry() {
	builtinPluginsMu.Lock()
	defer builtinPluginsMu.Unlock()
	builtinPluginsRegistry = map[string]BuiltinPluginDefinition{}
}

// GetBuiltinPluginDefinition returns a registered definition by name.
func GetBuiltinPluginDefinition(name string) (BuiltinPluginDefinition, bool) {
	builtinPluginsMu.RLock()
	defer builtinPluginsMu.RUnlock()
	d, ok := builtinPluginsRegistry[name]
	return d, ok
}

// IsBuiltinPluginID is true when pluginId ends with @builtin (TS isBuiltinPluginId).
func IsBuiltinPluginID(pluginID string) bool {
	return strings.HasSuffix(pluginID, "@"+BuiltinMarketplaceName)
}

// GetBuiltinPlugins returns enabled and disabled built-in plugins using merged enabledPlugins from settings.
// Mirrors src/plugins/builtinPlugins.ts getBuiltinPlugins.
func GetBuiltinPlugins(cwd string) (enabled []LoadedPlugin, disabled []LoadedPlugin, err error) {
	merged, err := settingsfile.MergeEnabledPlugins(cwd)
	if err != nil {
		return nil, nil, err
	}

	builtinPluginsMu.RLock()
	defer builtinPluginsMu.RUnlock()

	enabled = make([]LoadedPlugin, 0)
	disabled = make([]LoadedPlugin, 0)

	names := make([]string, 0, len(builtinPluginsRegistry))
	for n := range builtinPluginsRegistry {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		def := builtinPluginsRegistry[name]
		if def.IsAvailable != nil && !def.IsAvailable() {
			continue
		}
		pluginID := name + "@" + BuiltinMarketplaceName
		userVal, hasUser := merged[pluginID]
		isEnabled := true
		if hasUser {
			isEnabled = userVal
		} else if def.DefaultEnabled != nil {
			isEnabled = *def.DefaultEnabled
		}

		ver := def.Version
		lp := LoadedPlugin{
			Name: name,
			Manifest: PluginManifestLite{
				Name:        name,
				Description: def.Description,
				Version:     ver,
			},
			Path:        BuiltinMarketplaceName,
			Source:      pluginID,
			Repository:  pluginID,
			Enabled:     isEnabled,
			IsBuiltin:   true,
			HooksConfig: append([]byte(nil), def.HooksConfig...),
			MCPServers:  append([]byte(nil), def.MCPServers...),
		}
		if isEnabled {
			enabled = append(enabled, lp)
		} else {
			disabled = append(disabled, lp)
		}
	}
	return enabled, disabled, nil
}

// BuiltinPluginSkillCommands returns skills from enabled built-in plugins as Command rows (TS getBuiltinPluginSkillCommands).
func BuiltinPluginSkillCommands(cwd string) ([]types.Command, error) {
	enabled, _, err := GetBuiltinPlugins(cwd)
	if err != nil {
		return nil, err
	}
	builtinPluginsMu.RLock()
	defer builtinPluginsMu.RUnlock()

	out := make([]types.Command, 0)
	for _, p := range enabled {
		def, ok := builtinPluginsRegistry[p.Name]
		if !ok || len(def.Skills) == 0 {
			continue
		}
		for _, sk := range def.Skills {
			out = append(out, applyBundledBuiltinSkillDefaults(sk))
		}
	}
	return out, nil
}

// applyBundledBuiltinSkillDefaults mirrors skillDefinitionToCommand defaults in builtinPlugins.ts.
func applyBundledBuiltinSkillDefaults(c types.Command) types.Command {
	out := c
	t := true
	if out.HasUserSpecifiedDescription == nil {
		out.HasUserSpecifiedDescription = &t
	}
	if out.UserInvocable == nil {
		out.UserInvocable = &t
	}
	if out.IsHidden == nil {
		uv := out.UserInvocable != nil && *out.UserInvocable
		h := !uv
		out.IsHidden = &h
	}
	if out.DisableModelInvocation == nil {
		f := false
		out.DisableModelInvocation = &f
	}
	if out.ContentLength == nil {
		z := 0
		out.ContentLength = &z
	}
	bundled := "bundled"
	if out.Source == nil {
		out.Source = &bundled
	}
	if out.LoadedFrom == nil {
		out.LoadedFrom = &bundled
	}
	if out.ProgressMessage == nil {
		pm := "running"
		out.ProgressMessage = &pm
	}
	if out.Type == "" {
		out.Type = "prompt"
	}
	return out
}
