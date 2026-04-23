package tools

import (
	"testing"
)

func TestResolveAllowedToolsWithWildcard(t *testing.T) {
	available := []string{"read", "write", "shell", "grep"}
	
	tests := []struct {
		name        string
		definition  AgentDefinition
		expected    []string
		description string
	}{
		{
			name: "wildcard with no disallowed tools",
			definition: AgentDefinition{
				Tools: []string{"*"},
			},
			expected:    []string{"read", "write", "shell", "grep"},
			description: "Should include all available tools",
		},
		{
			name: "wildcard with disallowed tools",
			definition: AgentDefinition{
				Tools:           []string{"*"},
				DisallowedTools: []string{"shell"},
			},
			expected:    []string{"read", "write", "grep"},
			description: "Should include all tools except disallowed",
		},
		{
			name: "explicit tools only",
			definition: AgentDefinition{
				Tools: []string{"read", "write"},
			},
			expected:    []string{"read", "write"},
			description: "Should only include explicitly listed tools",
		},
		{
			name: "wildcard with additional explicit tools",
			definition: AgentDefinition{
				Tools: []string{"*", "custom-tool"},
			},
			expected:    []string{"read", "write", "shell", "grep", "custom-tool"},
			description: "Should include all available plus explicit tools",
		},
		{
			name: "empty tools list",
			definition: AgentDefinition{
				Tools: []string{},
			},
			expected:    []string{"read", "write", "shell", "grep"},
			description: "Empty tools should default to all available",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveAllowedTools(tt.definition, available)
			
			// Convert to map for easier comparison
			resultMap := make(map[string]bool)
			for _, tool := range result {
				resultMap[tool] = true
			}
			
			expectedMap := make(map[string]bool)
			for _, tool := range tt.expected {
				expectedMap[tool] = true
			}
			
			// Check lengths match
			if len(result) != len(tt.expected) {
				t.Errorf("Length mismatch: got %d, want %d. Got: %v, Want: %v", 
					len(result), len(tt.expected), result, tt.expected)
				return
			}
			
			// Check all expected tools are present
			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("Missing expected tool: %s. Got: %v, Want: %v", 
						expected, result, tt.expected)
				}
			}
			
			// Check no unexpected tools are present
			for _, actual := range result {
				if !expectedMap[actual] {
					t.Errorf("Unexpected tool: %s. Got: %v, Want: %v", 
						actual, result, tt.expected)
				}
			}
		})
	}
}

func TestAgentDefinitionFieldsPreservation(t *testing.T) {
	// Test that built-in agent definitions preserve all fields
	builtins := LoadAgentDefinitionsBuiltins()
	
	if len(builtins) == 0 {
		t.Skip("No built-in agents available")
	}
	
	// Find general-purpose agent
	var generalPurpose *AgentDefinition
	for i := range builtins {
		if builtins[i].AgentType == "general-purpose" {
			generalPurpose = &builtins[i]
			break
		}
	}
	
	if generalPurpose == nil {
		t.Fatal("general-purpose agent not found")
	}
	
	// Check that system prompt is preserved
	if generalPurpose.SystemPrompt == "" {
		t.Error("SystemPrompt should be preserved from built-in agent")
	}
	
	// Check that tools include wildcard
	hasWildcard := false
	for _, tool := range generalPurpose.Tools {
		if tool == "*" {
			hasWildcard = true
			break
		}
	}
	
	if !hasWildcard {
		t.Error("general-purpose agent should have wildcard tool access")
	}
}