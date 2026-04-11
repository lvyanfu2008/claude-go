// Package tscontext holds the JSON snapshot shape for optional in-process prompt/commands/tools
// injection (tests and parity helpers; no external Bun bridge).
package tscontext

import (
	"encoding/json"
)

// Snapshot is the JSON shape for default/user/system prompt parts plus commands and tools metadata.
type Snapshot struct {
	DefaultSystemPrompt []string          `json:"defaultSystemPrompt"`
	UserContext         map[string]string `json:"userContext"`
	SystemContext       map[string]string `json:"systemContext"`
	Commands            json.RawMessage   `json:"commands"`
	Tools               json.RawMessage   `json:"tools"`
	MainLoopModel       string            `json:"mainLoopModel"`
	// SkillToolCommands is TS getSkillToolCommands(cwd) serialized data-only (same shape as commands).
	SkillToolCommands json.RawMessage `json:"skillToolCommands,omitempty"`
	// SlashCommandToolSkills is TS getSlashCommandToolSkills(cwd) data-only.
	SlashCommandToolSkills json.RawMessage `json:"slashCommandToolSkills,omitempty"`
	// Agents is TS active agent definitions (JSON-serializable subset).
	Agents json.RawMessage `json:"agents,omitempty"`
}
