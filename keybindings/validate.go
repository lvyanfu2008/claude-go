package keybindings

import (
	"fmt"
	"strings"
)

// ValidateBindings validates user keybinding configuration
// Returns warnings for any issues found
func ValidateBindings(userBlocks []KeybindingBlock, allBindings []ParsedBinding) []KeybindingWarning {
	var warnings []KeybindingWarning
	
	// Validate each block
	for _, block := range userBlocks {
		blockWarnings := validateKeybindingBlock(block)
		warnings = append(warnings, blockWarnings...)
	}
	
	// Check for conflicts and duplicates
	conflictWarnings := checkKeybindingConflicts(allBindings)
	warnings = append(warnings, conflictWarnings...)
	
	return warnings
}

// validateKeybindingBlock validates a single keybinding block
func validateKeybindingBlock(block KeybindingBlock) []KeybindingWarning {
	var warnings []KeybindingWarning
	
	// Validate context name
	if !isValidContext(block.Context) {
		warnings = append(warnings, KeybindingWarning{
			Type:       "invalid_context",
			Severity:   "error",
			Message:    fmt.Sprintf("Unknown context %q", block.Context),
			Suggestion: "Valid contexts: " + strings.Join(getValidContexts(), ", "),
			Context:    string(block.Context),
		})
	}
	
	// Validate each binding
	for key, actionPtr := range block.Bindings {
		keyWarnings := validateKeybinding(key, actionPtr, string(block.Context))
		warnings = append(warnings, keyWarnings...)
	}
	
	return warnings
}

// validateKeybinding validates a single keybinding
func validateKeybinding(key string, actionPtr *KeybindingActionPtr, context string) []KeybindingWarning {
	var warnings []KeybindingWarning
	
	// Validate keystroke syntax
	_, err := ParseKeystroke(key)
	if err != nil {
		warnings = append(warnings, KeybindingWarning{
			Type:       "parse_error",
			Severity:   "error",
			Message:    fmt.Sprintf("Could not parse keystroke %q: %v", key, err),
			Suggestion: "Check syntax: use + between modifiers, valid key names",
			Context:    context,
			Key:        key,
		})
		return warnings // Don't continue validation if we can't parse the key
	}
	
	// Check for reserved keys
	if isReserved, reason := IsNonRebindableKey(key); isReserved {
		warnings = append(warnings, KeybindingWarning{
			Type:       "reserved_key",
			Severity:   "error",
			Message:    fmt.Sprintf("%q cannot be rebound: %s", key, reason),
			Suggestion: "Choose a different key combination",
			Context:    context,
			Key:        key,
		})
	} else if isReserved, reason := IsReservedKey(key); isReserved {
		warnings = append(warnings, KeybindingWarning{
			Type:       "reserved_key",
			Severity:   "warning",
			Message:    fmt.Sprintf("%q may not work: %s", key, reason),
			Suggestion: "Consider using a different key combination",
			Context:    context,
			Key:        key,
		})
	}
	
	// Validate action
	if actionPtr.Action != nil {
		if !isValidAction(string(*actionPtr.Action)) {
			warnings = append(warnings, KeybindingWarning{
				Type:       "invalid_action",
				Severity:   "error",
				Message:    fmt.Sprintf("Unknown action %q", *actionPtr.Action),
				Suggestion: "Use a valid action from the available actions list",
				Context:    context,
				Key:        key,
			})
		}
	}
	
	return warnings
}

// checkKeybindingConflicts checks for conflicts between keybindings
func checkKeybindingConflicts(bindings []ParsedBinding) []KeybindingWarning {
	var warnings []KeybindingWarning
	
	// Group bindings by context and chord for conflict detection
	contextBindings := make(map[KeybindingContextName]map[string][]ParsedBinding)
	
	for _, binding := range bindings {
		if contextBindings[binding.Context] == nil {
			contextBindings[binding.Context] = make(map[string][]ParsedBinding)
		}
		
		chordKey := chordToString(binding.Chord)
		contextBindings[binding.Context][chordKey] = append(
			contextBindings[binding.Context][chordKey], binding)
	}
	
	// Check for duplicates within each context
	for context, chordMap := range contextBindings {
		for chordKey, bindingList := range chordMap {
			if len(bindingList) > 1 {
				// Multiple bindings for the same chord in the same context
				actions := make([]string, len(bindingList))
				for i, binding := range bindingList {
					if binding.Action != nil {
						actions[i] = string(*binding.Action)
					} else {
						actions[i] = "null"
					}
				}
				
				warnings = append(warnings, KeybindingWarning{
					Type:     "duplicate_key",
					Severity: "warning",
					Message: fmt.Sprintf("Duplicate key %q in %s context (actions: %s)", 
						chordKey, context, strings.Join(actions, ", ")),
					Suggestion: "Only the last binding will be used",
					Context:    string(context),
					Key:        chordKey,
				})
			}
		}
	}
	
	return warnings
}

// chordToString converts a Chord to a string representation
func chordToString(chord Chord) string {
	parts := make([]string, len(chord))
	for i, keystroke := range chord {
		parts[i] = keystrokeToString(keystroke)
	}
	return strings.Join(parts, " ")
}

// keystrokeToString converts a ParsedKeystroke to a string representation
func keystrokeToString(keystroke ParsedKeystroke) string {
	var parts []string
	
	if keystroke.Ctrl {
		parts = append(parts, "ctrl")
	}
	if keystroke.Alt {
		parts = append(parts, "alt")
	}
	if keystroke.Shift {
		parts = append(parts, "shift")
	}
	if keystroke.Meta {
		parts = append(parts, "meta")
	}
	
	parts = append(parts, keystroke.Key)
	return strings.Join(parts, "+")
}

// isValidContext checks if a context name is valid
func isValidContext(context KeybindingContextName) bool {
	validContexts := getValidContexts()
	for _, valid := range validContexts {
		if string(context) == valid {
			return true
		}
	}
	return false
}

// getValidContexts returns a list of valid context names
func getValidContexts() []string {
	return []string{
		string(ContextGlobal),
		string(ContextChat),
		string(ContextAutocomplete),
		string(ContextConfirmation),
		string(ContextHelp),
		string(ContextTranscript),
		string(ContextHistorySearch),
		string(ContextTask),
		string(ContextThemePicker),
		string(ContextSettings),
		string(ContextTabs),
		string(ContextAttachments),
		string(ContextFooter),
		string(ContextMessageSelector),
		string(ContextDiffDialog),
		string(ContextModelPicker),
		string(ContextSelect),
		string(ContextPlugin),
	}
}

// isValidAction checks if an action name is valid
// For now, we'll accept any non-empty string as a valid action
// TODO: Implement comprehensive action validation
func isValidAction(action string) bool {
	return strings.TrimSpace(action) != ""
}