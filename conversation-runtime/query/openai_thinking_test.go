package query

import (
	"testing"
)

func TestIsOpenAIThinkingEnabled_V4ProAndFlash(t *testing.T) {
	t.Setenv("OPENAI_ENABLE_THINKING", "")
	if !IsOpenAIThinkingEnabled("DeepSeek-V4-Pro") {
		t.Fatal("expected V4-Pro to enable thinking")
	}
	if !IsOpenAIThinkingEnabled("vendor/deepseek-v4-pro") {
		t.Fatal("expected namespaced V4-Pro id to enable thinking")
	}
	if IsOpenAIThinkingEnabled("DeepSeek-V4-Flash") {
		t.Fatal("expected V4-Flash to disable thinking by default")
	}
	if IsOpenAIThinkingEnabled("deepseek-v4-flash") {
		t.Fatal("expected lowercase v4-flash to disable thinking")
	}
}

func TestIsOpenAIThinkingEnabled_R1AndV3(t *testing.T) {
	t.Setenv("OPENAI_ENABLE_THINKING", "")
	if !IsOpenAIThinkingEnabled("DeepSeek-R1-671B") {
		t.Fatal("expected DeepSeek-R1-671B to enable thinking (reasoner family)")
	}
	if !IsOpenAIThinkingEnabled("deepseek-r1-671b") {
		t.Fatal("expected lowercase deepseek-r1-671b to enable thinking")
	}
	if IsOpenAIThinkingEnabled("DeepSeek-V3-671B") {
		t.Fatal("expected DeepSeek-V3-671B to disable thinking (chat family)")
	}
	if IsOpenAIThinkingEnabled("deepseek-v3-671b") {
		t.Fatal("expected lowercase deepseek-v3-671b to disable thinking")
	}
}

func TestIsOpenAIThinkingEnabled_envOverridesFlash(t *testing.T) {
	t.Setenv("OPENAI_ENABLE_THINKING", "1")
	if !IsOpenAIThinkingEnabled("deepseek-v4-flash") {
		t.Fatal("explicit OPENAI_ENABLE_THINKING=1 should enable thinking for flash too")
	}
}

func TestIsOpenAIThinkingEnabled_envDisablesPro(t *testing.T) {
	t.Setenv("OPENAI_ENABLE_THINKING", "false")
	if IsOpenAIThinkingEnabled("deepseek-v4-pro") {
		t.Fatal("explicit disable should win over model detect")
	}
}

func TestIsOpenAIThinkingEnabled_Qwen(t *testing.T) {
	t.Setenv("OPENAI_ENABLE_THINKING", "")
	if !IsOpenAIThinkingEnabled("qwen3-32b") {
		t.Fatal("expected qwen3-32b to enable thinking")
	}
	if !IsOpenAIThinkingEnabled("Qwen/Qwen3-32B-Instruct") {
		t.Fatal("expected Qwen/Qwen3-32B-Instruct to enable thinking")
	}
	if !IsOpenAIThinkingEnabled("qwen2.5-coder-32b") {
		t.Fatal("expected qwen2.5-coder to enable thinking")
	}
}

func TestIsOpenAIThinkingEnabled_Qwen_envOff(t *testing.T) {
	t.Setenv("OPENAI_ENABLE_THINKING", "0")
	if IsOpenAIThinkingEnabled("qwen3-32b") {
		t.Fatal("expected OPENAI_ENABLE_THINKING=0 to disable for qwen")
	}
}

func TestOpenAIEnforcesReasoningInThinkingMode_Qwen(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DEEPSEEK_STRICT_THINKING", "")
	if OpenAIEnforcesReasoningInThinkingMode("qwen3-32b", true) {
		t.Fatal("expected qwen to NOT enforce reasoning")
	}
}

func TestMergeOpenAIThinkingBodyFields(t *testing.T) {
	t.Setenv("OPENAI_ENABLE_THINKING", "")
	req := map[string]any{"model": "deepseek-v4-pro", "max_tokens": 100}
	mergeOpenAIThinkingBodyFields(req, "deepseek-v4-pro")
	th, _ := req["thinking"].(map[string]any)
	if th["type"] != "enabled" {
		t.Fatalf("thinking: %#v", req["thinking"])
	}
	if req["enable_thinking"] != true {
		t.Fatal("expected enable_thinking")
	}
	kt, _ := req["chat_template_kwargs"].(map[string]any)
	if kt["thinking"] != true {
		t.Fatalf("chat_template_kwargs: %#v", req["chat_template_kwargs"])
	}

	req2 := map[string]any{"model": "deepseek-v4-flash"}
	mergeOpenAIThinkingBodyFields(req2, "deepseek-v4-flash")
	th2, _ := req2["thinking"].(map[string]any)
	if th2 == nil || th2["type"] != "disabled" {
		t.Fatalf("flash should request thinking disabled by default, got %#v", req2["thinking"])
	}
	if _, ok := req2["enable_thinking"]; ok {
		t.Fatal("flash should not set enable_thinking without OPENAI_ENABLE_THINKING")
	}
}
