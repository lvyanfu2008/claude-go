package config

import (
	"os"
	"path/filepath"
	"strings"

	"goc/ccb-engine/settingsfile"
	"goc/commands"
)

// ResolveToolProjectRoot returns CCB_ENGINE_PROJECT_ROOT if set, else the nearest Go project marker from cwd, else abs(cwd).
func ResolveToolProjectRoot(cwd string) string {
	if r := strings.TrimSpace(os.Getenv("CCB_ENGINE_PROJECT_ROOT")); r != "" {
		if a, err := filepath.Abs(r); err == nil {
			return a
		}
	}
	if pr, err := settingsfile.FindClaudeProjectRoot(cwd); err == nil {
		return pr
	}
	if a, err := filepath.Abs(cwd); err == nil {
		return a
	}
	return cwd
}

// MergedSystemLocale mirrors apiparity.GouDemo: user + project settings.go.json / settings.local.json language/outputStyle with env override.
func MergedSystemLocale() (lang, outputStyleName, outputStylePrompt string) {
	projRoot := settingsfile.ProjectRootLastResolved()
	locLang, locStyleKey, err := settingsfile.MergeGouDemoLocalePrefs(projRoot, true)
	if err != nil {
		Tracef("MergeGouDemoLocalePrefs: %v", err)
		locLang, locStyleKey = "", ""
	}
	lang = strings.TrimSpace(os.Getenv("CLAUDE_CODE_LANGUAGE"))
	if lang == "" {
		lang = locLang
	}
	on, op := commands.ResolveGouDemoOutputStyle(
		os.Getenv("CLAUDE_CODE_OUTPUT_STYLE_NAME"),
		os.Getenv("CLAUDE_CODE_OUTPUT_STYLE_PROMPT"),
		locStyleKey,
	)
	return lang, on, op
}
