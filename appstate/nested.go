package appstate

import (
	"goc/types"
)

// PluginMarketplaceInstall mirrors plugins.installationStatus.marketplaces[].
type PluginMarketplaceInstall struct {
	Name   string        `json:"name"`
	Status InstallStatus `json:"status"`
	Error  string        `json:"error,omitempty"`
}

// PluginInstall mirrors plugins.installationStatus.plugins[].
type PluginInstall struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Status InstallStatus `json:"status"`
	Error  string        `json:"error,omitempty"`
}

// PluginsInstallationStatus mirrors plugins.installationStatus.
type PluginsInstallationStatus struct {
	Marketplaces []PluginMarketplaceInstall `json:"marketplaces"`
	Plugins      []PluginInstall            `json:"plugins"`
}

// PluginsState mirrors AppState.plugins.
type PluginsState struct {
	Enabled            []LoadedPluginData        `json:"enabled"`
	Disabled           []LoadedPluginData        `json:"disabled"`
	Commands           []types.Command           `json:"commands"`
	Errors             []PluginErrorSnapshot     `json:"errors"`
	InstallationStatus PluginsInstallationStatus `json:"installationStatus"`
	NeedsRefresh       bool                      `json:"needsRefresh"`
}

// RemoteAgentTaskSuggestion mirrors remoteAgentTaskSuggestions entries.
type RemoteAgentTaskSuggestion struct {
	Summary string `json:"summary"`
	Task    string `json:"task"`
}

// NotificationsState mirrors AppState.notifications.
type NotificationsState struct {
	Current *NotificationSnapshot  `json:"current"`
	Queue   []NotificationSnapshot `json:"queue"`
}

// ElicitationState mirrors AppState.elicitation.
type ElicitationState struct {
	Queue []ElicitationRequestEventSnapshot `json:"queue"`
}

// InboxMessage mirrors inbox.messages[].
type InboxMessage struct {
	ID        string             `json:"id"`
	From      string             `json:"from"`
	Text      string             `json:"text"`
	Timestamp string             `json:"timestamp"`
	Status    InboxMessageStatus `json:"status"`
	Color     string             `json:"color,omitempty"`
	Summary   string             `json:"summary,omitempty"`
}

// InboxState mirrors AppState.inbox.
type InboxState struct {
	Messages []InboxMessage `json:"messages"`
}

// WorkerSandboxQueueItem mirrors workerSandboxPermissions.queue[].
type WorkerSandboxQueueItem struct {
	RequestID   string `json:"requestId"`
	WorkerID    string `json:"workerId"`
	WorkerName  string `json:"workerName"`
	WorkerColor string `json:"workerColor,omitempty"`
	Host        string `json:"host"`
	CreatedAt   int64  `json:"createdAt"`
}

// WorkerSandboxPermissionsState mirrors AppState.workerSandboxPermissions.
type WorkerSandboxPermissionsState struct {
	Queue         []WorkerSandboxQueueItem `json:"queue"`
	SelectedIndex int                      `json:"selectedIndex"`
}

// PendingWorkerRequest mirrors AppState.pendingWorkerRequest.
type PendingWorkerRequest struct {
	ToolName    string `json:"toolName"`
	ToolUseID   string `json:"toolUseId"`
	Description string `json:"description"`
}

// PendingSandboxRequest mirrors AppState.pendingSandboxRequest.
type PendingSandboxRequest struct {
	RequestID string `json:"requestId"`
	Host      string `json:"host"`
}

// PromptSuggestionState mirrors AppState.promptSuggestion.
type PromptSuggestionState struct {
	Text                *string                   `json:"text"`
	PromptID            *PromptSuggestionPromptID `json:"promptId"`
	ShownAt             int64                     `json:"shownAt"`
	AcceptedAt          int64                     `json:"acceptedAt"`
	GenerationRequestID *string                   `json:"generationRequestId"`
}

// SkillImprovementUpdate mirrors skillImprovement.suggestion.updates[].
type SkillImprovementUpdate struct {
	Section string `json:"section"`
	Change  string `json:"change"`
	Reason  string `json:"reason"`
}

// SkillImprovementSuggestion mirrors skillImprovement.suggestion when non-null.
type SkillImprovementSuggestion struct {
	SkillName string                   `json:"skillName"`
	Updates   []SkillImprovementUpdate `json:"updates"`
}

// SkillImprovementState mirrors AppState.skillImprovement.
type SkillImprovementState struct {
	Suggestion *SkillImprovementSuggestion `json:"suggestion"`
}

// StandaloneAgentContext mirrors AppState.standaloneAgentContext.
type StandaloneAgentContext struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// TungstenLastCommand mirrors tungstenLastCommand.
type TungstenLastCommand struct {
	Command   string `json:"command"`
	Timestamp int64  `json:"timestamp"`
}

// TungstenActiveSession mirrors tungstenActiveSession.
type TungstenActiveSession struct {
	SessionName string `json:"sessionName"`
	SocketName  string `json:"socketName"`
	Target      string `json:"target"`
}

// PendingPlanVerification mirrors pendingPlanVerification.
type PendingPlanVerification struct {
	Plan                  string `json:"plan"`
	VerificationStarted   bool   `json:"verificationStarted"`
	VerificationCompleted bool   `json:"verificationCompleted"`
}
