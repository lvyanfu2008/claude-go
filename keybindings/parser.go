package keybindings

import (
	"fmt"
	"strings"
)

// ParseBindings converts KeybindingBlocks to ParsedBindings
// Mirrors TypeScript parseBindings()
func ParseBindings(blocks []KeybindingBlock) ([]ParsedBinding, error) {
	var parsed []ParsedBinding
	
	for _, block := range blocks {
		for keyString, actionPtr := range block.Bindings {
			// Parse the keystroke string
			chord, err := ParseKeystroke(keyString)
			if err != nil {
				return nil, fmt.Errorf("failed to parse keystroke %q in context %s: %w", 
					keyString, block.Context, err)
			}
			
			// Create parsed binding
			binding := ParsedBinding{
				Context: block.Context,
				Chord:   chord,
				Action:  actionPtr.Action, // May be nil for unbinding
			}
			
			parsed = append(parsed, binding)
		}
	}
	
	return parsed, nil
}

// ParseKeystroke parses a keystroke string into a Chord
// Supports both single keystrokes and chords (space-separated)
// Examples: "ctrl+s", "ctrl+k ctrl+s", "meta+shift+p"
func ParseKeystroke(keystroke string) (Chord, error) {
	keystroke = strings.TrimSpace(keystroke)
	if keystroke == "" {
		return nil, fmt.Errorf("empty keystroke")
	}
	
	// Split by spaces to handle chords
	parts := strings.Fields(keystroke)
	chord := make(Chord, len(parts))
	
	for i, part := range parts {
		parsed, err := parseSingleKeystroke(part)
		if err != nil {
			return nil, fmt.Errorf("invalid keystroke part %q: %w", part, err)
		}
		chord[i] = parsed
	}
	
	return chord, nil
}

// parseSingleKeystroke parses a single keystroke (no spaces)
// Examples: "ctrl+s", "meta+shift+p", "escape", "f1"
func parseSingleKeystroke(keystroke string) (ParsedKeystroke, error) {
	keystroke = strings.TrimSpace(keystroke)
	parts := strings.Split(keystroke, "+")
	
	if len(parts) == 0 {
		return ParsedKeystroke{}, fmt.Errorf("empty keystroke")
	}
	
	parsed := ParsedKeystroke{}
	
	// The last part is always the key, everything before are modifiers
	keyPart := parts[len(parts)-1]
	modifierParts := parts[:len(parts)-1]
	
	// Parse modifiers
	for _, modifier := range modifierParts {
		modifier = strings.ToLower(strings.TrimSpace(modifier))
		switch modifier {
		case "ctrl", "control":
			parsed.Ctrl = true
		case "alt", "opt", "option":
			parsed.Alt = true
		case "shift":
			parsed.Shift = true
		case "meta", "cmd", "command":
			parsed.Meta = true
		default:
			return ParsedKeystroke{}, fmt.Errorf("unknown modifier: %s", modifier)
		}
	}
	
	// Parse key
	key, err := normalizeKey(keyPart)
	if err != nil {
		return ParsedKeystroke{}, fmt.Errorf("invalid key %q: %w", keyPart, err)
	}
	parsed.Key = key
	
	return parsed, nil
}

// normalizeKey normalizes key names to a standard form
func normalizeKey(key string) (string, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	
	// Handle special key aliases
	switch key {
	case "esc":
		return "escape", nil
	case "return":
		return "enter", nil
	case "del":
		return "delete", nil
	case "bs":
		return "backspace", nil
	}
	
	// Validate key name
	if !isValidKeyName(key) {
		return "", fmt.Errorf("invalid key name: %s", key)
	}
	
	return key, nil
}

// isValidKeyName checks if a key name is valid
func isValidKeyName(key string) bool {
	// Single characters
	if len(key) == 1 {
		c := key[0]
		return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || 
			   c == ' ' || c == '`' || c == '-' || c == '=' || c == '[' || c == ']' || 
			   c == '\\' || c == ';' || c == '\'' || c == ',' || c == '.' || c == '/' ||
			   c == '_'
	}
	
	// Special keys
	specialKeys := []string{
		"escape", "enter", "tab", "space", "backspace", "delete",
		"up", "down", "left", "right",
		"home", "end", "pageup", "pagedown",
		"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12",
		"insert", "menu", "pause", "scrolllock", "numlock", "capslock",
	}
	
	for _, special := range specialKeys {
		if key == special {
			return true
		}
	}
	
	return false
}