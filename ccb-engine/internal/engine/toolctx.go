package engine

import (
	"context"

	"goc/ccb-engine/internal/anthropic"
)

type toolRegistryCtxKey struct{}

// toolTurnContext carries registry + MCP pending hints for ToolSearch (mirrors TS getPendingServerNames on empty match).
type toolTurnContext struct {
	Tools                 []anthropic.ToolDefinition
	HasPendingMcpServers  bool
	PendingMcpServerNames []string
}

// ContextWithToolTurn attaches tools[] and optional MCP connecting metadata for this RunTurn.
func ContextWithToolTurn(ctx context.Context, tools []anthropic.ToolDefinition, hasPendingMcp bool, pendingMcpNames []string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	var names []string
	if len(pendingMcpNames) > 0 {
		names = append([]string(nil), pendingMcpNames...)
	}
	return context.WithValue(ctx, toolRegistryCtxKey{}, &toolTurnContext{
		Tools:                 tools,
		HasPendingMcpServers:  hasPendingMcp,
		PendingMcpServerNames: names,
	})
}

// ToolRegistryFromContext returns tools from [ContextWithToolTurn], or nil.
func ToolRegistryFromContext(ctx context.Context) []anthropic.ToolDefinition {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(toolRegistryCtxKey{}).(*toolTurnContext)
	if v == nil {
		return nil
	}
	return v.Tools
}

// MCPPendingsFromContext returns MCP connecting flag and optional server display names.
func MCPPendingsFromContext(ctx context.Context) (hasPending bool, serverNames []string) {
	if ctx == nil {
		return false, nil
	}
	v, _ := ctx.Value(toolRegistryCtxKey{}).(*toolTurnContext)
	if v == nil {
		return false, nil
	}
	return v.HasPendingMcpServers, v.PendingMcpServerNames
}
