package autodream

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReadLastConsolidatedAt_noFile(t *testing.T) {
	dir := t.TempDir()
	mtime, err := ReadLastConsolidatedAt(dir)
	if err != nil {
		t.Fatalf("expected no error for missing lock file, got %v", err)
	}
	if mtime != 0 {
		t.Fatalf("expected 0 for missing lock file, got %d", mtime)
	}
}

func TestReadLastConsolidatedAt_withFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, lockFileName)
	before := time.Now().Truncate(time.Millisecond)
	if err := os.WriteFile(path, []byte("1234"), 0644); err != nil {
		t.Fatal(err)
	}
	after := time.Now().Truncate(time.Millisecond)

	mtime, err := ReadLastConsolidatedAt(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mtime < before.UnixMilli() || mtime > after.UnixMilli() {
		t.Fatalf("expected mtime between %d and %d, got %d", before.UnixMilli(), after.UnixMilli(), mtime)
	}
}

func TestTryAcquireConsolidationLock_firstAcquire(t *testing.T) {
	dir := t.TempDir()
	prior, err := TryAcquireConsolidationLock(dir)
	if err != nil {
		t.Fatalf("expected no error on first acquire, got %v", err)
	}
	if prior != 0 {
		t.Fatalf("expected prior=0 for first acquire, got %d", prior)
	}

	// Verify lock file exists with current PID
	data, err := os.ReadFile(filepath.Join(dir, lockFileName))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("lock file should contain PID: %v", err)
	}
	if pid != os.Getpid() {
		t.Fatalf("expected PID %d, got %d", os.Getpid(), pid)
	}
}

func TestTryAcquireConsolidationLock_heldBySelf(t *testing.T) {
	dir := t.TempDir()
	// First acquire succeeds.
	_, err := TryAcquireConsolidationLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Second acquire should also succeed (same PID overwrites).
	prior, err := TryAcquireConsolidationLock(dir)
	if err != nil {
		t.Fatalf("expected re-acquire to succeed, got %v", err)
	}
	if prior <= 0 {
		t.Fatalf("expected prior > 0 on re-acquire, got %d", prior)
	}
}

func TestRollbackConsolidationLock_priorZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, lockFileName)

	// Create lock, then rollback with prior=0 (unlink).
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		t.Fatal(err)
	}
	RollbackConsolidationLock(dir, 0)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected lock file to be removed after rollback with prior=0")
	}
}

func TestRollbackConsolidationLock_priorNonZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, lockFileName)
	prior := time.Now().Add(-1 * time.Hour)

	// Create lock with past mtime.
	if err := os.WriteFile(path, []byte("0"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, prior, prior); err != nil {
		t.Fatal(err)
	}

	// Rollback should restore mtime.
	RollbackConsolidationLock(dir, prior.UnixMilli())

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.ModTime().UnixMilli() != prior.UnixMilli() {
		t.Fatalf("expected mtime %d, got %d", prior.UnixMilli(), fi.ModTime().UnixMilli())
	}
}

func TestListSessionsTouchedSince(t *testing.T) {
	dir := t.TempDir()

	// Create session files.
	now := time.Now()
	old := now.Add(-2 * time.Hour)
	sessions := map[string]time.Time{
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa": old,
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb": now,
	}

	for id, mtime := range sessions {
		path := filepath.Join(dir, id+".jsonl")
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}

	// Query for sessions touched in last hour.
	since := now.Add(-1 * time.Hour).UnixMilli()
	ids, err := listSessionIDsSince(dir, since)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 recent session, got %d: %v", len(ids), ids)
	}
	if ids[0] != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("expected recent session, got %s", ids[0])
	}
}

func TestListSessionsTouchedSince_nonUuidFile(t *testing.T) {
	dir := t.TempDir()

	// Non-UUID files should be skipped.
	if err := os.WriteFile(filepath.Join(dir, "not-a-uuid.jsonl"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	ids, err := listSessionIDsSince(dir, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected 0 sessions (non-UUID skipped), got %d", len(ids))
	}
}

func TestListSessionsTouchedSince_missingDir(t *testing.T) {
	ids, err := listSessionIDsSince("/nonexistent/dir", 0)
	if err != nil {
		t.Fatalf("expected nil for missing dir, got %v", err)
	}
	if ids != nil {
		t.Fatalf("expected nil result, got %v", ids)
	}
}

func TestRecordConsolidation(t *testing.T) {
	dir := t.TempDir()
	if err := RecordConsolidation(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(dir, lockFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("expected PID in lock file: %v", err)
	}
	if pid != os.Getpid() {
		t.Fatalf("expected PID %d, got %d", os.Getpid(), pid)
	}
}

func TestProjectDirForOriginalCwd(t *testing.T) {
	configHome := t.TempDir()
	projectPath := "/home/user/my-project"

	dir := ProjectDirForOriginalCwd(projectPath, configHome)
	expected := filepath.Join(configHome, "projects", projectPath)
	if dir != expected {
		t.Fatalf("expected %q, got %q", expected, dir)
	}
}

func TestSanitizePath_short(t *testing.T) {
	p := "/short/path"
	result := sanitizePath(p)
	if result != p {
		t.Fatalf("expected %q for short path, got %q", p, result)
	}
}

func TestSanitizePath_long(t *testing.T) {
	// Build a path > 240 chars.
	long := "/" + strings.Repeat("a", 250)
	result := sanitizePath(long)
	if len(result) <= 240 {
		t.Fatalf("expected truncated path with hash suffix > 240 chars, got %d", len(result))
	}
	if !strings.Contains(result, "-") {
		t.Fatalf("expected hash suffix in truncated path, got %q", result)
	}
	if !strings.HasPrefix(result, "/"+strings.Repeat("a", 239)) {
		t.Fatalf("expected first 240 chars of path as prefix (%d chars total), got %q", len(result), result)
	}
}

func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", true},
		{"00000000-0000-0000-0000-000000000000", true},
		{"", false},
		{"not-a-uuid", false},
		{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa", false},  // 35 chars
		{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaaa", false}, // 37 chars
		{"xxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", false},    // non-hex
	}
	for _, tc := range tests {
		got := looksLikeUUID(tc.input)
		if got != tc.want {
			t.Errorf("looksLikeUUID(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestLockFilePath(t *testing.T) {
	dir := "/some/dir"
	path := lockFilePath(dir)
	if path != filepath.Join(dir, lockFileName) {
		t.Fatalf("expected %q, got %q", filepath.Join(dir, lockFileName), path)
	}
}
