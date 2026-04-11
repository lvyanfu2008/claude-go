package query

// QueryConfigGates mirrors src/conversation-runtime/types/queryLoopConfig.ts QueryConfigGates.
type QueryConfigGates struct {
	// StreamingParityPath is set when GOU_QUERY_STREAMING_PARITY is truthy (host opt-in for HTTP SSE parity).
	StreamingParityPath bool
	StreamingToolExecution bool
	EmitToolUseSummaries   bool
	IsAnt                  bool
	FastModeEnabled        bool
}

// QueryConfig mirrors src/conversation-runtime/types/queryLoopConfig.ts QueryConfig.
// SessionID is the TS SessionId string brand (opaque here).
type QueryConfig struct {
	SessionID string
	Gates     QueryConfigGates
}
