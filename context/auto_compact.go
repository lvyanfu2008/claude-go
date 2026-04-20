package context

import (
	"os"
	"strconv"
)

// AutoCompactTrackingState 自动压缩跟踪状态
// Mirrors AutoCompactTrackingState in claude-code/src/services/compact/autoCompact.ts.
type AutoCompactTrackingState struct {
	Compacted           bool   // 是否已压缩
	TurnCounter         int    // 轮次计数器
	TurnId              string // 每轮的唯一ID
	ConsecutiveFailures int    // 连续失败次数（熔断器）
}

// Buffer/threshold constants — mirror claude-code/src/services/compact/autoCompact.ts.
const (
	// AutoCompactBufferTokens = AUTOCOMPACT_BUFFER_TOKENS (13_000).
	AutoCompactBufferTokens = 13_000
	// WarningThresholdBufferTokens = WARNING_THRESHOLD_BUFFER_TOKENS (20_000).
	WarningThresholdBufferTokens = 20_000
	// ErrorThresholdBufferTokens = ERROR_THRESHOLD_BUFFER_TOKENS (20_000).
	// NOTE: TS currently sets warning and error buffers to the same value, so
	// IsAboveWarningThreshold and IsAboveErrorThreshold flip at the same point.
	ErrorThresholdBufferTokens = 20_000
	// ManualCompactBufferTokens = MANUAL_COMPACT_BUFFER_TOKENS (3_000).
	ManualCompactBufferTokens = 3_000
	// MaxConsecutiveAutocompactFailures = MAX_CONSECUTIVE_AUTOCOMPACT_FAILURES (3).
	MaxConsecutiveAutocompactFailures = 3
)

// IsAutoCompactEnabled mirrors isAutoCompactEnabled() in TS.
// Go host has no persisted user config; defaults to enabled unless an env
// var disables it. DISABLE_COMPACT disables everything (manual + auto);
// DISABLE_AUTO_COMPACT disables only auto-compact while leaving /compact alive.
func IsAutoCompactEnabled() bool {
	if isEnvTruthy(os.Getenv("DISABLE_COMPACT")) {
		return false
	}
	if isEnvTruthy(os.Getenv("DISABLE_AUTO_COMPACT")) {
		return false
	}
	return true
}

// GetAutoCompactThreshold 获取自动压缩阈值
// Mirrors getAutoCompactThreshold(model) in TS.
func GetAutoCompactThreshold(model string, betas []string) int {
	effectiveContextWindow := GetEffectiveContextWindowSize(model, betas)

	autocompactThreshold := effectiveContextWindow - AutoCompactBufferTokens

	// CLAUDE_AUTOCOMPACT_PCT_OVERRIDE — makes auto-compact fire earlier for
	// testing. Mirrors TS: takes the min of (percent-of-effective, default).
	if envPercent := os.Getenv("CLAUDE_AUTOCOMPACT_PCT_OVERRIDE"); envPercent != "" {
		if parsed, err := strconv.ParseFloat(envPercent, 64); err == nil && parsed > 0 && parsed <= 100 {
			// TS uses Math.floor; int() conversion truncates toward zero which
			// matches floor for non-negative values.
			percentageThreshold := int(float64(effectiveContextWindow) * (parsed / 100))
			if percentageThreshold < autocompactThreshold {
				return percentageThreshold
			}
		}
	}

	return autocompactThreshold
}

// CalculateTokenWarningState 计算令牌警告状态
// Mirrors calculateTokenWarningState(tokenUsage, model) in TS. In particular,
// warning/error thresholds are derived from the adjusted `threshold`
// (autoCompactThreshold when auto-compact is enabled, else effective window),
// NOT from the raw effective window. percentLeft is also based on `threshold`.
func CalculateTokenWarningState(tokenUsage int, model string, betas []string) TokenWarningState {
	effectiveContextWindow := GetEffectiveContextWindowSize(model, betas)
	autoCompactThreshold := GetAutoCompactThreshold(model, betas)

	enabled := IsAutoCompactEnabled()
	threshold := autoCompactThreshold
	if !enabled {
		threshold = effectiveContextWindow
	}

	// percentLeft — mirror TS: Math.max(0, Math.round(((threshold - tokenUsage) / threshold) * 100)).
	percentLeft := 0
	if threshold > 0 {
		raw := float64(threshold-tokenUsage) / float64(threshold) * 100
		// Go has no built-in round-half-to-even for int conversion; use
		// Math.round semantics (round half away from zero). For non-negative
		// values this is: floor(x + 0.5).
		if raw < 0 {
			percentLeft = 0
		} else {
			percentLeft = int(raw + 0.5)
		}
	}

	warningThreshold := threshold - WarningThresholdBufferTokens
	errorThreshold := threshold - ErrorThresholdBufferTokens

	isAboveWarningThreshold := tokenUsage >= warningThreshold
	isAboveErrorThreshold := tokenUsage >= errorThreshold

	isAboveAutoCompactThreshold := enabled && tokenUsage >= autoCompactThreshold

	// Blocking limit — CLAUDE_CODE_BLOCKING_LIMIT_OVERRIDE allows tests/ops to
	// clamp. Default = effective window - MANUAL_COMPACT_BUFFER_TOKENS.
	defaultBlockingLimit := effectiveContextWindow - ManualCompactBufferTokens
	blockingLimit := defaultBlockingLimit
	if override := os.Getenv("CLAUDE_CODE_BLOCKING_LIMIT_OVERRIDE"); override != "" {
		if parsed, err := strconv.Atoi(override); err == nil && parsed > 0 {
			blockingLimit = parsed
		}
	}
	isAtBlockingLimit := tokenUsage >= blockingLimit

	return TokenWarningState{
		PercentLeft:                 percentLeft,
		IsAboveWarningThreshold:     isAboveWarningThreshold,
		IsAboveErrorThreshold:       isAboveErrorThreshold,
		IsAboveAutoCompactThreshold: isAboveAutoCompactThreshold,
		IsAtBlockingLimit:           isAtBlockingLimit,
		EffectiveContextWindow:      effectiveContextWindow,
		AutoCompactThreshold:        autoCompactThreshold,
		WarningThreshold:            warningThreshold,
		ErrorThreshold:              errorThreshold,
		BlockingLimit:               blockingLimit,
	}
}

// TokenWarningState 令牌警告状态
// Mirrors the object returned by calculateTokenWarningState in TS, plus a few
// derived thresholds that Go callers surface to the UI.
type TokenWarningState struct {
	PercentLeft                 int  // 剩余百分比（基于 threshold，和 TS 一致）
	IsAboveWarningThreshold     bool // 是否超过警告阈值
	IsAboveErrorThreshold       bool // 是否超过错误阈值
	IsAboveAutoCompactThreshold bool // 是否超过自动压缩阈值
	IsAtBlockingLimit           bool // 是否已达阻塞上限（mirrors TS isAtBlockingLimit）
	EffectiveContextWindow      int  // 有效上下文窗口
	AutoCompactThreshold        int  // 自动压缩阈值
	WarningThreshold            int  // 警告阈值
	ErrorThreshold              int  // 错误阈值
	BlockingLimit               int  // 阻塞上限
}

// ShouldAutoCompact 检查是否应该自动压缩
// Mirrors shouldAutoCompact(...) in TS. The Go signature is token-based rather
// than messages-based (the token count is computed elsewhere and passed in);
// tracking state carries the circuit breaker + compacted flag.
func ShouldAutoCompact(tokenUsage int, model string, betas []string, trackingState *AutoCompactTrackingState) bool {
	if trackingState == nil {
		return false
	}

	// Kill-switch gates — matches TS (which short-circuits via isAutoCompactEnabled).
	if !IsAutoCompactEnabled() {
		return false
	}

	// Already compacted for this turn.
	if trackingState.Compacted {
		return false
	}

	// Circuit breaker — TS MAX_CONSECUTIVE_AUTOCOMPACT_FAILURES.
	if trackingState.ConsecutiveFailures >= MaxConsecutiveAutocompactFailures {
		return false
	}

	// Threshold check — reuse CalculateTokenWarningState so the flag
	// respects IsAutoCompactEnabled() / env overrides uniformly.
	state := CalculateTokenWarningState(tokenUsage, model, betas)
	return state.IsAboveAutoCompactThreshold
}

// GetManualCompactThreshold 获取手动压缩阈值
// Mirrors the threshold used by the /compact blocking logic (effective - MANUAL_COMPACT_BUFFER_TOKENS).
func GetManualCompactThreshold(model string, betas []string) int {
	effectiveContextWindow := GetEffectiveContextWindowSize(model, betas)
	return effectiveContextWindow - ManualCompactBufferTokens
}

// UpdateAutoCompactTracking 更新自动压缩跟踪状态
func UpdateAutoCompactTracking(trackingState *AutoCompactTrackingState, compacted bool) {
	if trackingState == nil {
		return
	}

	trackingState.TurnCounter++
	trackingState.Compacted = compacted

	if compacted {
		// Success — reset the circuit breaker.
		trackingState.ConsecutiveFailures = 0
	} else {
		trackingState.ConsecutiveFailures++
	}
}

// GetCompactReservedTokens 获取压缩保留的令牌数
func GetCompactReservedTokens(model string) int {
	reservedTokens := CompactMaxOutputTokens
	maxOutputTokens := GetMaxOutputTokensForModel(model)
	if maxOutputTokens < reservedTokens {
		return maxOutputTokens
	}
	return reservedTokens
}
