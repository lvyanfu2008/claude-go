// Package sessionmemory mirrors src/services/SessionMemory/sessionMemoryUtils.ts
// and src/services/SessionMemory/sessionMemory.ts.
//
// Session Memory automatically maintains a markdown notes file (summary.md) about
// the current conversation using a forked subagent. It runs as a post-sampling hook
// after each user turn.
package sessionmemory

import (
	"sync"
	"time"

	"goc/growthbook"
)

// Config mirrors TS SessionMemoryConfig.
type Config struct {
	// MinimumMessageTokensToInit is the minimum context window tokens before
	// initializing session memory. Uses the same token counting as autocompact.
	MinimumMessageTokensToInit int
	// MinimumTokensBetweenUpdate is the minimum context window growth (in tokens)
	// between session memory updates.
	MinimumTokensBetweenUpdate int
	// ToolCallsBetweenUpdates is the number of tool calls between session memory updates.
	ToolCallsBetweenUpdates int
}

// DefaultConfig mirrors TS DEFAULT_SESSION_MEMORY_CONFIG.
var DefaultConfig = Config{
	MinimumMessageTokensToInit: 10000,
	MinimumTokensBetweenUpdate: 5000,
	ToolCallsBetweenUpdates:    3,
}

// SMCompactConfig mirrors TS SessionMemoryCompactConfig.
type SMCompactConfig struct {
	MinTokens            int
	MinTextBlockMessages int
	MaxTokens            int
}

// DefaultSMCompactConfig mirrors TS DEFAULT_SM_COMPACT_CONFIG.
var DefaultSMCompactConfig = SMCompactConfig{
	MinTokens:            10_000,
	MinTextBlockMessages: 5,
	MaxTokens:            40_000,
}

// extractionWaitTimeoutMs mirrors TS EXTRACTION_WAIT_TIMEOUT_MS.
const extractionWaitTimeoutMs = 15000

// extractionStaleThresholdMs mirrors TS EXTRACTION_STALE_THRESHOLD_MS.
const extractionStaleThresholdMs = 60000

// sessionMemoryFileBaseName is the name of the session memory file.
const sessionMemoryFileName = "summary.md"

// State holds the mutable module-level state for session memory extraction.
// Mirrors the module-level variables in sessionMemoryUtils.ts.
type State struct {
	mu sync.Mutex

	// Config holds the current session memory configuration.
	// Mirrors TS sessionMemoryConfig.
	Config Config

	// ConfigInitialized is set true after initConfigIfNeeded runs once.
	// Mirrors TS memoize() on initSessionMemoryConfigIfNeeded.
	ConfigInitialized bool

	// LastSummarizedMessageID is the UUID of the last message up to which
	// session memory is current. Mirrors TS lastSummarizedMessageId.
	LastSummarizedMessageID string

	// ExtractionStartedAt is set to the time when extraction starts (epoch ms),
	// and cleared when extraction completes. Mirrors TS extractionStartedAt.
	ExtractionStartedAt int64

	// TokensAtLastExtraction is the context window token count at the time
	// of the last extraction. Mirrors TS tokensAtLastExtraction.
	TokensAtLastExtraction int

	// SessionMemoryInitialized is set true after the first time the
	// initialization threshold is met. Mirrors TS sessionMemoryInitialized.
	SessionMemoryInitialized bool

	// LastMemoryMessageUUID tracks the UUID of the last message processed
	// during shouldExtractMemory. Mirrors TS lastMemoryMessageUuid.
	LastMemoryMessageUUID string

	// HasLoggedGateFailure guards against spamming the gate-disabled log event.
	// Mirrors TS hasLoggedGateFailure.
	HasLoggedGateFailure bool

	// SMCompactConfig holds the session memory compact configuration.
	// Mirrors TS smCompactConfig.
	SMCompactConfig SMCompactConfig

	// SMCompactConfigInitialized is set true after initSessionMemoryCompactConfig runs once.
	// Mirrors TS configInitialized.
	SMCompactConfigInitialized bool
}

// NewState creates an initialised State with default config.
func NewState() *State {
	return &State{
		Config: DefaultConfig,
	}
}

// SetConfig updates the session memory configuration.
// Mirrors TS setSessionMemoryConfig.
func (s *State) SetConfig(cfg Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config = cfg
}

// GetConfig returns a copy of the current session memory configuration.
// Mirrors TS getSessionMemoryConfig.
func (s *State) GetConfig() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Config
}

// GetLastSummarizedMessageID returns the ID of the last summarized message.
// Mirrors TS getLastSummarizedMessageId.
func (s *State) GetLastSummarizedMessageID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.LastSummarizedMessageID
}

// SetLastSummarizedMessageID sets the last summarized message ID.
// Mirrors TS setLastSummarizedMessageId.
func (s *State) SetLastSummarizedMessageID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastSummarizedMessageID = id
}

// MarkExtractionStarted records that extraction has started.
// Mirrors TS markExtractionStarted.
func (s *State) MarkExtractionStarted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ExtractionStartedAt = nowMS()
}

// MarkExtractionCompleted records that extraction has completed.
// Mirrors TS markExtractionCompleted.
func (s *State) MarkExtractionCompleted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ExtractionStartedAt = 0
}

// IsExtractionInProgress returns true if an extraction is currently running.
func (s *State) IsExtractionInProgress() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ExtractionStartedAt == 0 {
		return false
	}
	age := nowMS() - s.ExtractionStartedAt
	return age <= extractionStaleThresholdMs
}

// RecordExtractionTokenCount records the context size at extraction time.
// Mirrors TS recordExtractionTokenCount.
func (s *State) RecordExtractionTokenCount(tokenCount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TokensAtLastExtraction = tokenCount
}

// IsSessionMemoryInitialized returns true if session memory has met the init threshold.
// Mirrors TS isSessionMemoryInitialized.
func (s *State) IsSessionMemoryInitialized() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.SessionMemoryInitialized
}

// MarkSessionMemoryInitialized marks session memory as initialized.
// Mirrors TS markSessionMemoryInitialized.
func (s *State) MarkSessionMemoryInitialized() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SessionMemoryInitialized = true
}

// HasMetInitializationThreshold returns true if currentTokenCount meets the init threshold.
// Mirrors TS hasMetInitializationThreshold.
func (s *State) HasMetInitializationThreshold(currentTokenCount int) bool {
	cfg := s.GetConfig()
	return currentTokenCount >= cfg.MinimumMessageTokensToInit
}

// HasMetUpdateThreshold returns true if context window growth since last extraction
// meets the minimum tokens between updates threshold.
// Mirrors TS hasMetUpdateThreshold.
func (s *State) HasMetUpdateThreshold(currentTokenCount int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	tokensSince := currentTokenCount - s.TokensAtLastExtraction
	return tokensSince >= s.Config.MinimumTokensBetweenUpdate
}

// GetToolCallsBetweenUpdates returns the configured number of tool calls between updates.
// Mirrors TS getToolCallsBetweenUpdates.
func (s *State) GetToolCallsBetweenUpdates() int {
	cfg := s.GetConfig()
	return cfg.ToolCallsBetweenUpdates
}

// GetSMCompactConfig returns a copy of the current SM compact config.
// Mirrors TS getSessionMemoryCompactConfig.
func (s *State) GetSMCompactConfig() SMCompactConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.SMCompactConfig
}

// SetSMCompactConfig updates the SM compact config.
// Mirrors TS setSessionMemoryCompactConfig.
func (s *State) SetSMCompactConfig(cfg SMCompactConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SMCompactConfig = cfg
}

// InitSMCompactConfig initializes the SM compact config from remote
// (GrowthBook). Only runs once per session.
// Mirrors TS initSessionMemoryCompactConfig.
func (s *State) InitSMCompactConfig() {
	s.mu.Lock()
	if s.SMCompactConfigInitialized {
		s.mu.Unlock()
		return
	}
	s.SMCompactConfigInitialized = true
	s.mu.Unlock()

	// Try to load remote config from GrowthBook.
	// Falls back to DefaultSMCompactConfig if no remote config is set.
	remote := growthbook.DefaultManager().Get("tengu_sm_compact_config")

	cfg := DefaultSMCompactConfig

	if m, ok := remote.(map[string]any); ok {
		if v, ok := m["minTokens"].(float64); ok && v > 0 {
			cfg.MinTokens = int(v)
		}
		if v, ok := m["minTextBlockMessages"].(float64); ok && v > 0 {
			cfg.MinTextBlockMessages = int(v)
		}
		if v, ok := m["maxTokens"].(float64); ok && v > 0 {
			cfg.MaxTokens = int(v)
		}
	}

	s.SetSMCompactConfig(cfg)
}

// Reset resets all state to defaults (useful for testing).
// Mirrors TS resetSessionMemoryState.
func (s *State) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config = DefaultConfig
	s.ConfigInitialized = false
	s.LastSummarizedMessageID = ""
	s.ExtractionStartedAt = 0
	s.TokensAtLastExtraction = 0
	s.SessionMemoryInitialized = false
	s.LastMemoryMessageUUID = ""
	s.HasLoggedGateFailure = false
	s.SMCompactConfig = DefaultSMCompactConfig
	s.SMCompactConfigInitialized = false
}

// nowMS returns the current time in milliseconds since Unix epoch.
func nowMS() int64 {
	return time.Now().UnixMilli()
}
