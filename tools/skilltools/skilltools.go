// Package skilltools exposes tool list helpers for callers outside goc/internal/anthropic
// (e.g. gou/pui cannot import that package directly).
package skilltools

import (
	"encoding/json"
	"os"
	"strings"

	"goc/internal/anthropic"
)

// DiscoverSkillsToolNameFromEnv returns CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME (empty = TS default off / no tool).
func DiscoverSkillsToolNameFromEnv() string {
	return strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME"))
}

// SkillToolName matches TS SKILL_TOOL_NAME.
func SkillToolName() string {
	return anthropic.SkillToolName
}

// GouDemoDefaultToolsJSON marshals Skill + echo_stub for gou-demo parity.
func GouDemoDefaultToolsJSON() (json.RawMessage, error) {
	return anthropic.GouDemoDefaultToolsJSON()
}

// GouDemoParityToolsJSON returns tools[] from the Go tool wire (AssembleToolPoolFromGoWire), agent listing patch, optional
// DiscoverSkills when CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME is set, and default stub tools — see [anthropic.GouParityToolsJSON].
func GouDemoParityToolsJSON() (json.RawMessage, error) {
	return anthropic.GouParityToolsJSON()
}
