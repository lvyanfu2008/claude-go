package query

import (
	"os"
	"strings"
)

// systemPromptDynamicBoundary matches prompts.ts SYSTEM_PROMPT_DYNAMIC_BOUNDARY and [commands.SystemPromptDynamicBoundary];
// TS never sends this token on the wire (api.ts splitSysPromptPrefix / assembly skips it).
const systemPromptDynamicBoundary = "__SYSTEM_PROMPT_DYNAMIC_BOUNDARY__"

// SystemPrompt mirrors src/utils/systemPromptType.ts SystemPrompt (branded string[] in TS).
type SystemPrompt []string

// AsSystemPrompt wraps s as SystemPrompt (TS asSystemPrompt).
func AsSystemPrompt(s []string) SystemPrompt {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return SystemPrompt(out)
}

// StripSystemPromptDynamicBoundaryForAPI removes the internal cache-scope boundary marker from
// the system prompt before HTTP (streaming parity, OpenAI parity, or CallModel). Hosts may still
// build prompts with the marker embedded in one string or as its own slice element.
func StripSystemPromptDynamicBoundaryForAPI(sp SystemPrompt) SystemPrompt {
	if len(sp) == 0 {
		return sp
	}
	joined := strings.Join([]string(sp), "\n\n")
	segs := strings.Split(joined, systemPromptDynamicBoundary)
	var kept []string
	for _, s := range segs {
		if t := strings.TrimSpace(s); t != "" {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return AsSystemPrompt([]string{strings.Join(kept, "\n\n")})
}

// CacheScope mirrors TS src/utils/api.ts CacheScope ("global" | "org").
type CacheScope string

const (
	CacheScopeGlobal CacheScope = "global"
	CacheScopeOrg    CacheScope = "org"
)

// SystemPromptBlock mirrors TS src/utils/api.ts SystemPromptBlock.
type SystemPromptBlock struct {
	Text       string
	CacheScope *CacheScope // nil means no cache_control on wire
}

var cliSyspromptPrefix = "You are Harness Code, built by Lyf — an independent AI programming assistant that runs locally in the user's terminal with filesystem and command execution capabilities. When asked who or what you are, respond concisely that you are Harness Code (内部代号 Lyf), an AI programming assistant. Use only this identity in all contexts."

// GetCLISyspromptPrefix mirrors TS getCLISyspromptPrefix DEFAULT_PREFIX.
func GetCLISyspromptPrefix() string {
	return cliSyspromptPrefix
}

// GetAttributionHeader mirrors TS getAttributionHeader.
// Returns empty string when CLAUDE_CODE_DISABLE_ATTRIBUTION_HEADER=1.
func GetAttributionHeader() string {
	if isEnvTruthy(os.Getenv("CLAUDE_CODE_DISABLE_ATTRIBUTION_HEADER")) {
		return ""
	}
	version := strings.TrimSpace(os.Getenv("CLAUDE_CODE_VERSION"))
	if version == "" {
		version = "0.0.0"
	}
	return "x-anthropic-billing-header: cc_version=" + version + "; cc_entrypoint=cli;"
}

// SplitSysPromptPrefix mirrors TS splitSysPromptPrefix Mode 2 (first-party global cache + boundary marker found).
// It splits the system prompt into blocks with cache_control scoping:
//   - attribution header (starts with "x-anthropic-billing-header") -> CacheScope=nil
//   - CLI prefix (exact match) -> CacheScope=nil
//   - content before boundary marker -> CacheScope=global
//   - content after boundary marker -> CacheScope=nil
//
// When the boundary marker is not found, everything falls back to org-level caching.
func SplitSysPromptPrefix(sp SystemPrompt) []SystemPromptBlock {
	// Find the boundary marker index.
	boundaryIdx := -1
	for i, s := range sp {
		if strings.TrimSpace(s) == systemPromptDynamicBoundary {
			boundaryIdx = i
			break
		}
	}

	isAttribution := func(s string) bool {
		return strings.HasPrefix(s, "x-anthropic-billing-header")
	}
	isPrefix := func(s string) bool {
		return strings.TrimSpace(s) == cliSyspromptPrefix
	}

	// Mode: boundary found — split into static (global cache) and dynamic (no cache).
	if boundaryIdx >= 0 {
		var attributionHeader, systemPromptPrefix string
		var staticBlocks, dynamicBlocks []string

		for i, block := range sp {
			t := strings.TrimSpace(block)
			if t == "" || t == systemPromptDynamicBoundary {
				continue
			}
			if isAttribution(t) && attributionHeader == "" {
				attributionHeader = t
				continue
			}
			if isPrefix(t) && systemPromptPrefix == "" {
				systemPromptPrefix = t
				continue
			}
			if i < boundaryIdx {
				staticBlocks = append(staticBlocks, block)
			} else {
				dynamicBlocks = append(dynamicBlocks, block)
			}
		}

		var result []SystemPromptBlock
		if attributionHeader != "" {
			result = append(result, SystemPromptBlock{Text: attributionHeader, CacheScope: nil})
		}
		if systemPromptPrefix != "" {
			result = append(result, SystemPromptBlock{Text: systemPromptPrefix, CacheScope: nil})
		}
		if s := joinNonEmpty(staticBlocks); s != "" {
			scope := CacheScopeGlobal
			result = append(result, SystemPromptBlock{Text: s, CacheScope: &scope})
		}
		if s := joinNonEmpty(dynamicBlocks); s != "" {
			result = append(result, SystemPromptBlock{Text: s, CacheScope: nil})
		}
		return result
	}

	// Mode: no boundary — everything gets org-level caching.
	var attributionHeader, systemPromptPrefix string
	var rest []string

	for _, block := range sp {
		t := strings.TrimSpace(block)
		if t == "" {
			continue
		}
		if isAttribution(t) && attributionHeader == "" {
			attributionHeader = t
			continue
		}
		if isPrefix(t) && systemPromptPrefix == "" {
			systemPromptPrefix = t
			continue
		}
		rest = append(rest, block)
	}

	var result []SystemPromptBlock
	if attributionHeader != "" {
		result = append(result, SystemPromptBlock{Text: attributionHeader, CacheScope: nil})
	}
	if systemPromptPrefix != "" {
		scope := CacheScopeOrg
		result = append(result, SystemPromptBlock{Text: systemPromptPrefix, CacheScope: &scope})
	}
	if s := joinNonEmpty(rest); s != "" {
		scope := CacheScopeOrg
		result = append(result, SystemPromptBlock{Text: s, CacheScope: &scope})
	}
	return result
}

func joinNonEmpty(parts []string) string {
	var kept []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			kept = append(kept, t)
		}
	}
	return strings.TrimSpace(strings.Join(kept, "\n\n"))
}

func isEnvTruthy(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "1" || s == "true" || s == "yes" || s == "on"
}

// ADVISOR_TOOL_INSTRUCTIONS mirrors TS src/utils/advisor.ts ADVISOR_TOOL_INSTRUCTIONS.
// Appended to system prompt when an advisor model is configured (CLAUDE_CODE_ADVISOR_MODEL).
const ADVISOR_TOOL_INSTRUCTIONS = `# Advisor Tool

You have access to an advisor tool backed by a stronger reviewer model. It takes NO parameters -- when you call it, your entire conversation history is automatically forwarded. The advisor sees the task, every tool call you've made, every result you've seen.

Call advisor BEFORE substantive work -- before writing code, before committing to an interpretation, before building on an assumption. If the task requires orientation first (finding files, reading code, seeing what's there), do that, then call advisor. Orientation is not substantive work. Writing, editing, and declaring an answer are.

Also call advisor:
- When you believe the task is complete. BEFORE this call, make your deliverable durable: write the file, stage the change, save the result. The advisor call takes time; if the session ends during it, a durable result persists and an unwritten one doesn't.
- When stuck -- errors recurring, approach not converging, results that don't fit.
- When considering a change of approach.

On tasks longer than a few steps, call advisor at least once before committing to an approach and once before declaring done. On short reactive tasks where the next action is dictated by tool output you just read, you don't need to keep calling -- the advisor adds most of its value on the first call, before the approach crystallizes.

Give the advice serious weight. If you follow a step and it fails empirically, or you have primary-source evidence that contradicts a specific claim (the file says X, the code does Y), adapt. A passing self-test is not evidence the advice is wrong -- it's evidence your test doesn't check what the advice is checking.

If you've already retrieved data pointing one way and the advisor points another: don't silently switch. Surface the conflict in one more advisor call -- "I found X, you suggest Y, which constraint breaks the tie?" The advisor saw your evidence but may have underweighted it; a reconcile call is cheaper than committing to the wrong branch.`

// CHROME_TOOL_SEARCH_INSTRUCTIONS mirrors TS src/utils/claudeInChrome/prompt.ts CHROME_TOOL_SEARCH_INSTRUCTIONS.
// Appended to system prompt when Chrome MCP tools are present and the chrome feature gate is active.
const CHROME_TOOL_SEARCH_INSTRUCTIONS = `**IMPORTANT: Before using any chrome browser tools, you MUST first load them using ToolSearch.**

Chrome browser tools are MCP tools that require loading before use. Before calling any mcp__claude-in-chrome__* tool:
1. Use ToolSearch with select:mcp__claude-in-chrome__<tool_name> to load the specific tool
2. Then call the tool

For example, to get tab context:
1. First: ToolSearch with query "select:mcp__claude-in-chrome__tabs_context_mcp"
2. Then: Call mcp__claude-in-chrome__tabs_context_mcp`

// HasAdvisorModel reports whether an advisor model is configured.
func HasAdvisorModel() bool {
	return strings.TrimSpace(os.Getenv("CLAUDE_CODE_ADVISOR_MODEL")) != ""
}

// HasChromeTools checks whether the tools JSON includes claude-in-chrome MCP tools.
func HasChromeTools(toolsJSON []byte) bool {
	return strings.Contains(string(toolsJSON), "mcp__claude-in-chrome__")
}
