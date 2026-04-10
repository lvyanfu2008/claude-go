package toolpolicy

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDenyReason_enforcementOff(t *testing.T) {
	t.Setenv("CCB_ENGINE_ENFORCE_ALLOWED_TOOLS", "")
	if DenyReason(nil, "Bash") != "" {
		t.Fatal("expected allow when enforcement off")
	}
}

func TestDenyReason_enforcementOn_missingContext(t *testing.T) {
	t.Setenv("CCB_ENGINE_ENFORCE_ALLOWED_TOOLS", "1")
	if got := DenyReason(nil, "Bash"); got == "" {
		t.Fatal("expected deny")
	}
}

func TestDenyReason_enforcementOn_allowed(t *testing.T) {
	t.Setenv("CCB_ENGINE_ENFORCE_ALLOWED_TOOLS", "1")
	raw, _ := json.Marshal(map[string]any{"allowedTools": []string{"Read", "Bash"}})
	if got := DenyReason(raw, "Bash"); got != "" {
		t.Fatalf("want allow, got %q", got)
	}
}

func TestDenyReason_enforcementOn_denied(t *testing.T) {
	t.Setenv("CCB_ENGINE_ENFORCE_ALLOWED_TOOLS", "1")
	raw, _ := json.Marshal(map[string]any{"allowedTools": []string{"Read"}})
	if got := DenyReason(raw, "Bash"); got == "" {
		t.Fatal("expected deny")
	}
}

func TestEnforcementEnabled(t *testing.T) {
	_ = os.Unsetenv("CCB_ENGINE_ENFORCE_ALLOWED_TOOLS")
	if EnforcementEnabled() {
		t.Fatal("expected false")
	}
	t.Setenv("CCB_ENGINE_ENFORCE_ALLOWED_TOOLS", "1")
	if !EnforcementEnabled() {
		t.Fatal("expected true")
	}
}
