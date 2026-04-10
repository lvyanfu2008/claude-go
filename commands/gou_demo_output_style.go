package commands

import (
	_ "embed"
	"strings"
)

//go:embed testdata/builtin_output_style_explanatory.txt
var builtinOutputStyleExplanatory string

//go:embed testdata/builtin_output_style_learning.txt
var builtinOutputStyleLearning string

// ResolveGouDemoOutputStyle picks name+prompt for # Output Style in gou-demo / API parity.
//
// Precedence:
//  1. When both envName and envPrompt are non-empty after trim, returns them (shell / explicit override).
//  2. Otherwise, when settingsOutputStyleKey is a known built-in TS key (default, Explanatory, Learning),
//     returns the embedded prompt text from src/constants/outputStyles.ts OUTPUT_STYLE_CONFIG.
//  3. Unknown keys → empty name and prompt (same as TS missing style).
//
// Custom directory / plugin styles are not resolved here (TS uses getAllOutputStyles); use env pair or extend later.
func ResolveGouDemoOutputStyle(envName, envPrompt, settingsOutputStyleKey string) (name string, prompt string) {
	en := strings.TrimSpace(envName)
	ep := strings.TrimSpace(envPrompt)
	if en != "" && ep != "" {
		return en, ep
	}
	key := strings.TrimSpace(settingsOutputStyleKey)
	if key == "" || strings.EqualFold(key, "default") {
		return "", ""
	}
	switch strings.ToLower(key) {
	case "explanatory":
		return "Explanatory", strings.TrimSpace(builtinOutputStyleExplanatory)
	case "learning":
		return "Learning", strings.TrimSpace(builtinOutputStyleLearning)
	default:
		return "", ""
	}
}
