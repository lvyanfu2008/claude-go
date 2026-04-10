// Slash resolve result for Go TUI → ccb-engine (see docs/plans/go-slash-resolve.md).
package types

import (
	"encoding/json"

	"goc/utils"
)

// SlashResolveSource identifies how UserText was produced.
type SlashResolveSource string

const (
	SlashResolveDisk        SlashResolveSource = "disk"
	SlashResolveBundledEmbed SlashResolveSource = "bundled_embed"
	SlashResolveTSBridge    SlashResolveSource = "ts_bridge"
	SlashResolveUnknown     SlashResolveSource = "unknown"
)

// SlashBridgeMeta is required when Source is ts_bridge (observability).
type SlashBridgeMeta struct {
	BridgeVersion string `json:"bridgeVersion,omitempty"`
	LatencyMs     int64  `json:"latencyMs,omitempty"`
	RequestID     string `json:"requestId,omitempty"`
}

// SlashResolveResult is the structured payload after resolving a slash command (no bare string only).
type SlashResolveResult struct {
	UserText          string              `json:"userText"`
	AllowedTools      []string            `json:"allowedTools,omitempty"`
	Model             *string             `json:"model,omitempty"`
	Effort            *utils.EffortValue  `json:"effort,omitempty"`
	MaterializedPaths []string            `json:"materializedPaths,omitempty"`
	Source            SlashResolveSource  `json:"source"`
	BridgeMeta        *SlashBridgeMeta    `json:"bridgeMeta,omitempty"`
	SkippedReason     string              `json:"skippedReason,omitempty"`
	CommandJSON       json.RawMessage     `json:"commandJson,omitempty"` // optional echo of types.Command for TS bridge
}
