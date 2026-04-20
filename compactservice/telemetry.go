package compactservice

// CompactEvent mirrors the tengu_compact payload shape. We keep individual fields rather
// than a generic map so the host-side logger gets a typed, easily-searchable struct.
type CompactEvent struct {
	PreCompactTokenCount         int     `json:"preCompactTokenCount"`
	PostCompactTokenCount        int     `json:"postCompactTokenCount"`
	TruePostCompactTokenCount    int     `json:"truePostCompactTokenCount"`
	AutoCompactThreshold         int     `json:"autoCompactThreshold"`
	WillRetriggerNextTurn        bool    `json:"willRetriggerNextTurn"`
	IsAutoCompact                bool    `json:"isAutoCompact"`
	QuerySource                  string  `json:"querySource,omitempty"`
	QueryChainID                 string  `json:"queryChainId,omitempty"`
	QueryDepth                   int     `json:"queryDepth,omitempty"`
	IsRecompactionInChain        bool    `json:"isRecompactionInChain"`
	TurnsSincePreviousCompact    int     `json:"turnsSincePreviousCompact"`
	PreviousCompactTurnID        string  `json:"previousCompactTurnId,omitempty"`
	CompactionInputTokens        int     `json:"compactionInputTokens,omitempty"`
	CompactionOutputTokens       int     `json:"compactionOutputTokens,omitempty"`
	CompactionCacheReadTokens    int     `json:"compactionCacheReadTokens,omitempty"`
	CompactionCacheCreationTokens int    `json:"compactionCacheCreationTokens,omitempty"`
	CompactionTotalTokens        int     `json:"compactionTotalTokens,omitempty"`
	PromptCacheSharingEnabled    bool    `json:"promptCacheSharingEnabled"`
}

// CompactFailedEvent mirrors the tengu_compact_failed payload.
type CompactFailedEvent struct {
	Reason                    string `json:"reason"`
	PreCompactTokenCount      int    `json:"preCompactTokenCount"`
	PromptCacheSharingEnabled bool   `json:"promptCacheSharingEnabled"`
	PtlAttempts               int    `json:"ptlAttempts,omitempty"`
}

// CompactPTLRetryEvent mirrors tengu_compact_ptl_retry.
type CompactPTLRetryEvent struct {
	Attempt            int    `json:"attempt"`
	DroppedMessages    int    `json:"droppedMessages"`
	RemainingMessages  int    `json:"remainingMessages"`
	Path               string `json:"path,omitempty"`
}

// Logger is the injection point for telemetry. Default NoopLogger drops events.
type Logger interface {
	LogCompact(CompactEvent)
	LogCompactFailed(CompactFailedEvent)
	LogCompactPTLRetry(CompactPTLRetryEvent)
}

// NoopLogger is the safe default.
type NoopLogger struct{}

func (NoopLogger) LogCompact(CompactEvent)                {}
func (NoopLogger) LogCompactFailed(CompactFailedEvent)    {}
func (NoopLogger) LogCompactPTLRetry(CompactPTLRetryEvent) {}
