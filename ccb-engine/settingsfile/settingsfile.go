// Package settingsfile loads Claude Code-style settings `env` blocks for Go binaries:
// user ~/.claude/settings.json, project settings.go.json, project settings.local.json.
// Project .claude/settings.json is consumed by the TypeScript CLI only — not merged here.
package settingsfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// UserClaudeSettingsPath matches TS getClaudeConfigHomeDir + settings.json:
// $CLAUDE_CONFIG_DIR/settings.json, or $HOME/.claude/settings.json.
func UserClaudeSettingsPath() string {
	if d := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); d != "" {
		return filepath.Join(d, "settings.json")
	}
	h, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(h) == "" {
		return ""
	}
	return filepath.Join(h, ".claude", "settings.json")
}

// ReadUserSettingsEnv returns the "env" map from user settings.json only (no project merge).
func ReadUserSettingsEnv() (map[string]string, error) {
	return readEnvFromSettingsPath(UserClaudeSettingsPath())
}

// GlobalClaudeJSONPath is ~/.claude.json (TS global config file for auth/env subset).
func GlobalClaudeJSONPath() string {
	h, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(h) == "" {
		return ""
	}
	return filepath.Join(h, ".claude.json")
}

// ReadGlobalClaudeJSONEnv reads top-level "env" from ~/.claude.json if present.
func ReadGlobalClaudeJSONEnv() (map[string]string, error) {
	return readEnvFromSettingsPath(GlobalClaudeJSONPath())
}

// readEnvFromSettingsPath reads one settings file's top-level "env" as string map.
// Missing file → (nil, nil). Invalid JSON → error. Empty path → (nil, nil).
func readEnvFromSettingsPath(path string) (map[string]string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc struct {
		Env map[string]any `json:"env"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Env == nil {
		return nil, nil
	}
	out := make(map[string]string)
	for k, v := range doc.Env {
		if k == "" {
			continue
		}
		s, ok := envValueToString(v)
		if !ok {
			continue
		}
		out[k] = s
	}
	return out, nil
}

func applyEnvMapSkipExisting(m map[string]string) error {
	if m == nil {
		return nil
	}
	for k, v := range m {
		if k == "" {
			continue
		}
		if existing := os.Getenv(k); existing != "" {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("setenv %s: %w", k, err)
		}
	}
	return nil
}

// mergeEnvMany merges maps left-to-right; later maps override earlier keys.
func mergeEnvMany(maps ...map[string]string) map[string]string {
	var out map[string]string
	for _, m := range maps {
		if m == nil {
			continue
		}
		if out == nil {
			out = make(map[string]string)
		}
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// ApplyMergedClaudeSettingsEnv merges env from (later files override earlier for duplicate keys):
// 1) UserClaudeSettingsPath() — CLAUDE_CONFIG_DIR/settings.json or ~/.claude/settings.json
// 2) projectRoot/.claude/settings.go.json (Go / ccb-engine / gou-demo; optional)
// 3) projectRoot/.claude/settings.local.json (optional)
//
// Project .claude/settings.json is not read — it is for the TypeScript CLI only.
//
// Process env entries that are already non-empty are left unchanged (shell / parent wins).
func ApplyMergedClaudeSettingsEnv(projectRoot string) error {
	userPath := UserClaudeSettingsPath()
	goPath := filepath.Join(projectRoot, ".claude", "settings.go.json")
	localPath := filepath.Join(projectRoot, ".claude", "settings.local.json")

	userMap, err := readEnvFromSettingsPath(userPath)
	if err != nil {
		return err
	}
	goMap, err := readEnvFromSettingsPath(goPath)
	if err != nil {
		return err
	}
	localMap, err := readEnvFromSettingsPath(localPath)
	if err != nil {
		return err
	}
	return applyEnvMapSkipExisting(mergeEnvMany(userMap, goMap, localMap))
}

// ApplyUserAndProjectClaudeEnv is a deprecated alias for ApplyMergedClaudeSettingsEnv (home is ignored).
func ApplyUserAndProjectClaudeEnv(home, projectRoot string) error {
	_ = home
	return ApplyMergedClaudeSettingsEnv(projectRoot)
}

// ApplyProjectClaudeEnv reads root/.claude/settings.json and applies the top-level "env"
// map to the process environment. Variables that are already set in the environment
// (non-empty) are left unchanged so the shell and parent process keep precedence.
//
// If the file is missing, returns nil. If the file exists but is invalid JSON, returns an error.
func ApplyProjectClaudeEnv(root string) error {
	path := filepath.Join(root, ".claude", "settings.json")
	m, err := readEnvFromSettingsPath(path)
	if err != nil {
		return err
	}
	return applyEnvMapSkipExisting(m)
}

func envValueToString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		// JSON numbers decode as float64
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10), true
		}
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case bool:
		if t {
			return "1", true
		}
		return "0", true
	case nil:
		return "", false
	default:
		return fmt.Sprint(t), true
	}
}
