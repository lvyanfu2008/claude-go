package compactservice

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"goc/types"
)

// PostCompactAttachmentInput mirrors the inputs collected by the TS
// compactConversation right after clearing readFileState and before issuing
// createPostCompactFileAttachments + related calls.
type PostCompactAttachmentInput struct {
	// Model is mainLoopModel — passed through to per-provider routines (e.g. get tools delta).
	Model string
	// AgentID mirrors context.agentId in TS (plan / skill attachment scope).
	AgentID string
	// MessagesBeforeCompaction is the full pre-compact conversation (for discovering
	// previously-read files, invoked skills, deferred tools, agents, MCP tools).
	MessagesBeforeCompaction []any // typed as any because callers use domain-specific shapes; see host wiring

	// MessagesToKeep is the preserved tail (partial compact) — file paths in these
	// messages are skipped from re-injection (they're already visible to the model).
	MessagesToKeep []types.Message

	// ToolUseContext is optional runtime context for hosts that have richer state
	// (e.g. tool definitions for deferred_tools_delta, agent definitions, MCP clients).
	ToolUseContext *types.ToolUseContext
}

// PostCompactAttachmentProvider is the injection point that collectively produces the
// attachment messages TS appends after the summary. Returning an empty slice is the
// documented no-op default; hosts wire concrete providers (file re-read, plan, plan_mode,
// skills, agent listing, MCP, deferred tools) as those subsystems land Go parity.
type PostCompactAttachmentProvider func(ctx context.Context, in PostCompactAttachmentInput) ([]HookResultMessage, error)

// NoopPostCompactAttachmentProvider is the safe default.
func NoopPostCompactAttachmentProvider(_ context.Context, _ PostCompactAttachmentInput) ([]HookResultMessage, error) {
	return nil, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Extraction helpers: derive post-compact attachment data from messages alone.
// These are fallback utilities for hosts that don't have full runtime state;
// the canonical path is TS's readFileState / invokedSkills / etc. caches.
// ─────────────────────────────────────────────────────────────────────────────

// ExtractedReadFile represents a file path read during the conversation,
// discovered from Read tool_use / tool_result pairs in messages.
type ExtractedReadFile struct {
	Path      string
	Timestamp int64 // unix millis; 0 if unknown
}

// ExtractReadFilePaths scans messages for Read tool invocations and returns
// the file paths in most-recent-first order (deduped, capped at maxFiles).
// Paths present in preservedMessages are skipped (already visible to model).
// This mirrors TS createPostCompactFileAttachments's preCompactReadFileState
// derivation but operates on raw messages without a session cache.
func ExtractReadFilePaths(messages []types.Message, preservedMessages []types.Message, maxFiles int) []ExtractedReadFile {
	preservedPaths := collectReadToolFilePathsFromMessages(preservedMessages)

	type entry struct {
		path string
		ts   int64
		idx  int // message index for stable sort
	}
	seen := make(map[string]entry)

	for i, m := range messages {
		if m.Type != types.MessageTypeAssistant {
			continue
		}
		paths, ts := extractReadToolCallPaths(m)
		for _, p := range paths {
			norm := strings.TrimSpace(p)
			if norm == "" {
				continue
			}
			// Skip paths visible in preserved tail
			if _, ok := preservedPaths[norm]; ok {
				continue
			}
			// Later occurrence overwrites earlier
			seen[norm] = entry{path: norm, ts: ts, idx: i}
		}
	}

	// Sort by timestamp desc, then by index desc (most recent first)
	list := make([]entry, 0, len(seen))
	for _, e := range seen {
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].ts != list[j].ts {
			return list[i].ts > list[j].ts
		}
		return list[i].idx > list[j].idx
	})

	if maxFiles > 0 && len(list) > maxFiles {
		list = list[:maxFiles]
	}

	out := make([]ExtractedReadFile, len(list))
	for i, e := range list {
		out[i] = ExtractedReadFile{Path: e.path, Timestamp: e.ts}
	}
	return out
}

// extractReadToolCallPaths extracts file_path arguments from Read tool_use blocks
// in an assistant message. Returns paths and a timestamp (message timestamp or now).
func extractReadToolCallPaths(m types.Message) ([]string, int64) {
	if len(m.Message) == 0 {
		return nil, 0
	}
	var inner struct {
		Content []json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(m.Message, &inner); err != nil {
		return nil, 0
	}

	// Timestamp is *string in types.Message; parse if available, else use current time.
	var ts int64
	if m.Timestamp != nil && *m.Timestamp != "" {
		// Try parsing as ISO or unix millis string
		if t, err := time.Parse(time.RFC3339, *m.Timestamp); err == nil {
			ts = t.UnixMilli()
		}
	}
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}

	var paths []string
	for _, block := range inner.Content {
		var tb struct {
			Type  string          `json:"type"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(block, &tb); err != nil {
			continue
		}
		if tb.Type != "tool_use" || tb.Name != "Read" {
			continue
		}
		var inp struct {
			FilePath string `json:"file_path"`
		}
		if err := json.Unmarshal(tb.Input, &inp); err != nil {
			continue
		}
		if inp.FilePath != "" {
			paths = append(paths, inp.FilePath)
		}
	}
	return paths, ts
}

// collectReadToolFilePathsFromMessages mirrors TS collectReadToolFilePaths:
// returns a set of file paths that appear in Read tool_result blocks.
func collectReadToolFilePathsFromMessages(messages []types.Message) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range messages {
		// Read tool_use is in assistant, but the file path is in the tool_use input.
		// We also check user messages for tool_result that might reference paths.
		paths, _ := extractReadToolCallPaths(m)
		for _, p := range paths {
			if p != "" {
				out[strings.TrimSpace(p)] = struct{}{}
			}
		}
	}
	return out
}

// ExtractedSkill represents a skill invoked during the conversation.
type ExtractedSkill struct {
	Name    string
	Path    string
	Content string // may be empty if not extractable
}

// ExtractInvokedSkills scans messages for invoked_skills attachments and
// returns the union of skills found. This is a fallback when the host
// doesn't have a live skill store.
func ExtractInvokedSkills(messages []types.Message) []ExtractedSkill {
	seen := make(map[string]ExtractedSkill)
	for _, m := range messages {
		if m.Type != types.MessageTypeAttachment {
			continue
		}
		if len(m.Attachment) == 0 {
			continue
		}
		var att struct {
			Type   string `json:"type"`
			Skills []struct {
				Name    string `json:"name"`
				Path    string `json:"path"`
				Content string `json:"content"`
			} `json:"skills"`
		}
		if err := json.Unmarshal(m.Attachment, &att); err != nil {
			continue
		}
		if att.Type != "invoked_skills" {
			continue
		}
		for _, s := range att.Skills {
			if s.Name == "" {
				continue
			}
			// Later occurrence overwrites earlier (keeps most recent content)
			seen[s.Name] = ExtractedSkill{Name: s.Name, Path: s.Path, Content: s.Content}
		}
	}

	// Return sorted for determinism
	out := make([]ExtractedSkill, 0, len(seen))
	for _, s := range seen {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
