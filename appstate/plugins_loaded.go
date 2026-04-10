package appstate

import "encoding/json"

// PluginErrorSnapshot is JSON for a single TS PluginError (types/plugin.ts union).
type PluginErrorSnapshot = json.RawMessage

// LoadedPluginData mirrors types/plugin.ts LoadedPlugin (manifest and nested records opaque).
type LoadedPluginData struct {
	Name              string          `json:"name"`
	Manifest          json.RawMessage `json:"manifest"`
	Path              string          `json:"path"`
	Source            string          `json:"source"`
	Repository        string          `json:"repository"`
	Enabled           *bool           `json:"enabled,omitempty"`
	IsBuiltin         *bool           `json:"isBuiltin,omitempty"`
	Sha               string          `json:"sha,omitempty"`
	CommandsPath      string          `json:"commandsPath,omitempty"`
	CommandsPaths     []string        `json:"commandsPaths,omitempty"`
	CommandsMetadata  json.RawMessage `json:"commandsMetadata,omitempty"`
	AgentsPath        string          `json:"agentsPath,omitempty"`
	AgentsPaths       []string        `json:"agentsPaths,omitempty"`
	SkillsPath        string          `json:"skillsPath,omitempty"`
	SkillsPaths       []string        `json:"skillsPaths,omitempty"`
	OutputStylesPath  string          `json:"outputStylesPath,omitempty"`
	OutputStylesPaths []string        `json:"outputStylesPaths,omitempty"`
	HooksConfig       json.RawMessage `json:"hooksConfig,omitempty"`
	McpServers        json.RawMessage `json:"mcpServers,omitempty"`
	LspServers        json.RawMessage `json:"lspServers,omitempty"`
	Settings          json.RawMessage `json:"settings,omitempty"`
}
