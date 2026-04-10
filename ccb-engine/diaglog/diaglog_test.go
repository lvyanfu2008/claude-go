package diaglog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLine_writesToExplicitFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "diag.txt")
	t.Setenv("CLAUDE_CODE_DIAG_LOG_FILE", tmp)
	t.Setenv("CCB_ENGINE_DIAG_TO_STDERR", "")
	Line("test %d", 42)
	b, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "test 42") {
		t.Fatalf("got %q", s)
	}
}
