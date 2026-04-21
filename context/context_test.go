package context

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGetContextWindowForModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		betas    []string
		expected int
	}{
		{
			name:     "Default model",
			model:    "claude-3-haiku",
			betas:    []string{},
			expected: ModelContextWindowDefault,
		},
		{
			name:     "Model with 1m suffix",
			model:    "claude-3-sonnet[1m]",
			betas:    []string{},
			expected: 1_000_000,
		},
		{
			name:     "Sonnet 4.6 with 1m beta",
			model:    "claude-sonnet-4-6",
			betas:    []string{"context-1m"},
			expected: 1_000_000,
		},
		{
			name:     "Opus 4.6 with 1m beta",
			model:    "claude-opus-4-6",
			betas:    []string{"context-1m"},
			expected: 1_000_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetContextWindowForModel(tt.model, tt.betas)
			if result != tt.expected {
				t.Errorf("GetContextWindowForModel(%q, %v) = %d, want %d",
					tt.model, tt.betas, result, tt.expected)
			}
		})
	}
}

func TestGetMaxOutputTokensForModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected int
	}{
		{
			name:     "Opus 4.6",
			model:    "claude-opus-4-6",
			expected: 64_000,
		},
		{
			name:     "Sonnet 4.6",
			model:    "claude-sonnet-4-6",
			expected: 32_000,
		},
		{
			name:     "Haiku 4",
			model:    "claude-haiku-4",
			expected: 32_000,
		},
		{
			name:     "Claude 3 Opus",
			model:    "claude-3-opus",
			expected: 4_096,
		},
		{
			name:     "Default model",
			model:    "unknown-model",
			expected: MaxOutputTokensDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMaxOutputTokensForModel(tt.model)
			if result != tt.expected {
				t.Errorf("GetMaxOutputTokensForModel(%q) = %d, want %d",
					tt.model, result, tt.expected)
			}
		})
	}
}

func TestCalculateContextPercentages(t *testing.T) {
	tests := []struct {
		name                     string
		inputTokens              int
		cacheCreationInputTokens int
		cacheReadInputTokens     int
		contextWindowSize        int
		expectedUsed             int
		expectedRemaining        int
	}{
		{
			name:                     "50% usage",
			inputTokens:              50_000,
			cacheCreationInputTokens: 0,
			cacheReadInputTokens:     0,
			contextWindowSize:        100_000,
			expectedUsed:             50,
			expectedRemaining:        50,
		},
		{
			name:                     "With cache tokens",
			inputTokens:              40_000,
			cacheCreationInputTokens: 5_000,
			cacheReadInputTokens:     5_000,
			contextWindowSize:        100_000,
			expectedUsed:             50,
			expectedRemaining:        50,
		},
		{
			name:                     "Over 100% usage",
			inputTokens:              150_000,
			cacheCreationInputTokens: 0,
			cacheReadInputTokens:     0,
			contextWindowSize:        100_000,
			expectedUsed:             100,
			expectedRemaining:        0,
		},
		{
			name:                     "Zero window",
			inputTokens:              100_000,
			cacheCreationInputTokens: 0,
			cacheReadInputTokens:     0,
			contextWindowSize:        0,
			expectedUsed:             0,
			expectedRemaining:        100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			used, remaining := CalculateContextPercentages(
				tt.inputTokens,
				tt.cacheCreationInputTokens,
				tt.cacheReadInputTokens,
				tt.contextWindowSize,
			)

			if used != tt.expectedUsed {
				t.Errorf("CalculateContextPercentages() used = %d, want %d", used, tt.expectedUsed)
			}
			if remaining != tt.expectedRemaining {
				t.Errorf("CalculateContextPercentages() remaining = %d, want %d", remaining, tt.expectedRemaining)
			}
		})
	}
}

func TestGetAutoCompactThreshold(t *testing.T) {
	model := "claude-sonnet-4-6"
	betas := []string{}

	effectiveWindow := GetEffectiveContextWindowSize(model, betas)
	expectedThreshold := effectiveWindow - AutoCompactBufferTokens

	result := GetAutoCompactThreshold(model, betas)

	if result != expectedThreshold {
		t.Errorf("GetAutoCompactThreshold(%q, %v) = %d, want %d",
			model, betas, result, expectedThreshold)
	}
}

func TestCalculateTokenWarningState(t *testing.T) {
	model := "claude-sonnet-4-6"
	betas := []string{}

	effectiveWindow := GetEffectiveContextWindowSize(model, betas)
	autoCompactThreshold := GetAutoCompactThreshold(model, betas)
	// Mirrors TS: warning/error are derived from `threshold` (= autoCompactThreshold
	// when auto-compact is enabled), not from the raw effective window.
	warningThreshold := autoCompactThreshold - WarningThresholdBufferTokens
	errorThreshold := autoCompactThreshold - ErrorThresholdBufferTokens

	tests := []struct {
		name          string
		tokenUsage    int
		expectWarning bool
		expectError   bool
		expectCompact bool
	}{
		{
			name:          "Below all thresholds",
			tokenUsage:    effectiveWindow / 2,
			expectWarning: false,
			expectError:   false,
			expectCompact: false,
		},
		{
			name:       "Above auto-compact threshold",
			tokenUsage: autoCompactThreshold + 1_000,
			// Warning and error share the same buffer (20k) in TS, so both
			// flip well before auto-compact fires.
			expectWarning: true,
			expectError:   true,
			expectCompact: true,
		},
		{
			name:          "Above warning threshold (but below auto-compact)",
			tokenUsage:    warningThreshold + 1_000,
			expectWarning: true,
			expectError:   true, // warning buffer == error buffer in TS
			expectCompact: false,
		},
		{
			name:          "Above error threshold (same point as warning)",
			tokenUsage:    errorThreshold + 1_000,
			expectWarning: true,
			expectError:   true,
			expectCompact: false, // still below autoCompactThreshold
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := CalculateTokenWarningState(tt.tokenUsage, model, betas)

			if state.IsAboveWarningThreshold != tt.expectWarning {
				t.Errorf("IsAboveWarningThreshold = %v, want %v",
					state.IsAboveWarningThreshold, tt.expectWarning)
			}

			if state.IsAboveErrorThreshold != tt.expectError {
				t.Errorf("IsAboveErrorThreshold = %v, want %v",
					state.IsAboveErrorThreshold, tt.expectError)
			}

			if state.IsAboveAutoCompactThreshold != tt.expectCompact {
				t.Errorf("IsAboveAutoCompactThreshold = %v, want %v",
					state.IsAboveAutoCompactThreshold, tt.expectCompact)
			}

			// 验证阈值计算
			if state.AutoCompactThreshold != autoCompactThreshold {
				t.Errorf("AutoCompactThreshold = %d, want %d",
					state.AutoCompactThreshold, autoCompactThreshold)
			}

			if state.WarningThreshold != warningThreshold {
				t.Errorf("WarningThreshold = %d, want %d",
					state.WarningThreshold, warningThreshold)
			}

			if state.ErrorThreshold != errorThreshold {
				t.Errorf("ErrorThreshold = %d, want %d",
					state.ErrorThreshold, errorThreshold)
			}
		})
	}
}

func TestShouldAutoCompact(t *testing.T) {
	model := "claude-sonnet-4-6"
	betas := []string{}

	trackingState := &AutoCompactTrackingState{
		Compacted:           false,
		TurnCounter:         1,
		TurnId:              "test-turn-1",
		ConsecutiveFailures: 0,
	}

	_ = GetEffectiveContextWindowSize(model, betas) // Keep for documentation
	autoCompactThreshold := GetAutoCompactThreshold(model, betas)

	tests := []struct {
		name           string
		tokenUsage     int
		trackingState  *AutoCompactTrackingState
		expectedResult bool
	}{
		{
			name:           "Below threshold, should not compact",
			tokenUsage:     autoCompactThreshold - 1_000,
			trackingState:  trackingState,
			expectedResult: false,
		},
		{
			name:           "Above threshold, should compact",
			tokenUsage:     autoCompactThreshold + 1_000,
			trackingState:  trackingState,
			expectedResult: true,
		},
		{
			name:       "Already compacted, should not compact again",
			tokenUsage: autoCompactThreshold + 1_000,
			trackingState: &AutoCompactTrackingState{
				Compacted:           true,
				TurnCounter:         2,
				TurnId:              "test-turn-2",
				ConsecutiveFailures: 0,
			},
			expectedResult: false,
		},
		{
			name:       "Too many failures, should not compact",
			tokenUsage: autoCompactThreshold + 1_000,
			trackingState: &AutoCompactTrackingState{
				Compacted:           false,
				TurnCounter:         3,
				TurnId:              "test-turn-3",
				ConsecutiveFailures: MaxConsecutiveAutocompactFailures,
			},
			expectedResult: false,
		},
		{
			name:           "Nil tracking state, should not compact",
			tokenUsage:     autoCompactThreshold + 1_000,
			trackingState:  nil,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldAutoCompact(tt.tokenUsage, model, betas, tt.trackingState)
			if result != tt.expectedResult {
				t.Errorf("ShouldAutoCompact(%d, %q, %v, %+v) = %v, want %v",
					tt.tokenUsage, model, betas, tt.trackingState, result, tt.expectedResult)
			}
		})
	}
}

func TestTokenCountWithEstimation(t *testing.T) {
	// 创建测试消息
	messages := []Message{
		{
			Type:    "user",
			Message: json.RawMessage(`{"role":"user","content":"Hello, how are you?"}`),
		},
		{
			Type:    "assistant",
			Message: json.RawMessage(`{"role":"assistant","content":"I'm doing well, thank you!"}`),
		},
	}

	model := "claude-3-haiku"

	// 测试估算
	tokenCount := TokenCountWithEstimation(messages, model)

	// 验证令牌数大于0
	if tokenCount <= 0 {
		t.Errorf("TokenCountWithEstimation() = %d, want > 0", tokenCount)
	}

	// 验证对于短消息，令牌数应该合理
	expectedMax := 100 // 对于短消息，令牌数应该小于这个值
	if tokenCount > expectedMax {
		t.Errorf("TokenCountWithEstimation() = %d, want <= %d for short messages", tokenCount, expectedMax)
	}
}

func TestCompactConversation(t *testing.T) {
	// 创建测试消息 - 第一个消息非常长以确保触发压缩
	messages := make([]Message, 20)

	// 第一个消息非常长（约800,000字符）以确保超过压缩阈值
	longContent := `{"role":"user","content":"` + strings.Repeat("Very long message to trigger compression. ", 20000) + `"}`
	messages[0] = Message{
		Type:    "user",
		Message: json.RawMessage(longContent),
	}

	// 其余消息正常
	for i := 1; i < 20; i++ {
		content := json.RawMessage(`{"role":"user","content":"Test message ` + string(rune('A'+i)) + `"}`)
		messages[i] = Message{
			Type:    "user",
			Message: content,
		}
	}

	model := "claude-sonnet-4-6"
	config := DefaultCompactConfig(model)
	config.PreserveRecent = 5

	// 执行压缩
	result, err := CompactConversation(messages, config, nil)
	if err != nil {
		t.Fatalf("CompactConversation() error = %v", err)
	}

	if !result.Success {
		t.Errorf("CompactConversation() success = %v, want true", result.Success)
	}

	// 验证结果消息数量
	// 应该包含：1个压缩边界消息 + 1个摘要消息 + 5个保留的消息
	expectedMinMessages := 3 // 边界 + 摘要 + 至少1个保留消息
	if len(result.Messages) < expectedMinMessages {
		t.Errorf("CompactConversation() messages count = %d, want >= %d",
			len(result.Messages), expectedMinMessages)
	}

	// 验证第一个消息是压缩边界
	if result.Messages[0].Type != "system" || result.Messages[0].Subtype != "compact_boundary" {
		t.Errorf("First message should be compact boundary, got type=%q subtype=%q",
			result.Messages[0].Type, result.Messages[0].Subtype)
	}
}

func TestAnalyzeContext(t *testing.T) {
	// 创建测试消息
	messages := make([]Message, 10)
	for i := 0; i < 10; i++ {
		content := json.RawMessage(`{"role":"user","content":"Test message for analysis"}`)
		messages[i] = Message{
			Type:    "user",
			Message: content,
		}
	}

	model := "claude-sonnet-4-6"
	betas := []string{}

	analysis := AnalyzeContext(messages, model, betas)

	// 验证分析结果
	if analysis.MessagesCount != len(messages) {
		t.Errorf("AnalyzeContext() MessagesCount = %d, want %d",
			analysis.MessagesCount, len(messages))
	}

	if analysis.EffectiveWindow <= 0 {
		t.Errorf("AnalyzeContext() EffectiveWindow = %d, want > 0",
			analysis.EffectiveWindow)
	}

	// 使用百分比应该在0-100之间
	if analysis.UsagePercentage < 0 || analysis.UsagePercentage > 100 {
		t.Errorf("AnalyzeContext() UsagePercentage = %d, want between 0 and 100",
			analysis.UsagePercentage)
	}

	// 验证阈值计算
	if analysis.CompactThreshold != GetAutoCompactThreshold(model, betas) {
		t.Errorf("AnalyzeContext() CompactThreshold = %d, want %d",
			analysis.CompactThreshold, GetAutoCompactThreshold(model, betas))
	}
}

func TestUpdateAutoCompactTracking(t *testing.T) {
	tests := []struct {
		name           string
		initialState   *AutoCompactTrackingState
		compacted      bool
		expectedTurn   int
		expectedFailed int
	}{
		{
			name: "Successful compaction",
			initialState: &AutoCompactTrackingState{
				TurnCounter:         1,
				Compacted:           false,
				ConsecutiveFailures: 2,
			},
			compacted:      true,
			expectedTurn:   2,
			expectedFailed: 0, // 成功时重置失败计数器
		},
		{
			name: "Failed compaction",
			initialState: &AutoCompactTrackingState{
				TurnCounter:         1,
				Compacted:           false,
				ConsecutiveFailures: 1,
			},
			compacted:      false,
			expectedTurn:   2,
			expectedFailed: 2, // 失败时增加失败计数器
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 复制初始状态以避免修改原始数据
			state := &AutoCompactTrackingState{
				TurnCounter:         tt.initialState.TurnCounter,
				TurnId:              tt.initialState.TurnId,
				Compacted:           tt.initialState.Compacted,
				ConsecutiveFailures: tt.initialState.ConsecutiveFailures,
			}

			UpdateAutoCompactTracking(state, tt.compacted)

			if state.TurnCounter != tt.expectedTurn {
				t.Errorf("TurnCounter = %d, want %d", state.TurnCounter, tt.expectedTurn)
			}

			if state.Compacted != tt.compacted {
				t.Errorf("Compacted = %v, want %v", state.Compacted, tt.compacted)
			}

			if state.ConsecutiveFailures != tt.expectedFailed {
				t.Errorf("ConsecutiveFailures = %d, want %d",
					state.ConsecutiveFailures, tt.expectedFailed)
			}
		})
	}
}
