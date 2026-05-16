package processuserinput

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractAtMentionedFiles_Unquoted(t *testing.T) {
	result := extractAtMentionedFiles("check @main.go for bugs")
	if len(result) != 1 || result[0] != "main.go" {
		t.Errorf("expected [main.go], got %v", result)
	}
}

func TestExtractAtMentionedFiles_Multiple(t *testing.T) {
	result := extractAtMentionedFiles("compare @foo.go and @bar.go")
	if len(result) != 2 || result[0] != "foo.go" || result[1] != "bar.go" {
		t.Errorf("expected [foo.go bar.go], got %v", result)
	}
}

func TestExtractAtMentionedFiles_QuotedWithSpaces(t *testing.T) {
	result := extractAtMentionedFiles(`check @"my file.txt" please`)
	if len(result) != 1 || result[0] != "my file.txt" {
		t.Errorf("expected [my file.txt], got %v", result)
	}
}

func TestExtractAtMentionedFiles_Deduplicate(t *testing.T) {
	result := extractAtMentionedFiles("@main.go @main.go")
	if len(result) != 1 || result[0] != "main.go" {
		t.Errorf("expected deduplicated [main.go], got %v", result)
	}
}

func TestExtractAtMentionedFiles_SkipsAgentQuoted(t *testing.T) {
	result := extractAtMentionedFiles(`@"code-reviewer (agent)" review this`)
	for _, f := range result {
		if f == "code-reviewer (agent)" {
			t.Errorf("should skip agent mention, got %v", result)
		}
	}
}

func TestExtractAtMentionedFiles_LineRange(t *testing.T) {
	result := extractAtMentionedFiles("check @main.go#L10-20")
	if len(result) != 1 || result[0] != "main.go#L10-20" {
		t.Errorf("expected [main.go#L10-20], got %v", result)
	}
}

func TestExtractAtMentionedFiles_Empty(t *testing.T) {
	result := extractAtMentionedFiles("no mentions here")
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestParseAtMentionedFileLines_Plain(t *testing.T) {
	filename, start, end := parseAtMentionedFileLines("main.go")
	if filename != "main.go" || start != 0 || end != 0 {
		t.Errorf("expected (main.go, 0, 0), got (%s, %d, %d)", filename, start, end)
	}
}

func TestParseAtMentionedFileLines_SingleLine(t *testing.T) {
	filename, start, end := parseAtMentionedFileLines("main.go#L10")
	if filename != "main.go" || start != 10 || end != 10 {
		t.Errorf("expected (main.go, 10, 10), got (%s, %d, %d)", filename, start, end)
	}
}

func TestParseAtMentionedFileLines_Range(t *testing.T) {
	filename, start, end := parseAtMentionedFileLines("main.go#L10-20")
	if filename != "main.go" || start != 10 || end != 20 {
		t.Errorf("expected (main.go, 10, 20), got (%s, %d, %d)", filename, start, end)
	}
}

func TestParseAtMentionedFileLines_Heading(t *testing.T) {
	filename, _, _ := parseAtMentionedFileLines("main.go#heading")
	if filename != "main.go" {
		t.Errorf("expected main.go (heading stripped), got %s", filename)
	}
}

func TestParseAtMentionedFileLines_LineThenHeading(t *testing.T) {
	filename, start, end := parseAtMentionedFileLines("main.go#L10-20#section")
	if filename != "main.go" || start != 10 || end != 20 {
		t.Errorf("expected (main.go, 10, 20), got (%s, %d, %d)", filename, start, end)
	}
}

func TestResolveAtMentionedFiles_NoMatches(t *testing.T) {
	msgs, err := resolveAtMentionedFiles(t.Context(), "no matches", ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected no messages, got %d", len(msgs))
	}
}

func TestResolveAtMentionedFiles_FileExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(f, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatal(err)
	}
	msgs, err := resolveAtMentionedFiles(t.Context(), "check @hello.txt", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Type != "attachment" {
		t.Errorf("expected attachment type, got %q", msgs[0].Type)
	}
}

func TestResolveAtMentionedFiles_FileNotExists(t *testing.T) {
	msgs, err := resolveAtMentionedFiles(t.Context(), "check @nonexistent.txt", ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected no messages for missing file, got %d", len(msgs))
	}
}

func TestResolveAtMentionedFiles_Directory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0755)
	msgs, err := resolveAtMentionedFiles(t.Context(), "check @sub", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message for directory, got %d", len(msgs))
	}
	if msgs[0].Type != "attachment" {
		t.Errorf("expected attachment type, got %q", msgs[0].Type)
	}
}

func TestReadFileLines_Truncated(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "big.txt")
	var lines []string
	for i := 0; i < 2500; i++ {
		lines = append(lines, "line")
	}
	os.WriteFile(f, []byte(joinLines(lines)), 0644)

	content, err := readFileLines(f, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := countLines(content)
	if got > atMentionMaxLines {
		t.Errorf("expected at most %d lines, got %d", atMentionMaxLines, got)
	}
}

func TestReadFileLines_LineRange(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "lines.txt")
	os.WriteFile(f, []byte("a\nb\nc\nd\ne\n"), 0644)

	content, err := readFileLines(f, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if content != "b\nc\nd" {
		t.Errorf("expected lines 2-4, got %q", content)
	}
}

func TestNewDefaultGetAttachmentMessages(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)

	fn := NewDefaultGetAttachmentMessages(dir)
	msgs, err := fn(t.Context(), "read @test.txt", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
}

func joinLines(lines []string) string {
	s := ""
	for _, l := range lines {
		s += l + "\n"
	}
	return s
}

func countLines(s string) int {
	n := 0
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n
}
