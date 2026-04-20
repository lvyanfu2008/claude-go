package growthbook

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestGrowthBookManager(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	for _, env := range os.Environ() {
		key, value, _ := splitEnv(env)
		originalEnv[key] = value
	}
	defer func() {
		// Restore environment
		for key, value := range originalEnv {
			os.Setenv(key, value)
		}
	}()

	// Set test environment variables
	os.Setenv("FEATURE_TEST_FLAG", "1")
	os.Setenv("CLAUDE_CODE_TENGU_TEST_NUMERIC", "42")
	os.Setenv("CLAUDE_CODE_TENGU_TEST_BOOL", "true")
	os.Setenv("CLAUDE_CODE_TENGU_TEST_STRING", "hello")

	// Create fresh manager (not singleton)
	manager := &Manager{
		flags:      make(map[string]FeatureFlag),
		attributes: make(map[string]any),
	}
	manager.loadFromEnvironment()
	manager.loadFromConfigFile()

	// Test IsOn with FEATURE_ flag
	if !manager.IsOn("TEST_FLAG") {
		t.Error("FEATURE_TEST_FLAG should be enabled")
	}

	// Test Get with TENGU_ numeric flag
	value := manager.Get("test_numeric")
	// Convert to float64 for comparison since JSON numbers are float64
	if floatVal, ok := value.(float64); !ok || floatVal != 42 {
		t.Errorf("Expected test_numeric = 42, got %v (type: %T)", value, value)
	}

	// Test Get with TENGU_ bool flag
	value = manager.Get("test_bool")
	if value != true {
		t.Errorf("Expected test_bool = true, got %v", value)
	}

	// Test Get with TENGU_ string flag
	value = manager.Get("test_string")
	if value != "hello" {
		t.Errorf("Expected test_string = 'hello', got %v", value)
	}

	// Test GetWithDefault
	defaultValue := manager.GetWithDefault("non_existent", "default")
	if defaultValue != "default" {
		t.Errorf("Expected default value 'default', got %v", defaultValue)
	}
}

func TestGrowthBookAttributes(t *testing.T) {
	manager := &Manager{
		flags:      make(map[string]FeatureFlag),
		attributes: make(map[string]any),
	}

	// Set attributes
	manager.SetAttribute("user_id", "123")
	manager.SetAttribute("plan", "premium")
	manager.SetAttribute("country", "US")

	// Get attributes
	attrs := manager.GetAttributes()
	if attrs["user_id"] != "123" {
		t.Errorf("Expected user_id = '123', got %v", attrs["user_id"])
	}
	if attrs["plan"] != "premium" {
		t.Errorf("Expected plan = 'premium', got %v", attrs["plan"])
	}
	if attrs["country"] != "US" {
		t.Errorf("Expected country = 'US', got %v", attrs["country"])
	}
}

func TestGrowthBookEvaluate(t *testing.T) {
	manager := &Manager{
		flags:      make(map[string]FeatureFlag),
		attributes: make(map[string]any),
	}

	// Create a test flag with rules
	flag := FeatureFlag{
		Key:    "test_conditional",
		Value:  "default",
		Source: "test",
		Rules: []FeatureFlagRule{
			{
				Condition: map[string]any{"plan": "premium"},
				Value:     "premium_value",
			},
			{
				Condition: map[string]any{"country": "US"},
				Value:     "us_value",
			},
		},
		LastUpdated: time.Now(),
	}

	// Update manager with test flag
	manager.UpdateFlags(map[string]FeatureFlag{
		"test_conditional": flag,
	})

	// Test evaluation with no attributes (should return default)
	value, ok := manager.Evaluate("test_conditional")
	if !ok {
		t.Error("Should find test_conditional flag")
	}
	if value != "default" {
		t.Errorf("Expected default value 'default', got %v", value)
	}

	// Set attribute and test again
	manager.SetAttribute("plan", "premium")
	value, ok = manager.Evaluate("test_conditional")
	if !ok {
		t.Error("Should find test_conditional flag")
	}
	if value != "premium_value" {
		t.Errorf("Expected premium_value, got %v", value)
	}

	// Test with force rule
	flagWithForce := FeatureFlag{
		Key:    "test_force",
		Value:  "default",
		Source: "test",
		Rules: []FeatureFlagRule{
			{
				Force: true,
				Value: "forced_value",
			},
		},
		LastUpdated: time.Now(),
	}

	manager.UpdateFlags(map[string]FeatureFlag{
		"test_force": flagWithForce,
	})

	value, ok = manager.Evaluate("test_force")
	if !ok {
		t.Error("Should find test_force flag")
	}
	if value != "forced_value" {
		t.Errorf("Expected forced_value, got %v", value)
	}
}

func TestGrowthBookConfigFile(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "growthbook.json")
	configContent := `{
		"features": {
			"from_config": {
				"key": "from_config",
				"value": "config_value",
				"description": "Loaded from config file"
			},
			"numeric_flag": {
				"key": "numeric_flag",
				"value": 100
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Temporarily set HOME to temp directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create a new manager to load from config
	manager := &Manager{
		flags:      make(map[string]FeatureFlag),
		attributes: make(map[string]any),
	}
	manager.loadFromConfigFile()

	// Test flags loaded from config
	value := manager.Get("from_config")
	if value != "config_value" {
		t.Errorf("Expected from_config = 'config_value', got %v", value)
	}

	value = manager.Get("numeric_flag")
	// JSON numbers are float64
	if floatVal, ok := value.(float64); !ok || floatVal != 100 {
		t.Errorf("Expected numeric_flag = 100, got %v (type: %T)", value, value)
	}
}

func TestConvenienceFunctions(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	for _, env := range os.Environ() {
		key, value, _ := splitEnv(env)
		originalEnv[key] = value
	}
	defer func() {
		// Restore environment
		for key, value := range originalEnv {
			os.Setenv(key, value)
		}
	}()

	// Test convenience functions with environment variables
	os.Setenv("CLAUDE_CODE_TENGU_AMBER_STOAT", "true")
	os.Setenv("CLAUDE_CODE_TENGU_MOTH_CORPSE", "1")
	os.Setenv("CLAUDE_CODE_TENGU_PAPER_HALYARD", "false")
	os.Setenv("CLAUDE_CODE_TENGU_HIVE_EVIDENCE", "0")

	// Reset manager to pick up new environment variables
	managerOnce = sync.Once{}
	_ = DefaultManager() // Reset the singleton

	// Test IsTenguAmberStoat
	if !IsTenguAmberStoat() {
		t.Error("IsTenguAmberStoat should return true")
	}

	// Test IsTenguMothCorpse
	if !IsTenguMothCorpse() {
		t.Error("IsTenguMothCorpse should return true")
	}

	// Test IsTenguPaperHalyard
	if IsTenguPaperHalyard() {
		t.Error("IsTenguPaperHalyard should return false")
	}

	// Test IsTenguHiveEvidence
	if IsTenguHiveEvidence() {
		t.Error("IsTenguHiveEvidence should return false")
	}
}

func TestInitAndInitialization(t *testing.T) {
	// Reset singleton for test
	managerOnce = sync.Once{}
	defaultManager = nil

	// Test initialization
	if IsInitialized() {
		t.Error("Should not be initialized before Init()")
	}

	// Initialize with custom config
	config := Config{
		APIKey:    "test-api-key",
		ClientKey: "test-client-key",
		APIHost:   "https://test.growthbook.io",
		Attributes: map[string]any{
			"environment": "test",
		},
	}
	Init(config)

	if !IsInitialized() {
		t.Error("Should be initialized after Init()")
	}

	// Verify config was set
	manager := DefaultManager()
	attrs := manager.GetAttributes()
	if attrs["environment"] != "test" {
		t.Errorf("Expected environment attribute = 'test', got %v", attrs["environment"])
	}
}

// Helper function to split environment variable
func splitEnv(env string) (string, string, bool) {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return env[:i], env[i+1:], true
		}
	}
	return env, "", false
}