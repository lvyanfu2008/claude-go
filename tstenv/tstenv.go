// Package tstenv mirrors TS ENABLE_TOOL_SEARCH / ANTHROPIC_BASE_URL gates from
// src/utils/toolSearch.ts (getToolSearchMode, isToolSearchEnabledOptimistic proxy check)
// and src/utils/model/providers.ts (getAPIProvider, isFirstPartyAnthropicBaseUrl).
// It has no ccb-engine/internal dependency so messagesapi and ccb-engine can share it.
package tstenv

import (
	"net/url"
	"os"
	"strconv"
	"strings"
)

// APIProvider mirrors src/utils/model/providers.ts getAPIProvider().
type APIProvider string

const (
	FirstParty APIProvider = "firstParty"
	Bedrock    APIProvider = "bedrock"
	Vertex     APIProvider = "vertex"
	Foundry    APIProvider = "foundry"
	OpenAI     APIProvider = "openai"
)

func envTruthy(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func envDefinedFalsy(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	return v == "0" || v == "false" || v == "no" || v == "off"
}

func envTrim(k string) string {
	return strings.TrimSpace(os.Getenv(k))
}

// GetAPIProvider mirrors getAPIProvider (src/utils/model/providers.ts).
func GetAPIProvider() APIProvider {
	if envTruthy("CLAUDE_CODE_USE_OPENAI") {
		return OpenAI
	}
	if envTruthy("CLAUDE_CODE_USE_BEDROCK") {
		return Bedrock
	}
	if envTruthy("CLAUDE_CODE_USE_VERTEX") {
		return Vertex
	}
	if envTruthy("CLAUDE_CODE_USE_FOUNDRY") {
		return Foundry
	}
	return FirstParty
}

// IsFirstPartyAnthropicBaseUrl mirrors isFirstPartyAnthropicBaseUrl (providers.ts).
func IsFirstPartyAnthropicBaseUrl() bool {
	base := strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL"))
	if base == "" {
		return true
	}
	u, err := url.Parse(base)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	if host == "api.anthropic.com" {
		return true
	}
	if strings.TrimSpace(os.Getenv("USER_TYPE")) == "ant" && host == "api-staging.anthropic.com" {
		return true
	}
	return false
}

// GetToolSearchMode mirrors getToolSearchMode (src/utils/toolSearch.ts).
func GetToolSearchMode() string {
	if envTruthy("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS") {
		return "standard"
	}
	v := envTrim("ENABLE_TOOL_SEARCH")
	if envDefinedFalsy("ENABLE_TOOL_SEARCH") {
		return "standard"
	}
	if v == "" {
		return "tst"
	}
	lower := strings.ToLower(v)
	if p := parseAutoPercent(v); p != nil {
		if *p == 0 {
			return "tst"
		}
		if *p == 100 {
			return "standard"
		}
		return "tst-auto"
	}
	if lower == "auto" {
		return "tst-auto"
	}
	if envTruthy(v) {
		return "tst"
	}
	return "tst"
}

func parseAutoPercent(v string) *int {
	s := strings.TrimSpace(strings.ToLower(v))
	if !strings.HasPrefix(s, "auto:") {
		return nil
	}
	tail := strings.TrimSpace(strings.TrimPrefix(s, "auto:"))
	n, err := strconv.Atoi(tail)
	if err != nil {
		return nil
	}
	if n < 0 {
		n = 0
	}
	if n > 100 {
		n = 100
	}
	return &n
}

// AutoToolSearchPercentage mirrors getAutoToolSearchPercentage default 10.
func AutoToolSearchPercentage() int {
	v := envTrim("ENABLE_TOOL_SEARCH")
	if v == "" || strings.EqualFold(v, "auto") {
		return 10
	}
	if p := parseAutoPercent(v); p != nil {
		return *p
	}
	return 10
}

// ToolSearchEnabledOptimistic mirrors isToolSearchEnabledOptimistic (toolSearch.ts) without GrowthBook.
func ToolSearchEnabledOptimistic() bool {
	mode := GetToolSearchMode()
	if mode == "standard" {
		return false
	}
	// ENABLE_TOOL_SEARCH unset/empty + firstParty + non–first-party host → false (TS CC-30912 gate).
	if !envTruthy("ENABLE_TOOL_SEARCH") && !envDefinedFalsy("ENABLE_TOOL_SEARCH") {
		if envTrim("ENABLE_TOOL_SEARCH") == "" &&
			GetAPIProvider() == FirstParty &&
			!IsFirstPartyAnthropicBaseUrl() {
			return false
		}
	}
	return true
}
