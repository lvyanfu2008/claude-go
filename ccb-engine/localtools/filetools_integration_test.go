package localtools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadWriteEdit_sessionReadFileState(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.go")
	// No trailing newline so read content is a single logical line (avoids terminal empty-line join).
	if err := os.WriteFile(p, []byte("alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	roots := []string{dir}
	st := NewReadFileState()
	fp := "x.go"

	readIn := []byte(`{"file_path":"` + fp + `"}`)
	out, isErr, err := ReadFromJSON(readIn, roots, st, nil)
	if err != nil || isErr {
		t.Fatalf("read: %v isErr=%v %s", err, isErr, out)
	}
	var rd ReadTextOutput
	if err := json.Unmarshal([]byte(out), &rd); err != nil {
		t.Fatal(err)
	}
	if rd.File.Content != "alpha" {
		t.Fatalf("read content %q", rd.File.Content)
	}

	writeIn := []byte(`{"file_path":"` + fp + `","content":"beta"}`)
	_, isErr, err = WriteFromJSON(writeIn, roots, st)
	if err != nil || isErr {
		t.Fatalf("write: %v isErr=%v", err, isErr)
	}
	got, err := os.ReadFile(filepath.Join(dir, "x.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "beta" {
		t.Fatalf("disk %q", got)
	}

	// Read again so Edit sees fresh state (mirrors model turn).
	out2, _, _ := ReadFromJSON(readIn, roots, st, nil)
	if err := json.Unmarshal([]byte(out2), &rd); err != nil {
		t.Fatal(err)
	}
	editIn := []byte(`{"file_path":"` + fp + `","old_string":"beta","new_string":"gamma","replace_all":false}`)
	eout, isErr, err := EditFromJSON(editIn, roots, st, false)
	if err != nil || isErr {
		t.Fatalf("edit: %v isErr=%v %s", err, isErr, eout)
	}
	var ed EditOutput
	if err := json.Unmarshal([]byte(eout), &ed); err != nil {
		t.Fatal(err)
	}
	if ed.NewString != "gamma" {
		t.Fatalf("edit output %+v", ed)
	}
	got2, _ := os.ReadFile(filepath.Join(dir, "x.go"))
	if string(got2) != "gamma" {
		t.Fatalf("after edit %q", got2)
	}
}
