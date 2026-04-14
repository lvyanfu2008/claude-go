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
	if len(lines) < 4 {
		t.Fatalf("expected metadata + user + last-prompt at least, got %d lines", len(lines))
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &meta); err != nil {
		t.Fatal(err)
	}
	if meta["type"] != "custom-title" {
		t.Fatalf("first line: %#v", meta)
	}
	var assistantRow map[string]any
	for i := len(lines) - 1; i >= 0; i-- {
		var row map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &row); err != nil {
			continue
		}
		if row["type"] == "assistant" {
			assistantRow = row
			break
		}
	}
	if assistantRow == nil {
		t.Fatalf("no assistant row in %v", lines)
	}
	if assistantRow["parentUuid"] != u1 {
		t.Fatalf("assistant parent: %#v", assistantRow["parentUuid"])
	}
	var lastPromptTail map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &lastPromptTail); err != nil {
		t.Fatal(err)
	}
	if lastPromptTail["type"] != "last-prompt" || lastPromptTail["lastPrompt"] != "hello" {
		t.Fatalf("want last line last-prompt hello, got %#v", lastPromptTail)
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

func countJSONLType(t *testing.T, path, wantType string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, ln := range strings.Split(string(raw), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(ln), &row); err != nil {
			continue
		}
		if row["type"] == wantType {
			n++
		}
	}
	return n
}

func TestRemoveAllLastPromptJSONLRows_removesRun(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "strip.jsonl")
	sid := "11111111-2222-3333-4444-555555555555"
	lines := []string{
		`{"type":"user","uuid":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`,
		`{"type":"last-prompt","lastPrompt":"a","sessionId":"` + sid + `"}`,
		`{"type":"last-prompt","lastPrompt":"b","sessionId":"` + sid + `"}`,
	}
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removeAllLastPromptJSONLRows(p); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	got := strings.TrimSpace(string(raw))
	want := lines[0]
	if got != want {
		t.Fatalf("after strip want %q got %q", want, got)
	}
}

func TestRecordTranscript_twoBatches_oneLastPromptLine(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SKIP_PROMPT_HISTORY", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "twice.jsonl")
	sid := "11111111-2222-3333-4444-555555555555"
	st := &Store{
		SessionID:      sid,
		OriginalCwd:    "/proj/foo",
		ConfigHome:     filepath.Join(dir, "claude"),
		Cwd:            "/proj/foo",
		UserType:       "external",
		TranscriptFile: path,
		InitialMetadata: &SessionMetadata{
			LastPrompt:  "seed",
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
		Type:    types.MessageTypeUser,
		UUID:    u2,
		Message: json.RawMessage(`{"role":"user","content":"world"}`),
	}
	if _, err := st.RecordTranscript(context.Background(), []types.Message{m1}, RecordOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RecordTranscript(context.Background(), []types.Message{m2}, RecordOpts{}); err != nil {
		t.Fatal(err)
	}
	if n := countJSONLType(t, path, "last-prompt"); n != 1 {
		t.Fatalf("want exactly 1 last-prompt row, got %d", n)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	var tail map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &tail); err != nil {
		t.Fatal(err)
	}
	if tail["type"] != "last-prompt" || tail["lastPrompt"] != "world" {
		t.Fatalf("tail last-prompt: %#v", tail)
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
	var compact map[string]any
	for i := len(lines) - 1; i >= 0; i-- {
		var row map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &row); err != nil {
			continue
		}
		if row["type"] == "system" {
			compact = row
			break
		}
	}
	if compact == nil {
		t.Fatalf("no system row in %v", lines)
	}
	if compact["parentUuid"] != nil {
		t.Fatalf("compact boundary parentUuid want null got %#v", compact["parentUuid"])
	}
	if compact["logicalParentUuid"] != u0 {
		t.Fatalf("logicalParent: %#v", compact["logicalParentUuid"])
	}
}

func TestRecordTranscript_fileHistorySnapshot_skipsWhenCheckpointingDisabled(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SKIP_PROMPT_HISTORY", "")
	t.Setenv("CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING", "1")
	dir := t.TempDir()
	path := filepath.Join(dir, "no-fh.jsonl")
	st := &Store{
		SessionID:                 "11111111-2222-3333-4444-555555555555",
		OriginalCwd:               "/proj/foo",
		ConfigHome:                filepath.Join(dir, "claude"),
		Cwd:                       "/proj/foo",
		UserType:                  "external",
		TranscriptFile:            path,
		FileHistorySnapshotOnUser: true,
	}
	uid := "5b03942f-f884-4793-abfa-12aa4947fdf7"
	m1 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    uid,
		Message: json.RawMessage(`{"role":"user","content":"hello"}`),
	}
	_, err := st.RecordTranscript(context.Background(), []types.Message{m1}, RecordOpts{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "file-history-snapshot") {
		t.Fatalf("expected no file-history-snapshot when checkpointing disabled, got: %s", string(raw))
	}
}

func TestRecordTranscript_fileHistorySnapshotOnce_twoUsersOneStub(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SKIP_PROMPT_HISTORY", "")
	t.Setenv("CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "once.jsonl")
	st := &Store{
		SessionID:                 "11111111-2222-3333-4444-555555555555",
		OriginalCwd:               "/proj/foo",
		ConfigHome:                filepath.Join(dir, "claude"),
		Cwd:                       "/proj/foo",
		UserType:                  "external",
		TranscriptFile:            path,
		FileHistorySnapshotOnUser: true,
		FileHistorySnapshotOnce:   true,
	}
	u1 := "5b03942f-f884-4793-abfa-12aa4947fdf7"
	u2 := "6c14a53f-f884-4793-abfa-12aa4947fdf8"
	m1 := types.Message{Type: types.MessageTypeUser, UUID: u1, Message: json.RawMessage(`{"role":"user","content":"a"}`)}
	m2 := types.Message{Type: types.MessageTypeUser, UUID: u2, Message: json.RawMessage(`{"role":"user","content":"b"}`)}
	if _, err := st.RecordTranscript(context.Background(), []types.Message{m1}, RecordOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RecordTranscript(context.Background(), []types.Message{m2}, RecordOpts{StartingParentUUID: u1}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fhCount int
	for _, ln := range strings.Split(string(raw), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(ln), &row); err != nil {
			t.Fatal(err)
		}
		if row["type"] == "file-history-snapshot" {
			fhCount++
		}
	}
	if fhCount != 1 {
		t.Fatalf("want exactly 1 file-history-snapshot line, got %d", fhCount)
	}
}

func TestRecordTranscript_fileHistorySnapshotBeforeNewUser(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SKIP_PROMPT_HISTORY", "")
	t.Setenv("CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "fh.jsonl")
	st := &Store{
		SessionID:                 "11111111-2222-3333-4444-555555555555",
		OriginalCwd:               "/proj/foo",
		ConfigHome:                filepath.Join(dir, "claude"),
		Cwd:                       "/proj/foo",
		UserType:                  "external",
		TranscriptFile:            path,
		FileHistorySnapshotOnUser: true,
		FileHistorySnapshotOnce:   false,
	}
	uid := "5b03942f-f884-4793-abfa-12aa4947fdf7"
	m1 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    uid,
		Message: json.RawMessage(`{"role":"user","content":"hello"}`),
	}
	_, err := st.RecordTranscript(context.Background(), []types.Message{m1}, RecordOpts{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, ln := range strings.Split(string(raw), "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			lines = append(lines, ln)
		}
	}
	if len(lines) != 3 {
		t.Fatalf("want 3 jsonl lines (file-history-snapshot + user + last-prompt), got %d: %v", len(lines), lines)
	}
	var fhrow, urow, lp map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &fhrow); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &urow); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[2]), &lp); err != nil {
		t.Fatal(err)
	}
	if fhrow["type"] != "file-history-snapshot" {
		t.Fatalf("first line type: %#v", fhrow["type"])
	}
	if fhrow["messageId"] != uid {
		t.Fatalf("fh messageId: %#v", fhrow["messageId"])
	}
	if fhrow["isSnapshotUpdate"] != false {
		t.Fatalf("isSnapshotUpdate: %#v", fhrow["isSnapshotUpdate"])
	}
	snap, ok := fhrow["snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("snapshot: %#v", fhrow["snapshot"])
	}
	if snap["messageId"] != uid {
		t.Fatalf("snapshot.messageId: %#v", snap["messageId"])
	}
	tb, ok := snap["trackedFileBackups"].(map[string]any)
	if !ok || len(tb) != 0 {
		t.Fatalf("trackedFileBackups: %#v", snap["trackedFileBackups"])
	}
	if _, ok := snap["timestamp"].(string); !ok || snap["timestamp"] == "" {
		t.Fatalf("timestamp: %#v", snap["timestamp"])
	}
	if urow["uuid"] != uid || urow["type"] != "user" {
		t.Fatalf("second line: %#v", urow)
	}
	if lp["type"] != "last-prompt" || lp["lastPrompt"] != "hello" {
		t.Fatalf("third line last-prompt: %#v", lp)
	}
}

func TestRecordTranscript_skipsFileHistoryForMetaUser(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SKIP_PROMPT_HISTORY", "")
	t.Setenv("CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "meta.jsonl")
	meta := true
	st := &Store{
		SessionID:                 "11111111-2222-3333-4444-555555555555",
		OriginalCwd:               "/proj/foo",
		ConfigHome:                filepath.Join(dir, "claude"),
		Cwd:                       "/proj/foo",
		UserType:                  "external",
		TranscriptFile:            path,
		FileHistorySnapshotOnUser: true,
		FileHistorySnapshotOnce:   false,
	}
	m1 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "5b03942f-f884-4793-abfa-12aa4947fdf7",
		IsMeta:  &meta,
		Message: json.RawMessage(`{"role":"user","content":"x"}`),
	}
	_, err := st.RecordTranscript(context.Background(), []types.Message{m1}, RecordOpts{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	n := 0
	for _, ln := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(ln) != "" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("meta user should not get file-history line, got %d lines", n)
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
