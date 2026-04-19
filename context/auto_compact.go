package context

import (
	"os"
	"strconv"
)

// AutoCompactTrackingState 自动压缩跟踪状态
type AutoCompactTrackingState struct {
	Compacted          bool   // 是否已压缩
	TurnCounter        int    // 轮次计数器
	TurnId             string // 每轮的唯一ID
	ConsecutiveFailures int    // 连续失败次数
}

// 常量定义
const (
	AutoCompactBufferTokens        = 13_000  // 自动压缩缓冲区令牌
	WarningThresholdBufferTokens   = 20_000  // 警告阈值缓冲区令牌
	ErrorThresholdBufferTokens     = 10_000  // 错误阈值缓冲区令牌
	ManualCompactBufferTokens      = 3_000   // 手动压缩缓冲区令牌
	MaxConsecutiveAutocompactFailures = 3     // 最大连续自动压缩失败次数
)

// GetAutoCompactThreshold 获取自动压缩阈值
func GetAutoCompactThreshold(model string, betas []string) int {
	effectiveContextWindow := GetEffectiveContextWindowSize(model, betas)

	autocompactThreshold := effectiveContextWindow - AutoCompactBufferTokens

	// 允许通过环境变量覆盖自动压缩百分比
	if envPercent := os.Getenv("CLAUDE_AUTOCOMPACT_PCT_OVERRIDE"); envPercent != "" {
		if parsed, err := strconv.ParseFloat(envPercent, 64); err == nil && parsed > 0 && parsed <= 100 {
			percentageThreshold := int(float64(effectiveContextWindow) * (parsed / 100))
			if percentageThreshold < autocompactThreshold {
				return percentageThreshold
			}
		}
	}

	return autocompactThreshold
}

// CalculateTokenWarningState 计算令牌警告状态
func CalculateTokenWarningState(tokenUsage int, model string, betas []string) TokenWarningState {
	effectiveContextWindow := GetEffectiveContextWindowSize(model, betas)
	autoCompactThreshold := GetAutoCompactThreshold(model, betas)
	warningThreshold := effectiveContextWindow - WarningThresholdBufferTokens
	errorThreshold := effectiveContextWindow - ErrorThresholdBufferTokens

	percentLeft := 100
	if effectiveContextWindow > 0 {
		percentLeft = int(float64(effectiveContextWindow-tokenUsage) / float64(effectiveContextWindow) * 100)
		if percentLeft < 0 {
			percentLeft = 0
		}
	}

	return TokenWarningState{
		PercentLeft:               percentLeft,
		IsAboveWarningThreshold:   tokenUsage > warningThreshold,
		IsAboveErrorThreshold:     tokenUsage > errorThreshold,
		IsAboveAutoCompactThreshold: tokenUsage > autoCompactThreshold,
		EffectiveContextWindow:    effectiveContextWindow,
		AutoCompactThreshold:      autoCompactThreshold,
		WarningThreshold:          warningThreshold,
		ErrorThreshold:            errorThreshold,
	}
}

// TokenWarningState 令牌警告状态
type TokenWarningState struct {
	PercentLeft               int  // 剩余百分比
	IsAboveWarningThreshold   bool // 是否超过警告阈值
	IsAboveErrorThreshold     bool // 是否超过错误阈值
	IsAboveAutoCompactThreshold bool // 是否超过自动压缩阈值
	EffectiveContextWindow    int  // 有效上下文窗口
	AutoCompactThreshold      int  // 自动压缩阈值
	WarningThreshold          int  // 警告阈值
	ErrorThreshold            int  // 错误阈值
}

// ShouldAutoCompact 检查是否应该自动压缩
func ShouldAutoCompact(tokenUsage int, model string, betas []string, trackingState *AutoCompactTrackingState) bool {
	if trackingState == nil {
		return false
	}

	// 如果已经压缩过，则跳过
	if trackingState.Compacted {
		return false
	}

	// 如果连续失败次数超过限制，则跳过
	if trackingState.ConsecutiveFailures >= MaxConsecutiveAutocompactFailures {
		return false
	}

	// 检查是否超过自动压缩阈值
	state := CalculateTokenWarningState(tokenUsage, model, betas)
	return state.IsAboveAutoCompactThreshold
}

// GetManualCompactThreshold 获取手动压缩阈值
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
		// 压缩成功，重置失败计数器
		trackingState.ConsecutiveFailures = 0
	} else {
		// 压缩失败，增加失败计数器
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