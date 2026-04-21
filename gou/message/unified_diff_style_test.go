package message

import (
	"strings"
	"testing"

	"goc/gou/theme"
)

func TestFormatUnifiedDiffLineForDisplay_addDelMeta(t *testing.T) {
	p := theme.ActivePalette()
	s := FormatUnifiedDiffLineForDisplay("+foo", false, p)
	if !strings.Contains(s, "\x1b[") {
		t.Fatalf("expected ANSI from lipgloss: %q", s)
	}
	if FormatUnifiedDiffLineForDisplay(" plain", false, p) == "" {
		t.Fatal("empty")
	}
	_ = FormatUnifiedDiffLineForDisplay("--- a/x", false, p)
	_ = FormatUnifiedDiffLineForDisplay("+++ b/x", false, p)
	_ = FormatUnifiedDiffLineForDisplay("@@ -1,2 +1,2 @@", false, p)
}
