// Package diagnostics provides enhanced diagnostic logging and analytics functionality
// that mirrors TS-side implementations like logForDiagnosticsNoPII.
package diagnostics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"goc/ccb-engine/debugpath"
	"goc/ccb-engine/diaglog"
)

// PIIPatterns defines regular expressions for detecting potential PII
var PIIPatterns = []*regexp.Regexp{
	// Email addresses
	regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
	// IP addresses (IPv4)
	regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
	// Credit card numbers (simplified)
	regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`),
	// Social security numbers (US)
	regexp.MustCompile(`\b\d{3}[ -]?\d{2}[ -]?\d{4}\b`),
	// Phone numbers (various formats)
	regexp.MustCompile(`\b(?:\+?1[ -]?)?\(?\d{3}\)?[ -]?\d{3}[ -]?\d{4}\b`),
}

// scrubPII removes or masks potential personally identifiable information from text
func scrubPII(text string) string {
	result := text
	for _, pattern := range PIIPatterns {
		result = pattern.ReplaceAllString(result, "[REDACTED]")
	}
	return result
}

// LogForDiagnosticsNoPII logs a diagnostic message with PII scrubbing
// This mirrors TS logForDiagnosticsNoPII function
func LogForDiagnosticsNoPII(format string, args ...any) {
	// Format the message
	msg := fmt.Sprintf(format, args...)

	// Scrub PII from the message
	scrubbedMsg := scrubPII(msg)

	// Use existing diaglog infrastructure
	diaglog.Line("%s", scrubbedMsg)
}

// StructuredDiagnostic represents a structured diagnostic log entry
type StructuredDiagnostic struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Category  string         `json:"category,omitempty"`
	Message   string         `json:"message"`
	Data      map[string]any `json:"data,omitempty"`
	Duration  *int64         `json:"duration_ms,omitempty"` // Optional duration in milliseconds
}

// LogStructured logs a structured diagnostic entry
func LogStructured(level, category, message string, data map[string]any) {
	entry := StructuredDiagnostic{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Category:  category,
		Message:   scrubPII(message),
		Data:      make(map[string]any),
	}

	// Scrub PII from data values
	if data != nil {
		for k, v := range data {
			switch val := v.(type) {
			case string:
				entry.Data[k] = scrubPII(val)
			default:
				entry.Data[k] = v
			}
		}
	}

	// Marshal to JSON
	b, err := json.Marshal(entry)
	if err != nil {
		// Fall back to simple logging
		LogForDiagnosticsNoPII("STRUCTURED_LOG_FAILED: level=%s category=%s message=%s", level, category, message)
		return
	}

	// Log the JSON line
	diaglog.Line("%s", string(b))
}

// ContextLoadTracker tracks context loading times
type ContextLoadTracker struct {
	startTime time.Time
	phases    map[string]time.Time
	durations map[string]time.Duration
	mutex     sync.Mutex
}

// NewContextLoadTracker creates a new context load tracker
func NewContextLoadTracker() *ContextLoadTracker {
	return &ContextLoadTracker{
		startTime: time.Now(),
		phases:    make(map[string]time.Time),
		durations: make(map[string]time.Duration),
	}
}

// StartPhase marks the start of a loading phase
func (t *ContextLoadTracker) StartPhase(phase string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.phases[phase] = time.Now()
}

// EndPhase marks the end of a loading phase and records duration
func (t *ContextLoadTracker) EndPhase(phase string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	start, ok := t.phases[phase]
	if !ok {
		return
	}

	duration := time.Since(start)
	t.durations[phase] = duration

	// Log the phase completion
	LogStructured("INFO", "context_load", fmt.Sprintf("Phase completed: %s", phase), map[string]any{
		"phase":    phase,
		"duration": duration.Milliseconds(),
	})
}

// Complete marks the completion of context loading and logs summary
func (t *ContextLoadTracker) Complete(totalItems int, success bool) {
	totalDuration := time.Since(t.startTime)

	summary := map[string]any{
		"total_duration_ms": totalDuration.Milliseconds(),
		"total_items":       totalItems,
		"success":           success,
		"phases":            make(map[string]int64),
	}

	// Add phase durations
	for phase, duration := range t.durations {
		summary["phases"].(map[string]int64)[phase] = duration.Milliseconds()
	}

	// Log the completion
	LogStructured("INFO", "context_load", "Context loading completed", summary)
}

// AnalyticsEvent represents an analytics event for tracking
type AnalyticsEvent struct {
	Name      string         `json:"name"`
	Timestamp string         `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
	SessionID string         `json:"session_id,omitempty"`
}

var (
	analyticsMu      sync.Mutex
	analyticsWriters []analyticsWriter
)

type analyticsWriter interface {
	Write(event AnalyticsEvent) error
}

// RegisterAnalyticsWriter registers a writer for analytics events
func RegisterAnalyticsWriter(writer analyticsWriter) {
	analyticsMu.Lock()
	defer analyticsMu.Unlock()
	analyticsWriters = append(analyticsWriters, writer)
}

// EmitAnalyticsEvent emits an analytics event to all registered writers
func EmitAnalyticsEvent(name string, payload map[string]any) {
	event := AnalyticsEvent{
		Name:      name,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   make(map[string]any),
		SessionID: debugpath.SessionID(),
	}

	// Scrub PII from payload
	if payload != nil {
		for k, v := range payload {
			switch val := v.(type) {
			case string:
				event.Payload[k] = scrubPII(val)
			default:
				event.Payload[k] = v
			}
		}
	}

	// Write to all registered writers
	analyticsMu.Lock()
	writers := append([]analyticsWriter(nil), analyticsWriters...)
	analyticsMu.Unlock()

	for _, writer := range writers {
		_ = writer.Write(event) // Best effort
	}
}

// stderrWriter writes analytics events to stderr (existing compatibility)
type stderrWriter struct{}

func (w stderrWriter) Write(event AnalyticsEvent) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Use the existing format for compatibility
	_, err = fmt.Fprintf(os.Stderr, "GOC_ANALYTICS_EVENT:%s\n", string(b))
	return err
}

// fileWriter writes analytics events to a file
type fileWriter struct {
	path string
	mu   sync.Mutex
}

func (w *fileWriter) Write(event AnalyticsEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	b, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(w.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Append to file
	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", string(b))
	return err
}

// InitAnalytics initializes the analytics system with default writers
func InitAnalytics() {
	// Always write to stderr for compatibility
	RegisterAnalyticsWriter(stderrWriter{})

	// Optionally write to analytics file if configured
	if path := strings.TrimSpace(os.Getenv("CLAUDE_CODE_ANALYTICS_LOG_FILE")); path != "" {
		RegisterAnalyticsWriter(&fileWriter{path: path})
	}

	// Also use debug log path if no specific analytics file is set
	if os.Getenv("CLAUDE_CODE_ANALYTICS_LOG_FILE") == "" {
		if debugPath := debugpath.ResolveLogPath(); debugPath != "" {
			analyticsPath := filepath.Join(filepath.Dir(debugPath), "analytics.jsonl")
			RegisterAnalyticsWriter(&fileWriter{path: analyticsPath})
		}
	}
}