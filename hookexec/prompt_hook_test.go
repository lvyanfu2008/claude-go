package hookexec

import (
	"testing"
)

func TestParseHookResponseJSON(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantOk  bool
		wantErr bool
	}{
		{"ok true", `{"ok": true}`, true, false},
		{"ok false with reason", `{"ok": false, "reason": "not ready"}`, false, false},
		{"invalid json", `not json`, false, true},
		{"empty", "", false, true},
		{"missing ok", `{"reason": "x"}`, false, false}, // ok defaults to false (zero value)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseHookResponseJSON(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHookResponseJSON(%q) error = %v, wantErr = %v", tt.raw, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if parsed.Ok != tt.wantOk {
				t.Errorf("parseHookResponseJSON(%q).Ok = %v, want %v", tt.raw, parsed.Ok, tt.wantOk)
			}
		})
	}
}

func TestSubstituteArguments(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		jsonIn   string
		expected string
	}{
		{"$ARGUMENTS", "Check: $ARGUMENTS", `{"tool":"bash"}`, `Check: {"tool":"bash"}`},
		{"${ARGUMENTS}", "Check: ${ARGUMENTS}", `{"tool":"bash"}`, `Check: {"tool":"bash"}`},
		{"no args", "Check something", `{"tool":"bash"}`, "Check something"},
		{"multiple", "$ARGUMENTS and $ARGUMENTS", `x`, "x and x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := substituteArguments(tt.prompt, tt.jsonIn)
			if got != tt.expected {
				t.Errorf("substituteArguments(%q, %q) = %q, want %q", tt.prompt, tt.jsonIn, got, tt.expected)
			}
		})
	}
}
