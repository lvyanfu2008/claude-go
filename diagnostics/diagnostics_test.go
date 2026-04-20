package diagnostics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScrubPII(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "email address",
			input:    "Contact me at john.doe@example.com for details",
			expected: "Contact me at [REDACTED] for details",
		},
		{
			name:     "IP address",
			input:    "Server at 192.168.1.1 is down",
			expected: "Server at [REDACTED] is down",
		},
		{
			name:     "phone number",
			input:    "Call 555-123-4567 for support",
			expected: "Call [REDACTED] for support",
		},
		{
			name:     "SSN",
			input:    "SSN: 123-45-6789",
			expected: "SSN: [REDACTED]",
		},
		{
			name:     "credit card",
			input:    "Card: 4111-1111-1111-1111",
			expected: "Card: [REDACTED]",
		},
		{
			name:     "no PII",
			input:    "This is a normal log message",
			expected: "This is a normal log message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scrubPII(tt.input)
			if result != tt.expected {
				t.Errorf("scrubPII(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStructuredDiagnostic(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	oldEnv := os.Getenv("CLAUDE_CODE_DIAG_LOG_FILE")
	defer os.Setenv("CLAUDE_CODE_DIAG_LOG_FILE", oldEnv)

	testFile := filepath.Join(tempDir, "test.log")
	os.Setenv("CLAUDE_CODE_DIAG_LOG_FILE", testFile)

	// Test LogStructured
	LogStructured("INFO", "test", "Test message", map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": "email@example.com", // Should be scrubbed
	})

	// Read the log file
	time.Sleep(100 * time.Millisecond) // Give time for async write
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 {
		t.Fatal("No log lines written")
	}

	// Parse the last line as JSON
	var entry StructuredDiagnostic
	lastLine := lines[len(lines)-1]
	// diaglog.Line adds a timestamp prefix, extract JSON part
	// Format: "timestamp json"
	parts := strings.SplitN(lastLine, " ", 2)
	jsonStr := lastLine
	if len(parts) == 2 {
		jsonStr = parts[1]
	}
	if err := json.Unmarshal([]byte(jsonStr), &entry); err != nil {
		t.Fatalf("Failed to parse log line as JSON: %v\nLine: %s\nJSON: %s", err, lastLine, jsonStr)
	}

	// Verify the entry
	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO, got %s", entry.Level)
	}
	if entry.Category != "test" {
		t.Errorf("Expected category test, got %s", entry.Category)
	}
	if entry.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got %s", entry.Message)
	}

	// Check that PII was scrubbed from data
	if email, ok := entry.Data["key3"].(string); ok && email != "[REDACTED]" {
		t.Errorf("Expected email to be scrubbed to [REDACTED], got %s", email)
	}
}

func TestContextLoadTracker(t *testing.T) {
	tracker := NewContextLoadTracker()

	// Simulate context loading phases
	tracker.StartPhase("git_status")
	time.Sleep(10 * time.Millisecond)
	tracker.EndPhase("git_status")

	tracker.StartPhase("file_scan")
	time.Sleep(20 * time.Millisecond)
	tracker.EndPhase("file_scan")

	tracker.StartPhase("memory_load")
	time.Sleep(15 * time.Millisecond)
	tracker.EndPhase("memory_load")

	tracker.Complete(3, true)

	// The tracker should have logged the phases and completion
	// We can't easily verify the logs without mocking, but we can
	// verify the tracker internal state
	if len(tracker.durations) != 3 {
		t.Errorf("Expected 3 phase durations, got %d", len(tracker.durations))
	}

	// Check that durations are recorded
	for _, phase := range []string{"git_status", "file_scan", "memory_load"} {
		if _, ok := tracker.durations[phase]; !ok {
			t.Errorf("Missing duration for phase: %s", phase)
		}
	}
}

func TestAnalyticsEvent(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	oldEnv := os.Getenv("CLAUDE_CODE_ANALYTICS_LOG_FILE")
	defer os.Setenv("CLAUDE_CODE_ANALYTICS_LOG_FILE", oldEnv)

	testFile := filepath.Join(tempDir, "analytics.jsonl")
	os.Setenv("CLAUDE_CODE_ANALYTICS_LOG_FILE", testFile)

	// Re-initialize analytics to pick up the test file
	InitAnalytics()

	// Emit an analytics event
	EmitAnalyticsEvent("test_event", map[string]any{
		"action":   "button_click",
		"user_id":  "user123@example.com", // Should be scrubbed
		"count":    5,
		"metadata": "Some metadata",
	})

	// Give time for async write
	time.Sleep(100 * time.Millisecond)

	// Read the analytics file
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read analytics file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 {
		t.Fatal("No analytics lines written")
	}

	// Parse the last line as JSON
	var event AnalyticsEvent
	lastLine := lines[len(lines)-1]
	if err := json.Unmarshal([]byte(lastLine), &event); err != nil {
		t.Fatalf("Failed to parse analytics line as JSON: %v\nLine: %s", err, lastLine)
	}

	// Verify the event
	if event.Name != "test_event" {
		t.Errorf("Expected event name test_event, got %s", event.Name)
	}
	if event.Payload["action"] != "button_click" {
		t.Errorf("Expected action button_click, got %v", event.Payload["action"])
	}
	if event.Payload["user_id"] != "[REDACTED]" {
		t.Errorf("Expected user_id to be scrubbed to [REDACTED], got %v", event.Payload["user_id"])
	}
}

func TestLogForDiagnosticsNoPII(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	oldEnv := os.Getenv("CLAUDE_CODE_DIAG_LOG_FILE")
	defer os.Setenv("CLAUDE_CODE_DIAG_LOG_FILE", oldEnv)

	testFile := filepath.Join(tempDir, "test.log")
	os.Setenv("CLAUDE_CODE_DIAG_LOG_FILE", testFile)

	// Test with PII that should be scrubbed
	LogForDiagnosticsNoPII("User %s logged in from IP %s", "john@example.com", "192.168.1.100")

	// Give time for async write
	time.Sleep(100 * time.Millisecond)

	// Read the log file
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logLine := strings.TrimSpace(string(content))
	if !strings.Contains(logLine, "[REDACTED]") {
		t.Errorf("Expected PII to be scrubbed, got: %s", logLine)
	}
	if strings.Contains(logLine, "john@example.com") || strings.Contains(logLine, "192.168.1.100") {
		t.Errorf("PII should not appear in log: %s", logLine)
	}
}