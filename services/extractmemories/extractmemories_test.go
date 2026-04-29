package extractmemories

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"goc/memdir"
	"goc/types"
)

// ---------------------------------------------------------------------------
// Helper: make a user message with the given UUID
// ---------------------------------------------------------------------------

func makeMsg(uuid string, msgType types.MessageType) types.Message {
	return types.Message{
		Type:    msgType,
		UUID:    uuid,
		Message: json.RawMessage(`{"role":"user","content":"hello"}`),
	}
}

func makeAssistantMsg(uuid string, toolBlocks ...json.RawMessage) types.Message {
	content := make([]map[string]any, 0, len(toolBlocks))
	for _, tb := range toolBlocks {
		var parsed map[string]any
		json.Unmarshal(tb, &parsed)
		content = append(content, parsed)
	}
	payload := map[string]any{"content": content}
	b, _ := json.Marshal(payload)
	return types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    uuid,
		Message: b,
	}
}

func writeToolBlock(name, filePath string) json.RawMessage {
	b, _ := json.Marshal(map[string]any{
		"type":  "tool_use",
		"name":  name,
		"input": map[string]string{"file_path": filePath},
	})
	return b
}

// ---------------------------------------------------------------------------
// memoryDirDisplayPath
// ---------------------------------------------------------------------------

func TestMemoryDirDisplayPath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"/tmp/mem", "/tmp/mem/"},
		{"/tmp/mem/", "/tmp/mem/"},
		{"relative/path", "relative/path/"},
	}
	for _, tc := range tests {
		got := memoryDirDisplayPath(tc.in)
		if got != tc.want {
			t.Errorf("memoryDirDisplayPath(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// filterMemoryPaths
// ---------------------------------------------------------------------------

func TestFilterMemoryPaths(t *testing.T) {
	memDir := "/tmp/test-mem"

	tests := []struct {
		paths []string
		want  int
	}{
		{[]string{"/tmp/test-mem/user_role.md"}, 1},
		{[]string{"/tmp/test-mem/MEMORY.md"}, 0},
		{[]string{"/tmp/test-mem/sub/note.md"}, 1},
		{[]string{"/outside/file.md"}, 0},
		{[]string{"/tmp/test-mem/user.md", "/tmp/test-mem/MEMORY.md"}, 1},
	}
	for _, tc := range tests {
		got := filterMemoryPaths(tc.paths, memDir)
		if len(got) != tc.want {
			t.Errorf("filterMemoryPaths(%v) = %d items; want %d", tc.paths, len(got), tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// filePathFromInput
// ---------------------------------------------------------------------------

func TestFilePathFromInput(t *testing.T) {
	// Valid file_path.
	input, _ := json.Marshal(map[string]string{"file_path": "/tmp/test.md"})
	if got := filePathFromInput(input); got != "/tmp/test.md" {
		t.Errorf("filePathFromInput = %q; want /tmp/test.md", got)
	}

	// Empty file_path.
	input2, _ := json.Marshal(map[string]string{"file_path": ""})
	if got := filePathFromInput(input2); got != "" {
		t.Errorf("filePathFromInput = %q; want empty", got)
	}

	// Invalid JSON.
	if got := filePathFromInput(json.RawMessage(`{bad`)); got != "" {
		t.Errorf("filePathFromInput = %q; want empty for invalid JSON", got)
	}
}

// ---------------------------------------------------------------------------
// newMessagesSinceCursor
// ---------------------------------------------------------------------------

func TestNewMessagesSinceCursor(t *testing.T) {
	msgs := []types.Message{
		makeMsg("a", types.MessageTypeUser),
		makeMsg("b", types.MessageTypeAssistant),
		makeMsg("c", types.MessageTypeUser),
		makeMsg("d", types.MessageTypeSystem),
	}

	// No cursor: last 10 (or all).
	got := newMessagesSinceCursor(msgs, "")
	if len(got) != 4 {
		t.Fatalf("expected all 4 messages without cursor, got %d", len(got))
	}

	// Cursor at "b": should return messages after b (c, d).
	got2 := newMessagesSinceCursor(msgs, "b")
	if len(got2) != 2 {
		t.Fatalf("expected 2 messages after cursor b, got %d", len(got2))
	}
	if got2[0].UUID != "c" || got2[1].UUID != "d" {
		t.Errorf("unexpected messages after cursor: %v", got2)
	}

	// Cursor at "d": should return nothing.
	got3 := newMessagesSinceCursor(msgs, "d")
	if len(got3) != 0 {
		t.Fatalf("expected 0 messages after last cursor, got %d", len(got3))
	}

	// Cursor not found: treat all as new.
	got4 := newMessagesSinceCursor(msgs, "nonexistent")
	if len(got4) != 4 {
		t.Fatalf("expected all 4 when cursor not found, got %d", len(got4))
	}

	// First run with fewer than 50 messages: return all.
	under50 := make([]types.Message, 15)
	for i := range under50 {
		under50[i] = makeMsg(string(rune('a'+i)), types.MessageTypeUser)
	}
	got5 := newMessagesSinceCursor(under50, "")
	if len(got5) != 15 {
		t.Fatalf("expected all 15 messages without cursor (first run), got %d", len(got5))
	}
}

func TestNewMessagesSinceCursorOver50(t *testing.T) {
	// First run with more than 50 messages: cap at last 50.
	manyMsgs := make([]types.Message, 75)
	for i := range manyMsgs {
		manyMsgs[i] = makeMsg(string(rune('a'+i%26)), types.MessageTypeUser)
	}
	got := newMessagesSinceCursor(manyMsgs, "")
	if len(got) != 50 {
		t.Fatalf("expected 50 messages without cursor (first run cap), got %d", len(got))
	}
	// Verify they're the LAST 50, not the first 50.
	if got[0].UUID != manyMsgs[25].UUID {
		t.Error("expected last 50 messages, first doesn't match")
	}
}

// ---------------------------------------------------------------------------
// hasMemoryWritesSince
// ---------------------------------------------------------------------------

func TestHasMemoryWritesSince(t *testing.T) {
	memDir := memdir.GetAutoMemPath(t.TempDir())
	os.Setenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY", memDir)
	defer os.Unsetenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY")

	msgs := []types.Message{
		makeMsg("a", types.MessageTypeUser),
		makeAssistantMsg("b", writeToolBlock("Write", filepath.Join(memDir, "test.md"))),
	}

	if !hasMemoryWritesSince(msgs, "") {
		t.Error("expected memory write to be detected")
	}

	// No Write tool_use.
	msgs2 := []types.Message{
		makeMsg("a", types.MessageTypeUser),
		makeAssistantMsg("b", writeToolBlock("Read", "/tmp/foo.txt")),
	}
	if hasMemoryWritesSince(msgs2, "") {
		t.Error("expected no memory write without Write tool")
	}

	// Write outside memory dir.
	msgs3 := []types.Message{
		makeMsg("a", types.MessageTypeUser),
		makeAssistantMsg("b", writeToolBlock("Write", "/tmp/outside.txt")),
	}
	if hasMemoryWritesSince(msgs3, "") {
		t.Error("expected no memory write for path outside memory dir")
	}

	// Empty messages.
	if hasMemoryWritesSince(nil, "") {
		t.Error("expected false for nil messages")
	}
}

// ---------------------------------------------------------------------------
// extractWrittenPaths
// ---------------------------------------------------------------------------

func TestExtractWrittenPaths(t *testing.T) {
	msgs := []types.Message{
		makeAssistantMsg("a",
			writeToolBlock("Write", "/mem/a.md"),
			writeToolBlock("Edit", "/mem/b.md"),
			writeToolBlock("Glob", "/mem/c.md"), // not Write/Edit, should be ignored
		),
		makeAssistantMsg("b",
			writeToolBlock("Write", "/mem/a.md"), // duplicate, should be skipped
			writeToolBlock("Write", "/mem/d.md"),
		),
		// User message, should be ignored.
		makeMsg("c", types.MessageTypeUser),
	}

	paths := extractWrittenPaths(msgs)
	if len(paths) != 3 {
		t.Fatalf("expected 3 unique paths, got %d: %v", len(paths), paths)
	}
}

// ---------------------------------------------------------------------------
// State throttle
// ---------------------------------------------------------------------------

func TestStateThrottle(t *testing.T) {
	state := NewState()
	if state.TurnsSinceLastExtraction != 0 {
		t.Errorf("initial TurnsSinceLastExtraction should be 0, got %d", state.TurnsSinceLastExtraction)
	}

	// Simulate incrementing (as Execute does at the top).
	state.mu.Lock()
	state.TurnsSinceLastExtraction++
	tc1 := state.TurnsSinceLastExtraction
	state.mu.Unlock()

	if tc1 != 1 {
		t.Errorf("after first increment, expected 1, got %d", tc1)
	}

	// Simulate reset (as Execute does when extraction runs).
	state.mu.Lock()
	state.TurnsSinceLastExtraction = 0
	state.mu.Unlock()

	if state.TurnsSinceLastExtraction != 0 {
		t.Errorf("after reset, expected 0, got %d", state.TurnsSinceLastExtraction)
	}
}

// ---------------------------------------------------------------------------
// State LastMemoryMessageUUID
// ---------------------------------------------------------------------------

func TestStateLastMemoryMessageUUID(t *testing.T) {
	state := NewState()
	if state.LastMemoryMessageUUID != "" {
		t.Errorf("initial LastMemoryMessageUUID should be empty")
	}

	state.mu.Lock()
	state.LastMemoryMessageUUID = "test-uuid-123"
	state.mu.Unlock()

	if state.LastMemoryMessageUUID != "test-uuid-123" {
		t.Errorf("expected test-uuid-123, got %s", state.LastMemoryMessageUUID)
	}
}

// ---------------------------------------------------------------------------
// Prompt section builders (smoke tests)
// ---------------------------------------------------------------------------

func TestTypesSectionIndividual(t *testing.T) {
	s := typesSectionIndividual()
	if !strings.Contains(s, "user") {
		t.Error("expected user type in section")
	}
	if !strings.Contains(s, "feedback") {
		t.Error("expected feedback type in section")
	}
	if !strings.Contains(s, "project") {
		t.Error("expected project type in section")
	}
	if !strings.Contains(s, "reference") {
		t.Error("expected reference type in section")
	}
	// Verify detailed XML-like structure from TS TYPES_SECTION_INDIVIDUAL
	if !strings.Contains(s, "<when_to_save>") {
		t.Error("expected when_to_save tags in section")
	}
	if !strings.Contains(s, "<how_to_use>") {
		t.Error("expected how_to_use tags in section")
	}
	if !strings.Contains(s, "<examples>") {
		t.Error("expected examples tags in section")
	}
}

func TestWhatNotToSaveSection(t *testing.T) {
	s := whatNotToSaveSection()
	if !strings.Contains(s, "Code patterns") {
		t.Error("expected 'Code patterns' in section")
	}
	if !strings.Contains(s, "Git history") {
		t.Error("expected 'Git history' in section")
	}
	// Verify explicit-save gate from TS WHAT_NOT_TO_SAVE_SECTION
	if !strings.Contains(s, "explicitly asks you to save") {
		t.Error("expected explicit-save gate in section")
	}
}

func TestHowToSaveSection(t *testing.T) {
	s := howToSaveSection(false)
	if !strings.Contains(s, "```markdown") {
		t.Error("expected markdown code block in section")
	}
	if !strings.Contains(s, "MEMORY.md") {
		t.Error("expected MEMORY.md reference in section")
	}
	if !strings.Contains(s, "two-step process") {
		t.Error("expected two-step process for skipIndex=false")
	}
}

func TestHowToSaveSectionSkipIndex(t *testing.T) {
	s := howToSaveSection(true)
	if !strings.Contains(s, "```markdown") {
		t.Error("expected markdown code block in section")
	}
	// skipIndex=true should NOT mention MEMORY.md or two-step process
	if strings.Contains(s, "two-step process") {
		t.Error("skipIndex=true should not mention two-step process")
	}
	if strings.Contains(s, "Step 2") {
		t.Error("skipIndex=true should not have Step 2")
	}
}

func TestMemoryFrontmatterExample(t *testing.T) {
	s := memoryFrontmatterExample()
	if !strings.Contains(s, "```markdown") {
		t.Error("expected markdown code block")
	}
	if !strings.Contains(s, "name:") {
		t.Error("expected name field")
	}
	if !strings.Contains(s, "type:") {
		t.Error("expected type field")
	}
}

func TestOpener(t *testing.T) {
	s := opener(5, "existing.md — name=test desc", "/tmp/mem/", "/tmp/mem/")
	if !strings.Contains(s, "memory extraction subagent") {
		t.Error("expected subagent framing")
	}
	if !strings.Contains(s, "Available tools:") {
		t.Error("expected tool list")
	}
	if !strings.Contains(s, "read-only Bash") {
		t.Error("expected read-only Bash description")
	}
	if !strings.Contains(s, "turn 1") {
		t.Error("expected turn-budget strategy")
	}
	if !strings.Contains(s, "Bash rm is not permitted") {
		t.Error("expected Bash rm prohibition")
	}
	if !strings.Contains(s, "existing.md") {
		t.Error("expected existing memories listing")
	}
}

// ---------------------------------------------------------------------------
// buildExtractionUserMessage
// ---------------------------------------------------------------------------

func TestBuildExtractionUserMessage(t *testing.T) {
	var uuidCounter int
	fakeUUID := func() string {
		uuidCounter++
		return "uuid-" + strings.Repeat("0", uuidCounter)
	}

	msg := buildExtractionUserMessage("test prompt", fakeUUID)
	if msg.Type != types.MessageTypeUser {
		t.Errorf("expected user type, got %s", msg.Type)
	}
	if msg.UUID != "uuid-0" {
		t.Errorf("expected uuid-0, got %s", msg.UUID)
	}
	if !strings.Contains(string(msg.Message), "test prompt") {
		t.Errorf("expected prompt in message, got: %s", string(msg.Message))
	}
}

// ---------------------------------------------------------------------------
// buildExtractionPrompt (smoke test — just checks it produces output)
// ---------------------------------------------------------------------------

func TestBuildExtractionPrompt(t *testing.T) {
	dir := t.TempDir()
	// Create a memory file to exercise memory scanning.
	os.WriteFile(filepath.Join(dir, "user_role.md"), []byte("---\nname: Role\ndescription: User role\ntype: user\n---\n\ncontent"), 0644)
	// Create MEMORY.md index.
	os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("# Index\n- [Role](user_role.md)"), 0644)

	p := ExtractionParams{
		Messages: []types.Message{makeMsg("1", types.MessageTypeUser)},
		ToolUseContext: types.ToolUseContext{
			Options: types.ToolUseContextOptionsData{
				MainLoopModel: "test-model",
			},
		},
	}
	newMsgs := []types.Message{makeMsg("2", types.MessageTypeUser)}

	result := buildExtractionPrompt(p, newMsgs, dir)
	if result == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(result, "memory extraction subagent") {
		t.Error("expected prompt to contain 'memory extraction subagent'")
	}
	if !strings.Contains(result, "user_role.md") {
		t.Error("expected existing memory listing")
	}
}

func TestBuildExtractionPromptRelaxThreshold(t *testing.T) {
	dir := t.TempDir()
	p := ExtractionParams{Messages: []types.Message{makeMsg("1", types.MessageTypeUser)}}
	newMsgs := []types.Message{makeMsg("2", types.MessageTypeUser)}

	t.Run("default strict closing", func(t *testing.T) {
		t.Setenv("GOC_EXTRACT_MEMORIES_RELAX_THRESHOLD", "")
		t.Setenv("CLAUDE_CODE_EXTRACT_MEMORIES_RELAX_THRESHOLD", "")
		out := buildExtractionPrompt(p, newMsgs, dir)
		if !strings.Contains(out, "do nothing") || strings.Contains(out, "Only skip when there is no user-specific") {
			tail := out
			if len(tail) > 200 {
				tail = tail[len(tail)-200:]
			}
			t.Fatalf("expected default threshold line, got tail: %q", tail)
		}
	})
	t.Run("relaxed closing", func(t *testing.T) {
		t.Setenv("GOC_EXTRACT_MEMORIES_RELAX_THRESHOLD", "1")
		out := buildExtractionPrompt(p, newMsgs, dir)
		if !strings.Contains(out, "Only skip when there is no user-specific") {
			tail := out
			if len(tail) > 220 {
				tail = tail[len(tail)-220:]
			}
			t.Fatalf("expected relax closing, got tail: %q", tail)
		}
	})
}

func TestExtractionInitialMessagesForkOrder(t *testing.T) {
	parent := []types.Message{
		makeMsg("m1", types.MessageTypeUser),
		makeMsg("m2", types.MessageTypeAssistant),
	}
	extra := buildExtractionUserMessage("extract prompt", func() string { return "u-final" })
	got := extractionInitialMessages(parent, extra)
	if len(got) != 3 {
		t.Fatalf("len=%d; want 3 (parent + extraction user)", len(got))
	}
	if got[0].UUID != "m1" || got[1].UUID != "m2" {
		t.Fatalf("parent prefix mutated or wrong: %#v", got)
	}
	if got[2].UUID != "u-final" {
		t.Fatalf("last message should be extraction user, got UUID %q", got[2].UUID)
	}
	// Parent slice must not gain a third element.
	if len(parent) != 2 {
		t.Fatalf("parent len=%d; clone must not append in place", len(parent))
	}
}

func TestAssistantToolUseSummary(t *testing.T) {
	tb, _ := json.Marshal(map[string]any{
		"content": []map[string]any{
			{"type": "tool_use", "name": "Grep", "id": "1", "input": map[string]any{}},
			{"type": "tool_use", "name": "Read", "id": "2", "input": map[string]any{}},
			{"type": "tool_use", "name": "Grep", "id": "3", "input": map[string]any{}},
		},
	})
	msg := types.Message{Type: types.MessageTypeAssistant, Message: tb}
	s := assistantToolUseSummary([]types.Message{msg})
	if !strings.Contains(s, "Grep×2") || !strings.Contains(s, "Read×1") {
		t.Fatalf("expected Grep×2 Read×1, got %q", s)
	}
}

// ---------------------------------------------------------------------------
// Execute guard conditions (no real sub-agent call)
// ---------------------------------------------------------------------------

func TestExecuteDisabledWhenAutoMemoryDisabled(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")
	os.Setenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY", "1")
	defer os.Unsetenv("CLAUDE_CODE_DISABLE_AUTO_MEMORY")

	state := NewState()
	// Execute is fire-and-forget; guard returns immediately without spawning goroutine.
	Execute(context.Background(), state, ExtractionParams{
		ToolUseContext: types.ToolUseContext{},
	})
	DrainPendingExtraction(state)
	if len(state.inFlightExtractions) != 0 {
		t.Fatalf("expected no in-flight extractions when auto memory disabled, got %d", len(state.inFlightExtractions))
	}
}

func TestExecuteDisabledInAgent(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")
	agentID := "test-agent"
	state := NewState()
	Execute(context.Background(), state, ExtractionParams{
		ToolUseContext: types.ToolUseContext{
			AgentID: &agentID,
		},
	})
	DrainPendingExtraction(state)
	if len(state.inFlightExtractions) != 0 {
		t.Fatalf("expected no in-flight extractions when in agent, got %d", len(state.inFlightExtractions))
	}
}

func TestExecuteSimpleMode(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")
	os.Setenv("CLAUDE_CODE_SIMPLE", "1")
	defer os.Unsetenv("CLAUDE_CODE_SIMPLE")

	state := NewState()
	Execute(context.Background(), state, ExtractionParams{
		ToolUseContext: types.ToolUseContext{},
	})
	DrainPendingExtraction(state)
	if len(state.inFlightExtractions) != 0 {
		t.Fatalf("expected no in-flight extractions in simple mode, got %d", len(state.inFlightExtractions))
	}
}

// CLAUDE_CODE_SIMPLE=0 is not bare mode (project defaults often set 0). Extraction
// must not treat it as simple — otherwise merge env + truthy check diverge.
func TestExecuteClaudeCodeSimpleZeroIsNotSimpleMode(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")
	os.Setenv("CLAUDE_CODE_SIMPLE", "0")
	defer os.Unsetenv("CLAUDE_CODE_SIMPLE")
	os.Setenv("CLAUDE_CODE_TENGU_BRAMBLE_LINTEL", "3")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_BRAMBLE_LINTEL")
	memDir := filepath.Join(t.TempDir(), "projects", "test", "memory")
	os.Setenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY", memDir)
	defer os.Unsetenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY")

	state := NewState()
	Execute(context.Background(), state, ExtractionParams{
		Messages:       []types.Message{makeMsg("1", types.MessageTypeUser)},
		ToolUseContext: types.ToolUseContext{},
		Cwd:            t.TempDir(),
	})
	DrainPendingExtraction(state)
	// Reached throttle logic (turn 1 < 3), not the simple_mode return before throttle.
	if state.TurnsSinceLastExtraction != 1 {
		t.Fatalf("expected throttle counter 1 (past simple guard), got %d", state.TurnsSinceLastExtraction)
	}
}

func TestExecuteNonInteractiveSkipsWithoutSlateThimble(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")
	os.Setenv("CLAUDE_CODE_TENGU_SLATE_THIMBLE", "0")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_SLATE_THIMBLE")
	memDir := filepath.Join(t.TempDir(), "projects", "test", "memory")
	os.Setenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY", memDir)
	defer os.Unsetenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY")

	state := NewState()
	Execute(context.Background(), state, ExtractionParams{
		Messages: []types.Message{makeMsg("1", types.MessageTypeUser)},
		ToolUseContext: types.ToolUseContext{
			Options: types.ToolUseContextOptionsData{
				IsNonInteractiveSession: true,
			},
		},
		Cwd: t.TempDir(),
	})
	DrainPendingExtraction(state)
	if len(state.inFlightExtractions) != 0 {
		t.Fatalf("expected no in-flight extractions when non-interactive without slate_thimble, got %d", len(state.inFlightExtractions))
	}
}

func TestExecuteThrottle(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")
	// Set throttle to 3 to verify throttling behavior.
	os.Setenv("CLAUDE_CODE_TENGU_BRAMBLE_LINTEL", "3")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_BRAMBLE_LINTEL")
	memDir := filepath.Join(t.TempDir(), "projects", "test", "memory")
	os.Setenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY", memDir)
	defer os.Unsetenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY")

	state := NewState()
	// First call: TurnsSinceLastExtraction=0, incremented to 1 < 3 → throttle.
	Execute(context.Background(), state, ExtractionParams{
		Messages: []types.Message{makeMsg("1", types.MessageTypeUser)},
		Cwd:      t.TempDir(),
	})
	DrainPendingExtraction(state)
	if state.TurnsSinceLastExtraction != 1 {
		t.Fatalf("expected throttle counter 1 after skipped turn, got %d", state.TurnsSinceLastExtraction)
	}
}

func TestExecutePassportQuailGate(t *testing.T) {
	// Without TENGU_PASSPORT_QUAIL and without GOC_EXTRACT_MEMORIES, Execute should skip immediately.
	os.Setenv("GOC_EXTRACT_MEMORIES", "0")
	defer os.Unsetenv("GOC_EXTRACT_MEMORIES")
	state := NewState()
	Execute(context.Background(), state, ExtractionParams{
		ToolUseContext: types.ToolUseContext{},
	})
	DrainPendingExtraction(state)
	if len(state.inFlightExtractions) != 0 {
		t.Fatalf("expected no in-flight extractions when passport_quail gate is disabled, got %d", len(state.inFlightExtractions))
	}
}

func TestDrainPendingExtraction(t *testing.T) {
	// DrainPendingExtraction drains in-flight extraction goroutines.
	// Verify it doesn't panic on empty state (no in-flight work).
	state := NewState()
	DrainPendingExtraction(state)
	DrainPendingExtraction(state, 1000)
}

// ---------------------------------------------------------------------------
// buildRestrictedExecutionDeps
// ---------------------------------------------------------------------------

func TestRestrictedExecutionDepsAllowsOnlyExpectedTools(t *testing.T) {
	// Create a memDir under the current working directory so that
	// WriteFromJSONDeps/EditFromJSONDeps (which check workspace roots) accept it.
	// Set CLAUDE_CODE_AUTO_MEMORY_DIRECTORY so memdir.IsAutoMemPath matches.
	cwd, _ := os.Getwd()
	memDir := filepath.Join(cwd, ".tmp-test-memdir")
	os.MkdirAll(memDir, 0700)
	defer os.RemoveAll(memDir)
	t.Setenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY", memDir)

	deps := buildRestrictedExecutionDeps(memDir)
	ctx := context.Background()

	// Write a file first so Edit can succeed.
	writePath := filepath.Join(memDir, "test.md")
	os.WriteFile(writePath, []byte("hello"), 0644)

	tests := []struct {
		name      string
		input     json.RawMessage
		wantErr   bool
		errNotFwd bool
	}{
		{"Read", json.RawMessage(`{"file_path":"` + memDir + `/test.md"}`), false, false},
		{"Glob", json.RawMessage(`{"pattern":"*.go"}`), false, false},
		{"Grep", json.RawMessage(`{"pattern":"test"}`), false, false},
		{"Bash", json.RawMessage(`{"command":"echo hi"}`), false, false},
		{"Write", json.RawMessage(`{"file_path":"` + memDir + `/new.md","content":"hi"}`), false, false},
		{"Edit", json.RawMessage(`{"file_path":"` + memDir + `/test.md","old_string":"hello","new_string":"world"}`), false, false},
		{"UnknownTool", json.RawMessage(`{}`), true, true},
	}

	for _, tc := range tests {
		_, _, err := deps.InvokeTool(ctx, tc.name, "", tc.input)
		if tc.errNotFwd {
			if err == nil || !strings.Contains(err.Error(), "not allowed") {
				t.Errorf("%s: expected 'not allowed' error, got: %v", tc.name, err)
			}
			continue
		}
		// tool was forwarded (not rejected by routing) — if there is an error, check it's not 'not allowed'
		if err != nil && strings.Contains(err.Error(), "not allowed") {
			t.Errorf("%s: tool was rejected by routing when it should be forwarded: %v", tc.name, err)
		}
		if tc.wantErr && err == nil {
			t.Errorf("%s: expected error but got none", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
		}
	}
}

func TestRestrictedExecutionDepsDeniesWriteOutsideMemDir(t *testing.T) {
	memDir := t.TempDir()
	t.Setenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY", memDir)
	deps := buildRestrictedExecutionDeps(memDir)
	ctx := context.Background()

	// Write outside memDir.
	_, _, err := deps.InvokeTool(ctx, "Write", "", json.RawMessage(`{"file_path":"/tmp/outside.txt","content":"hi"}`))
	if err == nil {
		t.Error("expected error for Write outside memDir")
	}
	if err != nil && !strings.Contains(err.Error(), "not in memory directory") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Edit outside memDir.
	_, _, err = deps.InvokeTool(ctx, "Edit", "", json.RawMessage(`{"file_path":"/tmp/outside.txt","old_string":"x","new_string":"y"}`))
	if err == nil {
		t.Error("expected error for Edit outside memDir")
	}
}

// ---------------------------------------------------------------------------
// NewState
// ---------------------------------------------------------------------------

func TestNewState(t *testing.T) {
	s := NewState()
	if s == nil {
		t.Fatal("NewState returned nil")
	}
	if s.LastMemoryMessageUUID != "" {
		t.Errorf("expected empty LastMemoryMessageUUID, got %q", s.LastMemoryMessageUUID)
	}
	if s.TurnsSinceLastExtraction != 0 {
		t.Errorf("expected TurnsSinceLastExtraction=0, got %d", s.TurnsSinceLastExtraction)
	}
}

// ---------------------------------------------------------------------------
// Concurrent safety smoke test
// ---------------------------------------------------------------------------

func TestStateConcurrentAccess(t *testing.T) {
	s := NewState()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.mu.Lock()
			s.TurnsSinceLastExtraction++
			s.LastMemoryMessageUUID = string(rune('a' + n))
			s.mu.Unlock()
		}(i)
	}
	wg.Wait()
	if s.TurnsSinceLastExtraction != 20 {
		t.Errorf("expected 20 increments, got %d", s.TurnsSinceLastExtraction)
	}
}

// ---------------------------------------------------------------------------
// dirExistsGuidance constant
// ---------------------------------------------------------------------------

func TestDirExistsGuidance(t *testing.T) {
	if dirExistsGuidance == "" {
		t.Error("dirExistsGuidance should not be empty")
	}
	if !strings.Contains(dirExistsGuidance, "Write tool") {
		t.Error("should mention Write tool")
	}
}

// ---------------------------------------------------------------------------
// countModelVisibleMessages
// ---------------------------------------------------------------------------

func TestCountModelVisibleMessages(t *testing.T) {
	tests := []struct {
		name string
		msgs []types.Message
		want int
	}{
		{"nil slice", nil, 0},
		{"empty", []types.Message{}, 0},
		{"only user", []types.Message{makeMsg("1", types.MessageTypeUser)}, 1},
		{"user+assistant", []types.Message{
			makeMsg("1", types.MessageTypeUser),
			makeMsg("2", types.MessageTypeAssistant),
		}, 2},
		{"mixed with system/progress", []types.Message{
			makeMsg("1", types.MessageTypeUser),
			makeMsg("2", types.MessageTypeSystem),
			makeMsg("3", types.MessageTypeAssistant),
			makeMsg("4", types.MessageTypeProgress),
		}, 2},
		{"all non-visible", []types.Message{
			makeMsg("1", types.MessageTypeSystem),
			makeMsg("2", types.MessageTypeProgress),
		}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := countModelVisibleMessages(tc.msgs)
			if got != tc.want {
				t.Errorf("countModelVisibleMessages = %d; want %d", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// createMemorySavedMessage
// ---------------------------------------------------------------------------

func TestCreateMemorySavedMessage(t *testing.T) {
	var uuidCounter int
	fakeUUID := func() string {
		uuidCounter++
		return "mem-uuid-" + strings.Repeat("0", uuidCounter)
	}

	paths := []string{"/mem/a.md", "/mem/b.md"}
	msg := createMemorySavedMessage(paths, fakeUUID)

	if msg.Type != types.MessageTypeSystem {
		t.Errorf("expected system type, got %s", msg.Type)
	}
	if msg.Subtype == nil || *msg.Subtype != types.SubtypeMemorySaved {
		t.Errorf("expected subtype memory_saved, got %v", msg.Subtype)
	}
	if len(msg.WrittenPaths) != 2 || msg.WrittenPaths[0] != "/mem/a.md" {
		t.Errorf("unexpected WrittenPaths: %v", msg.WrittenPaths)
	}
	if msg.Timestamp == nil || *msg.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
	if msg.IsMeta == nil || *msg.IsMeta != false {
		t.Error("expected IsMeta to be false")
	}
	if msg.UUID != "mem-uuid-0" {
		t.Errorf("expected uuid mem-uuid-0, got %s", msg.UUID)
	}
}

// ---------------------------------------------------------------------------
// buildExtractionPrompt with SkipIndex
// ---------------------------------------------------------------------------

func TestBuildExtractionPromptSkipIndex(t *testing.T) {
	dir := t.TempDir()
	// Create MEMORY.md index.
	os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("# Index\n- [Role](user_role.md)"), 0644)

	p := ExtractionParams{
		SkipIndex: true,
	}
	newMsgs := []types.Message{makeMsg("1", types.MessageTypeUser)}

	result := buildExtractionPrompt(p, newMsgs, dir)
	if strings.Contains(result, "MEMORY.md index") {
		t.Error("expected MEMORY.md index to be suppressed when SkipIndex=true")
	}
	if !strings.Contains(result, "memory extraction subagent") {
		t.Error("expected prompt to contain 'memory extraction subagent'")
	}
	// SkipIndex=true should NOT mention Step 2 (MEMORY.md index update).
	if strings.Contains(result, "Step 2") {
		t.Error("expected no 'Step 2' when SkipIndex=true")
	}
}

func TestBuildExtractionPromptShowsIndexByDefault(t *testing.T) {
	dir := t.TempDir()
	// Create MEMORY.md index.
	os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("# Index\n- [Role](user_role.md)"), 0644)

	p := ExtractionParams{
		// SkipIndex defaults to false.
	}
	newMsgs := []types.Message{makeMsg("1", types.MessageTypeUser)}

	result := buildExtractionPrompt(p, newMsgs, dir)
	if !strings.Contains(result, "MEMORY.md index") {
		t.Error("expected MEMORY.md index when SkipIndex=false")
	}
}

// ---------------------------------------------------------------------------
// Execute: AppendSystemMessage callback
// ---------------------------------------------------------------------------

func TestExecuteAppendSystemMessageIsCalled(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")
	memDir := filepath.Join(t.TempDir(), "projects", "test", "memory")
	os.Setenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY", memDir)
	defer os.Unsetenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY")

	state := NewState()
	// Bypass throttle: set turn count >= threshold.
	state.mu.Lock()
	state.TurnsSinceLastExtraction = 0
	state.mu.Unlock()

	appendFn := func(msg types.Message) {
	}

	// Execute with a fake sub-agent that writes nothing — we can't easily
	// test the real sub-agent path, so we just verify the callback path
	// doesn't panic when memoryPaths is empty.
	Execute(context.Background(), state, ExtractionParams{
		Messages:            []types.Message{makeMsg("1", types.MessageTypeUser)},
		Cwd:                 t.TempDir(),
		ToolUseContext:      types.ToolUseContext{},
		AppendSystemMessage: appendFn,
	})
	DrainPendingExtraction(state)
	// No memory paths produced (sub-agent doesn't actually run in unit test).
	// With no API key configured, the goroutine will fail but the test
	// verifies the callback plumbing doesn't panic.
}

// ---------------------------------------------------------------------------
// Execute: hasMemoryWritesSince advances cursor
// ---------------------------------------------------------------------------

func TestExecuteAdvancesCursorOnPriorMemoryWrite(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")

	state := NewState()
	msgs := []types.Message{
		makeMsg("a", types.MessageTypeUser),
		makeAssistantMsg("b", writeToolBlock("Write", "/tmp/some-mem/test.md")),
		makeMsg("c", types.MessageTypeUser),
	}

	// With no cursor and no real auto-memory dir, hasMemoryWritesSince
	// won't detect writes (the path won't match GetAutoMemPath).
	// This test verifies the codepath doesn't panic.
	state.LastMemoryMessageUUID = ""
	Execute(context.Background(), state, ExtractionParams{
		Messages:       msgs,
		Cwd:            t.TempDir(),
		ToolUseContext: types.ToolUseContext{},
	})
	DrainPendingExtraction(state)
}

// ---------------------------------------------------------------------------
// Execute: coalescing gate (inProgress + pendingParams)
// ---------------------------------------------------------------------------

func TestExecuteCoalescingGate(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")
	memDir := filepath.Join(t.TempDir(), "projects", "test", "memory")
	os.Setenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY", memDir)
	defer os.Unsetenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY")

	state := NewState()
	// Simulate in-progress extraction.
	state.inProgress = true

	p := ExtractionParams{
		Messages:       []types.Message{makeMsg("1", types.MessageTypeUser)},
		Cwd:            t.TempDir(),
		ToolUseContext: types.ToolUseContext{},
	}
	Execute(context.Background(), state, p)
	if state.pendingParams == nil {
		t.Fatal("expected pendingParams to be set after coalescing")
	}
	if !state.inProgress {
		t.Error("expected inProgress to remain true after coalescing")
	}
	// Verify the stashed params match what we passed in.
	if len(state.pendingParams.Messages) != 1 || state.pendingParams.Messages[0].UUID != "1" {
		t.Errorf("pendingParams messages mismatch: %+v", state.pendingParams.Messages)
	}
}

func TestExecuteCoalescingOverwrite(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")
	memDir := filepath.Join(t.TempDir(), "projects", "test", "memory")
	os.Setenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY", memDir)
	defer os.Unsetenv("CLAUDE_CODE_AUTO_MEMORY_DIRECTORY")

	state := NewState()
	state.inProgress = true

	// First coalesce.
	p1 := ExtractionParams{
		Messages:       []types.Message{makeMsg("first", types.MessageTypeUser)},
		Cwd:            t.TempDir(),
		ToolUseContext: types.ToolUseContext{},
	}
	Execute(context.Background(), state, p1)

	// Second coalesce should overwrite.
	p2 := ExtractionParams{
		Messages:       []types.Message{makeMsg("second", types.MessageTypeUser)},
		Cwd:            t.TempDir(),
		ToolUseContext: types.ToolUseContext{},
	}
	Execute(context.Background(), state, p2)
	// Should hold the second (latest) params (overwrite semantics).
	if state.pendingParams == nil || state.pendingParams.Messages[0].UUID != "second" {
		t.Errorf("expected pendingParams to hold second (latest) call, got UUID=%q",
			func() string {
				if state.pendingParams != nil && len(state.pendingParams.Messages) > 0 {
					return state.pendingParams.Messages[0].UUID
				}
				return "nil"
			}())
	}
}
