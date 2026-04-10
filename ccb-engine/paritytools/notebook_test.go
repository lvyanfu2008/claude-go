package paritytools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNotebookEdit_replace(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "nb.ipynb")
	raw := `{
  "cells": [
    {"cell_type": "code", "metadata": {"id": "c1"}, "source": ["print(1)\n"]}
  ],
  "metadata": {"kernelspec": {"language": "python"}},
  "nbformat": 4,
  "nbformat_minor": 5
}`
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	roots := []string{dir}
	in := `{"notebook_path": "` + p + `", "cell_id": "c1", "new_source": "print(2)", "edit_mode": "replace"}`
	out, isErr, err := NotebookEditFromJSON([]byte(in), roots)
	if err != nil || isErr {
		t.Fatalf("replace: err=%v isErr=%v out=%s", err, isErr, out)
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var nb map[string]any
	if err := json.Unmarshal(b, &nb); err != nil {
		t.Fatal(err)
	}
	cells := nb["cells"].([]any)
	cell := cells[0].(map[string]any)
	src := cell["source"].([]any)
	if len(src) == 0 || src[0] != "print(2)\n" {
		t.Fatalf("unexpected source: %v", src)
	}
}
