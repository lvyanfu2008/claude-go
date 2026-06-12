// Package autodream mirrors src/services/autoDream/config.ts.
//
// Background memory consolidation (the /dream prompt) runs as a forked
// sub-agent when time-gate + session-count-gate + lock-gate all pass.
package autodream

import (
	"os"

	"goc/claudebase"
	"goc/growthbook"
)

// IsAutoDreamEnabled mirrors isAutoDreamEnabled() in config.ts.
// Checks the GrowthBook tengu_onyx_plover.enabled field, with env fallback
// for environments without GrowthBook (e.g. agentd).
func IsAutoDreamEnabled() bool {
	if v := os.Getenv("GOC_AUTO_DREAM"); v != "" {
		return claudebase.Truthy(v)
	}
	type onyxPloverCfg struct {
		Enabled bool `json:"enabled"`
	}
	raw := growthbook.DefaultManager().Get("onyx_plover")
	if raw == nil {
		return false
	}
	switch v := raw.(type) {
	case map[string]any:
		if e, ok := v["enabled"].(bool); ok {
			return e
		}
		return false
	case onyxPloverCfg:
		return v.Enabled
	case bool:
		return v
	default:
		return false
	}
}

// AutoDreamConfig holds scheduling thresholds from tengu_onyx_plover.
type AutoDreamConfig struct {
	MinHours    int
	MinSessions int
}

// DefaultAutoDreamConfig matches TS DEFAULTS in autoDream.ts.
var DefaultAutoDreamConfig = AutoDreamConfig{
	MinHours:    24,
	MinSessions: 5,
}

// GetConfig mirrors getConfig() in autoDream.ts.
// Reads MinHours and MinSessions from the tengu_onyx_plover GrowthBook flag.
func GetConfig() AutoDreamConfig {
	raw := growthbook.DefaultManager().Get("onyx_plover")
	if raw == nil {
		return DefaultAutoDreamConfig
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return DefaultAutoDreamConfig
	}
	cfg := DefaultAutoDreamConfig
	if v, ok := m["minHours"].(float64); ok && v > 0 {
		cfg.MinHours = int(v)
	}
	if v, ok := m["minSessions"].(float64); ok && v > 0 {
		cfg.MinSessions = int(v)
	}
	return cfg
}
