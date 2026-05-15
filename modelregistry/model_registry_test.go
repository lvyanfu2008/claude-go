package modelregistry

import "testing"

func TestLookup_DeepSeekV4Pro(t *testing.T) {
	caps, ok := Lookup("DeepSeek-V4-Pro")
	if !ok {
		t.Fatal("expected deepseek-v4-pro to match")
	}
	if !caps.SupportsThinking {
		t.Fatal("expected SupportsThinking=true")
	}
	if !caps.DefaultThinkingEnabled {
		t.Fatal("expected DefaultThinkingEnabled=true")
	}
	if !caps.EnforcesReasoningInThinking {
		t.Fatal("expected EnforcesReasoningInThinking=true")
	}
}

func TestLookup_DeepSeekV4Flash(t *testing.T) {
	caps, ok := Lookup("deepseek-v4-flash")
	if !ok {
		t.Fatal("expected v4-flash to match deepseek family")
	}
	if caps.DefaultThinkingEnabled {
		t.Fatal("expected DefaultThinkingEnabled=false for v4-flash variant")
	}
}

func TestLookup_DeepSeekReasoner(t *testing.T) {
	caps, ok := Lookup("deepseek-reasoner")
	if !ok {
		t.Fatal("expected deepseek-reasoner to match")
	}
	if !caps.DefaultThinkingEnabled {
		t.Fatal("expected DefaultThinkingEnabled=true for reasoner")
	}
}

func TestLookup_Qwen(t *testing.T) {
	caps, ok := Lookup("qwen3-32b")
	if !ok {
		t.Fatal("expected qwen3-32b to match")
	}
	if !caps.SupportsThinking {
		t.Fatal("expected SupportsThinking=true for qwen")
	}
	if !caps.DefaultThinkingEnabled {
		t.Fatal("expected DefaultThinkingEnabled=true for qwen")
	}
	if caps.EnforcesReasoningInThinking {
		t.Fatal("expected EnforcesReasoningInThinking=false for qwen")
	}
	if caps.MaxOutputTokens != 8192 {
		t.Fatalf("expected MaxOutputTokens=8192, got %d", caps.MaxOutputTokens)
	}
}

func TestLookup_Unknown(t *testing.T) {
	_, ok := Lookup("some-unknown-model")
	if ok {
		t.Fatal("expected unknown model to not match")
	}
}

func TestLookup_Claude(t *testing.T) {
	caps, ok := Lookup("claude-sonnet-4-6")
	if !ok {
		t.Fatal("expected claude to match")
	}
	if caps.DefaultThinkingEnabled {
		t.Fatal("expected DefaultThinkingEnabled=false for claude")
	}
}

func TestLookup_GPT(t *testing.T) {
	caps, ok := Lookup("gpt-4o")
	if !ok {
		t.Fatal("expected gpt-4o to match")
	}
	if caps.SupportsThinking {
		t.Fatal("expected SupportsThinking=false for gpt")
	}
}
