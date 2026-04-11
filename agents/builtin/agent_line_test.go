package builtin

import (
	"strings"
	"testing"
)

func TestFormatAgentLine_generalPurposeWildcard(t *testing.T) {
	s := FormatAgentLine(BuiltinAgent{
		AgentType: "general-purpose",
		WhenToUse: generalPurposeWhenToUse,
		Tools:     []string{"*"},
	})
	if !strings.Contains(s, "general-purpose:") || !strings.Contains(s, "not confident") || !strings.Contains(s, "(Tools: *)") {
		t.Fatalf("got %q", s)
	}
}
