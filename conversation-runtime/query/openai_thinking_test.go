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
