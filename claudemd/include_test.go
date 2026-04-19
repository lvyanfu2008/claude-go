package claudemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractIncludePaths(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		basePath  string // This should be a file path, not a directory
		expected  []string
	}{
		{
			name:      "simple relative path",
			content:   "See @./file.md",
			basePath:  "/test/main.md",
			expected:  []string{"/test/file.md"},
		},
		{
			name:      "relative path without ./",
			content:   "See @file.md",
			basePath:  "/test/main.md",
			expected:  []string{"/test/file.md"},
		},
		{
			name:      "home directory path",
			content:   "See @~/documents/file.md",
			basePath:  "/test/main.md",
			expected:  []string{filepath.Join(os.Getenv("HOME"), "documents/file.md")},
		},
		{
			name:      "absolute path",
			content:   "See @/absolute/path/file.md",
			basePath:  "/test/main.md",
			expected:  []string{"/absolute/path/file.md"},
		},
		{
			name:      "multiple includes",
			content:   "See @./file1.md and @./file2.md",
			basePath:  "/test/main.md",
			expected:  []string{"/test/file1.md", "/test/file2.md"},
		},
		{
			name:      "include with fragment",
			content:   "See @./file.md#section",
			basePath:  "/test/main.md",
			expected:  []string{"/test/file.md"},
		},
		{
			name:      "include with escaped space",
			content:   "See @./file\\ with\\ spaces.md",
			basePath:  "/test/main.md",
			expected:  []string{"/test/file with spaces.md"},
		},
		{
			name:      "ignore in code block",
			content:   "```\n@./ignored.md\n```",
			basePath:  "/test/main.md",
			expected:  []string{},
		},
		{
			name:      "ignore in inline code",
			content:   "This is `@./ignored.md` inline code",
			basePath:  "/test/main.md",
			expected:  []string{},
		},
		{
			name:      "include in HTML comment residue",
			content:   "<!-- comment --> @./file.md",
			basePath:  "/test/main.md",
			expected:  []string{"/test/file.md"},
		},
		{
			name:      "invalid path starting with @",
			content:   "See @@@invalid",
			basePath:  "/test/main.md",
			expected:  []string{},
		},
		{
			name:      "invalid path starting with special chars",
			content:   "See @#invalid",
			basePath:  "/test/main.md",
			expected:  []string{},
		},
		{
			name:      "empty path",
			content:   "See @",
			basePath:  "/test/main.md",
			expected:  []string{},
		},
		{
			name:      "path with only spaces",
			content:   "See @   ",
			basePath:  "/test/main.md",
			expected:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := ExtractIncludePathsFromMarkdown([]byte(tt.content), tt.basePath)

			// For home directory paths, we need to handle expansion
			var actual []string
			for _, path := range paths {
				actual = append(actual, filepath.Clean(path))
			}

			// Check if expected and actual match
			if len(tt.expected) != len(actual) {
				t.Errorf("expected %d paths, got %d: %v", len(tt.expected), len(actual), actual)
				return
			}

			for i := range tt.expected {
				expected := filepath.Clean(tt.expected[i])
				if expected != actual[i] {
					t.Errorf("path %d: expected %q, got %q", i, expected, actual[i])
				}
			}
		})
	}
}

func TestIncludeInMemoryFiles(t *testing.T) {
	dir := t.TempDir()

	// Create main CLAUDE.md with includes
	mainContent := "# Main File\nInclude external: @./external.md\nInclude nested: @./nested/nested.md\n"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create external.md
	externalContent := "# External File\nThis is included from main."
	if err := os.WriteFile(filepath.Join(dir, "external.md"), []byte(externalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create nested directory and file
	nestedDir := filepath.Join(dir, "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	nestedContent := "# Nested File\nThis is in a nested directory."
	if err := os.WriteFile(filepath.Join(nestedDir, "nested.md"), []byte(nestedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test BuildClaudeMdString with includes
	t.Setenv("CLAUDE_CODE_DISABLE_USER_MEMORY", "1")
	t.Setenv("CLAUDE_CODE_SIMPLE", "")

	out := BuildClaudeMdString(LoadOptions{OriginalCwd: dir})

	// Check that included content appears
	if !strings.Contains(out, "External File") {
		t.Error("Missing included external file content")
	}
	if !strings.Contains(out, "Nested File") {
		t.Error("Missing included nested file content")
	}

	// Check that main file content is present
	if !strings.Contains(out, "Main File") {
		t.Error("Missing main file content")
	}
}

func TestIncludeCircularReference(t *testing.T) {
	dir := t.TempDir()

	// Create file1.md that includes file2.md
	file1Content := "# File 1\nInclude @./file2.md"
	if err := os.WriteFile(filepath.Join(dir, "file1.md"), []byte(file1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create file2.md that includes file1.md (circular)
	file2Content := "# File 2\nInclude @./file1.md"
	if err := os.WriteFile(filepath.Join(dir, "file2.md"), []byte(file2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create CLAUDE.md that includes file1.md
	claudeContent := "# CLAUDE.md\nInclude @./file1.md"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(claudeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test that circular reference doesn't cause infinite loop
	t.Setenv("CLAUDE_CODE_DISABLE_USER_MEMORY", "1")
	t.Setenv("CLAUDE_CODE_SIMPLE", "")

	out := BuildClaudeMdString(LoadOptions{OriginalCwd: dir})

	// Should contain content from both files
	if !strings.Contains(out, "File 1") {
		t.Error("Missing file1 content")
	}
	if !strings.Contains(out, "File 2") {
		t.Error("Missing file2 content")
	}
}

func TestIncludeNonExistentFile(t *testing.T) {
	dir := t.TempDir()

	// Create CLAUDE.md with include to non-existent file
	claudeContent := "# CLAUDE.md\nInclude @./nonexistent.md"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(claudeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test that non-existent file is silently ignored
	t.Setenv("CLAUDE_CODE_DISABLE_USER_MEMORY", "1")
	t.Setenv("CLAUDE_CODE_SIMPLE", "")

	out := BuildClaudeMdString(LoadOptions{OriginalCwd: dir})

	// Should still contain main content
	if !strings.Contains(out, "CLAUDE.md") {
		t.Error("Missing main file content")
	}
	// Should not crash or error
}