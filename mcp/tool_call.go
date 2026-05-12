package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// ProcessMCPResult transforms an MCP CallToolResult into a string for the LLM.
// Mirrors TS processMCPResult / transformMCPResult.
func ProcessMCPResult(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	var parts []string

	for _, content := range result.Content {
		switch c := content.(type) {
		case mcp.TextContent:
			parts = append(parts, c.Text)
		case mcp.ImageContent:
			parts = append(parts, formatImageContent(c))
		case mcp.AudioContent:
			parts = append(parts, formatAudioContent(c))
		case mcp.EmbeddedResource:
			parts = append(parts, formatEmbeddedResource(c))
		default:
			if raw, err := json.Marshal(content); err == nil {
				parts = append(parts, string(raw))
			}
		}
	}

	// Include structured content if present.
	if result.StructuredContent != nil {
		structured, err := json.MarshalIndent(result.StructuredContent, "", "  ")
		if err == nil {
			parts = append(parts, fmt.Sprintf("\n[Structured Content]\n%s", string(structured)))
		}
	}

	output := strings.Join(parts, "\n")

	if result.IsError {
		return fmt.Sprintf("[MCP Tool Error]\n%s", output)
	}

	return output
}

// ProcessMCPResultStructured returns both the plain text and structured content separately.
func ProcessMCPResultStructured(result *mcp.CallToolResult) (plainText string, structured interface{}, isError bool) {
	if result == nil {
		return "", nil, false
	}

	var parts []string
	for _, content := range result.Content {
		switch c := content.(type) {
		case mcp.TextContent:
			parts = append(parts, c.Text)
		}
	}

	return strings.Join(parts, "\n"), result.StructuredContent, result.IsError
}

func formatImageContent(img mcp.ImageContent) string {
	return fmt.Sprintf("[Image: %s, mimeType: %s]", img.Data, img.MIMEType)
}

func formatAudioContent(audio mcp.AudioContent) string {
	return fmt.Sprintf("[Audio: %s, mimeType: %s]", audio.Data, audio.MIMEType)
}

func formatEmbeddedResource(res mcp.EmbeddedResource) string {
	// ResourceContents is an interface; extract what we can.
	raw, _ := json.Marshal(res.Resource)
	return fmt.Sprintf("[Resource: %s]", string(raw))
}

// TruncateMcpResult truncates a large MCP result to keep context usage manageable.
// Mirrors TS mcpContentNeedsTruncation / truncateMcpContentIfNeeded.
func TruncateMcpResult(result string, maxTokens int) string {
	if maxTokens <= 0 {
		maxTokens = 100000
	}
	// Rough token estimation: 1 token ≈ 4 chars.
	maxChars := maxTokens * 4
	if len(result) <= maxChars {
		return result
	}
	truncated := result[:maxChars]
	return truncated + fmt.Sprintf("\n\n[Output truncated. Original output was ~%d characters]", len(result))
}

// TokenCount estimates the token count of an MCP tool result.
func TokenCount(result string) int {
	// Rough estimate: 1 token ≈ 4 characters.
	return len(result) / 4
}

// MCPToolResultToJSON converts a CallToolResult to a JSON string for debugging/display.
func MCPToolResultToJSON(result *mcp.CallToolResult) (string, error) {
	if result == nil {
		return "{}", nil
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
