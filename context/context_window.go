package context

import (
	"os"
	"strconv"
	"strings"
)

// ModelContextWindowDefault 默认上下文窗口大小（200k令牌）
const ModelContextWindowDefault = 200_000

// CompactMaxOutputTokens 压缩操作的最大输出令牌
const CompactMaxOutputTokens = 20_000

// MaxOutputTokensDefault 默认最大输出令牌
const MaxOutputTokensDefault = 32_000

// MaxOutputTokensUpperLimit 最大输出令牌上限
const MaxOutputTokensUpperLimit = 64_000

// CappedDefaultMaxTokens  capped默认最大令牌
const CappedDefaultMaxTokens = 8_000

// EscalatedMaxTokens 升级后的最大令牌
const EscalatedMaxTokens = 64_000

// ContextWindow 表示模型的上下文窗口配置
type ContextWindow struct {
	// 模型名称
	Model string
	// 上下文窗口大小（令牌数）
	Size int
	// 最大输出令牌
	MaxOutputTokens int
	// 最大输出令牌上限
	MaxOutputTokensUpperLimit int
	// 是否支持1M上下文
	Supports1M bool
}

// Is1mContextDisabled 检查是否通过环境变量禁用1M上下文
func Is1mContextDisabled() bool {
	return isEnvTruthy(os.Getenv("CLAUDE_CODE_DISABLE_1M_CONTEXT"))
}

// Has1mContext 检查模型是否具有[1m]后缀
func Has1mContext(model string) bool {
	return strings.Contains(strings.ToLower(model), "[1m]")
}

// ModelSupports1M 检查模型是否支持1M上下文
func ModelSupports1M(model string) bool {
	if Is1mContextDisabled() {
		return false
	}

	// 简化实现：检查是否包含特定模型名称
	// 实际实现应该更复杂，检查模型能力
	modelLower := strings.ToLower(model)
	return strings.Contains(modelLower, "claude-sonnet-4") ||
		strings.Contains(modelLower, "opus-4-6") ||
		strings.Contains(modelLower, "sonnet-4-6")
}

// GetContextWindowForModel 获取模型的上下文窗口大小
func GetContextWindowForModel(model string, betas []string) int {
	// 允许通过环境变量覆盖（仅限ant用户）
	if os.Getenv("USER_TYPE") == "ant" {
		if overrideStr := os.Getenv("CLAUDE_CODE_MAX_CONTEXT_TOKENS"); overrideStr != "" {
			if override, err := strconv.Atoi(overrideStr); err == nil && override > 0 {
				return override
			}
		}
	}

	// [1m]后缀 - 显式客户端选择，优先级最高
	if Has1mContext(model) {
		return 1_000_000
	}

	// 检查是否在beta中启用了1M上下文
	if containsBeta(betas, "context-1m") && ModelSupports1M(model) {
		return 1_000_000
	}

	// 默认返回200k
	return ModelContextWindowDefault
}

// GetEffectiveContextWindowSize 获取有效的上下文窗口大小（减去保留的输出令牌）
func GetEffectiveContextWindowSize(model string, betas []string) int {
	contextWindow := GetContextWindowForModel(model, betas)

	// 为输出保留令牌
	reservedTokens := CompactMaxOutputTokens
	maxOutputTokens := GetMaxOutputTokensForModel(model)
	if maxOutputTokens < reservedTokens {
		reservedTokens = maxOutputTokens
	}

	// 允许通过环境变量覆盖自动压缩窗口
	if autoCompactWindow := os.Getenv("CLAUDE_CODE_AUTO_COMPACT_WINDOW"); autoCompactWindow != "" {
		if parsed, err := strconv.Atoi(autoCompactWindow); err == nil && parsed > 0 {
			if parsed < contextWindow {
				contextWindow = parsed
			}
		}
	}

	return contextWindow - reservedTokens
}

// GetMaxOutputTokensForModel 获取模型的最大输出令牌
func GetMaxOutputTokensForModel(model string) int {
	modelLower := strings.ToLower(model)

	// 简化实现，实际应该更复杂
	if strings.Contains(modelLower, "opus-4-6") {
		return 64_000
	} else if strings.Contains(modelLower, "sonnet-4-6") {
		return 32_000
	} else if strings.Contains(modelLower, "opus-4-5") ||
		strings.Contains(modelLower, "sonnet-4") ||
		strings.Contains(modelLower, "haiku-4") {
		return 32_000
	} else if strings.Contains(modelLower, "opus-4-1") || strings.Contains(modelLower, "opus-4") {
		return 32_000
	} else if strings.Contains(modelLower, "claude-3-opus") {
		return 4_096
	} else if strings.Contains(modelLower, "claude-3-sonnet") {
		return 8_192
	} else if strings.Contains(modelLower, "claude-3-haiku") {
		return 4_096
	} else if strings.Contains(modelLower, "3-5-sonnet") || strings.Contains(modelLower, "3-5-haiku") {
		return 8_192
	} else if strings.Contains(modelLower, "3-7-sonnet") {
		return 32_000
	}

	return MaxOutputTokensDefault
}

// GetMaxOutputTokensUpperLimit 获取模型的最大输出令牌上限
func GetMaxOutputTokensUpperLimit(model string) int {
	modelLower := strings.ToLower(model)

	// 简化实现
	if strings.Contains(modelLower, "opus-4-6") {
		return 128_000
	} else if strings.Contains(modelLower, "sonnet-4-6") {
		return 128_000
	} else if strings.Contains(modelLower, "opus-4-5") ||
		strings.Contains(modelLower, "sonnet-4") ||
		strings.Contains(modelLower, "haiku-4") {
		return 64_000
	} else if strings.Contains(modelLower, "opus-4-1") || strings.Contains(modelLower, "opus-4") {
		return 32_000
	}

	return MaxOutputTokensUpperLimit
}

// GetMaxThinkingTokensForModel 获取模型的最大思考令牌
func GetMaxThinkingTokensForModel(model string) int {
	// 最大思考令牌应该严格小于最大输出令牌
	return GetMaxOutputTokensUpperLimit(model) - 1
}

// CalculateContextPercentages 计算上下文使用百分比
func CalculateContextPercentages(
	inputTokens int,
	cacheCreationInputTokens int,
	cacheReadInputTokens int,
	contextWindowSize int,
) (usedPercentage int, remainingPercentage int) {
	totalInputTokens := inputTokens + cacheCreationInputTokens + cacheReadInputTokens

	if contextWindowSize == 0 {
		return 0, 100
	}

	usedPercentage = int(float64(totalInputTokens) / float64(contextWindowSize) * 100)
	if usedPercentage > 100 {
		usedPercentage = 100
	} else if usedPercentage < 0 {
		usedPercentage = 0
	}

	remainingPercentage = 100 - usedPercentage
	return usedPercentage, remainingPercentage
}

// 辅助函数
func isEnvTruthy(value string) bool {
	if value == "" {
		return false
	}
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func containsBeta(betas []string, beta string) bool {
	for _, b := range betas {
		if strings.EqualFold(b, beta) {
			return true
		}
	}
	return false
}