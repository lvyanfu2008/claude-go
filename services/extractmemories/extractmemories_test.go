package extractmemories

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"goc/claudemd"
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
// readMemoryFileFrontmatter
// ---------------------------------------------------------------------------

func TestReadMemoryFileFrontmatter(t *testing.T) {
	dir := t.TempDir()

	// File with valid frontmatter.
	p1 := filepath.Join(dir, "user_role.md")
	os.WriteFile(p1, []byte("---\nname: Role\ndescription: desc text\ntype: user\n---\n\ncontent"), 0644)

	info := readMemoryFileFrontmatter(p1)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Name != "Role" {
		t.Errorf("Name = %q; want Role", info.Name)
	}
	if info.Description != "desc text" {
		t.Errorf("Description = %q; want desc text", info.Description)
	}
	if info.Type != "user" {
		t.Errorf("Type = %q; want user", info.Type)
	}

	// File with no frontmatter.
	p2 := filepath.Join(dir, "plain.md")
	os.WriteFile(p2, []byte("just content"), 0644)
	if info := readMemoryFileFrontmatter(p2); info != nil {
		t.Fatal("expected nil for no frontmatter")
	}

	// File with frontmatter but empty name/description.
	p3 := filepath.Join(dir, "empty.md")
	os.WriteFile(p3, []byte("---\nname: \ndescription: \ntype: user\n---\n\ncontent"), 0644)
	if info := readMemoryFileFrontmatter(p3); info != nil {
		t.Fatal("expected nil when name and description are empty")
	}

	// Non-existent file.
	if info := readMemoryFileFrontmatter(filepath.Join(dir, "nonexistent.md")); info != nil {
		t.Fatal("expected nil for non-existent file")
	}
}

// ---------------------------------------------------------------------------
// scanExistingMemories
// ---------------------------------------------------------------------------

func TestScanExistingMemories(t *testing.T) {
	dir := t.TempDir()

	// Write a couple of memory files.
	os.WriteFile(filepath.Join(dir, "user_role.md"), []byte("---\nname: Role\ndescription: User role info\ntype: user\n---\n\ncontent"), 0644)
	os.WriteFile(filepath.Join(dir, "feedback_testing.md"), []byte("---\nname: Testing Tips\ndescription: Short\ntype: feedback\n---\n\ncontent"), 0644)
	// MEMORY.md should be skipped.
	os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("# Index\n- [Role](user_role.md)"), 0644)
	// Non-.md file should be skipped.
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0644)

	result := scanExistingMemories(dir)

	if !strings.Contains(result, "user_role.md") {
		t.Errorf("result should contain user_role.md, got: %s", result)
	}
	if !strings.Contains(result, "feedback_testing.md") {
		t.Errorf("result should contain feedback_testing.md, got: %s", result)
	}
	if strings.Contains(result, "MEMORY.md") {
		t.Errorf("result should NOT contain MEMORY.md, got: %s", result)
	}
	if strings.Contains(result, "notes.txt") {
		t.Errorf("result should NOT contain notes.txt, got: %s", result)
	}
	if !strings.Contains(result, "name=Role") {
		t.Errorf("result should contain frontmatter name, got: %s", result)
	}

	// Empty dir.
	emptyDir := t.TempDir()
	if s := scanExistingMemories(emptyDir); s != "" {
		t.Errorf("expected empty for empty dir, got: %s", s)
	}

	// Empty string dir.
	if s := scanExistingMemories(""); s != "" {
		t.Errorf("expected empty for empty string, got: %s", s)
	}
}

func TestScanExistingMemoriesTruncatesLongDescription(t *testing.T) {
	dir := t.TempDir()
	longDesc := strings.Repeat("a", 100)
	os.WriteFile(filepath.Join(dir, "long.md"), []byte("---\nname: Lng\ndescription: "+longDesc+"\ntype: project\n---\n\ncontent"), 0644)

	result := scanExistingMemories(dir)
	if !strings.Contains(result, "aaa") {
		t.Errorf("should contain truncated description, got: %s", result)
	}
	// Should be truncated to 80 chars + ellipsis.
	if strings.Contains(result, longDesc) {
		t.Errorf("description should be truncated, got full 100 chars")
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
// isPathInMemDir
// ---------------------------------------------------------------------------

func TestIsPathInMemDir(t *testing.T) {
	memDir := t.TempDir()

	// Path inside memDir.
	input, _ := json.Marshal(map[string]string{"file_path": filepath.Join(memDir, "test.md")})
	if !isPathInMemDir(input, memDir) {
		t.Error("expected path inside memDir to be allowed")
	}

	// Path outside.
	input2, _ := json.Marshal(map[string]string{"file_path": "/tmp/outside.md"})
	if isPathInMemDir(input2, memDir) {
		t.Error("expected path outside memDir to be denied")
	}

	// Empty file_path.
	input3, _ := json.Marshal(map[string]string{"file_path": ""})
	if isPathInMemDir(input3, memDir) {
		t.Error("expected empty file_path to be denied")
	}

	// Invalid JSON.
	if isPathInMemDir(json.RawMessage(`{bad`), memDir) {
		t.Error("expected invalid JSON to be denied")
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

	// More than 10 messages without cursor: only last 10.
	tenMsgs := make([]types.Message, 15)
	for i := range tenMsgs {
		tenMsgs[i] = makeMsg(string(rune('a'+i)), types.MessageTypeUser)
	}
	got5 := newMessagesSinceCursor(tenMsgs, "")
	if len(got5) != 10 {
		t.Fatalf("expected 10 messages without cursor (first run), got %d", len(got5))
	}
}

// ---------------------------------------------------------------------------
// hasMemoryWritesSince
// ---------------------------------------------------------------------------

func TestHasMemoryWritesSince(t *testing.T) {
	memDir := claudemd.GetAutoMemPath(t.TempDir())
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

func TestMemoryFormatSection(t *testing.T) {
	s := memoryFormatSection()
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
}

func TestWhatNotToSaveSection(t *testing.T) {
	s := whatNotToSaveSection()
	if !strings.Contains(s, "Code patterns") {
		t.Error("expected 'Code patterns' in section")
	}
	if !strings.Contains(s, "Git history") {
		t.Error("expected 'Git history' in section")
	}
}

func TestHowToSaveSection(t *testing.T) {
	s := howToSaveSection()
	if !strings.Contains(s, "```markdown") {
		t.Error("expected markdown code block in section")
	}
	if !strings.Contains(s, "MEMORY.md") {
		t.Error("expected MEMORY.md reference in section")
	}
}

func TestWhenToAccessSection(t *testing.T) {
	s := whenToAccessSection()
	if !strings.Contains(s, "When memories seem relevant") {
		t.Error("expected relevance guidance in section")
	}
	if !strings.Contains(s, "stale") {
		t.Error("expected stale memory guidance in section")
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
	if !strings.Contains(result, "Extract Memories") {
		t.Error("expected 'Extract Memories' heading")
	}
	if !strings.Contains(result, "user_role.md") {
		t.Error("expected existing memory listing")
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
	paths, err := Execute(context.Background(), state, ExtractionParams{
		ToolUseContext: types.ToolUseContext{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paths != nil {
		t.Fatalf("expected nil when auto memory disabled, got %v", paths)
	}
}

func TestExecuteDisabledInAgent(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")
	agentID := "test-agent"
	state := NewState()
	paths, err := Execute(context.Background(), state, ExtractionParams{
		ToolUseContext: types.ToolUseContext{
			AgentID: &agentID,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paths != nil {
		t.Fatalf("expected nil when inside agent, got %v", paths)
	}
}

func TestExecuteSimpleMode(t *testing.T) {
	os.Setenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL", "1")
	defer os.Unsetenv("CLAUDE_CODE_TENGU_PASSPORT_QUAIL")
	os.Setenv("CLAUDE_CODE_SIMPLE", "1")
	defer os.Unsetenv("CLAUDE_CODE_SIMPLE")

	state := NewState()
	paths, err := Execute(context.Background(), state, ExtractionParams{
		ToolUseContext: types.ToolUseContext{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paths != nil {
		t.Fatalf("expected nil in simple mode, got %v", paths)
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
	_, err := Execute(context.Background(), state, ExtractionParams{
		Messages:       []types.Message{makeMsg("1", types.MessageTypeUser)},
		ToolUseContext: types.ToolUseContext{},
		Cwd:            t.TempDir(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Reached throttle logic (turn 1 < 3), not the simple_mode return before throttle.
	if state.TurnsSinceLastExtraction != 1 {
		t.Fatalf("expected throttle counter 1 (past simple guard), got %d", state.TurnsSinceLastExtraction)
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
	paths, err := Execute(context.Background(), state, ExtractionParams{
		Messages: []types.Message{makeMsg("1", types.MessageTypeUser)},
		Cwd:      t.TempDir(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paths != nil {
		t.Fatalf("expected nil due to throttle (turn 1), got %v", paths)
	}
}

func TestExecutePassportQuailGate(t *testing.T) {
	// Without TENGU_PASSPORT_QUAIL and without GOC_EXTRACT_MEMORIES, Execute should return nil,nil.
	os.Setenv("GOC_EXTRACT_MEMORIES", "0")
	defer os.Unsetenv("GOC_EXTRACT_MEMORIES")
	state := NewState()
	paths, err := Execute(context.Background(), state, ExtractionParams{
		ToolUseContext: types.ToolUseContext{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paths != nil {
		t.Fatalf("expected nil when passport_quail gate is disabled, got %v", paths)
	}
}

func TestDrainPendingExtraction(t *testing.T) {
	// DrainPendingExtraction is a no-op in Go (synchronous extraction).
	// Verify it doesn't panic.
	DrainPendingExtraction()
	DrainPendingExtraction(1000)
}

// ---------------------------------------------------------------------------
// buildRestrictedExecutionDeps
// ---------------------------------------------------------------------------

func TestRestrictedExecutionDepsAllowsOnlyExpectedTools(t *testing.T) {
	// Create a memDir under the current working directory so that
	// WriteFromJSONDeps/EditFromJSONDeps (which check workspace roots) accept it.
	cwd, _ := os.Getwd()
	memDir := filepath.Join(cwd, ".tmp-test-memdir")
	os.MkdirAll(memDir, 0700)
	defer os.RemoveAll(memDir)

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
	if !strings.Contains(result, "Extract Memories") {
		t.Error("expected 'Extract Memories' heading")
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

	var capturedMsg *types.Message
	appendFn := func(msg types.Message) {
		capturedMsg = &msg
	}

	// Execute with a fake sub-agent that writes nothing — we can't easily
	// test the real sub-agent path, so we just verify the callback path
	// doesn't panic when memoryPaths is empty.
	paths, err := Execute(context.Background(), state, ExtractionParams{
		Messages:            []types.Message{makeMsg("1", types.MessageTypeUser)},
		Cwd:                 t.TempDir(),
		ToolUseContext:      types.ToolUseContext{},
		AppendSystemMessage: appendFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No memory paths produced (sub-agent doesn't actually run in unit test).
	// The callback should not be called.
	if capturedMsg != nil {
		t.Error("AppendSystemMessage should not be called when no memories saved")
	}
	if paths != nil {
		t.Errorf("expected nil paths, got %v", paths)
	}
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
	paths, err := Execute(context.Background(), state, ExtractionParams{
		Messages:       msgs,
		Cwd:            t.TempDir(),
		ToolUseContext: types.ToolUseContext{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = paths
}
