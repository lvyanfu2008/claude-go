package handlers

import (
	"fmt"
)

// LocalCommandHandler is a function that handles a local command
type LocalCommandHandler func() ([]byte, error)

// localCommandHandlers maps command names to their handlers
var localCommandHandlers = map[string]LocalCommandHandler{
	"keybindings": HandleKeybindingsCommand,
}

// HandleLocalCommand routes local commands to their appropriate handlers
func HandleLocalCommand(commandName string) ([]byte, error) {
	handler, exists := localCommandHandlers[commandName]
	if !exists {
		return nil, fmt.Errorf("no handler found for local command: %s", commandName)
	}
	
	return handler()
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