package appstate

// String unions from src/state/AppStateStore.ts (FooterItem, expandedView, viewSelectionMode, remoteConnectionStatus).

type FooterItem string

const (
	FooterTasks     FooterItem = "tasks"
	FooterTmux      FooterItem = "tmux"
	FooterBagel     FooterItem = "bagel"
	FooterTeams     FooterItem = "teams"
	FooterBridge    FooterItem = "bridge"
	FooterCompanion FooterItem = "companion"
)

type ExpandedView string

const (
	ExpandedNone      ExpandedView = "none"
	ExpandedTasks     ExpandedView = "tasks"
	ExpandedTeammates ExpandedView = "teammates"
)

type ViewSelectionMode string

const (
	ViewSelNone           ViewSelectionMode = "none"
	ViewSelSelectingAgent ViewSelectionMode = "selecting-agent"
	ViewSelViewingAgent   ViewSelectionMode = "viewing-agent"
)

type RemoteConnectionStatus string

const (
	RemoteConnecting   RemoteConnectionStatus = "connecting"
	RemoteConnected    RemoteConnectionStatus = "connected"
	RemoteReconnecting RemoteConnectionStatus = "reconnecting"
	RemoteDisconnected RemoteConnectionStatus = "disconnected"
)

type PromptSuggestionPromptID string

const (
	PromptIDUserIntent   PromptSuggestionPromptID = "user_intent"
	PromptIDStatedIntent PromptSuggestionPromptID = "stated_intent"
)

type InboxMessageStatus string

const (
	InboxPending    InboxMessageStatus = "pending"
	InboxProcessing InboxMessageStatus = "processing"
	InboxProcessed  InboxMessageStatus = "processed"
)

type InstallStatus string

const (
	InstallPending    InstallStatus = "pending"
	InstallInstalling InstallStatus = "installing"
	InstallInstalled  InstallStatus = "installed"
	InstallFailed     InstallStatus = "failed"
)
