// Package appstate mirrors the data shape of src/state/AppStateStore.ts AppState (DeepImmutable core + mutable edges).
// Callbacks, REPL vm.Context, and other non-JSON TS fields are omitted or held as json.RawMessage for IPC snapshots.
// Cross-check: [AppState] field names and json tags align with TS property names unless noted.
//
// Related TS exports also mirrored here: [CompletionBoundary], [SpeculationResult], [SpeculationState], [IdleSpeculationState],
// [ModelSetting], [EffortValue], [EffortLevel], [InitialMessage], [AllowedPrompt],
// [DenialTrackingState], [AttributionState], [EmptyAttributionState], [NormalizeAppState], [EmptySettingsJSON],
// [AgentDefinitionsResult], [NormalizeAgentDefinitionData], [FileHistoryState], [SettingsCommon], [ParseSettingsCommon],
// [TodosMap], [TodoList], [McpState], [MCPServerConnectionSnapshot], [MCPServerResourceSnapshot], [SessionHooksState], [SessionStoreSnapshot], [SessionHookEntrySnapshot],
// [NotificationSnapshot], [ElicitationRequestEventSnapshot], [LoadedPluginData], [PluginErrorSnapshot], [ComputerUseMcpState],
// [ReplContextState], [TeamContextState], and [Store].
package appstate
