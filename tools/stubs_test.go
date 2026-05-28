package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBriefFromJSON_BasicMessage(t *testing.T) {
	out, isErr, err := BriefFromJSON([]byte(`{"message":"hello","status":"normal"}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if isErr {
		t.Fatalf("unexpected error flag")
	}
	data := decodeData(t, out)
	if data["message"] != "hello" {
		t.Fatalf("expected message 'hello', got %v", data["message"])
	}
	if _, ok := data["sentAt"]; !ok {
		t.Fatalf("missing sentAt")
	}
	if _, ok := data["attachments"]; ok {
		t.Fatalf("unexpected attachments field when none provided")
	}
}

func TestBriefFromJSON_ProactiveStatus(t *testing.T) {
	out, isErr, err := BriefFromJSON([]byte(`{"message":"done","status":"proactive"}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if isErr {
		t.Fatalf("unexpected error flag")
	}
	data := decodeData(t, out)
	if data["message"] != "done" {
		t.Fatalf("expected message 'done', got %v", data["message"])
	}
}

func TestBriefFromJSON_EmptyMessage(t *testing.T) {
	_, isErr, err := BriefFromJSON([]byte(`{"message":"","status":"normal"}`))
	if err == nil {
		t.Fatalf("expected error for empty message")
	}
	if !isErr {
		t.Fatalf("expected isErr=true")
	}
}

func TestBriefFromJSON_InvalidStatus(t *testing.T) {
	_, isErr, err := BriefFromJSON([]byte(`{"message":"hi","status":"bad"}`))
	if err == nil {
		t.Fatalf("expected error for invalid status")
	}
	if !isErr {
		t.Fatalf("expected isErr=true")
	}
}

func TestBriefFromJSON_NonexistentAttachment(t *testing.T) {
	_, isErr, err := BriefFromJSON([]byte(`{"message":"hi","status":"normal","attachments":["/nonexistent/file/path"]}`))
	if err == nil {
		t.Fatalf("expected error for nonexistent attachment")
	}
	if !isErr {
		t.Fatalf("expected isErr=true")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected 'does not exist' in error, got: %v", err)
	}
}

func TestBriefFromJSON_DirectoryAsAttachment(t *testing.T) {
	dir := t.TempDir()
	_, isErr, err := BriefFromJSON([]byte(`{"message":"hi","status":"normal","attachments":["` + dir + `"]}`))
	if err == nil {
		t.Fatalf("expected error for directory as attachment")
	}
	if !isErr {
		t.Fatalf("expected isErr=true")
	}
	if !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("expected 'not a regular file' in error, got: %v", err)
	}
}

func TestBriefFromJSON_ValidAttachment(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(file, []byte("content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	out, isErr, err := BriefFromJSON([]byte(`{"message":"hi","status":"normal","attachments":["` + file + `"]}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if isErr {
		t.Fatalf("unexpected error flag")
	}
	data := decodeData(t, out)
	atts, ok := data["attachments"].([]any)
	if !ok || len(atts) != 1 {
		t.Fatalf("expected 1 attachment, got %v", data["attachments"])
	}
	a0, _ := atts[0].(map[string]any)
	if a0["path"] != file {
		t.Fatalf("expected path %q, got %v", file, a0["path"])
	}
	if sz, _ := a0["size"].(float64); sz != 7 {
		t.Fatalf("expected size 7, got %v", a0["size"])
	}
	if a0["isImage"] != false {
		t.Fatalf("expected isImage=false for .txt, got %v", a0["isImage"])
	}
}

func TestBriefFromJSON_ImageAttachment(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "photo.png")
	if err := os.WriteFile(file, []byte("fake-image-data"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	out, isErr, err := BriefFromJSON([]byte(`{"message":"hi","status":"normal","attachments":["` + file + `"]}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if isErr {
		t.Fatalf("unexpected error flag")
	}
	data := decodeData(t, out)
	atts := data["attachments"].([]any)
	a0, _ := atts[0].(map[string]any)
	if a0["isImage"] != true {
		t.Fatalf("expected isImage=true for .png, got %v", a0["isImage"])
	}
}

func TestBriefFromJSON_MultipleAttachments(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	f2 := filepath.Join(dir, "b.jpg")
	os.WriteFile(f1, []byte("aaa"), 0o644)
	os.WriteFile(f2, []byte("bbb"), 0o644)

	out, isErr, err := BriefFromJSON([]byte(`{"message":"hi","status":"normal","attachments":["` + f1 + `","` + f2 + `"]}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if isErr {
		t.Fatalf("unexpected error flag")
	}
	data := decodeData(t, out)
	atts := data["attachments"].([]any)
	if len(atts) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(atts))
	}
	a0, _ := atts[0].(map[string]any)
	a1, _ := atts[1].(map[string]any)
	if a0["path"] != f1 {
		t.Fatalf("expected first path %q, got %v", f1, a0["path"])
	}
	if a0["isImage"] != false {
		t.Fatalf("expected isImage=false for .txt")
	}
	if a1["path"] != f2 {
		t.Fatalf("expected second path %q, got %v", f2, a1["path"])
	}
	if a1["isImage"] != true {
		t.Fatalf("expected isImage=true for .jpg")
	}
}

func TestBriefFromJSON_RelativePath(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(cwd)

	if err := os.WriteFile(filepath.Join(dir, "local.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, isErr, err := BriefFromJSON([]byte(`{"message":"hi","status":"normal","attachments":["local.txt"]}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if isErr {
		t.Fatalf("unexpected error flag")
	}
	data := decodeData(t, out)
	atts := data["attachments"].([]any)
	a0, _ := atts[0].(map[string]any)
	gotPath := a0["path"].(string)
	// Resolve the temp dir to handle macOS /var → /private/var symlink
	absDir, _ := filepath.EvalSymlinks(dir)
	wantPath := filepath.Join(absDir, "local.txt")
	if gotPath != wantPath {
		t.Fatalf("expected absolute path %q, got %v", wantPath, gotPath)
	}
}
