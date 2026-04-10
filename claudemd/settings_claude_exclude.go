package claudemd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type settingsExcludesJSON struct {
	ClaudeMdExcludes []string `json:"claudeMdExcludes"`
}

// MergedClaudeMdExcludes mirrors TS-style layering for paths Go reads: user, settings.go.json,
// settings.local.json, then flag/policy from enabled sources. Project .claude/settings.json
// is TS-only and is not read here.
func MergedClaudeMdExcludes(originalCwd string) []string {
	absCwd, err := filepath.Abs(originalCwd)
	if err != nil {
		absCwd = originalCwd
	}
	var acc []string
	acc = mergeStringUniq(acc, readUserClaudeMdExcludes())
	acc = mergeStringUniq(acc, readClaudeMdExcludesFile(filepath.Join(absCwd, ".claude", "settings.go.json")))
	acc = mergeStringUniq(acc, readClaudeMdExcludesFile(filepath.Join(absCwd, ".claude", "settings.local.json")))

	for _, src := range EnabledSettingSources() {
		var next []string
		switch src {
		case SourceUserSettings, SourceProjectSettings, SourceLocalSettings:
			continue
		case SourceFlagSettings:
			if p := strings.TrimSpace(os.Getenv("CLAUDE_CODE_FLAG_SETTINGS_PATH")); p != "" {
				next = readClaudeMdExcludesFile(p)
			}
		case SourcePolicySettings:
			next = loadManagedClaudeMdExcludes()
		default:
			continue
		}
		if len(next) > 0 {
			acc = mergeStringUniq(acc, next)
		}
	}
	return acc
}

func mergeStringUniq(acc, next []string) []string {
	return dedupeKeepOrder(append(append([]string{}, acc...), next...))
}

func userSettingsFileName() string {
	if truthy(os.Getenv("CLAUDE_CODE_USE_COWORK_PLUGINS")) {
		return "cowork_settings.json"
	}
	return "settings.json"
}

func readUserClaudeMdExcludes() []string {
	cfg, err := ClaudeConfigHomeDir()
	if err != nil {
		return nil
	}
	return readClaudeMdExcludesFile(filepath.Join(cfg, userSettingsFileName()))
}

func readClaudeMdExcludesFile(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil || len(strings.TrimSpace(string(b))) == 0 {
		return nil
	}
	var s settingsExcludesJSON
	if err := json.Unmarshal(b, &s); err != nil {
		return nil
	}
	var out []string
	for _, p := range s.ClaudeMdExcludes {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func loadManagedClaudeMdExcludes() []string {
	basePath := filepath.Join(ManagedFilePath(), "managed-settings.json")
	var acc []string
	acc = mergeStringUniq(acc, readClaudeMdExcludesFile(basePath))
	dropDir := filepath.Join(ManagedFilePath(), "managed-settings.d")
	entries, err := os.ReadDir(dropDir)
	if err != nil {
		return acc
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		acc = mergeStringUniq(acc, readClaudeMdExcludesFile(filepath.Join(dropDir, name)))
	}
	return acc
}
