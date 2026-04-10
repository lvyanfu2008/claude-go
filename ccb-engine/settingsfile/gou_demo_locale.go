package settingsfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MergeGouDemoLocalePrefs reads top-level "language" and "outputStyle" from Claude settings files.
// Merge order matches [ApplyMergedClaudeSettingsEnv] for project files (later wins):
// when includeUser is true: user settings, then projectRoot/.claude/settings.go.json,
// settings.local.json. Project settings.json is TS-only and is not read here.
// When includeUser is false, only settings.go.json and settings.local.json under the project.
//
// Mirrors TS [SettingsJson] fields used by getSystemPrompt: language, outputStyle (string key).
func MergeGouDemoLocalePrefs(projectRoot string, includeUser bool) (language, outputStyleKey string, err error) {
	var paths []string
	if includeUser {
		if u := UserClaudeSettingsPath(); u != "" {
			paths = append(paths, u)
		}
	}
	cl := filepath.Join(projectRoot, ".claude")
	for _, name := range []string{"settings.go.json", "settings.local.json"} {
		paths = append(paths, filepath.Join(cl, name))
	}

	var lang, style string
	for _, p := range paths {
		l, s, errR := readGouDemoLocaleFromPath(p)
		if errR != nil {
			return "", "", errR
		}
		if l != nil {
			lang = strings.TrimSpace(*l)
		}
		if s != nil {
			style = strings.TrimSpace(*s)
		}
	}
	return lang, style, nil
}

func readGouDemoLocaleFromPath(path string) (language, outputStyle *string, err error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var doc struct {
		Language    *string `json:"language"`
		OutputStyle *string `json:"outputStyle"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return doc.Language, doc.OutputStyle, nil
}
