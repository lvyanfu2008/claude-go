package mcp

import (
	"fmt"
	"os"
	"strings"
)

// buildClaudeAIProxyURL builds the proxy URL for a claude.ai MCP proxy server.
// Mirrors TS: ${MCP_PROXY_URL}${MCP_PROXY_PATH.replace('{server_id}', id)}
func buildClaudeAIProxyURL(serverID string) string {
	base := strings.TrimRight(os.Getenv("CLAUDE_CODE_MCP_PROXY_URL"), "/")
	path := os.Getenv("CLAUDE_CODE_MCP_PROXY_PATH")
	if path == "" {
		path = "/mcp/{server_id}"
	}
	path = strings.ReplaceAll(path, "{server_id}", serverID)
	if base == "" {
		base = "https://claude.ai"
	}
	return fmt.Sprintf("%s%s", base, path)
}

// getClaudeAIOAuthToken returns the OAuth access token for claude.ai proxy requests.
func getClaudeAIOAuthToken() string {
	return os.Getenv("CLAUDE_CODE_OAUTH_ACCESS_TOKEN")
}
