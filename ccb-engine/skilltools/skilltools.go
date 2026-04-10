// Package skilltools exposes tool list helpers for ccb-engine callers outside internal/anthropic
// (e.g. gou/pui cannot import …/internal/anthropic).
package skilltools

import (
	"encoding/json"
	"os"
	"strings"

	"goc/ccb-engine/internal/anthropic"
)

// DiscoverSkillsToolNameFromEnv returns CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME (empty = TS default off / no tool).
func DiscoverSkillsToolNameFromEnv() string {
	return strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME"))
}

// SkillToolName matches TS SKILL_TOOL_NAME.
func SkillToolName() string {
	return anthropic.SkillToolName
}

// GouDemoDefaultToolsJSON marshals Skill + echo_stub for gou-demo / localturn parity.
func GouDemoDefaultToolsJSON() (json.RawMessage, error) {
	return anthropic.GouDemoDefaultToolsJSON()
}

// GouDemoParityToolsJSON marshals TS-shaped core tools + Skill + echo_stub (phase 2).
func GouDemoParityToolsJSON() (json.RawMessage, error) {
	return anthropic.GouParityToolsJSON()
}
