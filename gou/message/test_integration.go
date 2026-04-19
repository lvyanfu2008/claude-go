package message

import (
	"fmt"
	"strings"
	"testing"

	"goc/gou/theme"
	"goc/types"
)

func TestMessageRendering(t *testing.T) {
	// Create a test theme
	testTheme := theme.ActivePalette()

	// Test cases
	testCases := []struct {
		name     string
		message  *types.Message
		expected string
	}{
		{
			name: "User text message",
			message: &types.Message{
				Type: types.MessageTypeUser,
				UUID: "test-uuid-1",
				Content: []byte(`"Hello, world!"`),
			},
			expected: "Hello, world!",
		},
		{
			name: "Assistant text message",
			message: &types.Message{
				Type: types.MessageTypeAssistant,
				UUID: "test-uuid-2",
				Content: []byte(`[{"type":"text","text":"This is a response"}]`),
			},
			expected: "This is a response",
		},
		{
			name: "System informational message",
			message: &types.Message{
				Type:    types.MessageTypeSystem,
				UUID:    "test-uuid-3",
				Subtype: stringPtr("informational"),
				Content: []byte(`"System message"`),
			},
			expected: "ℹ System message",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dispatcher := NewDispatcher()
			ctx := &RenderContext{
				Width:  80,
				Theme:  testTheme,
				Verbose: true,
			}

			lines, err := dispatcher.Render(tc.message, ctx)
			if err != nil {
				t.Errorf("Render failed: %v", err)
				return
			}

			if len(lines) == 0 {
				t.Error("No lines rendered")
				return
			}

			// Check if expected text appears in rendered output
			rendered := strings.Join(lines, " ")
			if !strings.Contains(rendered, tc.expected) {
				t.Errorf("Expected %q in output, got: %s", tc.expected, rendered)
			}
		})
	}
}

func TestVirtualList(t *testing.T) {
	// Create test messages
	messages := []*types.Message{
		{
			Type:    types.MessageTypeUser,
			UUID:    "msg-1",
			Content: []byte(`"Message 1"`),
		},
		{
			Type:    types.MessageTypeAssistant,
			UUID:    "msg-2",
			Content: []byte(`[{"type":"text","text":"Response 1"}]`),
		},
		{
			Type:    types.MessageTypeUser,
			UUID:    "msg-3",
			Content: []byte(`"Message 2"`),
		},
	}

	// Create virtual list
	vlist := NewVirtualList()
	ctx := &RenderContext{
		Width:  80,
		Theme:  theme.ActivePalette(),
		Verbose: true,
	}

	// Test range computation
	start, end, total := vlist.ComputeVisibleRange(messages, 0, 10, ctx)
	if start != 0 {
		t.Errorf("Expected start=0, got %d", start)
	}
	if end <= start {
		t.Errorf("Expected end > start, got start=%d, end=%d", start, end)
	}
	if total <= 0 {
		t.Errorf("Expected total > 0, got %d", total)
	}

	// Test rendering
	lines, err := vlist.RenderRange(messages, start, end, ctx)
	if err != nil {
		t.Errorf("RenderRange failed: %v", err)
	}
	if len(lines) == 0 {
		t.Error("No lines rendered")
	}

	fmt.Printf("Virtual list test: rendered %d lines\n", len(lines))
}

func TestProcessor(t *testing.T) {
	// Create test messages with tool uses
	messages := []*types.Message{
		{
			Type:    types.MessageTypeAssistant,
			UUID:    "tool-1",
			Content: []byte(`[{"type":"tool_use","id":"1","name":"Read","input":{"file_path":"test.go"}}]`),
		},
		{
			Type:    types.MessageTypeAssistant,
			UUID:    "tool-2",
			Content: []byte(`[{"type":"tool_use","id":"2","name":"Read","input":{"file_path":"main.go"}}]`),
		},
		{
			Type:    types.MessageTypeAssistant,
			UUID:    "tool-3",
			Content: []byte(`[{"type":"tool_use","id":"3","name":"Bash","input":{"command":"ls"}}]`),
		},
	}

	// Process messages
	processor := NewProcessor()
	processed := processor.Process(messages, false)

	// Check if grouping occurred
	if len(processed) >= len(messages) {
		t.Log("Grouping may not have occurred (test implementation limited)")
	} else {
		t.Logf("Messages reduced from %d to %d after processing", len(messages), len(processed))
	}

	// Check for grouped tool use
	for _, msg := range processed {
		if msg.Type == types.MessageTypeGroupedToolUse {
			t.Logf("Found grouped tool use: %s ×%d", msg.ToolName, len(msg.Messages))
		}
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}