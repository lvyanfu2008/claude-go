package query

// BudgetTracker mirrors src/conversation-runtime/types/tokenBudgetLoop.ts BudgetTracker.
type BudgetTracker struct {
	ContinuationCount    int
	LastDeltaTokens      int
	LastGlobalTurnTokens int
	StartedAt            int64
}

// TokenBudgetContinueDecision mirrors TS TokenBudgetContinueDecision.
type TokenBudgetContinueDecision struct {
	Action            string // "continue"
	NudgeMessage      string
	ContinuationCount int
	Pct               int
	TurnTokens        int
	Budget            int
}

// TokenBudgetStopDecision mirrors TS TokenBudgetStopDecision completionEvent shape.
type TokenBudgetStopCompletion struct {
	ContinuationCount  int
	Pct                int
	TurnTokens         int
	Budget             int
	DiminishingReturns bool
	DurationMs         int
}

// TokenBudgetStopDecision mirrors TS TokenBudgetStopDecision.
type TokenBudgetStopDecision struct {
	Action          string // "stop"
	CompletionEvent *TokenBudgetStopCompletion
}

// TokenBudgetDecision is either continue or stop (TS union).
type TokenBudgetDecision struct {
	Continue *TokenBudgetContinueDecision
	Stop     *TokenBudgetStopDecision
}
