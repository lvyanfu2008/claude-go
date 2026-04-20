# Diagnostics and GrowthBook Integration

This document describes the new diagnostic logging and GrowthBook feature flag integration implemented in Go, mirroring TS-side functionality.

## Overview

Two new packages have been added:
1. `goc/diagnostics` - Enhanced diagnostic logging with PII scrubbing, structured logging, context timing, and analytics
2. `goc/growthbook` - GrowthBook-style feature flag management with rules and attributes

## Diagnostics Package

### PII Scrubbing

The package automatically detects and redacts Personally Identifiable Information (PII) including:
- Email addresses
- IP addresses (IPv4)
- Credit card numbers
- Social security numbers (US)
- Phone numbers

```go
import "goc/diagnostics"

// Log message with PII scrubbing
diagnostics.LogForDiagnosticsNoPII("User %s logged in from IP %s", "john@example.com", "192.168.1.100")
// Logs: "User [REDACTED] logged in from IP [REDACTED]"
```

### Structured Logging

Log structured diagnostic entries with automatic PII scrubbing:

```go
diagnostics.LogStructured("INFO", "auth", "User authentication", map[string]any{
    "user_id": "user123@example.com",  // Will be scrubbed to [REDACTED]
    "attempt": 3,
    "success": true,
})
```

### Context Load Timing

Track timing of different phases during context loading:

```go
tracker := diagnostics.NewContextLoadTracker()
tracker.StartPhase("git_status")
// ... do work ...
tracker.EndPhase("git_status")

tracker.StartPhase("file_scan")
// ... do work ...
tracker.EndPhase("file_scan")

tracker.Complete(totalItems, success)
```

### Analytics Events

Emit analytics events with PII scrubbing:

```go
// Initialize analytics system (automatically done in process-user-input)
diagnostics.InitAnalytics()

// Emit an event
diagnostics.EmitAnalyticsEvent("button_click", map[string]any{
    "button_id": "submit",
    "user_email": "test@example.com",  // Will be scrubbed
    "timestamp": time.Now().Unix(),
})
```

Analytics events are written to:
1. Stderr (for compatibility with existing systems)
2. File specified by `CLAUDE_CODE_ANALYTICS_LOG_FILE` environment variable
3. Default analytics file in debug log directory

## GrowthBook Package

### Feature Flag Management

GrowthBook-style feature flags with support for:
- Environment variable loading (`FEATURE_*` and `CLAUDE_CODE_TENGU_*`)
- Configuration file loading (`~/.claude/growthbook.json`)
- Rule-based evaluation with attributes
- Type conversion (bool, int, float, string)

### Basic Usage

```go
import "goc/growthbook"

// Initialize (automatically done in process-user-input)
growthbook.Init()

// Check if a feature is enabled
if growthbook.DefaultManager().IsOn("new_ui") {
    // Use new UI
}

// Get feature flag value
value := growthbook.DefaultManager().Get("api_timeout")
if timeout, ok := value.(int); ok {
    // Use timeout value
}

// Get with default
value := growthbook.DefaultManager().GetWithDefault("retry_count", 3)
```

### Environment Variables

```bash
# Legacy FEATURE_* format (boolean only)
FEATURE_NEW_UI=1
FEATURE_ENABLE_LOGGING=true

# GrowthBook-style TENGU_* format (supports types)
CLAUDE_CODE_TENGU_API_TIMEOUT=5000  # integer
CLAUDE_CODE_TENGU_ENABLE_CACHE=true  # boolean
CLAUDE_CODE_TENGU_DEFAULT_NAME="production"  # string
```

### Configuration File

Create `~/.claude/growthbook.json`:

```json
{
  "features": {
    "from_config": {
      "key": "from_config",
      "value": "config_value",
      "description": "Loaded from config file"
    },
    "numeric_flag": {
      "key": "numeric_flag",
      "value": 100
    },
    "conditional_flag": {
      "key": "conditional_flag",
      "value": "default",
      "rules": [
        {
          "condition": {"plan": "premium"},
          "value": "premium_value"
        },
        {
          "condition": {"country": "US"},
          "value": "us_value"
        }
      ]
    }
  }
}
```

### Rule-Based Evaluation

```go
manager := growthbook.DefaultManager()

// Set user attributes for rule evaluation
manager.SetAttribute("plan", "premium")
manager.SetAttribute("country", "US")

// Evaluate flag with rules
value, ok := manager.Evaluate("conditional_flag")
if ok {
    // value will be "premium_value" (first matching rule)
}
```

### Convenience Functions

Pre-defined convenience functions for common feature flags:

```go
if growthbook.IsTenguAmberStoat() {
    // amber_stoat flag is enabled
}

if growthbook.IsTenguMothCorpse() {
    // moth_corpse flag is enabled
}

if growthbook.IsTenguPaperHalyard() {
    // paper_halyard flag is enabled
}

if growthbook.IsTenguHiveEvidence() {
    // hive_evidence flag is enabled
}
```

## Integration with process-user-input

The `process-user-input` command has been updated to:

1. **Initialize analytics system** - Automatically calls `diagnostics.InitAnalytics()`
2. **Initialize GrowthBook** - Automatically calls `growthbook.Init()`
3. **Use enhanced analytics** - Events are sent through `diagnostics.EmitAnalyticsEvent()` with PII scrubbing
4. **Track context loading** - Measures timing of different processing phases

## Environment Variables

### Diagnostics
- `CLAUDE_CODE_DIAG_LOG_FILE` - Path for diagnostic log file
- `CLAUDE_CODE_ANALYTICS_LOG_FILE` - Path for analytics log file

### GrowthBook
- `FEATURE_*` - Legacy boolean feature flags
- `CLAUDE_CODE_TENGU_*` - GrowthBook-style feature flags with type support

### Process User Input Debugging
- `CLAUDE_DEBUG_PROCESS_USER_INPUT` - Enable debug logging
- `GOC_PROCESS_USER_INPUT_DEBUG_LOG` - Debug log file path
- `GOC_PROCESS_USER_INPUT_DEBUG_TO_STDERR` - Log to stderr

## Testing

Both packages include comprehensive tests:

```bash
# Run diagnostics tests
go test ./diagnostics/...

# Run growthbook tests  
go test ./growthbook/...
```

## Backward Compatibility

The implementation maintains backward compatibility:

1. **Diagnostics** - Still writes to stderr with `GOC_ANALYTICS_EVENT:` prefix
2. **GrowthBook** - Falls back to `featuregates.Feature()` for legacy feature checks
3. **Process-user-input** - Maintains existing `emitAnalyticsEventToStderr()` function

## Example: Complete Integration

```go
// In application startup
diagnostics.InitAnalytics()
growthbook.Init(growthbook.Config{
    Attributes: map[string]any{
        "environment": "production",
        "user_role": "admin",
    },
})

// During request processing
tracker := diagnostics.NewContextLoadTracker()
tracker.StartPhase("request_processing")

// Check feature flags
if growthbook.DefaultManager().IsOn("experimental_feature") {
    diagnostics.LogStructured("INFO", "feature", "Using experimental feature", nil)
    // Use experimental feature
}

tracker.EndPhase("request_processing")

// Emit analytics
diagnostics.EmitAnalyticsEvent("request_completed", map[string]any{
    "duration_ms": 150,
    "feature_used": "experimental_feature",
})

tracker.Complete(1, true)
```