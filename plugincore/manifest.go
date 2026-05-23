package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// manifestFileNames is the lookup order for plugin manifest files.
// plugin.json takes priority; package.json is fallback for TS/npm-origin plugins.
var manifestFileNames = []string{"plugin.json", "package.json"}

// LoadManifest reads a plugin manifest from dir. It tries plugin.json first,
// then falls back to package.json for compatibility with TS/npm-origin plugins.
func LoadManifest(dir string) (*PluginManifest, error) {
	for _, name := range manifestFileNames {
		p := filepath.Join(dir, name)
		data, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", p, err)
		}
		var m PluginManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		return &m, nil
	}
	return nil, fmt.Errorf("no manifest found in %s (tried plugin.json, package.json)", dir)
}
