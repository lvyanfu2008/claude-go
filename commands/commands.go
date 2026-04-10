// Package commands mirrors helpers from src/commands.ts: find/filter/availability, [GetSkills]/[GetSkillsAsync],
// and [LoadAllCommands]/[LoadAllCommandsAsync] for the same concat order as loadAllCommands (bundled → builtin-plugin → skill dirs → workflow → plugins → COMMANDS).
// getCommands-level filtering: use [FilterGetCommands] + [GetCommandsAuth] after [LoadAllCommands] (see get_commands.go).
// TS parity: load-all-commands-ts-parity.md; full roadmap: docs/plans/goc-load-all-commands.md; git stop / worktree: git_boundary.go.
package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"goc/types"
)

// RemoteSafeCommandNames matches REMOTE_SAFE_COMMANDS in src/commands.ts (command name keys).
var RemoteSafeCommandNames = []string{
	"session", "exit", "clear", "help", "theme", "color", "vim", "cost", "usage",
	"copy", "btw", "feedback", "plan", "keybindings", "statusline", "stickers", "mobile",
}

// BridgeSafeLocalCommandNames matches BRIDGE_SAFE_COMMANDS for type === "local" in src/commands.ts.
var BridgeSafeLocalCommandNames = []string{
	"compact", "clear", "cost", "summary", "releaseNotes", "files",
}

func remoteSafeSet() map[string]struct{} {
	m := make(map[string]struct{}, len(RemoteSafeCommandNames))
	for _, n := range RemoteSafeCommandNames {
		m[n] = struct{}{}
	}
	return m
}

func bridgeSafeLocalSet() map[string]struct{} {
	m := make(map[string]struct{}, len(BridgeSafeLocalCommandNames))
	for _, n := range BridgeSafeLocalCommandNames {
		m[n] = struct{}{}
	}
	return m
}

// MeetsAvailabilityRequirement mirrors src/commands.ts meetsAvailabilityRequirement.
// Pass auth/provider flags from the host (TS derives them from isClaudeAISubscriber, isUsing3PServices, isFirstPartyAnthropicBaseUrl).
func MeetsAvailabilityRequirement(cmd types.Command, isClaudeAISubscriber, isUsing3PServices, isFirstPartyAnthropicBaseURL bool) bool {
	if len(cmd.Availability) == 0 {
		return true
	}
	for _, a := range cmd.Availability {
		switch a {
		case types.CommandAvailabilityClaudeAI:
			if isClaudeAISubscriber {
				return true
			}
		case types.CommandAvailabilityConsole:
			if !isClaudeAISubscriber && !isUsing3PServices && isFirstPartyAnthropicBaseURL {
				return true
			}
		}
	}
	return false
}

// GetMcpSkillCommands mirrors src/commands.ts getMcpSkillCommands when MCP_SKILLS feature is on.
func GetMcpSkillCommands(mcpCommands []types.Command, mcpSkillsFeature bool) []types.Command {
	if !mcpSkillsFeature {
		return []types.Command{}
	}
	out := make([]types.Command, 0)
	for _, cmd := range mcpCommands {
		if cmd.Type != "prompt" {
			continue
		}
		if cmd.LoadedFrom == nil || *cmd.LoadedFrom != "mcp" {
			continue
		}
		if cmd.DisableModelInvocation != nil && *cmd.DisableModelInvocation {
			continue
		}
		out = append(out, cmd)
	}
	return out
}

// SkillToolCommands mirrors getSkillToolCommands filtering when given an already-resolved command list
// (TS normally calls getCommands(cwd) first). Omits isCommandEnabled — pre-filter in the caller if needed.
func SkillToolCommands(all []types.Command) []types.Command {
	out := make([]types.Command, 0)
	for _, cmd := range all {
		if cmd.Type != "prompt" {
			continue
		}
		if cmd.DisableModelInvocation != nil && *cmd.DisableModelInvocation {
			continue
		}
		if cmd.Source != nil && *cmd.Source == "builtin" {
			continue
		}
		lf := ""
		if cmd.LoadedFrom != nil {
			lf = *cmd.LoadedFrom
		}
		if lf == "bundled" || lf == "skills" || lf == "commands_DEPRECATED" {
			out = append(out, cmd)
			continue
		}
		if cmd.HasUserSpecifiedDescription != nil && *cmd.HasUserSpecifiedDescription {
			out = append(out, cmd)
			continue
		}
		if cmd.WhenToUse != nil && strings.TrimSpace(*cmd.WhenToUse) != "" {
			out = append(out, cmd)
		}
	}
	return out
}

// SlashCommandToolSkills mirrors getSlashCommandToolSkills filtering (src/commands.ts).
func SlashCommandToolSkills(all []types.Command) []types.Command {
	out := make([]types.Command, 0)
	for _, cmd := range all {
		if cmd.Type != "prompt" {
			continue
		}
		if cmd.Source != nil && *cmd.Source == "builtin" {
			continue
		}
		hasDesc := cmd.HasUserSpecifiedDescription != nil && *cmd.HasUserSpecifiedDescription
		hasWhen := cmd.WhenToUse != nil && strings.TrimSpace(*cmd.WhenToUse) != ""
		if !hasDesc && !hasWhen {
			continue
		}
		lf := ""
		if cmd.LoadedFrom != nil {
			lf = *cmd.LoadedFrom
		}
		dmi := cmd.DisableModelInvocation != nil && *cmd.DisableModelInvocation
		if lf == "skills" || lf == "plugin" || lf == "bundled" || dmi {
			out = append(out, cmd)
		}
	}
	return out
}

// IsBridgeSafeCommand mirrors src/commands.ts isBridgeSafeCommand.
func IsBridgeSafeCommand(cmd types.Command) bool {
	switch cmd.Type {
	case "local-jsx":
		return false
	case "prompt":
		return true
	case "local":
		_, ok := bridgeSafeLocalSet()[cmd.Name]
		return ok
	default:
		return false
	}
}

// FilterCommandsForRemoteMode mirrors src/commands.ts filterCommandsForRemoteMode.
func FilterCommandsForRemoteMode(cmds []types.Command) []types.Command {
	allow := remoteSafeSet()
	out := make([]types.Command, 0)
	for _, c := range cmds {
		if _, ok := allow[c.Name]; ok {
			out = append(out, c)
		}
	}
	return out
}

// FindCommand mirrors src/commands.ts findCommand.
func FindCommand(commandName string, commands []types.Command) *types.Command {
	for i := range commands {
		c := &commands[i]
		if c.Name == commandName {
			return c
		}
		if types.GetCommandName(*c) == commandName {
			return c
		}
		for _, a := range c.Aliases {
			if a == commandName {
				return c
			}
		}
	}
	return nil
}

// HasCommand mirrors src/commands.ts hasCommand.
func HasCommand(commandName string, commands []types.Command) bool {
	return FindCommand(commandName, commands) != nil
}

// GetCommand mirrors src/commands.ts getCommand.
func GetCommand(commandName string, commands []types.Command) (types.Command, error) {
	c := FindCommand(commandName, commands)
	if c == nil {
		return types.Command{}, fmt.Errorf("%w: %q (available: %s)", ErrCommandNotFound, commandName, formatAvailableList(commands))
	}
	return *c, nil
}

// ErrCommandNotFound is returned by GetCommand when no command matches.
var ErrCommandNotFound = errors.New("command not found")

// FormatDescriptionWithSource mirrors src/commands.ts formatDescriptionWithSource.
// Setting source names for non-builtin paths use the same labels as TS getSettingSourceName when possible.
func FormatDescriptionWithSource(cmd types.Command) string {
	if cmd.Type != "prompt" {
		return cmd.Description
	}
	if cmd.Kind != nil && *cmd.Kind == "workflow" {
		return cmd.Description + " (workflow)"
	}
	src := ""
	if cmd.Source != nil {
		src = *cmd.Source
	}
	if src == "plugin" {
		var meta struct {
			PluginManifest struct {
				Name string `json:"name"`
			} `json:"pluginManifest"`
		}
		if len(cmd.PluginInfo) > 0 && json.Unmarshal(cmd.PluginInfo, &meta) == nil && meta.PluginManifest.Name != "" {
			return "(" + meta.PluginManifest.Name + ") " + cmd.Description
		}
		return cmd.Description + " (plugin)"
	}
	if src == "builtin" || src == "mcp" {
		return cmd.Description
	}
	if src == "bundled" {
		return cmd.Description + " (bundled)"
	}
	if src != "" {
		return fmt.Sprintf("%s (%s)", cmd.Description, settingSourceLabel(src))
	}
	return cmd.Description
}

func settingSourceLabel(source string) string {
	switch source {
	case "userSettings":
		return "user"
	case "projectSettings":
		return "project"
	case "localSettings":
		return "project, gitignored"
	case "flagSettings":
		return "cli flag"
	case "policySettings":
		return "managed"
	default:
		return source
	}
}

func formatAvailableList(commands []types.Command) string {
	parts := make([]string, 0, len(commands))
	for _, c := range commands {
		name := types.GetCommandName(c)
		if len(c.Aliases) > 0 {
			parts = append(parts, fmt.Sprintf("%s (aliases: %s)", name, strings.Join(c.Aliases, ", ")))
		} else {
			parts = append(parts, name)
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}
