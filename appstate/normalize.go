package appstate

import (
	"encoding/json"

	"goc/types"
)

// NormalizeAppState fixes nil collections after json.Unmarshal of a partial [AppState] (TS Map / [] defaults).
// It also applies [EmptySettingsJSON] when settings is absent, [types.NormalizeToolPermissionContextData],
// active-speculation message/path slices, and initialMessage.allowedPrompts.
func NormalizeAppState(a *AppState) {
	if a == nil {
		return
	}
	if len(a.Settings) == 0 {
		a.Settings = EmptySettingsJSON
	}
	types.NormalizeToolPermissionContextData(&a.ToolPermissionContext)
	normalizeSpeculationCollections(&a.Speculation)
	if a.InitialMessage != nil && a.InitialMessage.AllowedPrompts == nil {
		a.InitialMessage.AllowedPrompts = []AllowedPrompt{}
	}
	if a.Tasks == nil {
		a.Tasks = make(map[string]json.RawMessage)
	}
	if a.AgentNameRegistry == nil {
		a.AgentNameRegistry = make(map[string]string)
	}
	if a.ActiveOverlays == nil {
		a.ActiveOverlays = []string{}
	}
	if a.RemoteAgentTaskSuggestions == nil {
		a.RemoteAgentTaskSuggestions = []RemoteAgentTaskSuggestion{}
	}
	if a.Inbox.Messages == nil {
		a.Inbox.Messages = []InboxMessage{}
	}
	if a.WorkerSandboxPermissions.Queue == nil {
		a.WorkerSandboxPermissions.Queue = []WorkerSandboxQueueItem{}
	}
	if a.Plugins.Commands == nil {
		a.Plugins.Commands = []types.Command{}
	}
	if a.Plugins.InstallationStatus.Marketplaces == nil {
		a.Plugins.InstallationStatus.Marketplaces = []PluginMarketplaceInstall{}
	}
	if a.Plugins.InstallationStatus.Plugins == nil {
		a.Plugins.InstallationStatus.Plugins = []PluginInstall{}
	}
	if a.SkillImprovement.Suggestion != nil && a.SkillImprovement.Suggestion.Updates == nil {
		a.SkillImprovement.Suggestion.Updates = []SkillImprovementUpdate{}
	}
	if a.Attribution.FileStates == nil {
		a.Attribution.FileStates = make(map[string]FileAttributionState)
	}
	if a.Attribution.SessionBaselines == nil {
		a.Attribution.SessionBaselines = make(map[string]SessionBaseline)
	}
	if a.AgentDefinitions.ActiveAgents == nil {
		a.AgentDefinitions.ActiveAgents = []AgentDefinitionData{}
	}
	if a.AgentDefinitions.AllAgents == nil {
		a.AgentDefinitions.AllAgents = []AgentDefinitionData{}
	}
	if a.AgentDefinitions.FailedFiles == nil {
		a.AgentDefinitions.FailedFiles = []AgentLoadFailure{}
	}
	if a.AgentDefinitions.AllowedAgentTypes == nil {
		a.AgentDefinitions.AllowedAgentTypes = []string{}
	}
	for i := range a.AgentDefinitions.ActiveAgents {
		NormalizeAgentDefinitionData(&a.AgentDefinitions.ActiveAgents[i])
	}
	for i := range a.AgentDefinitions.AllAgents {
		NormalizeAgentDefinitionData(&a.AgentDefinitions.AllAgents[i])
	}
	if a.FileHistory.Snapshots == nil {
		a.FileHistory.Snapshots = []FileHistorySnapshot{}
	}
	if a.FileHistory.TrackedFiles == nil {
		a.FileHistory.TrackedFiles = []string{}
	}
	for i := range a.FileHistory.Snapshots {
		if a.FileHistory.Snapshots[i].TrackedFileBackups == nil {
			s := a.FileHistory.Snapshots[i]
			s.TrackedFileBackups = make(map[string]FileHistoryBackup)
			a.FileHistory.Snapshots[i] = s
		}
	}
	if a.Todos == nil {
		a.Todos = TodosMap{}
	}
	for k, v := range a.Todos {
		if v == nil {
			a.Todos[k] = []TodoItem{}
		}
	}
	if a.SessionHooks == nil {
		a.SessionHooks = SessionHooksState{}
	}
	for k, v := range a.SessionHooks {
		a.SessionHooks[k] = SanitizeSessionStoreSnapshot(v)
	}
	if a.Mcp.Clients == nil {
		a.Mcp.Clients = []MCPServerConnectionSnapshot{}
	}
	if a.Mcp.Tools == nil {
		a.Mcp.Tools = []MCPSerializedTool{}
	}
	if a.Mcp.Commands == nil {
		a.Mcp.Commands = []types.Command{}
	}
	if a.Mcp.Resources == nil {
		a.Mcp.Resources = make(map[string][]MCPServerResourceSnapshot)
	}
	a.Mcp.Resources = normalizeMcpResourceMap(a.Mcp.Resources)
	if a.Plugins.Enabled == nil {
		a.Plugins.Enabled = []LoadedPluginData{}
	}
	if a.Plugins.Disabled == nil {
		a.Plugins.Disabled = []LoadedPluginData{}
	}
	if a.Plugins.Errors == nil {
		a.Plugins.Errors = []PluginErrorSnapshot{}
	}
	if a.Notifications.Queue == nil {
		a.Notifications.Queue = []NotificationSnapshot{}
	}
	if a.Elicitation.Queue == nil {
		a.Elicitation.Queue = []ElicitationRequestEventSnapshot{}
	}
	if a.ComputerUseMcpState != nil && a.ComputerUseMcpState.HiddenDuringTurn == nil {
		a.ComputerUseMcpState.HiddenDuringTurn = []string{}
	}
	if a.ComputerUseMcpState != nil && a.ComputerUseMcpState.AllowedApps == nil {
		a.ComputerUseMcpState.AllowedApps = []ComputerUseMcpAllowedApp{}
	}
	if a.ReplContext != nil && a.ReplContext.RegisteredTools == nil {
		a.ReplContext.RegisteredTools = make(map[string]ReplRegisteredToolSnapshot)
	}
	if a.TeamContext != nil && a.TeamContext.Teammates == nil {
		a.TeamContext.Teammates = make(map[string]TeamTeammateInfo)
	}
}
