package context

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestContextManagementIntegration(t *testing.T) {
	// 测试完整的上下文管理流程
	model := "claude-sonnet-4-6"
	betas := []string{}

	// 1. 测试上下文窗口计算
	contextWindow := GetContextWindowForModel(model, betas)
	if contextWindow != ModelContextWindowDefault {
		t.Errorf("GetContextWindowForModel() = %d, want %d", contextWindow, ModelContextWindowDefault)
	}

	// 2. 测试有效窗口大小（减去保留的输出令牌）
	effectiveWindow := GetEffectiveContextWindowSize(model, betas)
	if effectiveWindow <= 0 {
		t.Errorf("GetEffectiveContextWindowSize() = %d, want > 0", effectiveWindow)
	}

	// 3. 测试最大输出令牌
	maxOutputTokens := GetMaxOutputTokensForModel(model)
	if maxOutputTokens <= 0 {
		t.Errorf("GetMaxOutputTokensForModel() = %d, want > 0", maxOutputTokens)
	}

	// 4. 创建测试消息
	messages := make([]Message, 100)
	for i := 0; i < 100; i++ {
		content := json.RawMessage(fmt.Sprintf(`{"role":"user","content":"Message %d: This is a test message for context management integration testing."}`, i+1))
		messages[i] = Message{
			Type:    "user",
			Message: content,
		}
	}

	// 5. 分析上下文
	analysis := AnalyzeContext(messages, model, betas)
	if analysis.MessagesCount != len(messages) {
		t.Errorf("AnalyzeContext() MessagesCount = %d, want %d", analysis.MessagesCount, len(messages))
	}

	// 6. 测试令牌估算
	estimatedTokens := TokenCountWithEstimation(messages, model)
	if estimatedTokens <= 0 {
		t.Errorf("TokenCountWithEstimation() = %d, want > 0", estimatedTokens)
	}

	// 7. 测试警告状态计算
	warningState := CalculateTokenWarningState(estimatedTokens, model, betas)

	// 验证百分比计算
	if warningState.PercentLeft < 0 || warningState.PercentLeft > 100 {
		t.Errorf("PercentLeft = %d, want between 0 and 100", warningState.PercentLeft)
	}

	// 8. 测试自动压缩阈值
	autoCompactThreshold := GetAutoCompactThreshold(model, betas)
	if autoCompactThreshold <= 0 {
		t.Errorf("GetAutoCompactThreshold() = %d, want > 0", autoCompactThreshold)
	}

	// 9. 测试手动压缩阈值
	manualCompactThreshold := GetManualCompactThreshold(model, betas)
	if manualCompactThreshold <= 0 {
		t.Errorf("GetManualCompactThreshold() = %d, want > 0", manualCompactThreshold)
	}

	// 10. 测试压缩保留令牌
	compactReservedTokens := GetCompactReservedTokens(model)
	if compactReservedTokens <= 0 {
		t.Errorf("GetCompactReservedTokens() = %d, want > 0", compactReservedTokens)
	}

	// 11. 测试自动压缩决策
	trackingState := &AutoCompactTrackingState{
		Compacted:          false,
		TurnCounter:        1,
		TurnId:             "test-turn-1",
		ConsecutiveFailures: 0,
	}

	// 低于阈值时不应压缩
	if ShouldAutoCompact(autoCompactThreshold-1_000, model, betas, trackingState) {
		t.Errorf("ShouldAutoCompact() = true for token usage below threshold, want false")
	}

	// 高于阈值时应压缩
	if !ShouldAutoCompact(autoCompactThreshold+1_000, model, betas, trackingState) {
		t.Errorf("ShouldAutoCompact() = false for token usage above threshold, want true")
	}

	// 12. 测试压缩配置
	config := DefaultCompactConfig(model)
	if config.Model != model {
		t.Errorf("DefaultCompactConfig().Model = %s, want %s", config.Model, model)
	}
	if config.MaxOutputTokens != compactReservedTokens {
		t.Errorf("DefaultCompactConfig().MaxOutputTokens = %d, want %d", config.MaxOutputTokens, compactReservedTokens)
	}
	if config.PreserveRecent != 10 {
		t.Errorf("DefaultCompactConfig().PreserveRecent = %d, want 10", config.PreserveRecent)
	}

	// 13. 测试压缩边界消息检测
	// 创建压缩边界消息
	boundaryMessage := createCompactBoundaryMessage(messages, config)
	if !IsCompactBoundaryMessage(boundaryMessage) {
		t.Errorf("IsCompactBoundaryMessage() = false for boundary message, want true")
	}

	// 14. 测试获取压缩边界后的消息
	messagesWithBoundary := append([]Message{boundaryMessage}, messages...)
	messagesAfterBoundary := GetMessagesAfterCompactBoundary(messagesWithBoundary)
	if len(messagesAfterBoundary) != len(messages) {
		t.Errorf("GetMessagesAfterCompactBoundary() returned %d messages, want %d", len(messagesAfterBoundary), len(messages))
	}

	// 15. 测试令牌使用计算
	// 创建带有使用情况的消息
	usageMessage := Message{
		Type: "assistant",
		Message: json.RawMessage(`{
			"role": "assistant",
			"content": "Test response",
			"usage": {
				"input_tokens": 100,
				"output_tokens": 50,
				"cache_creation_input_tokens": 10,
				"cache_read_input_tokens": 5
			}
		}`),
	}

	usage := GetTokenUsage(usageMessage)
	if usage == nil {
		t.Errorf("GetTokenUsage() = nil, want usage data")
	} else {
		totalTokens := GetTokenCountFromUsage(*usage)
		expectedTokens := 100 + 50 + 10 + 5 // 165
		if totalTokens != expectedTokens {
			t.Errorf("GetTokenCountFromUsage() = %d, want %d", totalTokens, expectedTokens)
		}
	}

	// 16. 测试从最后一个API响应获取令牌数
	messagesWithUsage := append(messages, usageMessage)
	lastResponseTokens := TokenCountFromLastAPIResponse(messagesWithUsage)
	if lastResponseTokens <= 0 {
		t.Errorf("TokenCountFromLastAPIResponse() = %d, want > 0", lastResponseTokens)
	}

	// 17. 测试最终上下文令牌
	finalContextTokens := FinalContextTokensFromLastResponse(messagesWithUsage)
	if finalContextTokens <= 0 {
		t.Errorf("FinalContextTokensFromLastResponse() = %d, want > 0", finalContextTokens)
	}

	// 18. 测试令牌使用计算
	totalTokens, inputTokens, outputTokens := CalculateTokenUsage(messagesWithUsage)
	if totalTokens <= 0 {
		t.Errorf("CalculateTokenUsage() totalTokens = %d, want > 0", totalTokens)
	}
	if inputTokens <= 0 {
		t.Errorf("CalculateTokenUsage() inputTokens = %d, want > 0", inputTokens)
	}
	if outputTokens <= 0 {
		t.Errorf("CalculateTokenUsage() outputTokens = %d, want > 0", outputTokens)
	}

	// 19. 测试上下文百分比计算
	usedPct, remainingPct := CalculateContextPercentages(100_000, 10_000, 5_000, 200_000)
	if usedPct != 57 { // (100000 + 10000 + 5000) / 200000 * 100 = 57.5 -> 57
		t.Errorf("CalculateContextPercentages() usedPct = %d, want 57", usedPct)
	}
	if remainingPct != 43 { // 100 - 57 = 43
		t.Errorf("CalculateContextPercentages() remainingPct = %d, want 43", remainingPct)
	}

	t.Logf("Context management integration test passed:")
	t.Logf("  - Context window: %d tokens", contextWindow)
	t.Logf("  - Effective window: %d tokens", effectiveWindow)
	t.Logf("  - Max output tokens: %d", maxOutputTokens)
	t.Logf("  - Auto-compact threshold: %d tokens", autoCompactThreshold)
	t.Logf("  - Estimated tokens for %d messages: %d", len(messages), estimatedTokens)
	t.Logf("  - Context usage: %d%%", analysis.UsagePercentage)
}