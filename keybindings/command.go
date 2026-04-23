package keybindings

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// CommandResult represents the result of executing the keybindings command
type CommandResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ExecuteKeybindingsCommand handles the /keybindings command
// It creates the keybindings.json file if it doesn't exist and opens it in an editor
func ExecuteKeybindingsCommand() (*CommandResult, error) {
	// Check if keybinding customization is enabled
	if !isKeybindingCustomizationEnabled() {
		return &CommandResult{
			Type: "text",
			Value: "Keybinding customization is not enabled. This feature is currently in preview.",
		}, nil
	}
	
	keybindingsPath, err := GetKeybindingsPath()
	if err != nil {
		return &CommandResult{
			Type: "text",
			Value: fmt.Sprintf("Error getting keybindings path: %v", err),
		}, nil
	}
	
	// Check if file already exists
	fileExists := true
	if _, err := os.Stat(keybindingsPath); os.IsNotExist(err) {
		fileExists = false
		// Create the template file
		if err := SaveKeybindingsTemplate(keybindingsPath); err != nil {
			return &CommandResult{
				Type: "text",
				Value: fmt.Sprintf("Failed to create keybindings template: %v", err),
			}, nil
		}
	}
	
	// Try to open in editor
	editor, err := getEditor()
	if err != nil {
		message := fmt.Sprintf("%s %s", 
			getCreatedOrOpenedMessage(fileExists, keybindingsPath),
			fmt.Sprintf("Could not open in editor: %v", err))
		return &CommandResult{
			Type:  "text",
			Value: message,
		}, nil
	}
	
	// Launch editor
	if err := launchEditor(editor, keybindingsPath); err != nil {
		message := fmt.Sprintf("%s %s", 
			getCreatedOrOpenedMessage(fileExists, keybindingsPath),
			fmt.Sprintf("Could not open in editor: %v", err))
		return &CommandResult{
			Type:  "text",
			Value: message,
		}, nil
	}
	
	// Success message
	message := getCreatedOrOpenedMessage(fileExists, keybindingsPath) + " Opened in your editor."
	return &CommandResult{
		Type:  "text",
		Value: message,
	}, nil
}

// getCreatedOrOpenedMessage returns appropriate message based on whether file existed
func getCreatedOrOpenedMessage(fileExists bool, path string) string {
	if fileExists {
		return fmt.Sprintf("Opened %s.", path)
	}
	return fmt.Sprintf("Created %s with template.", path)
}

// getEditor gets the preferred editor from environment variables
func getEditor() (string, error) {
	// Check environment variables in order of preference
	editors := []string{"CLAUDE_EDITOR", "VISUAL", "EDITOR"}
	
	for _, envVar := range editors {
		if editor := os.Getenv(envVar); editor != "" {
			return editor, nil
		}
	}
	
	// Platform-specific defaults
	switch runtime.GOOS {
	case "darwin":
		// macOS defaults
		candidates := []string{"code", "subl", "atom", "vim", "nano"}
		for _, candidate := range candidates {
			if _, err := exec.LookPath(candidate); err == nil {
				return candidate, nil
			}
		}
	case "linux":
		// Linux defaults
		candidates := []string{"code", "gedit", "vim", "nano"}
		for _, candidate := range candidates {
			if _, err := exec.LookPath(candidate); err == nil {
				return candidate, nil
			}
		}
	case "windows":
		// Windows defaults
		candidates := []string{"code.cmd", "notepad.exe", "notepad++.exe"}
		for _, candidate := range candidates {
			if _, err := exec.LookPath(candidate); err == nil {
				return candidate, nil
			}
		}
	}
	
	return "", fmt.Errorf("no suitable editor found. Set EDITOR environment variable")
}

// launchEditor launches the specified editor with the given file
func launchEditor(editor, filepath string) error {
	cmd := exec.Command(editor, filepath)
	
	// For GUI editors, we can start them in the background
	// For terminal editors, we need to connect to stdin/stdout/stderr
	if isTerminalEditor(editor) {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	
	// For GUI editors, start in background
	return cmd.Start()
}

// isTerminalEditor checks if the editor is a terminal-based editor
func isTerminalEditor(editor string) bool {
	terminalEditors := []string{"vim", "vi", "nano", "emacs", "joe", "micro"}
	
	for _, termEditor := range terminalEditors {
		if editor == termEditor {
			return true
		}
	}
	
	return false
}