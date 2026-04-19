package main

import (
	"fmt"
	"os"

	"goc/gou/message"
	"goc/gou/theme"
	"goc/types"
)

func main() {
	// Test basic message rendering
	fmt.Println("Testing message rendering system...")

	// Create test messages
	testMessages := []struct {
		name    string
		message *types.Message
	}{
		{
			name: "User text message",
			message: &types.Message{
				Type:    types.MessageTypeUser,
				UUID:    "test-user-1",
				Content: []byte(`"Hello, this is a user message"`),
			},
		},
		{
			name: "Assistant text message",
			message: &types.Message{
				Type:    types.MessageTypeAssistant,
				UUID:    "test-assistant-1",
				Content: []byte(`[{"type":"text","text":"This is an assistant response with **markdown** support."}]`),
			},
		},
		{
			name: "System informational message",
			message: &types.Message{
				Type:    types.MessageTypeSystem,
				UUID:    "test-system-1",
				Subtype: stringPtr("informational"),
				Content: []byte(`"System notification"`),
			},
		},
		{
			name: "System error message",
			message: &types.Message{
				Type:    types.MessageTypeSystem,
				UUID:    "test-system-2",
				Subtype: stringPtr("api_error"),
				Content: []byte(`"API error: Invalid API key"`),
			},
		},
	}

	// Create renderer
	dispatcher := message.NewDispatcher()
	theme := theme.ActivePalette()

	for _, test := range testMessages {
		fmt.Printf("\n=== Testing: %s ===\n", test.name)
		fmt.Printf("Message type: %s\n", test.message.Type)

		ctx := &message.RenderContext{
			Width:         80,
			Theme:         theme,
			Verbose:       true,
			IsTranscript:  false,
			IsStatic:      false,
			ShouldAnimate: false,
			ShouldShowDot: false,
			AddMargin:     true,
		}

		lines, err := dispatcher.Render(test.message, ctx)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Rendered lines (%d):\n", len(lines))
		for i, line := range lines {
			fmt.Printf("  [%02d] %s\n", i+1, line)
		}
	}

	// Test virtual list
	fmt.Println("\n=== Testing Virtual List ===")
	vlist := message.NewVirtualList()

	testMessagesList := []*types.Message{
		{
			Type:    types.MessageTypeUser,
			UUID:    "msg-1",
			Content: []byte(`"First user message"`),
		},
		{
			Type:    types.MessageTypeAssistant,
			UUID:    "msg-2",
			Content: []byte(`[{"type":"text","text":"First assistant response"}]`),
		},
		{
			Type:    types.MessageTypeUser,
			UUID:    "msg-3",
			Content: []byte(`"Second user message"`),
		},
	}

	ctx := &message.RenderContext{
		Width:   80,
		Theme:   theme,
		Verbose: true,
	}

	start, end, total := vlist.ComputeVisibleRange(testMessagesList, 0, 10, ctx)
	fmt.Printf("Visible range: start=%d, end=%d, totalHeight=%d\n", start, end, total)

	lines, err := vlist.RenderRange(testMessagesList, start, end, ctx)
	if err != nil {
		fmt.Printf("Error rendering range: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Rendered %d lines:\n", len(lines))
	for i, line := range lines {
		fmt.Printf("  [%02d] %s\n", i+1, line)
	}

	// Test processor
	fmt.Println("\n=== Testing Message Processor ===")
	processor := message.NewProcessor()

	// Create messages that should be grouped
	toolMessages := []*types.Message{
		{
			Type:    types.MessageTypeAssistant,
			UUID:    "tool-1",
			Content: []byte(`[{"type":"tool_use","id":"1","name":"Read","input":{"file_path":"file1.txt"}}]`),
		},
		{
			Type:    types.MessageTypeAssistant,
			UUID:    "tool-2",
			Content: []byte(`[{"type":"tool_use","id":"2","name":"Read","input":{"file_path":"file2.txt"}}]`),
		},
		{
			Type:    types.MessageTypeAssistant,
			UUID:    "tool-3",
			Content: []byte(`[{"type":"tool_use","id":"3","name":"Bash","input":{"command":"ls -la"}}]`),
		},
	}

	processed := processor.Process(toolMessages, false)
	fmt.Printf("Original messages: %d, Processed: %d\n", len(toolMessages), len(processed))

	for i, msg := range processed {
		fmt.Printf("  [%d] Type: %s", i, msg.Type)
		if msg.Type == types.MessageTypeGroupedToolUse {
			fmt.Printf(" (Grouped: %s ×%d)", msg.ToolName, len(msg.Messages))
		}
		fmt.Println()
	}

	fmt.Println("\n=== Test Complete ===")
}

func stringPtr(s string) *string {
	return &s
}