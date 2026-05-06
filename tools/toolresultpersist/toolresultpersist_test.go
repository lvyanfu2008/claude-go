package toolresultpersist

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goc/types"
)

// --- GetPersistenceThreshold tests ---

func TestGetPersistenceThreshold_MaxInt64OptOut(t *testing.T) {
	th := GetPersistenceThreshold("Read", math.MaxInt64, DefaultProcessOptions())
	if th != math.MaxInt64 {
		t.Fatalf("expected math.MaxInt64, got %d", th)
	}
}

func TestGetPersistenceThreshold_ZeroDeclared(t *testing.T) {
	th := GetPersistenceThreshold("SomeTool", 0, DefaultProcessOptions())
	if th != 0 {
		t.Fatalf("expected 0, got %d", th)
	}
}

func TestGetPersistenceThreshold_NegativeDeclared(t *testing.T) {
	th := GetPersistenceThreshold("SomeTool", -1, DefaultProcessOptions())
	if th != -1 {
		t.Fatalf("expected -1, got %d", th)
	}
}

func TestGetPersistenceThreshold_UsesDeclaredWhenBelowFallback(t *testing.T) {
	th := GetPersistenceThreshold("Bash", 30000, DefaultProcessOptions())
	if th != 30000 {
		t.Fatalf("expected 30000, got %d", th)
	}
}

func TestGetPersistenceThreshold_UsesFallbackWhenDeclaredAboveFallback(t *testing.T) {
	th := GetPersistenceThreshold("Bash", 100_000, DefaultProcessOptions())
	if th != int64(DefaultMaxResultSizeChars) {
		t.Fatalf("expected fallback %d, got %d", DefaultMaxResultSizeChars, th)
	}
}

func TestGetPersistenceThreshold_Override(t *testing.T) {
	opts := ProcessOptions{
		PersistThresholdOverride: func(toolName string) (int64, bool) {
			if toolName == "Bash" {
				return 75_000, true
			}
			return 0, false
		},
	}
	th := GetPersistenceThreshold("Bash", 100_000, opts)
	if th != 75000 {
		t.Fatalf("expected 75000 from override, got %d", th)
	}
}

func TestGetPersistenceThreshold_OverrideNotMatching(t *testing.T) {
	opts := ProcessOptions{
		PersistThresholdOverride: func(toolName string) (int64, bool) {
			if toolName == "Bash" {
				return 75_000, true
			}
			return 0, false
		},
	}
	th := GetPersistenceThreshold("Grep", 100_000, opts)
	if th != int64(DefaultMaxResultSizeChars) {
		t.Fatalf("expected fallback %d, got %d", DefaultMaxResultSizeChars, th)
	}
}

// --- IsToolResultContentEmpty tests ---

func TestIsToolResultContentEmpty_Nil(t *testing.T) {
	if !IsToolResultContentEmpty(nil) {
		t.Fatal("expected true for nil")
	}
}

func TestIsToolResultContentEmpty_EmptyString(t *testing.T) {
	if !IsToolResultContentEmpty("") {
		t.Fatal("expected true for empty string")
	}
}

func TestIsToolResultContentEmpty_WhitespaceString(t *testing.T) {
	if !IsToolResultContentEmpty("  \n\t  ") {
		t.Fatal("expected true for whitespace-only string")
	}
}

func TestIsToolResultContentEmpty_NonEmptyString(t *testing.T) {
	if IsToolResultContentEmpty("hello") {
		t.Fatal("expected false for non-empty string")
	}
}

func TestIsToolResultContentEmpty_EmptyArray(t *testing.T) {
	if !IsToolResultContentEmpty([]any{}) {
		t.Fatal("expected true for empty array")
	}
}

func TestIsToolResultContentEmpty_ArrayWithEmptyTextBlock(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": ""},
	}
	if !IsToolResultContentEmpty(content) {
		t.Fatal("expected true for array with empty text block")
	}
}

func TestIsToolResultContentEmpty_ArrayWithWhitespaceTextBlock(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": "   "},
	}
	if !IsToolResultContentEmpty(content) {
		t.Fatal("expected true for array with whitespace text block")
	}
}

func TestIsToolResultContentEmpty_ArrayWithNonTextBlock(t *testing.T) {
	content := []any{
		map[string]any{"type": "image", "source": "x"},
	}
	if IsToolResultContentEmpty(content) {
		t.Fatal("expected false for array with image block")
	}
}

func TestIsToolResultContentEmpty_ArrayWithContent(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": "hello"},
	}
	if IsToolResultContentEmpty(content) {
		t.Fatal("expected false for array with text content")
	}
}

// --- contentSize tests ---

func TestContentSize_String(t *testing.T) {
	if s := contentSize("hello"); s != 5 {
		t.Fatalf("expected 5, got %d", s)
	}
}

func TestContentSize_StringEmpty(t *testing.T) {
	if s := contentSize(""); s != 0 {
		t.Fatalf("expected 0, got %d", s)
	}
}

func TestContentSize_ArrayWithTextBlocks(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": "hello"},
		map[string]any{"type": "text", "text": "world"},
	}
	if s := contentSize(content); s != 10 {
		t.Fatalf("expected 10, got %d", s)
	}
}

func TestContentSize_ArrayMixed(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": "hello"},
		map[string]any{"type": "image", "source": "x"},
	}
	if s := contentSize(content); s != 5 {
		t.Fatalf("expected 5, got %d", s)
	}
}

// --- hasImageBlock tests ---

func TestHasImageBlock_String(t *testing.T) {
	if hasImageBlock("hello") {
		t.Fatal("expected false for string")
	}
}

func TestHasImageBlock_ArrayWithoutImage(t *testing.T) {
	content := []any{
		map[string]any{"type": "text", "text": "hello"},
	}
	if hasImageBlock(content) {
		t.Fatal("expected false for array without image")
	}
}

func TestHasImageBlock_ArrayWithImage(t *testing.T) {
	content := []any{
		map[string]any{"type": "image", "source": "data:..."},
	}
	if !hasImageBlock(content) {
		t.Fatal("expected true for array with image")
	}
}

// --- GeneratePreview tests ---

func TestGeneratePreview_ShortContent(t *testing.T) {
	content := "short"
	preview, hasMore := GeneratePreview(content, 1000)
	if preview != "short" {
		t.Fatalf("expected 'short', got %q", preview)
	}
	if hasMore {
		t.Fatal("expected hasMore=false")
	}
}

func TestGeneratePreview_LongContentNoNewline(t *testing.T) {
	content := strings.Repeat("a", 2000)
	preview, hasMore := GeneratePreview(content, 1000)
	if len(preview) != 1000 {
		t.Fatalf("expected 1000 chars, got %d", len(preview))
	}
	if !hasMore {
		t.Fatal("expected hasMore=true")
	}
	if preview != content[:1000] {
		t.Fatal("expected first 1000 chars")
	}
}

func TestGeneratePreview_NewlineNearBoundary(t *testing.T) {
	// Create content with a newline at position 700 (within 50% of 1000)
	content := strings.Repeat("a", 699) + "\n" + strings.Repeat("b", 500)
	preview, hasMore := GeneratePreview(content, 1000)
	if preview != content[:700] {
		t.Fatalf("expected truncation at newline (700), got %d", len(preview))
	}
	if !hasMore {
		t.Fatal("expected hasMore=true")
	}
}

func TestGeneratePreview_NewlineTooEarly(t *testing.T) {
	// Newline at 100, which is below 50% of 1000
	content := strings.Repeat("a", 99) + "\n" + strings.Repeat("b", 2000)
	preview, hasMore := GeneratePreview(content, 1000)
	if len(preview) != 1000 {
		t.Fatalf("expected 1000 chars (newline too early), got %d", len(preview))
	}
	if !hasMore {
		t.Fatal("expected hasMore=true")
	}
}

// --- BuildLargeToolResultMessage tests ---

func TestBuildLargeToolResultMessage_Format(t *testing.T) {
	result := &PersistedToolResult{
		FilePath:     "/tmp/tool-results/abc123.json",
		OriginalSize: 100_000,
		IsJSON:       true,
		Preview:      "preview content here",
		HasMore:      true,
	}
	msg := BuildLargeToolResultMessage(result)
	if !strings.Contains(msg, PersistedOutputTag) {
		t.Fatal("expected persisted output tag")
	}
	if !strings.Contains(msg, PersistedOutputClosingTag) {
		t.Fatal("expected closing tag")
	}
	if !strings.Contains(msg, "97.7KB") {
		t.Fatal("expected formatted original size in message")
	}
	if !strings.Contains(msg, "preview content here") {
		t.Fatal("expected preview content in message")
	}
	if !strings.Contains(msg, "...") {
		t.Fatal("expected continuation indicator")
	}
}

func TestBuildLargeToolResultMessage_NoMore(t *testing.T) {
	result := &PersistedToolResult{
		FilePath:     "/tmp/tool-results/abc123.txt",
		OriginalSize: 100,
		IsJSON:       false,
		Preview:      "short",
		HasMore:      false,
	}
	msg := BuildLargeToolResultMessage(result)
	if strings.Contains(msg, "...\n"+PersistedOutputClosingTag) {
		t.Fatal("expected no continuation indicator when HasMore=false")
	}
}

// --- PersistToolResult tests ---

func TestPersistToolResult_String(t *testing.T) {
	dir := t.TempDir()
	info := SessionInfo{
		SessionID: "test-session",
		Cwd:       dir,
	}
	content := "hello world test content"
	res, perr := PersistToolResult(info, content, "tool-use-1")
	if perr != nil {
		t.Fatalf("unexpected error: %s", perr.Error)
	}
	if res == nil {
		t.Fatal("expected result")
	}
	if res.OriginalSize != len(content) {
		t.Fatalf("expected OriginalSize=%d, got %d", len(content), res.OriginalSize)
	}
	if res.IsJSON {
		t.Fatal("expected IsJSON=false for string content")
	}
	if !strings.Contains(res.FilePath, "test-session") {
		t.Fatalf("expected session ID in path, got %s", res.FilePath)
	}

	// Verify file on disk
	data, err := os.ReadFile(res.FilePath)
	if err != nil {
		t.Fatalf("failed to read persisted file: %v", err)
	}
	if string(data) != content {
		t.Fatalf("expected %q, got %q", content, string(data))
	}
}

func TestPersistToolResult_JSONArray(t *testing.T) {
	dir := t.TempDir()
	info := SessionInfo{
		SessionID: "test-session",
		Cwd:       dir,
	}
	content := []any{
		map[string]any{"type": "text", "text": "hello"},
		map[string]any{"type": "text", "text": "world"},
	}
	res, perr := PersistToolResult(info, content, "tool-use-2")
	if perr != nil {
		t.Fatalf("unexpected error: %s", perr.Error)
	}
	if !res.IsJSON {
		t.Fatal("expected IsJSON=true for array content")
	}
	if !strings.HasSuffix(res.FilePath, ".json") {
		t.Fatalf("expected .json extension, got %s", filepath.Ext(res.FilePath))
	}

	// Verify JSON file on disk
	data, err := os.ReadFile(res.FilePath)
	if err != nil {
		t.Fatalf("failed to read persisted file: %v", err)
	}
	var arr []any
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("failed to unmarshal persisted JSON: %v", err)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr))
	}
}

func TestPersistToolResult_EEXIST(t *testing.T) {
	dir := t.TempDir()
	info := SessionInfo{
		SessionID: "test-session",
		Cwd:       dir,
	}
	content := "first write"

	// Write first time
	_, perr := PersistToolResult(info, content, "tool-use-eexist")
	if perr != nil {
		t.Fatalf("first PersistToolResult failed: %s", perr.Error)
	}

	// Write second time with different content — should not overwrite
	res2, perr2 := PersistToolResult(info, "second write", "tool-use-eexist")
	if perr2 != nil {
		t.Fatalf("second PersistToolResult failed: %s", perr2.Error)
	}
	// Should still have original content on disk
	data, _ := os.ReadFile(res2.FilePath)
	if string(data) != "first write" {
		t.Fatalf("file was overwritten, expected 'first write', got %q", string(data))
	}
}

func TestPersistToolResult_NonTextBlock(t *testing.T) {
	dir := t.TempDir()
	info := SessionInfo{SessionID: "test", Cwd: dir}
	content := []any{
		map[string]any{"type": "image", "source": "data:..."},
	}
	_, perr := PersistToolResult(info, content, "tool-use-img")
	if perr == nil {
		t.Fatal("expected error for non-text content")
	}
	if !strings.Contains(perr.Error, "non-text") {
		t.Fatalf("expected 'non-text' error, got %s", perr.Error)
	}
}

// --- ProcessToolResultBlock tests ---

func TestProcessToolResultBlock_UnderThreshold(t *testing.T) {
	dir := t.TempDir()
	info := SessionInfo{SessionID: "test", Cwd: dir}
	opts := DefaultProcessOptions()
	opts.EmptyResultMarkerEnabled = false

	content, isErr := ProcessToolResultBlock(info, "Bash", 1000, "short result", "tool-use-1", opts)
	if isErr {
		t.Fatal("expected isError=false")
	}
	if content != "short result" {
		t.Fatalf("expected 'short result', got %v", content)
	}
}

func TestProcessToolResultBlock_OverThreshold(t *testing.T) {
	dir := t.TempDir()
	info := SessionInfo{SessionID: "test", Cwd: dir}
	opts := DefaultProcessOptions()
	opts.EmptyResultMarkerEnabled = false

	// Threshold is 10 chars, content is 30 chars
	longContent := strings.Repeat("x", 30)
	content, isErr := ProcessToolResultBlock(info, "Bash", 10, longContent, "tool-use-persist", opts)
	if isErr {
		t.Fatal("expected isError=false")
	}
	s, ok := content.(string)
	if !ok {
		t.Fatalf("expected string content, got %T", content)
	}
	if !strings.Contains(s, PersistedOutputTag) {
		t.Fatal("expected persisted output tag in content")
	}
}

func TestProcessToolResultBlock_EmptyResult(t *testing.T) {
	dir := t.TempDir()
	info := SessionInfo{SessionID: "test", Cwd: dir}
	opts := DefaultProcessOptions()

	content, isErr := ProcessToolResultBlock(info, "Bash", 1000, "", "tool-use-empty", opts)
	if isErr {
		t.Fatal("expected isError=false for empty result")
	}
	s, ok := content.(string)
	if !ok {
		t.Fatalf("expected string, got %T", content)
	}
	if !strings.Contains(s, "Bash completed with no output") {
		t.Fatalf("expected empty marker, got %q", s)
	}
}

func TestProcessToolResultBlock_EmptyResultMarkerDisabled(t *testing.T) {
	dir := t.TempDir()
	info := SessionInfo{SessionID: "test", Cwd: dir}
	opts := DefaultProcessOptions()
	opts.EmptyResultMarkerEnabled = false

	content, isErr := ProcessToolResultBlock(info, "Bash", 1000, "", "tool-use-empty", opts)
	if isErr {
		t.Fatal("expected isError=false")
	}
	if content != "" {
		t.Fatalf("expected empty string, got %v", content)
	}
}

func TestProcessPreMappedToolResultBlock_UnderThreshold(t *testing.T) {
	dir := t.TempDir()
	info := SessionInfo{SessionID: "test", Cwd: dir}
	opts := DefaultProcessOptions()
	opts.EmptyResultMarkerEnabled = false

	result := ProcessPreMappedToolResultBlock(info, "Bash", 1000, "already mapped", "tool-use-1", opts)
	s, ok := result.(string)
	if !ok || s != "already mapped" {
		t.Fatalf("expected 'already mapped', got %v", result)
	}
}

// --- ContentReplacementState tests ---

func TestContentReplacementState_New(t *testing.T) {
	s := NewContentReplacementState()
	if s == nil {
		t.Fatal("expected non-nil state")
	}
	if len(s.SeenIDs) != 0 {
		t.Fatal("expected empty seen IDs")
	}
	if len(s.Replacements) != 0 {
		t.Fatal("expected empty replacements")
	}
}

func TestContentReplacementState_Clone(t *testing.T) {
	s := NewContentReplacementState()
	s.SeenIDs["id1"] = true
	s.Replacements["id1"] = "preview1"

	cloned := s.Clone()
	if !cloned.SeenIDs["id1"] {
		t.Fatal("expected id1 in cloned seen IDs")
	}
	if cloned.Replacements["id1"] != "preview1" {
		t.Fatalf("expected preview1, got %q", cloned.Replacements["id1"])
	}

	// Verify independence
	cloned.SeenIDs["id2"] = true
	if s.SeenIDs["id2"] {
		t.Fatal("mutating clone should not affect original")
	}
}

func TestContentReplacementState_CloneNil(t *testing.T) {
	var s *ContentReplacementState
	if s.Clone() != nil {
		t.Fatal("expected nil from Clone on nil")
	}
}

func TestContentReplacementState_ToJSON(t *testing.T) {
	s := NewContentReplacementState()
	s.SeenIDs["id1"] = true
	s.SeenIDs["id2"] = true
	s.Replacements["id1"] = "preview1"

	raw := s.ToJSON()
	if len(raw) == 0 {
		t.Fatal("expected non-empty JSON")
	}

	var w contentReplacementWire
	if err := json.Unmarshal(raw, &w); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(w.SeenIDs) != 2 {
		t.Fatalf("expected 2 seen IDs, got %d", len(w.SeenIDs))
	}
	if w.Replacements["id1"] != "preview1" {
		t.Fatalf("expected preview1, got %q", w.Replacements["id1"])
	}
}

func TestContentReplacementState_ToJSON_Nil(t *testing.T) {
	var s *ContentReplacementState
	if s.ToJSON() != nil {
		t.Fatal("expected nil")
	}
}

// --- EnforceToolResultBudget tests ---

func makeTestMessages(toolUseBlockID, content string) []types.Message {
	// Assistant message with tool_use block (needed for buildToolNameMap)
	assistantBlocks := []map[string]any{{
		"type": "tool_use",
		"id":   toolUseBlockID,
		"name": "Bash",
	}}
	asstContent, _ := json.Marshal(assistantBlocks)
	asstMsg, _ := json.Marshal(map[string]any{
		"role":    "assistant",
		"content": assistantBlocks,
	})
	asst := types.Message{
		Type:    types.MessageTypeAssistant,
		Message: asstMsg,
		Content: asstContent,
	}

	// User message with tool_result block
	userBlocks := []map[string]any{{
		"type":        "tool_result",
		"content":     content,
		"is_error":    false,
		"tool_use_id": toolUseBlockID,
	}}
	userContent, _ := json.Marshal(userBlocks)
	userMsg, _ := json.Marshal(map[string]any{
		"role":    "user",
		"content": userBlocks,
	})
	user := types.Message{
		Type:    types.MessageTypeUser,
		Message: userMsg,
		Content: userContent,
	}

	return []types.Message{asst, user}
}

func TestEnforceToolResultBudget_NilState(t *testing.T) {
	msgs := makeTestMessages("tu1", "hello")
	result := EnforceToolResultBudget(msgs, nil, SessionInfo{}, 1000, nil, nil)
	if len(result.Messages) != len(msgs) {
		t.Fatal("expected messages unchanged when state is nil")
	}
}

func TestEnforceToolResultBudget_UnderBudget(t *testing.T) {
	msgs := makeTestMessages("tu1", "short")
	state := NewContentReplacementState()
	info := SessionInfo{SessionID: "test", Cwd: t.TempDir()}

	result := EnforceToolResultBudget(msgs, state, info, 100_000, nil, buildToolNameMap(msgs))
	if len(result.NewlyReplaced) != 0 {
		t.Fatal("expected no replacements when under budget")
	}
	if !state.SeenIDs["tu1"] {
		t.Fatal("expected tu1 to be marked seen")
	}
}

func TestEnforceToolResultBudget_OverBudget(t *testing.T) {
	dir := t.TempDir()
	// Long content that exceeds the per-message budget
	longContent := strings.Repeat("x", 5000)
	msgs := makeTestMessages("tu-big", longContent)
	state := NewContentReplacementState()
	info := SessionInfo{SessionID: "test", Cwd: dir}

	// Set a very small budget to force replacement
	result := EnforceToolResultBudget(msgs, state, info, 10, nil, buildToolNameMap(msgs))
	if len(result.NewlyReplaced) != 1 {
		t.Fatalf("expected 1 replacement, got %d", len(result.NewlyReplaced))
	}
	if result.NewlyReplaced[0].ToolUseID != "tu-big" {
		t.Fatalf("expected tu-big, got %s", result.NewlyReplaced[0].ToolUseID)
	}
	// Verify the message content was replaced
	replacedContent := extractFirstToolResultContent(t, result.Messages)
	if !strings.Contains(replacedContent, PersistedOutputTag) {
		t.Fatal("expected persisted output tag in replaced content")
	}
}

func TestEnforceToolResultBudget_ReapplyPriorDecision(t *testing.T) {
	msgs := makeTestMessages("tu-reapply", "should not need to re-persist")
	state := NewContentReplacementState()
	state.SeenIDs["tu-reapply"] = true
	state.Replacements["tu-reapply"] = "<persisted-output>\ncached preview\n</persisted-output>"

	result := EnforceToolResultBudget(msgs, state, SessionInfo{}, 10, nil, buildToolNameMap(msgs))
	if len(result.NewlyReplaced) != 0 {
		t.Fatalf("expected 0 new replacements, got %d (re-apply should not persist again)", len(result.NewlyReplaced))
	}
	replacedContent := extractFirstToolResultContent(t, result.Messages)
	if !strings.Contains(replacedContent, "cached preview") {
		t.Fatal("expected cached preview in replaced content")
	}
}

func TestEnforceToolResultBudget_FrozenNotReplaced(t *testing.T) {
	msgs := makeTestMessages("tu-frozen", "frozen content")
	state := NewContentReplacementState()
	state.SeenIDs["tu-frozen"] = true
	// No replacement → seen but not replaced (frozen)

	result := EnforceToolResultBudget(msgs, state, SessionInfo{}, 10, nil, buildToolNameMap(msgs))
	if len(result.NewlyReplaced) != 0 {
		t.Fatal("expected 0 replacements for frozen (already seen) content")
	}
	// Frozen content stays as-is
	replacedContent := extractFirstToolResultContent(t, result.Messages)
	if replacedContent != "frozen content" {
		t.Fatalf("expected 'frozen content' unchanged, got %q", replacedContent)
	}
}

func TestEnforceToolResultBudget_SkipToolName(t *testing.T) {
	dir := t.TempDir()
	longContent := strings.Repeat("y", 5000)
	msgs := makeTestMessages("tu-read", longContent)
	state := NewContentReplacementState()
	info := SessionInfo{SessionID: "test", Cwd: dir}
	skip := map[string]bool{"Bash": true} // tool is named "Bash" in makeTestMessages

	result := EnforceToolResultBudget(msgs, state, info, 10, skip, buildToolNameMap(msgs))
	if len(result.NewlyReplaced) != 0 {
		t.Fatalf("expected 0 replacements when tool is skipped, got %d", len(result.NewlyReplaced))
	}
}

func TestApplyToolResultBudget_NilState(t *testing.T) {
	msgs := makeTestMessages("tu1", "hello")
	out := ApplyToolResultBudget(msgs, nil, SessionInfo{}, 1000, nil)
	if len(out) != len(msgs) {
		t.Fatal("expected messages unchanged when state is nil")
	}
}

func TestApplyToolResultBudget_EmptyMessages(t *testing.T) {
	out := ApplyToolResultBudget(nil, NewContentReplacementState(), SessionInfo{}, 1000, nil)
	if out != nil {
		t.Fatal("expected nil for empty input")
	}
}

// --- Helpers ---

func extractFirstToolResultContent(t *testing.T, msgs []types.Message) string {
	t.Helper()
	for _, m := range msgs {
		if m.Type != types.MessageTypeUser {
			continue
		}
		var blocks []map[string]any
		if json.Unmarshal(m.Content, &blocks) != nil {
			continue
		}
		for _, b := range blocks {
			if typ, _ := b["type"].(string); typ == "tool_result" {
				if s, ok := b["content"].(string); ok {
					return s
				}
			}
		}
	}
	t.Fatal("no tool_result block found in messages")
	return ""
}
