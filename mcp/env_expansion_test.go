package mcp

import (
	"os"
	"testing"
)

func TestExpandEnvVars(t *testing.T) {
	os.Setenv("TEST_VAR", "hello")
	defer os.Unsetenv("TEST_VAR")

	tests := []struct{ input, want string }{
		{"${TEST_VAR} world", "hello world"},
		{"${MISSING:-default}", "default"},
		{"${TEST_VAR:-fallback}", "hello"},
		{"no vars here", "no vars here"},
		{"", ""},
	}

	for _, tt := range tests {
		got := ExpandEnvVars(tt.input)
		if got != tt.want {
			t.Errorf("ExpandEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExpandMapEnvVars(t *testing.T) {
	os.Setenv("HOME_VAR", "/home/user")
	defer os.Unsetenv("HOME_VAR")

	input := map[string]string{
		"path": "${HOME_VAR}/projects",
		"name": "static",
	}
	result := ExpandMapEnvVars(input)
	if result["path"] != "/home/user/projects" {
		t.Errorf("expected /home/user/projects, got %q", result["path"])
	}
	if result["name"] != "static" {
		t.Errorf("expected static, got %q", result["name"])
	}
}
