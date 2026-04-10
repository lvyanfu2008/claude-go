// Mirrors src/types/command.ts Command / CommandBase / PromptCommand (data fields only).
// Callable fields (getPromptForCommand, load, call) are runtime-only in TS and omitted here.
package types

import (
	"encoding/json"

	"goc/utils"
)

// CommandAvailability mirrors src/types/command.ts CommandAvailability.
type CommandAvailability string

const (
	CommandAvailabilityClaudeAI CommandAvailability = "claude-ai"
	CommandAvailabilityConsole  CommandAvailability = "console"
)

// CommandBase mirrors src/types/command.ts CommandBase.
type CommandBase struct {
	Availability                []CommandAvailability `json:"availability,omitempty"`
	Description                 string                `json:"description"`
	HasUserSpecifiedDescription *bool                 `json:"hasUserSpecifiedDescription,omitempty"`
	IsHidden                    *bool                 `json:"isHidden,omitempty"`
	Name                        string                `json:"name"`
	Aliases                     []string              `json:"aliases,omitempty"`
	IsMcp                       *bool                 `json:"isMcp,omitempty"`
	ArgumentHint                *string               `json:"argumentHint,omitempty"`
	WhenToUse                   *string               `json:"whenToUse,omitempty"`
	Version                     *string               `json:"version,omitempty"`
	DisableModelInvocation      *bool                 `json:"disableModelInvocation,omitempty"`
	UserInvocable               *bool                 `json:"userInvocable,omitempty"`
	LoadedFrom                  *string               `json:"loadedFrom,omitempty"`
	Kind                        *string               `json:"kind,omitempty"`
	Immediate                   *bool                 `json:"immediate,omitempty"`
	IsSensitive                 *bool                 `json:"isSensitive,omitempty"`
}

// Command mirrors src/types/command.ts Command (discriminated by type).
// Fields are grouped by variant; unset fields should be omitted in JSON.
type Command struct {
	CommandBase
	Type string `json:"type"` // prompt | local | local-jsx

	// --- prompt ---
	ProgressMessage       *string            `json:"progressMessage,omitempty"`
	ContentLength         *int               `json:"contentLength,omitempty"`
	ArgNames              []string           `json:"argNames,omitempty"`
	AllowedTools          []string           `json:"allowedTools,omitempty"`
	Model                 *string            `json:"model,omitempty"`
	Source                *string            `json:"source,omitempty"`
	PluginInfo            json.RawMessage    `json:"pluginInfo,omitempty"`
	DisableNonInteractive *bool              `json:"disableNonInteractive,omitempty"`
	Hooks                 json.RawMessage    `json:"hooks,omitempty"`
	SkillRoot             *string            `json:"skillRoot,omitempty"`
	Context               *string            `json:"context,omitempty"` // inline | fork
	Agent                 *string            `json:"agent,omitempty"`
	Effort                *utils.EffortValue `json:"effort,omitempty"`
	Paths                 []string           `json:"paths,omitempty"`

	// --- local ---
	SupportsNonInteractive *bool `json:"supportsNonInteractive,omitempty"`

	// local-jsx has no extra data fields beyond CommandBase + type
}

// GetCommandName mirrors src/types/command.ts getCommandName (data model has no userFacingName callback — use Name).
func GetCommandName(cmd Command) string {
	return cmd.Name
}
