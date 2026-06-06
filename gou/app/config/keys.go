package config

import (
	"fmt"
	"os"
	"strings"

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	"goc/modelenv"
)

// AnthropicAPIKey returns the ANTHROPIC_API_KEY or ANTHROPIC_AUTH_TOKEN env var, trimmed.
func AnthropicAPIKey() string {
	k := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN"))
}

// HasLLMKeys reports whether at least one LLM API key is set (Anthropic or OpenAI).
func HasLLMKeys() bool {
	for _, k := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "OPENAI_API_KEY"} {
		if strings.TrimSpace(os.Getenv(k)) != "" {
			return true
		}
	}
	return false
}

// PreferQueryStreamingParity is true when env gates parity and an Anthropic key is present (HTTP path usable).
func PreferQueryStreamingParity() bool {
	if AnthropicAPIKey() == "" {
		return false
	}
	cfg := query.BuildQueryConfig()
	return query.StreamingParityPathEnabled(cfg)
}

// QueryMainLoopModel is the model id for HTTP streaming parity + ParityToolRunner.
// /model sets CLAUDE_CODE_MODEL in-process; that must override ToolUseContext.Options from
// [pui.BuildDemoParams] when they disagree (otherwise the API keeps an older id).
func QueryMainLoopModel(params *processuserinput.ProcessUserInputParams) string {
	if cm := strings.TrimSpace(os.Getenv("CLAUDE_CODE_MODEL")); cm != "" {
		return cm
	}
	if params != nil && params.RuntimeContext != nil {
		if m := strings.TrimSpace(params.RuntimeContext.ToolUseContext.Options.MainLoopModel); m != "" {
			return m
		}
	}
	return modelenv.EffectiveMainLoopModel()
}

// UserContextMapForQuery copies live user context for [query.PrependUserContext].
// Values must be raw (no <system-reminder> wrapper): TS prependUserContext wraps once per #key/value.
// Do not pass [querycontext.FormatUserContextReminder] here — that string is already wrapped for ccbhydrate lead-in only.
func UserContextMapForQuery(uc map[string]string) map[string]string {
	if len(uc) == 0 {
		return nil
	}
	out := make(map[string]string)
	for k, v := range uc {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// WarnAPILogExpectations prints stderr hints when CLAUDE_CODE_LOG_API_* cannot produce HTTP body logs.
func WarnAPILogExpectations(ccbInline bool) {
	if !EnvWantsAPIBodyLog() {
		return
	}
	if !ccbInline {
		fmt.Fprintf(os.Stderr,
			"[gou-demo] CLAUDE_CODE_LOG_API_* is set, but this run has real HTTP / streaming parity disabled (GOU_DEMO_CCB_INLINE=0).\n"+
				"           No HTTP → apilog will not append request/response lines. Unset GOU_DEMO_CCB_INLINE and set ANTHROPIC_API_KEY plus GOU_QUERY_STREAMING_PARITY=1 or GOU_DEMO_STREAMING_TOOL_EXECUTION=1 for real API logs.\n")
		return
	}
	if !HasLLMKeys() {
		fmt.Fprintf(os.Stderr,
			"[gou-demo] CLAUDE_CODE_LOG_API_* is set, but no ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, or OPENAI_API_KEY is set.\n"+
				"           Put keys in ~/.claude/settings.go.json or project .claude/settings.go.json env, or export them.\n")
	}
}
