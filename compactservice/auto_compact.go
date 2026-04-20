package compactservice

import (
	"context"
	"os"
	"strconv"

	"goc/types"
)

// Threshold buffer constants mirror src/services/compact/autoCompact.ts.
const (
	AutoCompactBufferTokens          = 13_000
	WarningThresholdBufferTokens     = 20_000
	ErrorThresholdBufferTokens       = 20_000
	ManualCompactBufferTokens        = 3_000
	MaxConsecutiveAutoCompactFailures = 3

	// MaxOutputTokensForSummary mirrors MAX_OUTPUT_TOKENS_FOR_SUMMARY in TS.
	MaxOutputTokensForSummary = 20_000
)

// ContextWindowResolver returns the model's context window. Mirrors getContextWindowForModel(model, betas) in TS.
// Callers pass betas via a ContextWindowInput struct to keep the signature compat with future overrides.
type ContextWindowResolver func(model string, betas []string) int

// MaxOutputTokensResolver returns the model's max-output-tokens cap (getMaxOutputTokensForModel in TS).
type MaxOutputTokensResolver func(model string) int

// Defaults: these keep in-package callers functional without wiring. Hosts provide the real
// per-model tables via CompactThresholds.
const defaultContextWindow = 200_000
const defaultMaxOutputTokens = 64_000

// CompactThresholds captures the subset of TS module-level dependencies that threshold math reads.
type CompactThresholds struct {
	ResolveContextWindow  ContextWindowResolver
	ResolveMaxOutputTokens MaxOutputTokensResolver
}

// resolve returns a ready-to-use ResolveX pair, falling back to documented defaults
// (200_000 context / 64_000 output) when hosts do not wire model-specific tables.
func (t CompactThresholds) resolve() (ContextWindowResolver, MaxOutputTokensResolver) {
	cw := t.ResolveContextWindow
	if cw == nil {
		cw = func(string, []string) int { return defaultContextWindow }
	}
	mt := t.ResolveMaxOutputTokens
	if mt == nil {
		mt = func(string) int { return defaultMaxOutputTokens }
	}
	return cw, mt
}

// GetEffectiveContextWindowSize mirrors getEffectiveContextWindowSize(model) in TS.
// Respects the CLAUDE_CODE_AUTO_COMPACT_WINDOW env override.
func GetEffectiveContextWindowSize(model string, betas []string, t CompactThresholds) int {
	cw, mt := t.resolve()
	reserved := mt(model)
	if reserved > MaxOutputTokensForSummary {
		reserved = MaxOutputTokensForSummary
	}
	contextWindow := cw(model, betas)
	if override := os.Getenv("CLAUDE_CODE_AUTO_COMPACT_WINDOW"); override != "" {
		if parsed, err := strconv.Atoi(override); err == nil && parsed > 0 {
			if parsed < contextWindow {
				contextWindow = parsed
			}
		}
	}
	return contextWindow - reserved
}

// GetAutoCompactThreshold mirrors getAutoCompactThreshold in TS.
// Respects CLAUDE_AUTOCOMPACT_PCT_OVERRIDE (0 < p ≤ 100 → floor(effectiveWindow * p/100)).
func GetAutoCompactThreshold(model string, betas []string, t CompactThresholds) int {
	effective := GetEffectiveContextWindowSize(model, betas, t)
	threshold := effective - AutoCompactBufferTokens

	if pct := os.Getenv("CLAUDE_AUTOCOMPACT_PCT_OVERRIDE"); pct != "" {
		if parsed, err := strconv.ParseFloat(pct, 64); err == nil && parsed > 0 && parsed <= 100 {
			pctThreshold := int(float64(effective) * parsed / 100.0)
			if pctThreshold < threshold {
				return pctThreshold
			}
		}
	}
	return threshold
}

// TokenWarningState mirrors the return shape of calculateTokenWarningState in TS.
type TokenWarningState struct {
	PercentLeft                  int
	IsAboveWarningThreshold      bool
	IsAboveErrorThreshold        bool
	IsAboveAutoCompactThreshold  bool
	IsAtBlockingLimit            bool
}

// CalculateTokenWarningState mirrors calculateTokenWarningState in TS.
// When auto-compact is enabled the threshold base is getAutoCompactThreshold; otherwise
// it is getEffectiveContextWindowSize. Comparisons use >= (TS parity).
func CalculateTokenWarningState(tokenUsage int, model string, betas []string, t CompactThresholds) TokenWarningState {
	autoCompactThreshold := GetAutoCompactThreshold(model, betas, t)
	enabled := IsAutoCompactEnabled()
	threshold := autoCompactThreshold
	if !enabled {
		threshold = GetEffectiveContextWindowSize(model, betas, t)
	}
	percentLeft := 0
	if threshold > 0 {
		left := float64(threshold-tokenUsage) / float64(threshold) * 100.0
		if left < 0 {
			left = 0
		}
		percentLeft = int(roundHalfUp(left))
	}

	warningThreshold := threshold - WarningThresholdBufferTokens
	errorThreshold := threshold - ErrorThresholdBufferTokens

	actualCW := GetEffectiveContextWindowSize(model, betas, t)
	defaultBlockingLimit := actualCW - ManualCompactBufferTokens
	blockingLimit := defaultBlockingLimit
	if override := os.Getenv("CLAUDE_CODE_BLOCKING_LIMIT_OVERRIDE"); override != "" {
		if parsed, err := strconv.Atoi(override); err == nil && parsed > 0 {
			blockingLimit = parsed
		}
	}

	return TokenWarningState{
		PercentLeft:                 percentLeft,
		IsAboveWarningThreshold:     tokenUsage >= warningThreshold,
		IsAboveErrorThreshold:       tokenUsage >= errorThreshold,
		IsAboveAutoCompactThreshold: enabled && tokenUsage >= autoCompactThreshold,
		IsAtBlockingLimit:           tokenUsage >= blockingLimit,
	}
}

func roundHalfUp(f float64) float64 {
	if f >= 0 {
		return float64(int(f + 0.5))
	}
	return float64(int(f - 0.5))
}

// IsAutoCompactEnabled mirrors isAutoCompactEnabled in TS, minus the global-config
// `userConfig.autoCompactEnabled` check (Go's global config is read via host).
// Hosts that need config-driven disable wire this through CompactThresholds in a
// follow-up; for now the env kill-switches are the only gate.
func IsAutoCompactEnabled() bool {
	if IsEnvTruthy("DISABLE_COMPACT") {
		return false
	}
	if IsEnvTruthy("DISABLE_AUTO_COMPACT") {
		return false
	}
	return true
}

// AutoCompactTrackingState mirrors AutoCompactTrackingState in TS.
type AutoCompactTrackingState struct {
	Compacted            bool
	TurnCounter          int
	TurnID               string
	ConsecutiveFailures  int
}

// ShouldAutoCompactInput groups the inputs to ShouldAutoCompact to avoid a long argument list.
type ShouldAutoCompactInput struct {
	Messages        []types.Message
	Model           string
	Betas           []string
	QuerySource     string
	SnipTokensFreed int
	Thresholds      CompactThresholds
}

// ShouldAutoCompact mirrors shouldAutoCompact(messages, model, querySource, snipTokensFreed).
// Recursion guards + killswitches + enabled check + calculateTokenWarningState.
func ShouldAutoCompact(in ShouldAutoCompactInput) bool {
	switch in.QuerySource {
	case "session_memory", "compact":
		return false
	}
	if !IsAutoCompactEnabled() {
		return false
	}
	tokenCount := TokenCountWithEstimation(in.Messages) - in.SnipTokensFreed
	return CalculateTokenWarningState(tokenCount, in.Model, in.Betas, in.Thresholds).IsAboveAutoCompactThreshold
}

// AutoCompactIfNeededInput groups inputs to AutoCompactIfNeeded to mirror the TS call shape.
type AutoCompactIfNeededInput struct {
	Messages        []types.Message
	Model           string
	AgentID         string
	Betas           []string
	QuerySource     string
	Tracking        *AutoCompactTrackingState
	SnipTokensFreed int
	Thresholds      CompactThresholds
	Deps            Deps
	// ToolUseContext is optional runtime context passed through to CompactConversation
	// for post-compact attachment re-injection.
	ToolUseContext *types.ToolUseContext
}

// AutoCompactIfNeededResult mirrors the TS return shape.
type AutoCompactIfNeededResult struct {
	WasCompacted        bool
	CompactionResult    *CompactionResult
	ConsecutiveFailures *int
}

// AutoCompactIfNeeded mirrors autoCompactIfNeeded in services/compact/autoCompact.ts.
// Circuit-breaker, kill-switch, and query-source guards are honored; trySessionMemoryCompaction
// is NOT yet ported (TODO: Go has no session-memory subsystem — hosts override Deps.Summarize
// with a memory-aware implementation if they need that optimization).
func AutoCompactIfNeeded(ctx context.Context, in AutoCompactIfNeededInput) (AutoCompactIfNeededResult, error) {
	if IsEnvTruthy("DISABLE_COMPACT") {
		return AutoCompactIfNeededResult{}, nil
	}

	if in.Tracking != nil && in.Tracking.ConsecutiveFailures >= MaxConsecutiveAutoCompactFailures {
		return AutoCompactIfNeededResult{}, nil
	}

	if !ShouldAutoCompact(ShouldAutoCompactInput{
		Messages:        in.Messages,
		Model:           in.Model,
		Betas:           in.Betas,
		QuerySource:     in.QuerySource,
		SnipTokensFreed: in.SnipTokensFreed,
		Thresholds:      in.Thresholds,
	}) {
		return AutoCompactIfNeededResult{}, nil
	}

	recomp := &RecompactionInfo{
		IsRecompactionInChain:     in.Tracking != nil && in.Tracking.Compacted,
		TurnsSincePreviousCompact: -1,
		AutoCompactThreshold:      GetAutoCompactThreshold(in.Model, in.Betas, in.Thresholds),
		QuerySource:               in.QuerySource,
	}
	if in.Tracking != nil {
		recomp.TurnsSincePreviousCompact = in.Tracking.TurnCounter
		recomp.PreviousCompactTurnID = in.Tracking.TurnID
	}

	result, err := CompactConversation(ctx, in.Messages, in.Deps, CompactOptions{
		SuppressFollowUpQuestions: true,
		IsAutoCompact:             true,
		RecompactionInfo:          recomp,
		Model:                     in.Model,
		AgentID:                   in.AgentID,
		ToolUseContext:            in.ToolUseContext,
	})
	if err != nil {
		prev := 0
		if in.Tracking != nil {
			prev = in.Tracking.ConsecutiveFailures
		}
		next := prev + 1
		return AutoCompactIfNeededResult{
			WasCompacted:        false,
			ConsecutiveFailures: &next,
		}, err
	}
	zero := 0
	return AutoCompactIfNeededResult{
		WasCompacted:        true,
		CompactionResult:    &result,
		ConsecutiveFailures: &zero,
	}, nil
}
