package appstate

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"goc/types"
)

func boolPtr(b bool) *bool { return &b }

// defaultThinkingEnabled approximates src/utils/thinking.ts shouldEnableThinkingByDefault (no settings merge).
func defaultThinkingEnabled() *bool {
	if v := strings.TrimSpace(os.Getenv("MAX_THINKING_TOKENS")); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return boolPtr(n > 0)
		}
	}
	// TS: default true unless settings.alwaysThinkingEnabled === false (requires settings load).
	return boolPtr(true)
}

// defaultPromptSuggestionEnabled approximates src/services/PromptSuggestion/promptSuggestion.ts (env only in Go).
func defaultPromptSuggestionEnabled() bool {
	v := strings.TrimSpace(os.Getenv("CLAUDE_CODE_ENABLE_PROMPT_SUGGESTION"))
	if v == "" {
		return false
	}
	vl := strings.ToLower(v)
	if vl == "0" || vl == "false" || vl == "no" || vl == "off" {
		return false
	}
	return true
}

// DefaultAppState returns a new state aligned with src/state/AppStateStore.ts getDefaultAppState.
// Settings remain opaque ([EmptySettingsJSON]) until full SettingsJson is ported; [AttributionState] is structured.
func DefaultAppState() AppState {
	tpc := types.EmptyToolPermissionContextData()
	return AppState{
		Settings:                   EmptySettingsJSON,
		Verbose:                    false,
		MainLoopModel:              nil,
		MainLoopModelForSession:    nil,
		StatusLineText:             nil,
		ExpandedView:               ExpandedNone,
		IsBriefOnly:                false,
		ShowTeammateMessagePreview: boolPtr(false),
		SelectedIPAgentIndex:       -1,
		CoordinatorTaskIndex:       -1,
		ViewSelectionMode:          ViewSelNone,
		FooterSelection:            nil,
		ToolPermissionContext:      tpc,
		SpinnerTip:                 nil,
		Agent:                      nil,
		KairosEnabled:              false,
		RemoteSessionURL:           nil,
		RemoteConnectionStatus:     RemoteConnecting,
		RemoteBackgroundTaskCount:  0,
		ReplBridgeEnabled:          false,
		ReplBridgeExplicit:         false,
		ReplBridgeOutboundOnly:     false,
		ReplBridgeConnected:        false,
		ReplBridgeSessionActive:    false,
		ReplBridgeReconnecting:     false,
		ReplBridgeConnectURL:       nil,
		ReplBridgeSessionURL:       nil,
		ReplBridgeEnvironmentID:    nil,
		ReplBridgeSessionID:        nil,
		ReplBridgeError:            nil,
		ReplBridgeInitialName:      nil,
		ShowRemoteCallout:          false,

		Tasks:              make(map[string]json.RawMessage),
		AgentNameRegistry:  make(map[string]string),
		ForegroundedTaskID: nil,
		ViewingAgentTaskID: nil,
		CompanionReaction:  nil,
		CompanionPetAt:     nil,

		Mcp: EmptyMcpState(),
		Plugins: PluginsState{
			Enabled:  []LoadedPluginData{},
			Disabled: []LoadedPluginData{},
			Commands: []types.Command{},
			Errors:   []PluginErrorSnapshot{},
			InstallationStatus: PluginsInstallationStatus{
				Marketplaces: []PluginMarketplaceInstall{},
				Plugins:      []PluginInstall{},
			},
			NeedsRefresh: false,
		},
		AgentDefinitions:              EmptyAgentDefinitionsResult(),
		FileHistory:                   EmptyFileHistoryState(),
		Attribution:                   EmptyAttributionState(),
		Todos:                         TodosMap{},
		RemoteAgentTaskSuggestions:    []RemoteAgentTaskSuggestion{},
		Notifications:                 NotificationsState{Current: nil, Queue: []NotificationSnapshot{}},
		Elicitation:                   ElicitationState{Queue: []ElicitationRequestEventSnapshot{}},
		ThinkingEnabled:               defaultThinkingEnabled(),
		PromptSuggestionEnabled:       defaultPromptSuggestionEnabled(),
		SessionHooks:                  SessionHooksState{},
		TungstenActiveSession:         nil,
		TungstenLastCapturedTime:      nil,
		TungstenLastCommand:           nil,
		TungstenPanelVisible:          nil,
		TungstenPanelAutoHidden:       nil,
		BagelActive:                   nil,
		BagelURL:                      nil,
		BagelPanelVisible:             nil,
		ComputerUseMcpState:           nil,
		ReplContext:                   nil,
		TeamContext:                   nil,
		StandaloneAgentContext:        nil,
		Inbox:                         InboxState{Messages: []InboxMessage{}},
		WorkerSandboxPermissions:      WorkerSandboxPermissionsState{Queue: []WorkerSandboxQueueItem{}, SelectedIndex: 0},
		PendingWorkerRequest:          nil,
		PendingSandboxRequest:         nil,
		PromptSuggestion:              PromptSuggestionState{Text: nil, PromptID: nil, ShownAt: 0, AcceptedAt: 0, GenerationRequestID: nil},
		Speculation:                   IdleSpeculationState(),
		SpeculationSessionTimeSavedMs: 0,
		SkillImprovement:              SkillImprovementState{Suggestion: nil},
		AuthVersion:                   0,
		InitialMessage:                nil,
		PendingPlanVerification:       nil,
		DenialTracking:                nil,
		ActiveOverlays:                []string{},
		FastMode:                      boolPtr(false),
		AdvisorModel:                  nil,
		EffortValue:                   nil,
	}
}
