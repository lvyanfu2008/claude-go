package autodream

import (
	"testing"
)

func TestNewState(t *testing.T) {
	s := NewState()
	if s == nil {
		t.Fatal("expected non-nil state")
	}
}

func TestFilterMemoryPaths_empty(t *testing.T) {
	result := filterMemoryPaths(nil, "/mem")
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %v", result)
	}
	result = filterMemoryPaths([]string{}, "/mem")
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %v", result)
	}
}

func TestFilterMemoryPaths_excludesEntrypoint(t *testing.T) {
	memDir := "/tmp/memory"
	paths := []string{
		"/tmp/memory/MEMORY.md",
		"/tmp/memory/user_role.md",
	}
	result := filterMemoryPaths(paths, memDir)
	if len(result) != 1 {
		t.Fatalf("expected 1 path (excluded MEMORY.md), got %d: %v", len(result), result)
	}
	if result[0] != "/tmp/memory/user_role.md" {
		t.Fatalf("expected user_role.md, got %s", result[0])
	}
}

func TestFilterMemoryPaths_outsideDir(t *testing.T) {
	memDir := "/tmp/memory"
	paths := []string{
		"/tmp/memory/good.md",
		"/tmp/other/outside.md",
		"/tmp/memory/../outside2.md",
	}
	result := filterMemoryPaths(paths, memDir)
	if len(result) != 1 {
		t.Fatalf("expected 1 path (inside memory dir), got %d: %v", len(result), result)
	}
	if result[0] != "/tmp/memory/good.md" {
		t.Fatalf("expected good.md, got %s", result[0])
	}
}

func TestBuildExtraSection_basic(t *testing.T) {
	sessions := []string{"aaa", "bbb", "ccc"}
	extra := buildExtraSection(sessions)
	if extra == "" {
		t.Fatal("expected non-empty extra section")
	}
	if len(extra) < 100 {
		t.Fatal("expected extra section to contain tool constraints and session list")
	}
}

func TestBuildExtraSection_empty(t *testing.T) {
	extra := buildExtraSection(nil)
	if extra == "" {
		t.Fatal("expected non-empty even with nil sessions")
	}
}

func TestBuildExtraSection_sessionList(t *testing.T) {
	sessions := []string{"session-1", "session-2"}
	extra := buildExtraSection(sessions)
	// Should contain the count
	if len(extra) < 10 {
		t.Fatal("expected content")
	}
}

func TestBuildUserMessage(t *testing.T) {
	prompt := "test prompt content"
	uuid := func() string { return "test-uuid-123" }

	msg := buildUserMessage(prompt, uuid)
	if msg.Type != "user" {
		t.Fatalf("expected message type 'user', got %q", msg.Type)
	}
	if msg.UUID != "test-uuid-123" {
		t.Fatalf("expected UUID 'test-uuid-123', got %q", msg.UUID)
	}
	if len(msg.Message) == 0 {
		t.Fatal("expected non-empty message payload")
	}
}

func TestIsPathInMemDir_inside(t *testing.T) {
	input := []byte(`{"file_path": "/mem/dir/file.md"}`)
	if !isPathInMemDir(input, "/mem/dir") {
		t.Fatal("expected path inside mem dir to be allowed")
	}
}

func TestIsPathInMemDir_outside(t *testing.T) {
	input := []byte(`{"file_path": "/other/file.md"}`)
	if isPathInMemDir(input, "/mem/dir") {
		t.Fatal("expected path outside mem dir to be denied")
	}
}

func TestIsPathInMemDir_traversal(t *testing.T) {
	input := []byte(`{"file_path": "/mem/dir/../outside.md"}`)
	if isPathInMemDir(input, "/mem/dir") {
		t.Fatal("expected path traversal outside mem dir to be denied")
	}
}

func TestIsPathInMemDir_emptyFilePath(t *testing.T) {
	input := []byte(`{}`)
	if isPathInMemDir(input, "/mem/dir") {
		t.Fatal("expected empty file_path to be denied")
	}
}

func TestIsPathInMemDir_emptyMemDir(t *testing.T) {
	input := []byte(`{"file_path": "/some/file.md"}`)
	if isPathInMemDir(input, "") {
		t.Fatal("expected empty memDir to deny all paths")
	}
}

func TestExtractWrittenPaths_empty(t *testing.T) {
	paths := extractWrittenPaths(nil)
	if len(paths) != 0 {
		t.Fatalf("expected nil input to yield empty, got %v", paths)
	}
}

func TestExtractWrittenPaths_none(t *testing.T) {
	paths := extractWrittenPaths(nil)
	if len(paths) != 0 {
		t.Fatalf("expected no paths from nil messages, got %v", paths)
	}
}
