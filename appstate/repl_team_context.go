package appstate

import "encoding/json"

// ReplRegisteredToolSnapshot mirrors replContext.registeredTools map values (handler omitted).
type ReplRegisteredToolSnapshot struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
}

// ReplContextState mirrors AppStateStore.ts replContext JSON-safe subset.
// vmContext and console are omitted (non-JSON / functions in TS).
type ReplContextState struct {
	RegisteredTools map[string]ReplRegisteredToolSnapshot `json:"registeredTools"`
}

// TeamTeammateInfo mirrors teamContext.teammates entries.
type TeamTeammateInfo struct {
	Name            string `json:"name"`
	AgentType       string `json:"agentType,omitempty"`
	Color           string `json:"color,omitempty"`
	TmuxSessionName string `json:"tmuxSessionName"`
	TmuxPaneID      string `json:"tmuxPaneId"`
	Cwd             string `json:"cwd"`
	WorktreePath    string `json:"worktreePath,omitempty"`
	SpawnedAt       int64  `json:"spawnedAt"`
}

// TeamContextState mirrors AppStateStore.ts teamContext.
type TeamContextState struct {
	TeamName       string                      `json:"teamName"`
	TeamFilePath   string                      `json:"teamFilePath"`
	LeadAgentID    string                      `json:"leadAgentId"`
	SelfAgentID    string                      `json:"selfAgentId,omitempty"`
	SelfAgentName  string                      `json:"selfAgentName,omitempty"`
	IsLeader       *bool                       `json:"isLeader,omitempty"`
	SelfAgentColor string                      `json:"selfAgentColor,omitempty"`
	Teammates      map[string]TeamTeammateInfo `json:"teammates"`
}
