# Memory Management Behavior Alignment

## Summary

This document describes the changes made to align Go and TypeScript memory management behavior.

## Changes Made

### 1. Default Memory Skip Index Behavior

**File**: `commands/gou_demo_system.go`
- Changed `MemorySkipIndex` to default to `true` (matching TypeScript's `tengu_moth_copse: true`)
- Can be overridden with `CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX=1`

**Before**:
```go
o.MemorySkipIndex = featuregates.Feature("MOTH_COPSE")
```

**After**:
```go
// Default to true to match TypeScript behavior where tengu_moth_copse defaults to true
// This skips the manual MEMORY.md maintenance instruction since memory files are 
// prefetched via attachments instead
o.MemorySkipIndex = featuregates.Feature("MOTH_COPSE") || !envTruthyGo("CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX")
```

### 2. Memory File Filtering Alignment

**File**: `claudemd/format.go`
- Updated `FilterInjectedMemoryFiles` to default to filtering out `MEMORY.md` index files
- Aligns with TypeScript behavior where memory files are prefetched via attachments

**Before**:
```go
if !truthy(os.Getenv("CLAUDE_CODE_TENGU_MOTH_COPSE")) {
    return files
}
```

**After**:
```go
// Default to filtering (skip index) unless explicitly disabled
skipIndex := truthy(os.Getenv("CLAUDE_CODE_TENGU_MOTH_COPSE")) || !truthy(os.Getenv("CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX"))
if !skipIndex {
    return files
}
```

## Impact

### Positive Changes
1. **Behavior Consistency**: Go and TypeScript now have the same default memory management behavior
2. **User Experience**: Users no longer see manual `MEMORY.md` maintenance instructions by default
3. **Future-Ready**: Aligns with the direction toward automated memory management

### Migration Path
- **Existing Users**: No action needed - new behavior is more user-friendly
- **Advanced Users**: Can restore old behavior with `CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX=1`

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FEATURE_MOTH_COPSE` | `false` | Legacy feature flag for memory prefetching |
| `CLAUDE_CODE_TENGU_MOTH_COPSE` | `false` | Legacy environment override |
| `CLAUDE_CODE_GO_DISABLE_MEMORY_SKIP_INDEX` | `false` | New opt-out flag for skipping memory index |

## Testing

Added comprehensive tests:
- `commands/gou_demo_system_test.go`: Verifies default behavior and environment overrides
- `claudemd/format_test.go`: Verifies memory file filtering behavior

Run tests with:
```bash
go test ./commands -run TestMemorySkipIndexDefaultBehavior
go test ./claudemd -run TestFilterInjectedMemoryFilesDefaultBehavior
```