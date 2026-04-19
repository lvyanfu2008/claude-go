package context

import (
	"encoding/json"
	"strings"
)

// Usage 表示API使用情况
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	OutputTokens             int `json:"output_tokens"`
}

// Message 表示消息
type Message struct {
	Type            string          `json:"type,omitempty"`
	Subtype         string          `json:"subtype,omitempty"`
	Message         json.RawMessage `json:"message,omitempty"`
	Model           string          `json:"model,omitempty"`
	CompactMetadata json.RawMessage `json:"compactMetadata,omitempty"`
}

// GetTokenUsage 从消息中获取令牌使用情况
func GetTokenUsage(message Message) *Usage {
	if message.Type != "assistant" || len(message.Message) == 0 {
		return nil
	}

	// 检查是否是合成消息
	if isSyntheticMessage(message) {
		return nil
	}

	// 解析消息以获取使用情况
	var msgData map[string]interface{}
	if err := json.Unmarshal(message.Message, &msgData); err != nil {
		return nil
	}

	usageData, ok := msgData["usage"]
	if !ok {
		return nil
	}

	usageJSON, err := json.Marshal(usageData)
	if err != nil {
		return nil
	}

	var usage Usage
	if err := json.Unmarshal(usageJSON, &usage); err != nil {
		return nil
	}

	return &usage
}

// GetTokenCountFromUsage 从使用情况计算总令牌数
func GetTokenCountFromUsage(usage Usage) int {
	return usage.InputTokens +
		usage.CacheCreationInputTokens +
		usage.CacheReadInputTokens +
		usage.OutputTokens
}

// TokenCountFromLastAPIResponse 从最后一个API响应计算令牌数
func TokenCountFromLastAPIResponse(messages []Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		usage := GetTokenUsage(messages[i])
		if usage != nil {
			return GetTokenCountFromUsage(*usage)
		}
	}
	return 0
}

// FinalContextTokensFromLastResponse 从最后一个响应获取最终上下文令牌数
func FinalContextTokensFromLastResponse(messages []Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		usage := GetTokenUsage(messages[i])
		if usage != nil {
			// 简化实现：返回输入+输出令牌（不包括缓存令牌）
			return usage.InputTokens + usage.OutputTokens
		}
	}
	return 0
}

// TokenCountWithEstimation 使用估算计算消息的令牌数
func TokenCountWithEstimation(messages []Message, model string) int {
	// 首先尝试从最后一个API响应获取准确计数
	lastResponseTokens := TokenCountFromLastAPIResponse(messages)
	if lastResponseTokens > 0 {
		return lastResponseTokens
	}

	// 如果没有API响应数据，使用估算
	return estimateTokenCount(messages, model)
}

// 估算令牌数（简化实现）
func estimateTokenCount(messages []Message, model string) int {
	// 简化实现：基于消息内容长度估算
	// 实际实现应该使用更准确的令牌化
	totalTokens := 0

	for _, msg := range messages {
		// 估算文本内容的令牌数
		if len(msg.Message) > 0 {
			// 粗略估算：每4个字符约1个令牌
			textLength := len(msg.Message)
			tokens := textLength / 4
			if tokens < 1 {
				tokens = 1
			}
			totalTokens += tokens
		}

		// 添加消息类型的开销
		totalTokens += 10 // 消息格式开销
	}

	return totalTokens
}

// 检查是否是合成消息
func isSyntheticMessage(message Message) bool {
	if message.Model == "synthetic" {
		return true
	}

	// 检查消息内容是否包含合成标记
	var content []interface{}
	if err := json.Unmarshal(message.Message, &map[string]interface{}{
		"content": &content,
	}); err == nil {
		if len(content) > 0 {
			if firstItem, ok := content[0].(map[string]interface{}); ok {
				if text, ok := firstItem["text"].(string); ok {
					// 检查是否是已知的合成消息
					syntheticTexts := []string{
						"SYNTHETIC_MESSAGE",
						"SYNTHETIC_TOOL_RESULT",
						"SYNTHETIC_ERROR",
					}
					for _, syntheticText := range syntheticTexts {
						if strings.Contains(text, syntheticText) {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// CalculateTokenUsage 计算总令牌使用情况
func CalculateTokenUsage(messages []Message) (totalTokens int, inputTokens int, outputTokens int) {
	for _, msg := range messages {
		usage := GetTokenUsage(msg)
		if usage != nil {
			totalTokens += GetTokenCountFromUsage(*usage)
			inputTokens += usage.InputTokens + usage.CacheCreationInputTokens + usage.CacheReadInputTokens
			outputTokens += usage.OutputTokens
		}
	}
	return totalTokens, inputTokens, outputTokens
}