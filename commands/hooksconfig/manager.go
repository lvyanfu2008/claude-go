// Package hooksconfig provides hooks configuration grouping and metadata
// for the /hooks interactive menu, porting TS hooksConfigManager.ts logic.
package hooksconfig

import (
	"sort"
	"strings"

	"goc/tools/hookstypes"
)

// HookSource mirrors TS HookSource.
type HookSource string

const (
	SourceUserSettings    HookSource = "userSettings"
	SourceProjectSettings HookSource = "projectSettings"
	SourceLocalSettings   HookSource = "localSettings"
	SourcePluginHook      HookSource = "pluginHook"
	SourceSessionHook     HookSource = "sessionHook"
	SourceBuiltinHook     HookSource = "builtinHook"
	SourcePolicySettings  HookSource = "policySettings"
)

// sourcePriority maps sources to display priority (lower = higher priority).
var sourcePriority = map[HookSource]int{
	SourceLocalSettings:   0,
	SourceProjectSettings: 1,
	SourceUserSettings:    2,
	SourcePluginHook:      999,
	SourceBuiltinHook:     999,
	SourceSessionHook:     998,
	SourcePolicySettings:  997,
}

// MatcherMetadata describes how an event's hooks are matched.
type MatcherMetadata struct {
	FieldToMatch string
	Values       []string
}

// HookEventMetadata mirrors TS HookEventMetadata.
type HookEventMetadata struct {
	Summary         string
	Description     string
	MatcherMetadata *MatcherMetadata
}

// IndividualHookConfig mirrors TS IndividualHookConfig.
type IndividualHookConfig struct {
	Event      hookstypes.HookEvent
	Config     hookstypes.HookCommand
	Matcher    string
	Source     HookSource
	PluginName string
}

// GroupedHooks is map[event]map[matcher][]IndividualHookConfig.
type GroupedHooks map[hookstypes.HookEvent]map[string][]IndividualHookConfig

// description strings ported from TS hooksConfigManager.ts
var eventDescriptions = map[hookstypes.HookEvent]string{
	hookstypes.PreToolUse:         "Before tool execution. The hook receives the tool name and input, and can block or modify the call. Exit code 2 blocks the tool and continues the turn. Exit code 0 returns additional context (from stdout). Any other exit code blocks and stops the conversation.",
	hookstypes.PostToolUse:        "After tool execution. The hook receives the tool name, input, and response. It can inject additional context into the conversation.",
	hookstypes.PostToolUseFailure: "After tool execution fails. The hook receives the tool name, input, and error information.",
	hookstypes.PermissionDenied:   "After the auto mode classifier denies a tool call. The hook receives the tool name and the reason for denial.",
	hookstypes.Notification:       "When notifications are sent (permission prompts, idle prompts, auth successes, elicitation dialogs). The hook receives the notification type and message.",
	hookstypes.UserPromptSubmit:   "When the user submits a prompt. The hook receives the user's input text. It can inject additional context or modify the input.",
	hookstypes.SessionStart:       "When a new session is started. The hook receives the source (startup, resume, clear, compact). Can be used for initialization.",
	hookstypes.Stop:               "Right before Claude concludes its response. The hook receives the stop reason. Exit code 2 prevents the stop and continues the conversation.",
	hookstypes.StopFailure:        "When the turn ends due to an API error. The hook receives the error type (rate_limit, authentication_failed, billing_error, etc.).",
	hookstypes.SubagentStart:      "When a subagent (Agent tool call) is started. The hook receives the agent type and the prompt.",
	hookstypes.SubagentStop:       "Right before a subagent concludes its response. The hook receives the agent type and the final message.",
	hookstypes.PreCompact:         "Before conversation compaction. The hook receives the trigger (manual or auto). Exit code 2 blocks the compaction.",
	hookstypes.PostCompact:        "After conversation compaction. The hook receives the trigger and the compaction result.",
	hookstypes.SessionEnd:         "When a session is ending. The hook receives the reason (clear, logout, prompt_input_exit, other).",
	hookstypes.PermissionRequest:  "When a permission dialog is displayed to the user. The hook receives the tool name and permission prompt.",
	hookstypes.Setup:              "Repo setup hooks for initialization and maintenance. The hook receives the trigger (init or maintenance).",
	hookstypes.TeammateIdle:       "When a teammate is about to go idle.",
	hookstypes.TaskCreated:        "When a task is being created. The hook receives the task subject and description.",
	hookstypes.TaskCompleted:      "When a task is being marked as completed. The hook receives the task subject.",
	hookstypes.Elicitation:        "When an MCP server requests user input (elicitation dialog). The hook receives the MCP server name and the elicitation request.",
	hookstypes.ElicitationResult:  "After a user responds to an MCP elicitation. The hook receives the MCP server name and the user's response.",
	hookstypes.ConfigChange:       "When configuration files change during a session. The hook receives the source (user_settings, project_settings, local_settings, policy_settings, skills).",
	hookstypes.WorktreeCreate:     "When a worktree is created for isolation.",
	hookstypes.WorktreeRemove:     "When a previously created worktree is removed.",
	hookstypes.InstructionsLoaded: "When an instruction file (CLAUDE.md or rule) is loaded. The hook receives the load reason (session_start, nested_traversal, path_glob_match, include, compact).",
	hookstypes.CwdChanged:         "After the working directory changes.",
	hookstypes.FileChanged:        "When a watched file changes. The hook receives the file path.",
}

// matcherValueGroups maps events to their matcher value lists (static values).
var matcherValueGroups = map[hookstypes.HookEvent][]string{
	hookstypes.Notification:       {"permission_prompt", "idle_prompt", "auth_success", "elicitation_dialog", "elicitation_complete", "elicitation_response"},
	hookstypes.SessionStart:       {"startup", "resume", "clear", "compact"},
	hookstypes.StopFailure:        {"rate_limit", "authentication_failed", "billing_error", "invalid_request", "server_error", "max_output_tokens", "unknown"},
	hookstypes.PreCompact:         {"manual", "auto"},
	hookstypes.PostCompact:        {"manual", "auto"},
	hookstypes.SessionEnd:         {"clear", "logout", "prompt_input_exit", "other"},
	hookstypes.Setup:              {"init", "maintenance"},
	hookstypes.ConfigChange:       {"user_settings", "project_settings", "local_settings", "policy_settings", "skills"},
	hookstypes.InstructionsLoaded: {"session_start", "nested_traversal", "path_glob_match", "include", "compact"},
}

// eventsWithoutMatcher are events that don't have matcherMetadata in TS.
var eventsWithoutMatcher = map[hookstypes.HookEvent]bool{
	hookstypes.UserPromptSubmit: true,
	hookstypes.Stop:             true,
	hookstypes.TeammateIdle:     true,
	hookstypes.TaskCreated:      true,
	hookstypes.TaskCompleted:    true,
	hookstypes.WorktreeCreate:   true,
	hookstypes.WorktreeRemove:   true,
	hookstypes.CwdChanged:       true,
	hookstypes.FileChanged:      true,
}

// eventsWithDynamicValues are events whose matcher values are populated at runtime.
var eventsWithDynamicValues = map[hookstypes.HookEvent]bool{
	hookstypes.SubagentStart:    true,
	hookstypes.SubagentStop:     true,
	hookstypes.Elicitation:      true,
	hookstypes.ElicitationResult: true,
}

// GetHookEventMetadata returns metadata for all hook events.
// toolNames is used for events whose matcher values are tool names.
func GetHookEventMetadata(toolNames []string) map[hookstypes.HookEvent]HookEventMetadata {
	result := make(map[hookstypes.HookEvent]HookEventMetadata)

	for _, ev := range hookstypes.AllHookEvents {
		summary := eventSummary(ev)
		desc := eventDescriptions[ev]
		if desc == "" {
			desc = summary
		}

		meta := HookEventMetadata{
			Summary:     summary,
			Description: desc,
		}

		if eventsWithoutMatcher[ev] {
			// No matcher
		} else if eventsWithDynamicValues[ev] {
			meta.MatcherMetadata = &MatcherMetadata{
				FieldToMatch: matcherField(ev),
				Values:       []string{},
			}
		} else if vals, ok := matcherValueGroups[ev]; ok {
			meta.MatcherMetadata = &MatcherMetadata{
				FieldToMatch: matcherField(ev),
				Values:       vals,
			}
		} else {
			// Tool-name based matchers
			meta.MatcherMetadata = &MatcherMetadata{
				FieldToMatch: matcherField(ev),
				Values:       toolNames,
			}
		}

		result[ev] = meta
	}

	return result
}

func eventSummary(ev hookstypes.HookEvent) string {
	switch ev {
	case hookstypes.PreToolUse:
		return "Before tool execution"
	case hookstypes.PostToolUse:
		return "After tool execution"
	case hookstypes.PostToolUseFailure:
		return "After tool execution fails"
	case hookstypes.PermissionDenied:
		return "After auto mode denies a tool call"
	case hookstypes.Notification:
		return "When notifications are sent"
	case hookstypes.UserPromptSubmit:
		return "When user submits a prompt"
	case hookstypes.SessionStart:
		return "When a new session starts"
	case hookstypes.Stop:
		return "Before Claude concludes its response"
	case hookstypes.StopFailure:
		return "When turn ends due to API error"
	case hookstypes.SubagentStart:
		return "When a subagent starts"
	case hookstypes.SubagentStop:
		return "Before a subagent concludes"
	case hookstypes.PreCompact:
		return "Before conversation compaction"
	case hookstypes.PostCompact:
		return "After conversation compaction"
	case hookstypes.SessionEnd:
		return "When a session is ending"
	case hookstypes.PermissionRequest:
		return "When a permission dialog is displayed"
	case hookstypes.Setup:
		return "Repo setup hooks"
	case hookstypes.TeammateIdle:
		return "When a teammate goes idle"
	case hookstypes.TaskCreated:
		return "When a task is created"
	case hookstypes.TaskCompleted:
		return "When a task is completed"
	case hookstypes.Elicitation:
		return "When MCP server requests input"
	case hookstypes.ElicitationResult:
		return "After user responds to MCP elicitation"
	case hookstypes.ConfigChange:
		return "When config files change"
	case hookstypes.WorktreeCreate:
		return "When a worktree is created"
	case hookstypes.WorktreeRemove:
		return "When a worktree is removed"
	case hookstypes.InstructionsLoaded:
		return "When instruction file is loaded"
	case hookstypes.CwdChanged:
		return "After working directory changes"
	case hookstypes.FileChanged:
		return "When a watched file changes"
	default:
		return string(ev)
	}
}

func matcherField(ev hookstypes.HookEvent) string {
	switch ev {
	case hookstypes.Notification:
		return "notification_type"
	case hookstypes.SessionStart:
		return "source"
	case hookstypes.StopFailure:
		return "error"
	case hookstypes.SubagentStart, hookstypes.SubagentStop:
		return "agent_type"
	case hookstypes.PreCompact, hookstypes.PostCompact:
		return "trigger"
	case hookstypes.SessionEnd:
		return "reason"
	case hookstypes.PermissionRequest, hookstypes.PermissionDenied:
		return "tool_name"
	case hookstypes.Setup:
		return "trigger"
	case hookstypes.Elicitation, hookstypes.ElicitationResult:
		return "mcp_server_name"
	case hookstypes.ConfigChange:
		return "source"
	case hookstypes.InstructionsLoaded:
		return "load_reason"
	default:
		return "tool_name"
	}
}

// SourceDisplayString returns the full description for a hook source (TS hookSourceDescriptionDisplayString).
func SourceDisplayString(source HookSource) string {
	switch source {
	case SourceUserSettings:
		return "User settings (~/.harness/settings.go.json)"
	case SourceProjectSettings:
		return "Project settings (.harness/settings.go.json)"
	case SourceLocalSettings:
		return "Local settings (.harness/settings.local.json)"
	case SourcePluginHook:
		return "Plugin hooks (~/.claude/plugins/*/hooks/hooks.json)"
	case SourceSessionHook:
		return "Session hooks (in-memory, temporary)"
	case SourceBuiltinHook:
		return "Built-in hooks (registered internally by Claude Code)"
	case SourcePolicySettings:
		return "Policy settings (managed)"
	default:
		return string(source)
	}
}

// SourceHeaderString returns the short header for a hook source (TS hookSourceHeaderDisplayString).
func SourceHeaderString(source HookSource) string {
	switch source {
	case SourceUserSettings:
		return "User Settings"
	case SourceProjectSettings:
		return "Project Settings"
	case SourceLocalSettings:
		return "Local Settings"
	case SourcePluginHook:
		return "Plugin Hooks"
	case SourceSessionHook:
		return "Session Hooks"
	case SourceBuiltinHook:
		return "Built-in Hooks"
	case SourcePolicySettings:
		return "Policy Settings"
	default:
		return string(source)
	}
}

// SourceInlineString returns the inline label for a hook source (TS hookSourceInlineDisplayString).
func SourceInlineString(source HookSource) string {
	switch source {
	case SourceUserSettings:
		return "User"
	case SourceProjectSettings:
		return "Project"
	case SourceLocalSettings:
		return "Local"
	case SourcePluginHook:
		return "Plugin"
	case SourceSessionHook:
		return "Session"
	case SourceBuiltinHook:
		return "Built-in"
	case SourcePolicySettings:
		return "Policy"
	default:
		return string(source)
	}
}

// HookTypeLabel returns a short label for the hook command type.
func HookTypeLabel(config hookstypes.HookCommand) string {
	switch config.Type {
	case "command":
		return "command"
	case "prompt":
		return "prompt"
	case "agent":
		return "agent"
	case "http":
		return "http"
	default:
		return config.Type
	}
}

// HookDisplayText returns the main content text for a hook (TS getHookDisplayText).
func HookDisplayText(config hookstypes.HookCommand) string {
	if config.StatusMessage != "" {
		return config.StatusMessage
	}
	switch config.Type {
	case "command":
		return config.Command
	case "prompt", "agent":
		return config.Prompt
	case "http":
		return config.URL
	default:
		return ""
	}
}

// GroupHooksByEventAndMatcher groups hooks by event then by matcher.
func GroupHooksByEventAndMatcher(hooks []IndividualHookConfig, toolNames []string) GroupedHooks {
	grouped := make(GroupedHooks)

	// Initialize empty groups for all events
	meta := GetHookEventMetadata(toolNames)
	for _, ev := range hookstypes.AllHookEvents {
		grouped[ev] = make(map[string][]IndividualHookConfig)
	}

	for _, hook := range hooks {
		ev := hook.Event
		if _, ok := grouped[ev]; !ok {
			grouped[ev] = make(map[string][]IndividualHookConfig)
		}

		matcherKey := ""
		if _, ok := meta[ev]; ok && meta[ev].MatcherMetadata != nil {
			matcherKey = hook.Matcher
		}
		// For events without matcherMetadata, matcherKey stays ""

		grouped[ev][matcherKey] = append(grouped[ev][matcherKey], hook)
	}

	return grouped
}

// SortedMatchersForEvent returns matcher keys sorted by source priority then alphabetically.
func SortedMatchersForEvent(grouped GroupedHooks, ev hookstypes.HookEvent) []string {
	matchers, ok := grouped[ev]
	if !ok || len(matchers) == 0 {
		return nil
	}

	type matcherInfo struct {
		key      string
		priority int
	}

	var infos []matcherInfo
	for key, hooks := range matchers {
		priority := 999
		for _, h := range hooks {
			if p, ok := sourcePriority[h.Source]; ok && p < priority {
				priority = p
			}
		}
		infos = append(infos, matcherInfo{key: key, priority: priority})
	}

	sort.Slice(infos, func(i, j int) bool {
		if infos[i].priority != infos[j].priority {
			return infos[i].priority < infos[j].priority
		}
		return strings.ToLower(infos[i].key) < strings.ToLower(infos[j].key)
	})

	result := make([]string, len(infos))
	for i, info := range infos {
		result[i] = info.key
	}
	return result
}

// MatcherSourceSet returns the set of sources for hooks matching a given matcher key.
func MatcherSourceSet(matchers map[string][]IndividualHookConfig, matcherKey string) map[HookSource]bool {
	sources := make(map[HookSource]bool)
	for _, hook := range matchers[matcherKey] {
		sources[hook.Source] = true
	}
	return sources
}

// TotalHooksCount returns the total number of hooks across all events.
func TotalHooksCount(grouped GroupedHooks) int {
	count := 0
	for _, matchers := range grouped {
		for _, hooks := range matchers {
			count += len(hooks)
		}
	}
	return count
}

// EventHasMatcher reports whether an event supports matchers.
func EventHasMatcher(toolNames []string, ev hookstypes.HookEvent) bool {
	meta := GetHookEventMetadata(toolNames)
	if m, ok := meta[ev]; ok {
		return m.MatcherMetadata != nil
	}
	return false
}

// HookContentLabel returns the content field label for a hook type.
func HookContentLabel(config hookstypes.HookCommand) string {
	switch config.Type {
	case "command":
		return "Command"
	case "prompt", "agent":
		return "Prompt"
	case "http":
		return "URL"
	default:
		return ""
	}
}

// HookContentValue returns the content field value for a hook type.
func HookContentValue(config hookstypes.HookCommand) string {
	switch config.Type {
	case "command":
		return config.Command
	case "prompt", "agent":
		return config.Prompt
	case "http":
		return config.URL
	default:
		return ""
	}
}
