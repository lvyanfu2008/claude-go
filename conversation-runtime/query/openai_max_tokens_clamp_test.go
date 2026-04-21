package query

import (
	"testing"

	"goc/compactservice"
)

func TestClampOpenAICompatibleMaxTokens_TSDefault8192(t *testing.T) {
	t.Setenv(envOpenAIMaxOutputTokensCapTS, "")
	t.Setenv(envOpenAIMaxOutputTokensCapGo, "")
	if got := ClampOpenAICompatibleMaxTokens(20_000); got != 8192 {
		t.Fatalf("want 8192, got %d", got)
	}
	if got := ClampOpenAICompatibleMaxTokens(0); got != 1 {
		t.Fatalf("want min 1, got %d", got)
	}
	if got := ClampOpenAICompatibleMaxTokens(4000); got != 4000 {
		t.Fatalf("want 4000, got %d", got)
	}
}

func TestClampOpenAICompatibleMaxTokens_TSEnvCap(t *testing.T) {
	t.Setenv(envOpenAIMaxOutputTokensCapTS, "16384")
	t.Setenv(envOpenAIMaxOutputTokensCapGo, "")
	if got := ClampOpenAICompatibleMaxTokens(20_000); got != 16384 {
		t.Fatalf("want 16384, got %d", got)
	}
}

func TestClampOpenAICompatibleMaxTokens_goAliasWhenTSEnvUnset(t *testing.T) {
	t.Setenv(envOpenAIMaxOutputTokensCapTS, "")
	t.Setenv(envOpenAIMaxOutputTokensCapGo, "4096")
	if got := ClampOpenAICompatibleMaxTokens(20_000); got != 4096 {
		t.Fatalf("want 4096 from Go alias, got %d", got)
	}
}

func TestClampOpenAICompatibleMaxTokens_TSPrecedenceOverGoAlias(t *testing.T) {
	t.Setenv(envOpenAIMaxOutputTokensCapTS, "8192")
	t.Setenv(envOpenAIMaxOutputTokensCapGo, "4096")
	if got := ClampOpenAICompatibleMaxTokens(20_000); got != 8192 {
		t.Fatalf("TS env should win, want 8192 got %d", got)
	}
}

func TestAutocompactOpenAIMaxWire_minWithModelCap(t *testing.T) {
	t.Setenv(envOpenAIMaxOutputTokensCapTS, "")
	t.Setenv(envOpenAIMaxOutputTokensCapGo, "")
	in := compactservice.SummaryStreamInput{
		Model:           "claude-3-haiku-20240307",
		MaxOutputTokens: compactservice.CompactMaxOutputTokens,
	}
	// Go context table: haiku 3 → 4096 max output; min(20_000, 4096)=4096, clamp cap 8192 → 4096
	if got := autocompactOpenAIMaxWire(in); got != 4096 {
		t.Fatalf("want 4096, got %d", got)
	}
}
