// Package modelregistry provides a centralized registry of AI model family
// capabilities. It replaces scattered substring matching across the codebase
// with a single source of truth.
package modelregistry

import "strings"

// ModelCapabilities describes what a model family supports.
type ModelCapabilities struct {
	// Thinking / reasoning mode
	SupportsThinking            bool
	DefaultThinkingEnabled      bool
	EnforcesReasoningInThinking bool
	ThinkingDisabledVariants    []string // variant substrings where DefaultThinkingEnabled flips to false
	ThinkingEnabledVariants     []string // allowlist: when non-empty, only these variants get thinking

	// Token limits
	MaxOutputTokens int
}

type modelFamilyEntry struct {
	Family       string
	Capabilities ModelCapabilities
	Matchers     []string // model ID substrings, first match wins
}

var registry = []modelFamilyEntry{
	{
		Family: "deepseek",
		Capabilities: ModelCapabilities{
			SupportsThinking:            true,
			DefaultThinkingEnabled:      false,
			EnforcesReasoningInThinking: true,
			ThinkingEnabledVariants:     []string{"reasoner", "r1-671b", "v3.2", "v4-pro"},
			MaxOutputTokens:             32768,
		},
		Matchers: []string{"deepseek"},
	},
	{
		Family: "qwen",
		Capabilities: ModelCapabilities{
			SupportsThinking:            true,
			DefaultThinkingEnabled:      true,
			EnforcesReasoningInThinking: false,
			MaxOutputTokens:             8192,
		},
		Matchers: []string{"qwen"},
	},
	{
		Family: "claude",
		Capabilities: ModelCapabilities{
			SupportsThinking:            true,
			DefaultThinkingEnabled:      false,
			EnforcesReasoningInThinking: false,
			MaxOutputTokens:             32000,
		},
		Matchers: []string{"claude"},
	},
	{
		Family: "gpt",
		Capabilities: ModelCapabilities{
			SupportsThinking:            false,
			DefaultThinkingEnabled:      false,
			EnforcesReasoningInThinking: false,
			MaxOutputTokens:             16384,
		},
		Matchers: []string{"gpt-", "o1", "o3", "o4"},
	},
}

// Lookup finds the ModelCapabilities for a model ID. Returns zero value and
// false when no family matches.
func Lookup(modelID string) (ModelCapabilities, bool) {
	low := strings.ToLower(modelID)
	for _, entry := range registry {
		for _, m := range entry.Matchers {
			if strings.Contains(low, m) {
				caps := entry.Capabilities
				// Apply allowlist first (if set, thinking defaults to false)
				if len(caps.ThinkingEnabledVariants) > 0 {
					caps.DefaultThinkingEnabled = false
					for _, variant := range caps.ThinkingEnabledVariants {
						if strings.Contains(low, variant) {
							caps.DefaultThinkingEnabled = true
							break
						}
					}
				}
				// Then apply denylist overrides
				for _, variant := range caps.ThinkingDisabledVariants {
					if strings.Contains(low, variant) {
						caps.DefaultThinkingEnabled = false
						break
					}
				}
				return caps, true
			}
		}
	}
	return ModelCapabilities{}, false
}
