// Package tscontext loads a one-shot Bun bridge snapshot (system prompt parts, commands, tools)
// for gou-demo parity with src/utils/queryContext.ts + TS tool definitions.
package tscontext

import (
	"encoding/json"
)

// Snapshot is the JSON body emitted by scripts/go-context-bridge.ts on success.
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
