package hookexec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"goc/ccb-engine/settingsfile"
)

// MatcherGroup mirrors one entry in settings "hooks"."EVENT_NAME"[].
type MatcherGroup struct {
	Matcher string          `json:"matcher"`
	Hooks   []json.RawMessage `json:"hooks"`
}

// HooksTable is hooks keyed by event name (e.g. SessionStart, InstructionsLoaded).
type HooksTable map[string][]MatcherGroup

func readHooksTable(path string) (HooksTable, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc struct {
		Hooks HooksTable `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Hooks == nil {
		return nil, nil
	}
	return doc.Hooks, nil
}

func mergeHooksTable(dst HooksTable, src HooksTable) HooksTable {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = HooksTable{}
	}
	for ev, matchers := range src {
		dst[ev] = append(dst[ev], matchers...)
	}
	return dst
}

// MergedHooksFromPaths loads and concatenates hook matcher groups in merge order:
// user settings.json, then project settings.go.json, then project settings.local.json.
// Project .claude/settings.json (TS-only) is not read.
func MergedHooksFromPaths(projectRoot string) (HooksTable, error) {
	var merged HooksTable
	userPath := settingsfile.UserClaudeSettingsPath()
	tUser, err := readHooksTable(userPath)
	if err != nil {
		return nil, err
	}
	merged = mergeHooksTable(merged, tUser)

	root := strings.TrimSpace(projectRoot)
	if root != "" {
		goPath := filepath.Join(root, ".claude", "settings.go.json")
		tGo, err := readHooksTable(goPath)
		if err != nil {
			return nil, err
		}
		merged = mergeHooksTable(merged, tGo)

		localPath := filepath.Join(root, ".claude", "settings.local.json")
		tLoc, err := readHooksTable(localPath)
		if err != nil {
			return nil, err
		}
		merged = mergeHooksTable(merged, tLoc)
	}
	return merged, nil
}

// MergedHooksForCwd resolves project root from cwd (FindClaudeProjectRoot) then merges hooks.
func MergedHooksForCwd(cwd string) (HooksTable, error) {
	wd := strings.TrimSpace(cwd)
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			wd = "."
		}
	}
	root, err := settingsfile.FindClaudeProjectRoot(wd)
	if err != nil || strings.TrimSpace(root) == "" {
		root = wd
	}
	return MergedHooksFromPaths(root)
}
