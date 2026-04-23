// Package keybindings provides keyboard shortcut configuration management for Claude Code.
// It mirrors the TypeScript keybindings system functionality.
package keybindings

import "encoding/json"

// KeybindingContextName represents the context where a keybinding is active
type KeybindingContextName string

const (
	ContextGlobal         KeybindingContextName = "Global"
	ContextChat           KeybindingContextName = "Chat"
	ContextAutocomplete   KeybindingContextName = "Autocomplete"
	ContextConfirmation   KeybindingContextName = "Confirmation"
	ContextHelp           KeybindingContextName = "Help"
	ContextTranscript     KeybindingContextName = "Transcript"
	ContextHistorySearch  KeybindingContextName = "HistorySearch"
	ContextTask           KeybindingContextName = "Task"
	ContextThemePicker    KeybindingContextName = "ThemePicker"
	ContextSettings       KeybindingContextName = "Settings"
	ContextTabs           KeybindingContextName = "Tabs"
	ContextAttachments    KeybindingContextName = "Attachments"
	ContextFooter         KeybindingContextName = "Footer"
	ContextMessageSelector KeybindingContextName = "MessageSelector"
	ContextDiffDialog     KeybindingContextName = "DiffDialog"
	ContextModelPicker    KeybindingContextName = "ModelPicker"
	ContextSelect         KeybindingContextName = "Select"
	ContextPlugin         KeybindingContextName = "Plugin"
)

// KeybindingAction represents an action that can be triggered by a keybinding
type KeybindingAction string

// ParsedKeystroke represents a parsed keystroke with modifiers and key
type ParsedKeystroke struct {
	Ctrl  bool   `json:"ctrl"`
	Alt   bool   `json:"alt"`
	Shift bool   `json:"shift"`
	Meta  bool   `json:"meta"`
	Key   string `json:"key"`
}

// Chord represents a sequence of keystrokes (for chord bindings like "ctrl+k ctrl+s")
type Chord []ParsedKeystroke

// ParsedBinding represents a fully parsed keybinding
type ParsedBinding struct {
	Context KeybindingContextName `json:"context"`
	Chord   Chord                 `json:"chord"`
	Action  *KeybindingAction     `json:"action"` // null for unbinding
}

// KeybindingBlock represents a block of keybindings for a specific context
type KeybindingBlock struct {
	Context  KeybindingContextName            `json:"context"`
	Bindings map[string]*KeybindingActionPtr `json:"bindings"` // string or null
}

// KeybindingActionPtr is a helper type for JSON marshaling of nullable actions
type KeybindingActionPtr struct {
	Action *KeybindingAction
}

// MarshalJSON implements json.Marshaler for KeybindingActionPtr
func (k *KeybindingActionPtr) MarshalJSON() ([]byte, error) {
	if k.Action == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(*k.Action)
}

// UnmarshalJSON implements json.Unmarshaler for KeybindingActionPtr
func (k *KeybindingActionPtr) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		k.Action = nil
		return nil
	}
	var action KeybindingAction
	if err := json.Unmarshal(data, &action); err != nil {
		return err
	}
	k.Action = &action
	return nil
}

// KeybindingConfig represents the structure of keybindings.json
type KeybindingConfig struct {
	Schema   string             `json:"$schema"`
	Docs     string             `json:"$docs"`
	Bindings []KeybindingBlock  `json:"bindings"`
}

// KeybindingWarning represents a validation warning for keybindings
type KeybindingWarning struct {
	Type       string `json:"type"`       // "parse_error", "conflict", "reserved", etc.
	Severity   string `json:"severity"`   // "error", "warning"
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Context    string `json:"context,omitempty"`
	Key        string `json:"key,omitempty"`
}

// KeybindingsLoadResult represents the result of loading keybindings
type KeybindingsLoadResult struct {
	Bindings []ParsedBinding     `json:"bindings"`
	Warnings []KeybindingWarning `json:"warnings"`
}

// ReservedShortcut represents a keyboard shortcut that cannot be rebound
type ReservedShortcut struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}