package handlers

import (
	"encoding/json"
	
	"goc/keybindings"
)

// HandleKeybindingsCommand handles the /keybindings local command
func HandleKeybindingsCommand() ([]byte, error) {
	result, err := keybindings.ExecuteKeybindingsCommand()
	if err != nil {
		return nil, err
	}
	
	// Return JSON response compatible with the expected format
	response, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	
	return response, nil
}