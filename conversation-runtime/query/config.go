package query

// QueryConfigGates mirrors src/conversation-runtime/types/queryLoopConfig.ts QueryConfigGates.
type QueryConfigGates struct {
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
