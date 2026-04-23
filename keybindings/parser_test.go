package keybindings

import (
	"reflect"
	"testing"
)

func TestParseKeystroke(t *testing.T) {
	tests := []struct {
		name        string
		keystroke   string
		expected    Chord
		expectError bool
	}{
		{
			name:      "simple key",
			keystroke: "a",
			expected: Chord{
				{Key: "a"},
			},
		},
		{
			name:      "ctrl modifier",
			keystroke: "ctrl+s",
			expected: Chord{
				{Ctrl: true, Key: "s"},
			},
		},
		{
			name:      "multiple modifiers",
			keystroke: "ctrl+shift+p",
			expected: Chord{
				{Ctrl: true, Shift: true, Key: "p"},
			},
		},
		{
			name:      "meta key alias",
			keystroke: "cmd+v",
			expected: Chord{
				{Meta: true, Key: "v"},
			},
		},
		{
			name:      "alt key alias",
			keystroke: "opt+enter",
			expected: Chord{
				{Alt: true, Key: "enter"},
			},
		},
		{
			name:      "chord binding",
			keystroke: "ctrl+k ctrl+s",
			expected: Chord{
				{Ctrl: true, Key: "k"},
				{Ctrl: true, Key: "s"},
			},
		},
		{
			name:      "special key",
			keystroke: "escape",
			expected: Chord{
				{Key: "escape"},
			},
		},
		{
			name:      "key alias",
			keystroke: "esc",
			expected: Chord{
				{Key: "escape"},
			},
		},
		{
			name:        "empty keystroke",
			keystroke:   "",
			expectError: true,
		},
		{
			name:        "invalid modifier",
			keystroke:   "invalid+s",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseKeystroke(tt.keystroke)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "regular key",
			input:    "s",
			expected: "s",
		},
		{
			name:     "escape alias",
			input:    "esc",
			expected: "escape",
		},
		{
			name:     "return alias",
			input:    "return",
			expected: "enter",
		},
		{
			name:     "case insensitive",
			input:    "ESCAPE",
			expected: "escape",
		},
		{
			name:     "whitespace trimmed",
			input:    "  enter  ",
			expected: "enter",
		},
		{
			name:        "invalid key",
			input:       "invalid-key-name",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeKey(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}