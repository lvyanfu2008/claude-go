package keybindings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// KeybindingsFileName is the name of the keybindings configuration file
	KeybindingsFileName = "keybindings.json"
	
	// SchemaURL is the JSON schema URL for keybindings.json
	SchemaURL = "https://www.schemastore.org/claude-code-keybindings.json"
	
	// DocsURL is the documentation URL for keybindings
	DocsURL = "https://code.claude.com/docs/en/keybindings"
)

// GetKeybindingsPath returns the path to the user keybindings file
// Mirrors TypeScript getKeybindingsPath()
func GetKeybindingsPath() (string, error) {
	homeDir, err := getClaudeConfigHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get Claude config directory: %w", err)
	}
	return filepath.Join(homeDir, KeybindingsFileName), nil
}

// getClaudeConfigHomeDir mirrors TypeScript getClaudeConfigHomeDir()
// Returns $CLAUDE_CONFIG_DIR if set, otherwise $HOME/.claude
func getClaudeConfigHomeDir() (string, error) {
	if configDir := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); configDir != "" {
		return configDir, nil
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	return filepath.Join(homeDir, ".claude"), nil
}

// LoadKeybindings loads and parses keybindings from user config file
// Returns merged default + user bindings along with validation warnings
func LoadKeybindings() (*KeybindingsLoadResult, error) {
	defaultBindings, err := ParseBindings(DefaultBindings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse default bindings: %w", err)
	}
	
	// Check if keybinding customization is enabled (feature gate)
	if !isKeybindingCustomizationEnabled() {
		return &KeybindingsLoadResult{
			Bindings: defaultBindings,
			Warnings: []KeybindingWarning{},
		}, nil
	}
	
	userPath, err := GetKeybindingsPath()
	if err != nil {
		return &KeybindingsLoadResult{
			Bindings: defaultBindings,
			Warnings: []KeybindingWarning{
				{
					Type:     "config_error",
					Severity: "error",
					Message:  fmt.Sprintf("Failed to determine keybindings path: %v", err),
				},
			},
		}, nil
	}
	
	// Try to read user config file
	content, err := os.ReadFile(userPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - use defaults (user can run /keybindings to create)
			return &KeybindingsLoadResult{
				Bindings: defaultBindings,
				Warnings: []KeybindingWarning{},
			}, nil
		}
		// Other error - log and return defaults with warning
		return &KeybindingsLoadResult{
			Bindings: defaultBindings,
			Warnings: []KeybindingWarning{
				{
					Type:     "parse_error",
					Severity: "error",
					Message:  fmt.Sprintf("Failed to read keybindings.json: %v", err),
				},
			},
		}, nil
	}
	
	// Parse JSON content
	var config KeybindingConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return &KeybindingsLoadResult{
			Bindings: defaultBindings,
			Warnings: []KeybindingWarning{
				{
					Type:     "parse_error",
					Severity: "error",
					Message:  fmt.Sprintf("Failed to parse keybindings.json: %v", err),
					Suggestion: "Check JSON syntax and format",
				},
			},
		}, nil
	}
	
	// Validate structure
	if len(config.Bindings) == 0 {
		return &KeybindingsLoadResult{
			Bindings: defaultBindings,
			Warnings: []KeybindingWarning{
				{
					Type:       "parse_error",
					Severity:   "error",
					Message:    "keybindings.json must have a \"bindings\" array",
					Suggestion: "Use format: { \"bindings\": [...] }",
				},
			},
		}, nil
	}
	
	// Parse user bindings
	userParsed, err := ParseBindings(config.Bindings)
	if err != nil {
		return &KeybindingsLoadResult{
			Bindings: defaultBindings,
			Warnings: []KeybindingWarning{
				{
					Type:     "parse_error",
					Severity: "error",
					Message:  fmt.Sprintf("Failed to parse user bindings: %v", err),
				},
			},
		}, nil
	}
	
	// User bindings come after defaults, so they override
	mergedBindings := append(defaultBindings, userParsed...)
	
	// Run validation on user config
	warnings := ValidateBindings(config.Bindings, mergedBindings)
	
	return &KeybindingsLoadResult{
		Bindings: mergedBindings,
		Warnings: warnings,
	}, nil
}

// SaveKeybindingsTemplate creates a keybindings.json template file
func SaveKeybindingsTemplate(path string) error {
	// Generate template config
	template := GenerateKeybindingsTemplate()
	
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	
	// Write template to file
	content, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}
	
	// Add newline at end
	content = append(content, '\n')
	
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}
	
	return nil
}

// GenerateKeybindingsTemplate creates a template keybindings configuration
// Mirrors TypeScript generateKeybindingsTemplate()
func GenerateKeybindingsTemplate() *KeybindingConfig {
	// Filter out reserved shortcuts that cannot be rebound
	filteredBindings := filterReservedShortcuts(DefaultBindings)
	
	return &KeybindingConfig{
		Schema:   SchemaURL,
		Docs:     DocsURL,
		Bindings: filteredBindings,
	}
}

// filterReservedShortcuts removes reserved shortcuts from default bindings
// These would cause validation warnings, so we exclude them from the template
func filterReservedShortcuts(blocks []KeybindingBlock) []KeybindingBlock {
	var filtered []KeybindingBlock
	
	for _, block := range blocks {
		filteredBindings := make(map[string]*KeybindingActionPtr)
		
		for key, action := range block.Bindings {
			if isReserved, _ := IsNonRebindableKey(key); !isReserved {
				filteredBindings[key] = action
			}
		}
		
		if len(filteredBindings) > 0 {
			filtered = append(filtered, KeybindingBlock{
				Context:  block.Context,
				Bindings: filteredBindings,
			})
		}
	}
	
	return filtered
}

// isKeybindingCustomizationEnabled checks if keybinding customization is enabled
// This would typically check a feature gate - for now, we'll enable it for all users
// TODO: Implement proper feature gate integration
func isKeybindingCustomizationEnabled() bool {
	// For now, always enable keybinding customization
	// In the future, this should check a feature gate like the TypeScript version
	return true
}