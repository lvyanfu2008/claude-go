package appstate

import (
	"encoding/json"

	"goc/types"
)

// AgentLoadFailure mirrors AgentDefinitionsResult.failedFiles entries.
type AgentLoadFailure struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// AgentDefinitionData mirrors the JSON-serializable subset of src/tools/AgentTool/loadAgentsDir.ts
// AgentDefinition (BaseAgentDefinition + source discriminator). Omits getSystemPrompt / callback.
type AgentDefinitionData struct {
	AgentType string `json:"agentType"`
	Source    string `json:"source"` // built-in | userSettings | projectSettings | policySettings | flagSettings | plugin
	WhenToUse string `json:"whenToUse"`

	Tools                  []string              `json:"tools,omitempty"`
	DisallowedTools        []string              `json:"disallowedTools,omitempty"`
	Skills                 []string              `json:"skills,omitempty"`
	McpServers             []json.RawMessage     `json:"mcpServers,omitempty"`
	Hooks                  json.RawMessage       `json:"hooks,omitempty"`
	Color                  string                `json:"color,omitempty"`
	Model                  string                `json:"model,omitempty"`
	Effort                 *EffortValue          `json:"effort,omitempty"`
	PermissionMode         *types.PermissionMode `json:"permissionMode,omitempty"`
	MaxTurns               *int                  `json:"maxTurns,omitempty"`
	Filename               string                `json:"filename,omitempty"`
	BaseDir                string                `json:"baseDir,omitempty"`
	CriticalSystemReminder string                `json:"criticalSystemReminder_EXPERIMENTAL,omitempty"`
	RequiredMcpServers     []string              `json:"requiredMcpServers,omitempty"`
	Background             *bool                 `json:"background,omitempty"`
	InitialPrompt          string                `json:"initialPrompt,omitempty"`
	Memory                 json.RawMessage       `json:"memory,omitempty"`
	Isolation              *string               `json:"isolation,omitempty"`
	PendingSnapshotUpdate  json.RawMessage       `json:"pendingSnapshotUpdate,omitempty"`
	OmitClaudeMd           *bool                 `json:"omitClaudeMd,omitempty"`

	Plugin string `json:"plugin,omitempty"` // plugin agents only
}

// AgentDefinitionsResult mirrors src/tools/AgentTool/loadAgentsDir.ts AgentDefinitionsResult.
type AgentDefinitionsResult struct {
	ActiveAgents      []AgentDefinitionData `json:"activeAgents"`
	AllAgents         []AgentDefinitionData `json:"allAgents"`
	FailedFiles       []AgentLoadFailure    `json:"failedFiles,omitempty"`
	AllowedAgentTypes []string              `json:"allowedAgentTypes,omitempty"`
}

// EmptyAgentDefinitionsResult matches TS getDefaultAppState agentDefinitions default.
func EmptyAgentDefinitionsResult() AgentDefinitionsResult {
	return AgentDefinitionsResult{
		ActiveAgents: []AgentDefinitionData{},
		AllAgents:    []AgentDefinitionData{},
	}
}

// MarshalJSON encodes nil slices as [] (TS default empty arrays, not null).
func (r AgentDefinitionsResult) MarshalJSON() ([]byte, error) {
	type out struct {
		ActiveAgents      []AgentDefinitionData `json:"activeAgents"`
		AllAgents         []AgentDefinitionData `json:"allAgents"`
		FailedFiles       []AgentLoadFailure    `json:"failedFiles,omitempty"`
		AllowedAgentTypes []string              `json:"allowedAgentTypes,omitempty"`
	}
	aa := r.ActiveAgents
	if aa == nil {
		aa = []AgentDefinitionData{}
	}
	all := r.AllAgents
	if all == nil {
		all = []AgentDefinitionData{}
	}
	return json.Marshal(out{
		ActiveAgents:      aa,
		AllAgents:         all,
		FailedFiles:       r.FailedFiles,
		AllowedAgentTypes: r.AllowedAgentTypes,
	})
}

// UnmarshalJSON normalizes nil active/all slices to empty.
func (r *AgentDefinitionsResult) UnmarshalJSON(data []byte) error {
	var s struct {
		ActiveAgents      []AgentDefinitionData `json:"activeAgents"`
		AllAgents         []AgentDefinitionData `json:"allAgents"`
		FailedFiles       []AgentLoadFailure    `json:"failedFiles"`
		AllowedAgentTypes []string              `json:"allowedAgentTypes"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*r = AgentDefinitionsResult{
		ActiveAgents:      s.ActiveAgents,
		AllAgents:         s.AllAgents,
		FailedFiles:       s.FailedFiles,
		AllowedAgentTypes: s.AllowedAgentTypes,
	}
	if r.ActiveAgents == nil {
		r.ActiveAgents = []AgentDefinitionData{}
	}
	if r.AllAgents == nil {
		r.AllAgents = []AgentDefinitionData{}
	}
	return nil
}

// NormalizeAgentDefinitionData fills nil slices after json.Unmarshal (TS defaults to []).
func NormalizeAgentDefinitionData(d *AgentDefinitionData) {
	if d == nil {
		return
	}
	if d.Tools == nil {
		d.Tools = []string{}
	}
	if d.DisallowedTools == nil {
		d.DisallowedTools = []string{}
	}
	if d.Skills == nil {
		d.Skills = []string{}
	}
	if d.McpServers == nil {
		d.McpServers = []json.RawMessage{}
	}
	if d.RequiredMcpServers == nil {
		d.RequiredMcpServers = []string{}
	}
}
