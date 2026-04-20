package context

import (
	"encoding/json"
	"fmt"
	"time"
)

// CompactResult 压缩结果
type CompactResult struct {
	Success      bool      // 是否成功
	ErrorMessage string    // 错误信息
	CompactedAt  time.Time // 压缩时间
	Messages     []Message // 压缩后的消息
}

// RecompactionInfo 重新压缩信息
type RecompactionInfo struct {
	PreviousCompactId string // 前一个压缩ID
	Reason            string // 重新压缩原因
}

// CompactMetadata 压缩元数据
type CompactMetadata struct {
	PreservedSegment *PreservedSegment `json:"preservedSegment,omitempty"`
	Direction        string            `json:"direction,omitempty"`
	Model            string            `json:"model,omitempty"`
	Timestamp        time.Time         `json:"timestamp,omitempty"`
}

// PreservedSegment 保留的片段
type PreservedSegment struct {
	TailUuid string `json:"tailUuid,omitempty"`
	Count    int    `json:"count,omitempty"`
}

// CompactDirection 压缩方向
type CompactDirection string

const (
	CompactDirectionForward  CompactDirection = "forward"
	CompactDirectionBackward CompactDirection = "backward"
)

// CompactConfig 压缩配置
type CompactConfig struct {
	Model            string           // 模型名称
	MaxOutputTokens  int              // 最大输出令牌
	Direction        CompactDirection // 压缩方向
	PreserveRecent   int              // 保留最近的消息数
	IncludeAttachments bool           // 是否包含附件
}

// DefaultCompactConfig 默认压缩配置
func DefaultCompactConfig(model string) CompactConfig {
	return CompactConfig{
		Model:            model,
		MaxOutputTokens:  GetCompactReservedTokens(model),
		Direction:        CompactDirectionForward,
		PreserveRecent:   10, // 默认保留最近10条消息
		IncludeAttachments: true,
	}
}

// CompactConversation 压缩对话
func CompactConversation(
	messages []Message,
	config CompactConfig,
	toolContext interface{}, // 简化：实际应该是ToolUseContext
) (CompactResult, error) {
	if len(messages) == 0 {
		return CompactResult{
			Success:     true,
			CompactedAt: time.Now(),
			Messages:    messages,
		}, nil
	}

	// 检查是否需要压缩
	if !shouldCompact(messages, config) {
		return CompactResult{
			Success:     true,
			CompactedAt: time.Now(),
			Messages:    messages,
		}, nil
	}

	// 执行压缩逻辑
	compactedMessages, err := performCompaction(messages, config)
	if err != nil {
		return CompactResult{
			Success:      false,
			ErrorMessage: err.Error(),
			CompactedAt:  time.Now(),
		}, err
	}

	// 创建压缩边界消息
	boundaryMessage := createCompactBoundaryMessage(messages, config)

	// 将压缩边界消息添加到结果中
	resultMessages := append([]Message{boundaryMessage}, compactedMessages...)

	return CompactResult{
		Success:     true,
		CompactedAt: time.Now(),
		Messages:    resultMessages,
	}, nil
}

// 检查是否需要压缩
func shouldCompact(messages []Message, config CompactConfig) bool {
	// 估算当前消息的令牌数
	estimatedTokens := TokenCountWithEstimation(messages, config.Model)

	// 获取有效上下文窗口
	effectiveWindow := GetEffectiveContextWindowSize(config.Model, []string{})

	// 检查是否超过阈值
	return estimatedTokens > (effectiveWindow - AutoCompactBufferTokens)
}

// 执行压缩
func performCompaction(messages []Message, config CompactConfig) ([]Message, error) {
	// 简化实现：保留最近的消息
	preserveCount := config.PreserveRecent
	if preserveCount > len(messages) {
		preserveCount = len(messages)
	}

	// 保留最近的消息
	preservedMessages := messages[len(messages)-preserveCount:]

	// 创建摘要消息（简化实现）
	summaryMessage, err := createSummaryMessage(messages, config)
	if err != nil {
		return nil, err
	}

	// 组合结果：摘要 + 保留的消息
	result := []Message{summaryMessage}
	result = append(result, preservedMessages...)

	return result, nil
}

// 创建摘要消息
func createSummaryMessage(messages []Message, config CompactConfig) (Message, error) {
	// 简化实现：创建文本摘要
	summaryText := fmt.Sprintf("之前的对话已压缩。保留了最近%d条消息中的关键信息。", len(messages))

	// 创建消息结构
	messageData := map[string]interface{}{
		"role":    "assistant",
		"content": summaryText,
	}

	messageJSON, err := json.Marshal(messageData)
	if err != nil {
		return Message{}, err
	}

	return Message{
		Type:    "assistant",
		Message: messageJSON,
	}, nil
}

// 创建压缩边界消息
func createCompactBoundaryMessage(messages []Message, config CompactConfig) Message {
	metadata := CompactMetadata{
		Direction: string(config.Direction),
		Model:     config.Model,
		Timestamp: time.Now(),
	}

	// 如果有保留的片段，设置元数据
	if config.PreserveRecent > 0 && len(messages) > 0 {
		metadata.PreservedSegment = &PreservedSegment{
			Count: config.PreserveRecent,
		}
	}

	metadataJSON, _ := json.Marshal(metadata)

	return Message{
		Type:            "system",
		Subtype:         "compact_boundary",
		Message:         json.RawMessage(`{"role":"system","content":"对话压缩边界"}`),
		CompactMetadata: metadataJSON,
	}
}

// GetMessagesAfterCompactBoundary 获取压缩边界后的消息
func GetMessagesAfterCompactBoundary(messages []Message) []Message {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == "system" && messages[i].Subtype == "compact_boundary" {
			if i+1 < len(messages) {
				return messages[i+1:]
			}
			return []Message{}
		}
	}
	return messages
}

// IsCompactBoundaryMessage 检查是否是压缩边界消息
func IsCompactBoundaryMessage(message Message) bool {
	return message.Type == "system" && message.Subtype == "compact_boundary"
}

// AnalyzeContext 分析上下文
// Mirrors the threshold semantics of calculateTokenWarningState in
// claude-code/src/services/compact/autoCompact.ts: warning/error thresholds
// are derived from autoCompactThreshold (when auto-compact is enabled),
// not the raw effective window. ShouldCompact uses ">=" like TS.
func AnalyzeContext(messages []Message, model string, betas []string) ContextAnalysis {
	tokenCount := TokenCountWithEstimation(messages, model)
	effectiveWindow := GetEffectiveContextWindowSize(model, betas)
	autoCompactThreshold := GetAutoCompactThreshold(model, betas)

	threshold := autoCompactThreshold
	if !IsAutoCompactEnabled() {
		threshold = effectiveWindow
	}

	usagePercentage := 0
	if effectiveWindow > 0 {
		usagePercentage = int(float64(tokenCount) / float64(effectiveWindow) * 100)
		if usagePercentage > 100 {
			usagePercentage = 100
		}
	}

	return ContextAnalysis{
		TokenCount:       tokenCount,
		EffectiveWindow:  effectiveWindow,
		UsagePercentage:  usagePercentage,
		MessagesCount:    len(messages),
		ShouldCompact:    IsAutoCompactEnabled() && tokenCount >= autoCompactThreshold,
		CompactThreshold: autoCompactThreshold,
		WarningThreshold: threshold - WarningThresholdBufferTokens,
		ErrorThreshold:   threshold - ErrorThresholdBufferTokens,
	}
}

// ContextAnalysis 上下文分析结果
type ContextAnalysis struct {
	TokenCount       int  // 令牌数
	EffectiveWindow  int  // 有效窗口大小
	UsagePercentage  int  // 使用百分比
	MessagesCount    int  // 消息数量
	ShouldCompact    bool // 是否应该压缩
	CompactThreshold int  // 压缩阈值
	WarningThreshold int  // 警告阈值
	ErrorThreshold   int  // 错误阈值
}