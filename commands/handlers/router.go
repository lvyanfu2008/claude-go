package handlers

import (
	"fmt"
)

// LocalCommandHandler is a function that handles a local command.
// The args parameter carries the typed argument string (empty if none).
type LocalCommandHandler func(args string) ([]byte, error)

// localCommandHandlers maps command names to their handlers
// Handlers with an Args suffix (e.g. args string) receive the raw argument string.
var localCommandHandlers = map[string]LocalCommandHandler{
	"keybindings":   func(args string) ([]byte, error) { return HandleKeybindingsCommand() },
	"cost":          func(args string) ([]byte, error) { return HandleCostCommand() },
	"version":       func(args string) ([]byte, error) { return HandleVersionCommand() },
	"release-notes": func(args string) ([]byte, error) { return HandleReleaseNotesCommand() },
	"context":       func(args string) ([]byte, error) { return HandleContextCommand() },
	"vim":           func(args string) ([]byte, error) { return HandleVimCommand(args) },
	"all":           func(args string) ([]byte, error) { return HandleAllCommand() },
	"doctor":        func(args string) ([]byte, error) { return HandleDoctorCommand() },
	"effort":        func(args string) ([]byte, error) { return HandleEffortCommand(args) },
	"help":          func(args string) ([]byte, error) { return HandleHelpCommand() },
	"model":         func(args string) ([]byte, error) { return HandleModelCommand(args) },
	"plugins":       func(args string) ([]byte, error) { return HandlePluginsCommand() },
	"session":       func(args string) ([]byte, error) { return HandleSessionCommand() },
	"status":        func(args string) ([]byte, error) { return HandleStatusCommand() },
	"stickers":      func(args string) ([]byte, error) { return HandleStickersCommand() },
	"mobile":        func(args string) ([]byte, error) { return HandleMobileCommand() },
		"reload-plugins": func(args string) ([]byte, error) { return HandleReloadPluginsCommand() },
		"extra-usage":    func(args string) ([]byte, error) { return HandleExtraUsageCommand() },
		"rewind":         func(args string) ([]byte, error) { return HandleRewindCommand(args) },
}

// HandleLocalCommand routes local commands to their appropriate handlers
func HandleLocalCommand(commandName string, args string) ([]byte, error) {
	handler, exists := localCommandHandlers[commandName]
	if !exists {
		return nil, fmt.Errorf("no handler found for local command: %s", commandName)
	}

	return handler(args)
}

// RegisterLocalCommand registers a new local command handler
func RegisterLocalCommand(name string, handler LocalCommandHandler) {
	localCommandHandlers[name] = handler
}

// GetSupportedLocalCommands returns a list of supported local command names
func GetSupportedLocalCommands() []string {
	var commands []string
	for name := range localCommandHandlers {
		commands = append(commands, name)
	}
	return commands
}
