package processuserinput

import (
	"strings"

	"goc/types"
)

// ParsedSlashCommand mirrors src/utils/slashCommandParsing.ts ParsedSlashCommand.
type ParsedSlashCommand struct {
	CommandName string
	Args        string
	IsMcp       bool
}

// ParseSlashCommand mirrors src/utils/slashCommandParsing.ts parseSlashCommand.
func ParseSlashCommand(input string) *ParsedSlashCommand {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return nil
	}
	without := strings.TrimPrefix(trimmed, "/")
	words := strings.Fields(without)
	if len(words) == 0 {
		return nil
	}
	commandName := words[0]
	isMcp := false
	argsStart := 1
	if len(words) > 1 && words[1] == "(MCP)" {
		commandName = commandName + " (MCP)"
		isMcp = true
		argsStart = 2
	}
	args := strings.Join(words[argsStart:], " ")
	return &ParsedSlashCommand{CommandName: commandName, Args: args, IsMcp: isMcp}
}

// GetCommandName mirrors src/types/command.ts getCommandName (no userFacingName in Go).
func GetCommandName(cmd types.Command) string {
	return cmd.Name
}

// FindCommand mirrors src/commands.ts findCommand.
func FindCommand(commandName string, commands []types.Command) *types.Command {
	for i := range commands {
		c := &commands[i]
		if c.Name == commandName || GetCommandName(*c) == commandName {
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

// bridgeSafeLocalNames matches src/commands.ts BRIDGE_SAFE_COMMANDS (local command names only).
var bridgeSafeLocalNames = map[string]struct{}{
	"compact":       {},
	"clear":         {},
	"cost":          {},
	"summary":       {},
	"release-notes": {},
	"files":         {},
}

// IsBridgeSafeCommand mirrors src/commands.ts isBridgeSafeCommand.
func IsBridgeSafeCommand(cmd types.Command) bool {
	switch cmd.Type {
	case "local-jsx":
		return false
	case "prompt":
		return true
	case "local":
		_, ok := bridgeSafeLocalNames[cmd.Name]
		return ok
	default:
		return false
	}
}
