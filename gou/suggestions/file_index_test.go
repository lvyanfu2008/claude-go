package suggestions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScoredSearch_ExactMatch(t *testing.T) {
	entries := []string{
		"src/components/Button.tsx",
		"src/hooks/useAuth.ts",
		"src/utils/format.ts",
		"README.md",
	}
	results := scoredSearch(entries, "src/components/Button.tsx", 15)
	if len(results) == 0 {
		t.Fatal("expected at least one result for exact match")
	}
	if results[0].Label != "src/components/Button.tsx" {
		t.Errorf("expected exact match first, got %q", results[0].Label)
	}
}

func TestScoredSearch_PrefixMatch(t *testing.T) {
	entries := []string{
		"src/components/Button.tsx",
		"src/components/Modal.tsx",
		"README.md",
	}
	results := scoredSearch(entries, "src/components", 15)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 prefix matches, got %d", len(results))
	}
}

func TestScoredSearch_SubsequenceMatch(t *testing.T) {
	entries := []string{
		"src/components/Button.tsx",
		"src/hooks/useAuth.ts",
		"README.md",
	}
	results := scoredSearch(entries, "btn", 15)
	if len(results) == 0 {
		t.Fatal("expected at least one subsequence match for 'btn'")
	}
	if results[0].Label != "src/components/Button.tsx" {
		t.Errorf("expected Button.tsx first, got %q", results[0].Label)
	}
}

func TestScoredSearch_EmptyQuery(t *testing.T) {
	entries := []string{"a.go", "b.go", "c.go"}
	results := scoredSearch(entries, "", 15)
	if len(results) != 3 {
		t.Errorf("expected 3 results for empty query, got %d", len(results))
	}
}

func TestGetTopLevelPaths_DirectoriesFirst(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "src", "lib"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "src", "utils.go"), []byte("package src"), 0644)

	fi := &FileIndex{cwd: tmpDir}
	results := fi.GetTopLevelPaths("src")
	if len(results) == 0 {
		t.Fatal("expected results for src directory")
	}
	if results[0].Type != SuggestionTypeDirectory {
		t.Errorf("expected first result to be directory, got type %v label %q", results[0].Type, results[0].Label)
	}
	if results[0].Label != "src/lib/" {
		t.Errorf("expected 'src/lib/', got %q", results[0].Label)
	}
}

func TestGetTopLevelPaths_HidesDotFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("secret"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("hello"), 0644)

	fi := &FileIndex{cwd: tmpDir}
	results := fi.GetTopLevelPaths(".")
	for _, r := range results {
		if filepath.Base(r.Label)[0] == '.' {
			t.Errorf("expected no dot-files, got %q", r.Label)
		}
	}
}
