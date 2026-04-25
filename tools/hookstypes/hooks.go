// Package hookstypes provides hooks types shared between tools and agents/builtin
// packages, mirroring TS schemas/hooks.ts + entrypoints/agentSdkTypes.js HookEvent values.
package hookstypes

// HookEvent mirrors TS HOOK_EVENTS union type.
type HookEvent string

const (
	PreToolUse         HookEvent = "PreToolUse"
	PostToolUse        HookEvent = "PostToolUse"
	PostToolUseFailure HookEvent = "PostToolUseFailure"
	Notification       HookEvent = "Notification"
	UserPromptSubmit   HookEvent = "UserPromptSubmit"
	SessionStart       HookEvent = "SessionStart"
	SessionEnd         HookEvent = "SessionEnd"
	Stop               HookEvent = "Stop"
	StopFailure        HookEvent = "StopFailure"
	SubagentStart      HookEvent = "SubagentStart"
	SubagentStop       HookEvent = "SubagentStop"
	PreCompact         HookEvent = "PreCompact"
	PostCompact        HookEvent = "PostCompact"
	PermissionRequest  HookEvent = "PermissionRequest"
	PermissionDenied   HookEvent = "PermissionDenied"
	Setup              HookEvent = "Setup"
	TeammateIdle       HookEvent = "TeammateIdle"
	TaskCreated        HookEvent = "TaskCreated"
	TaskCompleted      HookEvent = "TaskCompleted"
	Elicitation        HookEvent = "Elicitation"
	ElicitationResult  HookEvent = "ElicitationResult"
	ConfigChange       HookEvent = "ConfigChange"
	WorktreeCreate     HookEvent = "WorktreeCreate"
	WorktreeRemove     HookEvent = "WorktreeRemove"
	InstructionsLoaded HookEvent = "InstructionsLoaded"
	CwdChanged         HookEvent = "CwdChanged"
	FileChanged        HookEvent = "FileChanged"
)

// AllHookEvents lists all valid hook events in order (mirrors TS HOOK_EVENS array).
var AllHookEvents = []HookEvent{
	PreToolUse,
	PostToolUse,
	PostToolUseFailure,
	Notification,
	UserPromptSubmit,
	SessionStart,
	SessionEnd,
	Stop,
	StopFailure,
	SubagentStart,
	SubagentStop,
	PreCompact,
	PostCompact,
	PermissionRequest,
	PermissionDenied,
	Setup,
	TeammateIdle,
	TaskCreated,
	TaskCompleted,
	Elicitation,
	ElicitationResult,
	ConfigChange,
	WorktreeCreate,
	WorktreeRemove,
	InstructionsLoaded,
	CwdChanged,
	FileChanged,
}

// KnownHookEvent reports whether the given string is a valid [HookEvent].
func KnownHookEvent(s string) bool {
	for _, e := range AllHookEvents {
		if string(e) == s {
			return true
		}
	}
	return false
}

// HookCommand mirrors TS HookCommand (discriminated union: command / prompt / http / agent).
type HookCommand struct {
	Type          string            `json:"type"`
	Command       string            `json:"command,omitempty"`
	Prompt        string            `json:"prompt,omitempty"`
	URL           string            `json:"url,omitempty"`
	If            string            `json:"if,omitempty"`
	Shell         string            `json:"shell,omitempty"`
	Timeout       int               `json:"timeout,omitempty"`
	StatusMessage string            `json:"statusMessage,omitempty"`
	Once          bool              `json:"once,omitempty"`
	Async         bool              `json:"async,omitempty"`
	AsyncRewake   bool              `json:"asyncRewake,omitempty"`
	Model         string            `json:"model,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	AllowedEnvVars []string         `json:"allowedEnvVars,omitempty"`
}

// HookMatcher mirrors TS HookMatcher.
type HookMatcher struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []HookCommand `json:"hooks"`
}

// HooksSettings mirrors TS HooksSettings = Partial<Record<HookEvent, HookMatcher[]>>.
type HooksSettings map[HookEvent][]HookMatcher
