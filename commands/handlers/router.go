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
	"keybindings":    func(args string) ([]byte, error) { return HandleKeybindingsCommand() },
	"compact":        func(args string) ([]byte, error) { return HandleCompactCommand() },
	"cost":           func(args string) ([]byte, error) { return HandleCostCommand() },
	"version":        func(args string) ([]byte, error) { return HandleVersionCommand() },
	"release-notes":  func(args string) ([]byte, error) { return HandleReleaseNotesCommand() },
	"context":        func(args string) ([]byte, error) { return HandleContextCommand() },
	"vim":            func(args string) ([]byte, error) { return HandleVimCommand(args) },
	"all":            func(args string) ([]byte, error) { return HandleAllCommand() },
	"doctor":         func(args string) ([]byte, error) { return HandleDoctorCommand() },
	"effort":         func(args string) ([]byte, error) { return HandleEffortCommand(args) },
	"help":           func(args string) ([]byte, error) { return HandleHelpCommand() },
	"model":          func(args string) ([]byte, error) { return HandleModelCommand(args) },
	"plugins":        func(args string) ([]byte, error) { return HandlePluginsCommand() },
	"session":        func(args string) ([]byte, error) { return HandleSessionCommand() },
	"status":         func(args string) ([]byte, error) { return HandleStatusCommand() },
	"stickers":       func(args string) ([]byte, error) { return HandleStickersCommand() },
	"mobile":         func(args string) ([]byte, error) { return HandleMobileCommand() },
	"reload-plugins": func(args string) ([]byte, error) { return HandleReloadPluginsCommand() },
	"extra-usage":    func(args string) ([]byte, error) { return HandleExtraUsageCommand() },
	"rewind":         func(args string) ([]byte, error) { return HandleRewindCommand(args) },
	// P1 Phase 1 additions
	"permissions":  func(args string) ([]byte, error) { return HandlePermissionsCommand(args) },
	"theme":        func(args string) ([]byte, error) { return HandleThemeCommand(args) },
	"color":        func(args string) ([]byte, error) { return HandleColorCommand(args) },
	"output-style": func(args string) ([]byte, error) { return HandleOutputStyleCommand(args) },
	"statusline":   func(args string) ([]byte, error) { return HandleStatuslineCommand(args) },
	"fast":         func(args string) ([]byte, error) { return HandleFastCommand(args) },
	"ide":          func(args string) ([]byte, error) { return HandleIDECommand(args) },
	"add-dir":      func(args string) ([]byte, error) { return HandleAddDirCommand(args) },
	"stats":        func(args string) ([]byte, error) { return HandleStatsCommand(args) },
	"usage":        func(args string) ([]byte, error) { return HandleUsageCommand(args) },
	"logout":       func(args string) ([]byte, error) { return HandleLogoutCommand(args) },
	"login":        func(args string) ([]byte, error) { return HandleLoginCommand(args) },
	"export":       func(args string) ([]byte, error) { return HandleExportCommand(args) },
	"tasks":        func(args string) ([]byte, error) { return HandleTasksCommand(args) },
	"memory":       func(args string) ([]byte, error) { return HandleMemoryCommand(args) },
	// P1 Phase 3 additions
	"rename":      func(args string) ([]byte, error) { return HandleRenameCommand(args) },
	"resume":      func(args string) ([]byte, error) { return HandleResumeCommand(args) },
	"clear":       func(args string) ([]byte, error) { return HandleClearCommand(args) },
	"config":      func(args string) ([]byte, error) { return HandleConfigCommand(args) },
	"hooks":       func(args string) ([]byte, error) { return HandleHooksCommand(args) },
	"mcp":         func(args string) ([]byte, error) { return HandleMcpCommand(args) },
	"skills":      func(args string) ([]byte, error) { return HandleSkillsCommand(args) },
	"tag":         func(args string) ([]byte, error) { return HandleTagCommand(args) },
	"plan":        func(args string) ([]byte, error) { return HandlePlanCommand(args) },
	"review":      func(args string) ([]byte, error) { return HandleReviewCommand(args) },
	"pr-comments": func(args string) ([]byte, error) { return HandlePRCommentsCommand(args) },
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
