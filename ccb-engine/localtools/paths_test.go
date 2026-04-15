package localtools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUnderRoots_relativeUnderRoot(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	roots := []string{dir}
	got, err := ResolveUnderRoots(filepath.Join("pkg", "a.txt"), roots)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(sub, "a.txt")
	wantAbs, _ := filepath.Abs(want)
	if got != wantAbs {
		t.Fatalf("got %q want %q", got, wantAbs)
	}
}

func TestResolveUnderRoots_rejectsEscape(t *testing.T) {
	dir := t.TempDir()
	roots := []string{dir}
	_, err := ResolveUnderRoots("/etc/passwd", roots)
	if err == nil {
		t.Fatal("expected error outside roots")
	}
}

func TestReadFromJSON_lineSlice(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("a\nb\nc\nd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	roots := []string{dir}
	raw := []byte(`{"file_path":"f.txt","offset":2,"limit":2}`)
	s, isErr, err := ReadFromJSON(raw, roots, NewReadFileState(), nil)
	if err != nil || isErr {
		t.Fatalf("err=%v isErr=%v", err, isErr)
	}
	var got ReadTextOutput
	if err := json.Unmarshal([]byte(s), &got); err != nil {
		t.Fatal(err)
	}
	if got.Type != "text" || got.File.Content != "b\nc" || got.File.StartLine != 2 || got.File.NumLines != 2 {
		t.Fatalf("got %+v / raw %q", got, s)
	}
}
