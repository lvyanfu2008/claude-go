package appstate

import (
	"encoding/json"

	"goc/types"
)

// AppState mirrors src/state/AppStateStore.ts AppState (data-carrying fields).
// Omitted: replBridgePermissionCallbacks, channelPermissionCallbacks (functions).
// Opaque: settings and per-task values (json.RawMessage). [DenialTracking] is optional struct.
type AppState struct {
	Settings                   json.RawMessage                 `json:"settings"`
	Verbose                    bool                            `json:"verbose"`
	MainLoopModel              ModelSetting                    `json:"mainLoopModel"`
	MainLoopModelForSession    ModelSetting                    `json:"mainLoopModelForSession"`
	StatusLineText             *string                         `json:"statusLineText"`
	ExpandedView               ExpandedView                    `json:"expandedView"`
	IsBriefOnly                bool                            `json:"isBriefOnly"`
	ShowTeammateMessagePreview *bool                           `json:"showTeammateMessagePreview,omitempty"`
	SelectedIPAgentIndex       int                             `json:"selectedIPAgentIndex"`
	CoordinatorTaskIndex       int                             `json:"coordinatorTaskIndex"`
	ViewSelectionMode          ViewSelectionMode               `json:"viewSelectionMode"`
	FooterSelection            *FooterItem                     `json:"footerSelection"`
	ToolPermissionContext      types.ToolPermissionContextData `json:"toolPermissionContext"`
	SpinnerTip                 *string                         `json:"spinnerTip,omitempty"`
	Agent                      *string                         `json:"agent"`
	KairosEnabled              bool                            `json:"kairosEnabled"`
	RemoteSessionURL           *string                         `json:"remoteSessionUrl"`
	RemoteConnectionStatus     RemoteConnectionStatus          `json:"remoteConnectionStatus"`
	RemoteBackgroundTaskCount  int                             `json:"remoteBackgroundTaskCount"`
	ReplBridgeEnabled          bool                            `json:"replBridgeEnabled"`
	ReplBridgeExplicit         bool                            `json:"replBridgeExplicit"`
	ReplBridgeOutboundOnly     bool                            `json:"replBridgeOutboundOnly"`
	ReplBridgeConnected        bool                            `json:"replBridgeConnected"`
	ReplBridgeSessionActive    bool                            `json:"replBridgeSessionActive"`
	ReplBridgeReconnecting     bool                            `json:"replBridgeReconnecting"`
	ReplBridgeConnectURL       *string                         `json:"replBridgeConnectUrl"`
	ReplBridgeSessionURL       *string                         `json:"replBridgeSessionUrl"`
	ReplBridgeEnvironmentID    *string                         `json:"replBridgeEnvironmentId"`
	ReplBridgeSessionID        *string                         `json:"replBridgeSessionId"`
	ReplBridgeError            *string                         `json:"replBridgeError"`
	ReplBridgeInitialName      *string                         `json:"replBridgeInitialName"`
	ShowRemoteCallout          bool                            `json:"showRemoteCallout"`

	// Tasks maps taskId → opaque TaskState (TS includes function fields).
	Tasks map[string]json.RawMessage `json:"tasks"`
	// AgentNameRegistry: TS Map<name, AgentId>.
	AgentNameRegistry  map[string]string `json:"agentNameRegistry"`
	ForegroundedTaskID *string           `json:"foregroundedTaskId,omitempty"`
	ViewingAgentTaskID *string           `json:"viewingAgentTaskId,omitempty"`
	CompanionReaction  *string           `json:"companionReaction,omitempty"`
	CompanionPetAt     *int64            `json:"companionPetAt,omitempty"`

	Mcp                        McpState                    `json:"mcp"`
	Plugins                    PluginsState                `json:"plugins"`
	AgentDefinitions           AgentDefinitionsResult      `json:"agentDefinitions"`
	FileHistory                FileHistoryState            `json:"fileHistory"`
	Attribution                AttributionState            `json:"attribution"`
	Todos                      TodosMap                    `json:"todos"`
	RemoteAgentTaskSuggestions []RemoteAgentTaskSuggestion `json:"remoteAgentTaskSuggestions"`
	Notifications              NotificationsState          `json:"notifications"`
	Elicitation                ElicitationState            `json:"elicitation"`
	ThinkingEnabled            *bool                       `json:"thinkingEnabled"`
	PromptSuggestionEnabled    bool                        `json:"promptSuggestionEnabled"`
	// SessionHooks: TS Map<sessionId, SessionStore> — JSON-safe snapshot only.
	SessionHooks SessionHooksState `json:"sessionHooks"`

	TungstenActiveSession    *TungstenActiveSession `json:"tungstenActiveSession,omitempty"`
	TungstenLastCapturedTime *int64                 `json:"tungstenLastCapturedTime,omitempty"`
	TungstenLastCommand      *TungstenLastCommand   `json:"tungstenLastCommand,omitempty"`
	TungstenPanelVisible     *bool                  `json:"tungstenPanelVisible,omitempty"`
	TungstenPanelAutoHidden  *bool                  `json:"tungstenPanelAutoHidden,omitempty"`
	BagelActive              *bool                  `json:"bagelActive,omitempty"`
	BagelURL                 *string                `json:"bagelUrl,omitempty"`
	BagelPanelVisible        *bool                  `json:"bagelPanelVisible,omitempty"`

	ComputerUseMcpState *ComputerUseMcpState `json:"computerUseMcpState,omitempty"`
	// ReplContext: TS also holds vm.Context + console — [ReplContextState] is the JSON-safe subset.
	ReplContext *ReplContextState `json:"replContext,omitempty"`
	TeamContext *TeamContextState `json:"teamContext,omitempty"`

	StandaloneAgentContext   *StandaloneAgentContext       `json:"standaloneAgentContext,omitempty"`
	Inbox                    InboxState                    `json:"inbox"`
	WorkerSandboxPermissions WorkerSandboxPermissionsState `json:"workerSandboxPermissions"`
	PendingWorkerRequest     *PendingWorkerRequest         `json:"pendingWorkerRequest"`
	PendingSandboxRequest    *PendingSandboxRequest        `json:"pendingSandboxRequest"`
	PromptSuggestion         PromptSuggestionState         `json:"promptSuggestion"`
	// Speculation: TS active branch has abort/refs/functions — [SpeculationState] carries JSON-safe fields only.
	Speculation                   SpeculationState         `json:"speculation"`
	SpeculationSessionTimeSavedMs int64                    `json:"speculationSessionTimeSavedMs"`
	SkillImprovement              SkillImprovementState    `json:"skillImprovement"`
	AuthVersion                   int                      `json:"authVersion"`
	InitialMessage                *InitialMessage          `json:"initialMessage"`
	PendingPlanVerification       *PendingPlanVerification `json:"pendingPlanVerification,omitempty"`
	DenialTracking                *DenialTrackingState     `json:"denialTracking,omitempty"`
	// ActiveOverlays: TS ReadonlySet<string> — list for JSON.
	ActiveOverlays []string     `json:"activeOverlays"`
	FastMode       *bool        `json:"fastMode,omitempty"`
	AdvisorModel   *string      `json:"advisorModel,omitempty"`
	EffortValue    *EffortValue `json:"effortValue,omitempty"`
}
