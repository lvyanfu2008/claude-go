package toolresultpersist

import (
	"math"

	"goc/ccb-engine/diaglog"
)

// GetPersistenceThreshold mirrors TS getPersistenceThreshold(toolName, declaredMaxResultSizeChars).
//
// When declaredMaxResultSizeChars is +Inf (MaxInt64 sentinel), the tool opts out of persistence
// entirely (Read tool reads its own bounds — saving output to a file the model reads with Read
// is circular). Returns math.MaxInt64 unchanged.
//
// Hosts that need GrowthBook overrides wire a PerToolOverrideFn into ProcessOptions.
// When absent, falls back to min(declaredMax, DefaultMaxResultSizeChars).
func GetPersistenceThreshold(toolName string, declaredMaxResultSizeChars int64, opts ProcessOptions) int64 {
	if declaredMaxResultSizeChars <= 0 || declaredMaxResultSizeChars == math.MaxInt64 {
		return declaredMaxResultSizeChars
	}
	if opts.PersistThresholdOverride != nil {
		if override, ok := opts.PersistThresholdOverride(toolName); ok && override > 0 {
			return override
		}
	}
	fallback := int64(DefaultMaxResultSizeChars)
	if declaredMaxResultSizeChars < fallback {
		return declaredMaxResultSizeChars
	}
	return fallback
}

// IsToolResultContentEmpty mirrors TS isToolResultContentEmpty.
// Returns true for nil, empty string, whitespace-only string, empty arrays,
// and arrays whose only blocks are text blocks with empty/whitespace text.
func IsToolResultContentEmpty(content any) bool {
	if content == nil {
		return true
	}
	switch v := content.(type) {
	case string:
		return len(v) == 0 || allWhitespace(v)
	case []any:
		if len(v) == 0 {
			return true
		}
		for _, block := range v {
			bm, ok := block.(map[string]any)
			if !ok {
				return false
			}
			typ, _ := bm["type"].(string)
			if typ != "text" {
				return false
			}
			txt, _ := bm["text"].(string)
			if len(txt) > 0 && !allWhitespace(txt) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func allWhitespace(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

// contentSize returns the character length of tool result content.
// For string content: len(string). For array content: sum of text block lengths.
func contentSize(content any) int {
	switch v := content.(type) {
	case string:
		return len(v)
	case []any:
		sum := 0
		for _, block := range v {
			bm, ok := block.(map[string]any)
			if !ok {
				continue
			}
			if typ, _ := bm["type"].(string); typ == "text" {
				txt, _ := bm["text"].(string)
				sum += len(txt)
			}
		}
		return sum
	default:
		return 0
	}
}

// hasImageBlock checks whether array content contains an image block.
func hasImageBlock(content any) bool {
	arr, ok := content.([]any)
	if !ok {
		return false
	}
	for _, block := range arr {
		bm, ok := block.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := bm["type"].(string); typ == "image" {
			return true
		}
	}
	return false
}

// ProcessOptions configures the per-tool result processing behavior.
// Mirrors the closure of getPersistenceThreshold + GrowthBook environment in TS.
type ProcessOptions struct {
	// PersistThresholdOverride is an optional per-tool threshold override (TS GrowthBook tengu_satin_quoll).
	// Returns (threshold, true) to override, (0, false) to use the default computation.
	PersistThresholdOverride func(toolName string) (int64, bool)

	// EmptyResultMarkerEnabled: when true, injects "(toolName completed with no output)" for empty results.
	// Defaults to true (TS: always enabled via logEvent gate).
	EmptyResultMarkerEnabled bool
}

// DefaultProcessOptions returns reasonable defaults for ProcessOptions.
func DefaultProcessOptions() ProcessOptions {
	return ProcessOptions{
		EmptyResultMarkerEnabled: true,
	}
}

// ProcessToolResultBlock mirrors TS processToolResultBlock.
// Maps a tool result to a tool_result block, persisting large outputs to disk.
//
// Parameters:
//   - info: session identity for path resolution
//   - toolName: name of the tool that produced the result
//   - maxResultSizeChars: declared max from ToolSpec (may be math.MaxInt64 for opt-out)
//   - result: the tool result content (string or []any of content blocks)
//   - toolUseID: the tool_use id linking to the assistant's tool_use block
//   - opts: optional threshold overrides
//
// Returns (content, isError) for the tool_result block.
func ProcessToolResultBlock(
	info SessionInfo,
	toolName string,
	maxResultSizeChars int64,
	result any,
	toolUseID string,
	opts ProcessOptions,
) (content any, isError bool) {
	return maybePersistLargeToolResult(info, toolName, result, toolUseID,
		GetPersistenceThreshold(toolName, maxResultSizeChars, opts), opts)
}

// ProcessPreMappedToolResultBlock mirrors TS processPreMappedToolResultBlock.
// Applies persistence for large results when the content block is already mapped.
func ProcessPreMappedToolResultBlock(
	info SessionInfo,
	toolName string,
	maxResultSizeChars int64,
	content any,
	toolUseID string,
	opts ProcessOptions,
) any {
	result, _ := maybePersistLargeToolResult(info, toolName, content, toolUseID,
		GetPersistenceThreshold(toolName, maxResultSizeChars, opts), opts)
	return result
}

// maybePersistLargeToolResult mirrors TS maybePersistLargeToolResult.
func maybePersistLargeToolResult(
	info SessionInfo,
	toolName string,
	result any,
	toolUseID string,
	persistenceThreshold int64,
	opts ProcessOptions,
) (content any, isError bool) {
	// inc-4586: empty result guard — some models break on empty tool_results
	if IsToolResultContentEmpty(result) {
		diaglog.Line("[toolresultpersist] empty result for tool=%s toolUseID=%s", toolName, toolUseID)
		if opts.EmptyResultMarkerEnabled {
			return "(" + toolName + " completed with no output)", false
		}
		return result, false
	}

	// Skip persistence for image content blocks
	if hasImageBlock(result) {
		return result, false
	}

	size := contentSize(result)

	// Use tool-specific threshold or fall back to global limit
	threshold := persistenceThreshold
	if threshold <= 0 {
		threshold = MaxToolResultBytes
	}
	if int64(size) <= threshold {
		return result, false
	}

	// Persist the entire content as a unit
	persisted, persistErr := PersistToolResult(info, result, toolUseID)
	if persistErr != nil {
		diaglog.Line("[toolresultpersist] persist failed for tool=%s toolUseID=%s: %s", toolName, toolUseID, persistErr.Error)
		// If persistence failed, return original content unchanged
		return result, false
	}

	message := BuildLargeToolResultMessage(persisted)
	diaglog.Line("[toolresultpersist] persisted tool=%s toolUseID=%s size=%d threshold=%d", toolName, toolUseID, size, threshold)
	return message, false
}
