package autodream

import (
	"testing"

	"goc/growthbook"
)

func TestIsAutoDreamEnabled_enabledByLocalGateDefault(t *testing.T) {
	// When no flag is set and local gate defaults are active,
	// onyx_plover.enabled defaults to true (mirrors TS LOCAL_GATE_DEFAULTS).
	if !IsAutoDreamEnabled() {
		t.Fatal("expected auto-dream enabled via local gate default (onyx_plover.enabled=true)")
	}
}

func TestIsAutoDreamEnabled_enabledViaMap(t *testing.T) {
	growthbook.DefaultManager().UpdateFlags(map[string]growthbook.FeatureFlag{
		"onyx_plover": {
			Key:   "onyx_plover",
			Value: map[string]any{"enabled": true},
		},
	})
	if !IsAutoDreamEnabled() {
		t.Fatal("expected auto-dream enabled when flag.enabled=true")
	}
}

func TestIsAutoDreamEnabled_disabledViaMap(t *testing.T) {
	growthbook.DefaultManager().UpdateFlags(map[string]growthbook.FeatureFlag{
		"onyx_plover": {
			Key:   "onyx_plover",
			Value: map[string]any{"enabled": false},
		},
	})
	if IsAutoDreamEnabled() {
		t.Fatal("expected auto-dream disabled when flag.enabled=false")
	}
}

func TestIsAutoDreamEnabled_boolValue(t *testing.T) {
	growthbook.DefaultManager().UpdateFlags(map[string]growthbook.FeatureFlag{
		"onyx_plover": {
			Key:   "onyx_plover",
			Value: true,
		},
	})
	if !IsAutoDreamEnabled() {
		t.Fatal("expected auto-dream enabled when flag value is bool(true)")
	}
}

func TestGetConfig_defaults(t *testing.T) {
	// Remove any onyx_plover flag by setting it to nil-equivalent.
	// We can't delete from update, so test with a flag that has no config fields.
	growthbook.DefaultManager().UpdateFlags(map[string]growthbook.FeatureFlag{
		"onyx_plover": {
			Key:   "onyx_plover",
			Value: map[string]any{}, // empty config
		},
	})
	cfg := GetConfig()
	if cfg.MinHours != DefaultAutoDreamConfig.MinHours {
		t.Fatalf("expected MinHours %d, got %d", DefaultAutoDreamConfig.MinHours, cfg.MinHours)
	}
	if cfg.MinSessions != DefaultAutoDreamConfig.MinSessions {
		t.Fatalf("expected MinSessions %d, got %d", DefaultAutoDreamConfig.MinSessions, cfg.MinSessions)
	}
}

func TestGetConfig_custom(t *testing.T) {
	growthbook.DefaultManager().UpdateFlags(map[string]growthbook.FeatureFlag{
		"onyx_plover": {
			Key: "onyx_plover",
			Value: map[string]any{
				"minHours":    12.0,
				"minSessions": 3.0,
			},
		},
	})
	cfg := GetConfig()
	if cfg.MinHours != 12 {
		t.Fatalf("expected MinHours 12, got %d", cfg.MinHours)
	}
	if cfg.MinSessions != 3 {
		t.Fatalf("expected MinSessions 3, got %d", cfg.MinSessions)
	}
}

func TestGetConfig_partialOverride(t *testing.T) {
	growthbook.DefaultManager().UpdateFlags(map[string]growthbook.FeatureFlag{
		"onyx_plover": {
			Key: "onyx_plover",
			Value: map[string]any{
				"minHours": 6.0,
				// minSessions not set — should use default
			},
		},
	})
	cfg := GetConfig()
	if cfg.MinHours != 6 {
		t.Fatalf("expected MinHours 6, got %d", cfg.MinHours)
	}
	if cfg.MinSessions != DefaultAutoDreamConfig.MinSessions {
		t.Fatalf("expected MinSessions %d (default), got %d", DefaultAutoDreamConfig.MinSessions, cfg.MinSessions)
	}
}

func TestGetConfig_zeroValues(t *testing.T) {
	// Zero values in config should be ignored (use defaults).
	growthbook.DefaultManager().UpdateFlags(map[string]growthbook.FeatureFlag{
		"onyx_plover": {
			Key: "onyx_plover",
			Value: map[string]any{
				"minHours":    0.0,
				"minSessions": 0.0,
			},
		},
	})
	cfg := GetConfig()
	if cfg.MinHours != DefaultAutoDreamConfig.MinHours {
		t.Fatalf("expected MinHours %d (default), got %d", DefaultAutoDreamConfig.MinHours, cfg.MinHours)
	}
	if cfg.MinSessions != DefaultAutoDreamConfig.MinSessions {
		t.Fatalf("expected MinSessions %d (default), got %d", DefaultAutoDreamConfig.MinSessions, cfg.MinSessions)
	}
}
