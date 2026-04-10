package sessiontranscript

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goc/types"
)

func TestRecordTranscript_incrementalAndParent(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SKIP_PROMPT_HISTORY", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "sess.jsonl")
	st := &Store{
		SessionID:      "11111111-2222-3333-4444-555555555555",
		OriginalCwd:    "/proj/foo",
		ConfigHome:     filepath.Join(dir, "claude"),
		Cwd:            "/proj/foo",
		UserType:       "external",
		TranscriptFile: path,
		InitialMetadata: &SessionMetadata{
			CustomTitle: "t",
		},
	}
	u1 := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	u2 := "bbbbbbbb-cccc-dddd-eeee-ffffffffffff"
	m1 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    u1,
		Message: json.RawMessage(`{"role":"user","content":"hello"}`),
	}
	m2 := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    u2,
		Message: json.RawMessage(`{"role":"assistant","content":"yo"}`),
	}
	last, err := st.RecordTranscript(context.Background(), []types.Message{m1}, RecordOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if last != u1 {
		t.Fatalf("last %q", last)
	}
	last2, err := st.RecordTranscript(context.Background(), []types.Message{m2}, RecordOpts{StartingParentUUID: last})
	if err != nil {
		t.Fatal(err)
	}
	if last2 != u2 {
		t.Fatalf("last2 %q", last2)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected metadata + 2 msgs, got %d lines", len(lines))
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &meta); err != nil {
		t.Fatal(err)
	}
	if meta["type"] != "custom-title" {
		t.Fatalf("first line: %#v", meta)
	}
	var row map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &row); err != nil {
		t.Fatal(err)
	}
	if row["parentUuid"] != u1 {
		t.Fatalf("assistant parent: %#v", row["parentUuid"])
	}

	// Dedup: same UUID again adds no line
	nBefore := len(lines)
	_, err = st.RecordTranscript(context.Background(), []types.Message{m1, m2}, RecordOpts{})
	if err != nil {
		t.Fatal(err)
	}
	data2, _ := os.ReadFile(path)
	lines2 := strings.Split(strings.TrimSpace(string(data2)), "\n")
	if len(lines2) != nBefore {
		t.Fatalf("dedup failed: before %d after %d", nBefore, len(lines2))
	}
}

func TestRecordTranscript_invalidSessionID(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SKIP_PROMPT_HISTORY", "")
	st := &Store{
		SessionID:      "not-a-uuid",
		OriginalCwd:    "/p",
		ConfigHome:     t.TempDir(),
		TranscriptFile: filepath.Join(t.TempDir(), "x.jsonl"),
	}
	_, err := st.RecordTranscript(context.Background(), []types.Message{
		{Type: types.MessageTypeUser, UUID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Message: json.RawMessage(`{"role":"user","content":"x"}`)},
	}, RecordOpts{})
	if err == nil {
		t.Fatal("expected error for invalid SessionID")
	}
}

func TestRecordTranscript_skipPersistence(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SKIP_PROMPT_HISTORY", "1")
	dir := t.TempDir()
	path := filepath.Join(dir, "x.jsonl")
	st := &Store{
		SessionID:      "11111111-2222-3333-4444-555555555555",
		OriginalCwd:    "/p",
		ConfigHome:     filepath.Join(dir, "c"),
		TranscriptFile: path,
	}
	_, err := st.RecordTranscript(context.Background(), []types.Message{
		{Type: types.MessageTypeUser, UUID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Message: json.RawMessage(`{"role":"user","content":"x"}`)},
	}, RecordOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file should not exist")
	}
}

func TestRecordTranscript_compactBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.jsonl")
	st := &Store{
		SessionID:      "11111111-2222-3333-4444-555555555555",
		OriginalCwd:    "/p",
		ConfigHome:     filepath.Join(dir, "c"),
		TranscriptFile: path,
	}
	sub := "compact_boundary"
	u0 := "00000000-0000-0000-0000-000000000001"
	u1 := "00000000-0000-0000-0000-000000000002"
	prior, err := st.RecordTranscript(context.Background(), []types.Message{
		{Type: types.MessageTypeUser, UUID: u0, Message: json.RawMessage(`{"role":"user","content":"a"}`)},
	}, RecordOpts{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.RecordTranscript(context.Background(), []types.Message{
		{Type: types.MessageTypeSystem, UUID: u1, Subtype: &sub},
	}, RecordOpts{StartingParentUUID: prior})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	var last map[string]any
	json.Unmarshal([]byte(lines[len(lines)-1]), &last)
	if last["parentUuid"] != nil {
		t.Fatalf("compact boundary parentUuid want null got %#v", last["parentUuid"])
	}
	if last["logicalParentUuid"] != u0 {
		t.Fatalf("logicalParent: %#v", last["logicalParentUuid"])
	}
}

func TestRecordSidechainTranscript(t *testing.T) {
	dir := t.TempDir()
	st := &Store{
		SessionID:   "11111111-2222-3333-4444-555555555555",
		OriginalCwd: "/proj/x",
		ConfigHome:  filepath.Join(dir, "claude"),
		Cwd:         "/proj/x",
	}
	st.TranscriptFile = filepath.Join(dir, "main.jsonl")
	agent := "agent-uuid-1234-5678-90ab-cdef01234567"
	path := st.sidechainPath(agent)
	if !strings.Contains(path, "subagents") || !strings.HasSuffix(path, "agent-"+agent+".jsonl") {
		t.Fatalf("bad path %q", path)
	}
	err := st.RecordSidechainTranscript(context.Background(), agent, []types.Message{
		{Type: types.MessageTypeUser, UUID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Message: json.RawMessage(`{"role":"user","content":"s"}`)},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	var row map[string]any
	json.Unmarshal(raw, &row)
	if row["isSidechain"] != true {
		t.Fatalf("%#v", row["isSidechain"])
	}
}
