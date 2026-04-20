// Package growthbook provides GrowthBook-style feature flag management
// that mirrors TS-side GrowthBook integration for feature flags.
package growthbook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"goc/commands/featuregates"
)

// FeatureFlag represents a GrowthBook-style feature flag
type FeatureFlag struct {
	Key         string                 `json:"key"`
	Value       any                    `json:"value"`
	Source      string                 `json:"source"` // "environment", "config", "default"
	Description string                 `json:"description,omitempty"`
	Attributes  map[string]any         `json:"attributes,omitempty"`
	Rules       []FeatureFlagRule      `json:"rules,omitempty"`
	LastUpdated time.Time              `json:"last_updated"`
}

// FeatureFlagRule defines a rule for conditional feature flag evaluation
type FeatureFlagRule struct {
	Condition map[string]any `json:"condition,omitempty"`
	Value     any            `json:"value"`
	Force     bool           `json:"force,omitempty"`
}

// Config represents GrowthBook configuration
type Config struct {
	APIKey         string `json:"api_key,omitempty"`
	ClientKey      string `json:"client_key,omitempty"`
	DecryptionKey  string `json:"decryption_key,omitempty"`
	APIHost        string `json:"api_host,omitempty"`
	Attributes     map[string]any `json:"attributes,omitempty"`
	TrackingCallback func(experimentKey, result string) `json:"-"`
}

// Manager manages GrowthBook feature flags
type Manager struct {
	config      Config
	flags       map[string]FeatureFlag
	attributes  map[string]any
	mu          sync.RWMutex
	initialized bool
}

var (
	defaultManager *Manager
	managerOnce    sync.Once
)

// DefaultManager returns the default GrowthBook manager instance
func DefaultManager() *Manager {
	managerOnce.Do(func() {
		defaultManager = &Manager{
			flags:      make(map[string]FeatureFlag),
			attributes: make(map[string]any),
			config: Config{
				APIHost: "https://cdn.growthbook.io",
			},
		}
		defaultManager.loadFromEnvironment()
		defaultManager.loadFromConfigFile()
	})
	return defaultManager
}

// loadFromEnvironment loads feature flags from environment variables
func (m *Manager) loadFromEnvironment() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load from FEATURE_* environment variables (compatibility with featuregates)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "FEATURE_") {
			key, value, ok := strings.Cut(env, "=")
			if !ok {
				continue
			}

			flagKey := strings.TrimPrefix(key, "FEATURE_")
			boolValue := strings.ToLower(strings.TrimSpace(value)) == "1" ||
				strings.ToLower(strings.TrimSpace(value)) == "true" ||
				strings.ToLower(strings.TrimSpace(value)) == "yes" ||
				strings.ToLower(strings.TrimSpace(value)) == "on"

			m.flags[flagKey] = FeatureFlag{
				Key:         flagKey,
				Value:       boolValue,
				Source:      "environment",
				LastUpdated: time.Now(),
			}
		}
	}

	// Load from CLAUDE_CODE_TENGU_* environment variables (GrowthBook-style)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "CLAUDE_CODE_TENGU_") {
			key, value, ok := strings.Cut(env, "=")
			if !ok {
				continue
			}

			flagKey := strings.TrimPrefix(key, "CLAUDE_CODE_TENGU_")
			flagKey = strings.ToLower(flagKey)

			// Try to parse as JSON first, then as boolean, then as string
			var parsedValue any
			if err := json.Unmarshal([]byte(value), &parsedValue); err == nil {
				// Successfully parsed as JSON
			} else if strings.ToLower(value) == "true" || value == "1" {
				parsedValue = true
			} else if strings.ToLower(value) == "false" || value == "0" {
				parsedValue = false
			} else {
				// Try to parse as number
				if intVal, err := parseInt(value); err == nil {
					parsedValue = intVal
				} else if floatVal, err := parseFloat(value); err == nil {
					parsedValue = floatVal
				} else {
					parsedValue = value
				}
			}

			m.flags[flagKey] = FeatureFlag{
				Key:         flagKey,
				Value:       parsedValue,
				Source:      "environment",
				LastUpdated: time.Now(),
			}
		}
	}
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func parseFloat(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

// loadFromConfigFile loads feature flags from configuration file
func (m *Manager) loadFromConfigFile() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to load from ~/.claude/growthbook.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	configPath := filepath.Join(homeDir, ".claude", "growthbook.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// File doesn't exist or can't be read
		return
	}

	var config struct {
		Features map[string]FeatureFlag `json:"features"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		// Invalid JSON
		return
	}

	for key, flag := range config.Features {
		flag.Source = "config"
		flag.LastUpdated = time.Now()
		m.flags[key] = flag
	}
}

// IsOn returns true if a feature flag is enabled (truthy)
// This mirrors TS GrowthBook feature() function
func (m *Manager) IsOn(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// First check GrowthBook flags
	if flag, ok := m.flags[key]; ok {
		switch v := flag.Value.(type) {
		case bool:
			return v
		case string:
			return strings.ToLower(v) == "true" || v == "1"
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return v != 0
		case float32, float64:
			return v != 0.0
		default:
			return false
		}
	}

	// Fall back to legacy featuregates for compatibility
	return featuregates.Feature(key)
}

// Get returns the value of a feature flag
func (m *Manager) Get(key string) any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if flag, ok := m.flags[key]; ok {
		return flag.Value
	}

	// Fall back to environment variable check
	if featuregates.Feature(key) {
		return true
	}

	return nil
}

// GetWithDefault returns the value of a feature flag with a default value
func (m *Manager) GetWithDefault(key string, defaultValue any) any {
	value := m.Get(key)
	if value == nil {
		return defaultValue
	}
	return value
}

// SetAttribute sets a user attribute for feature flag evaluation
func (m *Manager) SetAttribute(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attributes[key] = value
}

// GetAttributes returns all user attributes
func (m *Manager) GetAttributes() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	attrs := make(map[string]any)
	for k, v := range m.attributes {
		attrs[k] = v
	}
	return attrs
}

// GetAllFlags returns all feature flags
func (m *Manager) GetAllFlags() map[string]FeatureFlag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flags := make(map[string]FeatureFlag)
	for k, v := range m.flags {
		flags[k] = v
	}
	return flags
}

// UpdateFlags updates feature flags from an external source
func (m *Manager) UpdateFlags(flags map[string]FeatureFlag) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, flag := range flags {
		flag.LastUpdated = time.Now()
		m.flags[key] = flag
	}
}

// Evaluate evaluates a feature flag with attributes
func (m *Manager) Evaluate(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	flag, ok := m.flags[key]
	if !ok {
		return nil, false
	}

	// If flag has rules, evaluate them
	if len(flag.Rules) > 0 {
		for _, rule := range flag.Rules {
			if m.evaluateRule(rule) {
				return rule.Value, true
			}
		}
	}

	// Return default value
	return flag.Value, true
}

// evaluateRule evaluates a feature flag rule against current attributes
func (m *Manager) evaluateRule(rule FeatureFlagRule) bool {
	if rule.Force {
		return true
	}

	if rule.Condition == nil {
		return false
	}

	// Simple condition evaluation - can be extended for more complex logic
	for attrKey, expectedValue := range rule.Condition {
		actualValue, ok := m.attributes[attrKey]
		if !ok {
			return false
		}

		// Simple equality check - can be extended for more operators
		if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
			return false
		}
	}

	return true
}

// Convenience functions for common feature flags

// IsTenguAmberStoat returns true if the "tengu_amber_stoat" flag is enabled
func IsTenguAmberStoat() bool {
	return DefaultManager().IsOn("amber_stoat")
}

// IsTenguMothCorpse returns true if the "tengu_moth_corpse" flag is enabled
func IsTenguMothCorpse() bool {
	return DefaultManager().IsOn("moth_corpse")
}

// IsTenguPaperHalyard returns true if the "tengu_paper_halyard" flag is enabled
func IsTenguPaperHalyard() bool {
	return DefaultManager().IsOn("paper_halyard")
}

// IsTenguHiveEvidence returns true if the "tengu_hive_evidence" flag is enabled
func IsTenguHiveEvidence() bool {
	return DefaultManager().IsOn("hive_evidence")
}

// Init initializes the GrowthBook manager
func Init(config ...Config) {
	manager := DefaultManager()
	if len(config) > 0 {
		manager.mu.Lock()
		manager.config = config[0]
		// Copy attributes from config to manager
		if config[0].Attributes != nil {
			for k, v := range config[0].Attributes {
				manager.attributes[k] = v
			}
		}
		manager.mu.Unlock()
	}
	manager.initialized = true
}

// IsInitialized returns true if GrowthBook has been initialized
func IsInitialized() bool {
	return DefaultManager().initialized
}