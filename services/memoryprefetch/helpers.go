package memoryprefetch

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"goc/memdir"
	"goc/types"
)

const (
	// MAX_MEMORY_LINES mirrors RELEVANT_MEMORIES_CONFIG.MAX_MEMORY_LINES in TS attachments.ts.
	MAX_MEMORY_LINES = 200

	// MAX_MEMORY_BYTES mirrors RELEVANT_MEMORIES_CONFIG.MAX_MEMORY_BYTES in TS attachments.ts.
	MAX_MEMORY_BYTES = 4096

	// MAX_SESSION_BYTES mirrors RELEVANT_MEMORIES_CONFIG.MAX_SESSION_BYTES in TS attachments.ts.
	MAX_SESSION_BYTES = 60 * 1024
)

// SurfacedMemory mirrors the per-file result from readMemoriesForSurfacing in TS attachments.ts.
type SurfacedMemory struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	MtimeMs int64  `json:"mtimeMs"`
	Header  string `json:"header"`
}

// ---- memory age helpers (mirror src/memdir/memoryAge.ts) ----

func memoryAgeDays(mtimeMs int64) int {
	d := int((time.Now().UnixMilli() - mtimeMs) / 86_400_000)
	if d < 0 {
		return 0
	}
	return d
}

func memoryAge(mtimeMs int64) string {
	d := memoryAgeDays(mtimeMs)
	if d == 0 {
		return "today"
	}
	if d == 1 {
		return "yesterday"
	}
	return fmt.Sprintf("%d days ago", d)
}

func memoryFreshnessText(mtimeMs int64) string {
	d := memoryAgeDays(mtimeMs)
	if d <= 1 {
		return ""
	}
	return fmt.Sprintf(
		"This memory is %d days old. "+
			"Memories are point-in-time observations, not live state — "+
			"claims about code behavior or file:line citations may be outdated. "+
			"Verify against current code before asserting as fact.",
		d,
	)
}

func memoryHeader(path string, mtimeMs int64) string {
	staleness := memoryFreshnessText(mtimeMs)
	if staleness != "" {
		return staleness + "\n\nMemory: " + path + ":"
	}
	return "Memory (saved " + memoryAge(mtimeMs) + "): " + path + ":"
}

// ---- reading memory files ----

// readMemoryFile reads up to maxLines and maxBytes from path.
// Returns content and whether it was truncated.
func readMemoryFile(path string, maxLines, maxBytes int) (content string, truncated bool, _ error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}

	// Apply byte limit first
	raw := data
	if len(raw) > maxBytes {
		raw = raw[:maxBytes]
		truncated = true
	}

	// Apply line limit
	lines := strings.Split(string(raw), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}

	content = strings.Join(lines, "\n")
	if truncated {
		content += fmt.Sprintf(
			"\n\n> This memory file was truncated (first %d lines). Use the FileRead tool to view the complete file at: %s",
			maxLines, path,
		)
	}
	return content, truncated, nil
}

// readMemoriesForSurfacing reads selected memory files with line/byte limits.
// Mirrors readMemoriesForSurfacing in TS attachments.ts.
func readMemoriesForSurfacing(selected []RelevantMemory) []SurfacedMemory {
	var results []SurfacedMemory
	for _, sel := range selected {
		content, _, err := readMemoryFile(sel.Path, MAX_MEMORY_LINES, MAX_MEMORY_BYTES)
		if err != nil {
			continue
		}
		results = append(results, SurfacedMemory{
			Path:    sel.Path,
			Content: content,
			MtimeMs: sel.MtimeMs,
			Header:  memoryHeader(sel.Path, sel.MtimeMs),
		})
	}
	return results
}

// ---- message scanning helpers ----

// collectSurfacedMemories scans messages for previously injected relevant_memories
// attachments. Returns set of surfaced paths and cumulative byte count.
// Mirrors collectSurfacedMemories in TS attachments.ts.
func collectSurfacedMemories(messages []types.Message) (paths map[string]bool, totalBytes int) {
	paths = make(map[string]bool)
	for _, m := range messages {
		if m.Type != types.MessageTypeAttachment || len(m.Attachment) == 0 {
			continue
		}
		var att struct {
			Type     string           `json:"type"`
			Memories []SurfacedMemory `json:"memories"`
		}
		if json.Unmarshal(m.Attachment, &att) != nil || att.Type != "relevant_memories" {
			continue
		}
		for _, mem := range att.Memories {
			paths[mem.Path] = true
			totalBytes += len(mem.Content)
		}
	}
	return
}

// getLastUserMessage finds the most recent non-isMeta user message.
// Mirrors messages.findLast(m => m.type === 'user' && !m.isMeta) in TS.
func getLastUserMessage(messages []types.Message) (types.Message, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if m.Type == types.MessageTypeUser && (m.IsMeta == nil || !*m.IsMeta) {
			return m, true
		}
	}
	return types.Message{}, false
}

// getUserMessageText extracts the text content from a user message.
// Mirrors getUserMessageText in TS attachments.ts.
func getUserMessageText(m types.Message) string {
	raw := m.Message
	if len(raw) == 0 {
		raw = m.Content
	}
	if len(raw) == 0 {
		return ""
	}
	// Try string content first
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return strings.TrimSpace(text)
	}
	// Try array of content blocks
	//{
	//  "content" : "你好",
	//  "role" : "user"
	//}
	var block struct {
		Role string `json:"role"`
		Text string `json:"content"`
	}
	if json.Unmarshal(raw, &block) == nil {
		return strings.TrimSpace(block.Text)
	}
	return ""
}

// RelevantMemory is a simple path+mtime pair used before reading full content.
// Mirrors the return type of findRelevantMemories in TS.
type RelevantMemory struct {
	Path    string
	MtimeMs int64
}

// ---- agent mentions (mirrors TS extractAgentMentions) ----

// AgentMemoryDef mirrors the subset of AgentDefinition needed by
// GetRelevantMemoryAttachments: agentType and memory scope only.
type AgentMemoryDef struct {
	AgentType string                `json:"agentType"`
	Memory    memdir.AgentMemoryScope `json:"memory,omitempty"`
}

// ReadFileStateChecker mirrors TS FileStateCache.has(): checks whether
// a file path has been read via FileRead/FileWrite/FileEdit tools (any
// iteration this turn), so the memory selector can de-dup.
type ReadFileStateChecker interface {
	Has(absPath string) bool
}

var (
	// quotedAgentRe matches @"<type> (agent)" — autocomplete selection format.
	quotedAgentRe = regexp.MustCompile(`(^|\s)@"([\w:.@-]+) \(agent\)"`)
	// unquotedAgentRe matches @agent-<type> — legacy/manual typing format.
	unquotedAgentRe = regexp.MustCompile(`(^|\s)@(agent-[\w:.@-]+)`)
)

// extractAgentMentions extracts agent type mentions from user input content.
// Supports two formats:
//  1. @"<type> (agent)" — autocomplete selection (e.g. @"code-reviewer (agent)")
//  2. @agent-<type> — manual typing (e.g. @agent-code-reviewer)
//
// Mirrors TS extractAgentMentions in attachments.ts:2811-2837.
func extractAgentMentions(content string) []string {
	seen := make(map[string]bool)
	var results []string

	// Match quoted format: @"<type> (agent)"
	for _, match := range quotedAgentRe.FindAllStringSubmatch(content, -1) {
		if len(match) >= 3 && match[2] != "" && !seen[match[2]] {
			seen[match[2]] = true
			results = append(results, match[2])
		}
	}

	// Match unquoted format: @agent-<type>
	for _, match := range unquotedAgentRe.FindAllStringSubmatch(content, -1) {
		if len(match) >= 3 && match[2] != "" && !seen[match[2]] {
			seen[match[2]] = true
			results = append(results, match[2])
		}
	}

	return results
}

// ---- recent successful tools (mirrors TS collectRecentSuccessfulTools) ----

// isHumanTurn mirrors TS isHumanTurn(m): m.type === 'user' && !m.isMeta && m.toolUseResult === undefined.
func isHumanTurn(m types.Message) bool {
	return m.Type == types.MessageTypeUser &&
		(m.IsMeta == nil || !*m.IsMeta) &&
		m.ToolUseResult == nil
}

// isToolResultBlock mirrors TS isToolResultBlock: type === 'tool_result' && typeof tool_use_id === 'string'.
func isToolResultBlock(block map[string]any) bool {
	if block == nil {
		return false
	}
	t, _ := block["type"].(string)
	if t != "tool_result" {
		return false
	}
	_, ok := block["tool_use_id"].(string)
	return ok
}

// collectRecentSuccessfulTools returns tool names that succeeded (and never
// errored) since the previous real turn boundary. The memory selector uses
// this to suppress docs about tools that are already working.
//
// Backward scan: tool_use lives in assistant content; tool_result in user
// content. Results are seen before uses in backward order, so we collect
// both by ID and resolve after the scan.
//
// Mirrors TS collectRecentSuccessfulTools in attachments.ts:2474-2512.
func collectRecentSuccessfulTools(messages []types.Message, lastUserMessage types.Message) []string {
	useIDToName := make(map[string]string)
	resultByUseID := make(map[string]bool)

	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if isHumanTurn(m) && m.UUID != lastUserMessage.UUID {
			break
		}
		if m.Type == types.MessageTypeAssistant && len(m.Message) > 0 {
			// Try to parse m.Message as array of content blocks
			var blocks []map[string]any
			if json.Unmarshal(m.Message, &blocks) == nil {
				for _, block := range blocks {
					if t, _ := block["type"].(string); t == "tool_use" {
						if id, _ := block["id"].(string); id != "" {
							if name, _ := block["name"].(string); name != "" {
								useIDToName[id] = name
							}
						}
					}
				}
			}
		} else if m.Type == types.MessageTypeUser && len(m.Message) > 0 {
			var blocks []map[string]any
			if json.Unmarshal(m.Message, &blocks) == nil {
				for _, block := range blocks {
					if isToolResultBlock(block) {
						id, _ := block["tool_use_id"].(string)
						isError, _ := block["is_error"].(bool)
						resultByUseID[id] = isError
					}
				}
			}
		}
	}

	failed := make(map[string]bool)
	succeeded := make(map[string]bool)
	for id, name := range useIDToName {
		errored, ok := resultByUseID[id]
		if !ok {
			continue
		}
		if errored {
			failed[name] = true
		} else {
			succeeded[name] = true
		}
	}

	var results []string
	for name := range succeeded {
		if !failed[name] {
			results = append(results, name)
		}
	}
	return results
}

// newRelevantMemoriesAttachment creates an attachment message for the memories.
// Mirrors createAttachmentMessage({ type: 'relevant_memories', memories }) in TS.
func newRelevantMemoriesAttachment(memories []SurfacedMemory) types.Message {
	att := struct {
		Type     string           `json:"type"`
		Memories []SurfacedMemory `json:"memories"`
	}{
		Type:     "relevant_memories",
		Memories: memories,
	}
	data, _ := json.Marshal(att)
	isMeta := true
	return types.Message{
		Type:       types.MessageTypeAttachment,
		Attachment: json.RawMessage(data),
		IsMeta:     &isMeta,
	}
}
