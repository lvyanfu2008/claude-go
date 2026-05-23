package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPluginHooksTable(t *testing.T) {
	tmp := t.TempDir()

	os.MkdirAll(filepath.Join(tmp, "hooks"), 0755)
	hooksJSON := `{
		"hooks": {
			"SessionStart": [{
				"matcher": "startup|clear|compact",
				"hooks": [{"type": "command", "command": "echo test"}]
			}]
		}
	}`
	os.WriteFile(filepath.Join(tmp, "hooks", "hooks.json"), []byte(hooksJSON), 0644)

	table, err := LoadPluginHooksTable(tmp)
	if err != nil {
		t.Fatalf("LoadPluginHooksTable: %v", err)
	}
	if len(table) == 0 {
		t.Fatal("expected non-empty hooks table")
	}
	if _, ok := table["SessionStart"]; !ok {
		t.Error("expected SessionStart hook")
	}
}

func TestLoadPluginHooksTable_NoHooks(t *testing.T) {
	tmp := t.TempDir()
	table, err := LoadPluginHooksTable(tmp)
	if err != nil {
		t.Fatalf("LoadPluginHooksTable: %v", err)
	}
	if table != nil {
		t.Error("expected nil table when no hooks.json")
	}
}

func TestLoadPluginHooksTable_ResolvesPluginRoot(t *testing.T) {
	tmp := t.TempDir()

	os.MkdirAll(filepath.Join(tmp, "hooks"), 0755)
	hooksJSON := `{
		"hooks": {
			"SessionStart": [{
				"matcher": "startup",
				"hooks": [{"type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/hooks/session-start"}]
			}]
		}
	}`
	os.WriteFile(filepath.Join(tmp, "hooks", "hooks.json"), []byte(hooksJSON), 0644)

	table, err := LoadPluginHooksTable(tmp)
	if err != nil {
		t.Fatalf("LoadPluginHooksTable: %v", err)
	}
	// Verify ${CLAUDE_PLUGIN_ROOT} was resolved
	sessionStart := table["SessionStart"]
	hooksData := string(sessionStart[0].Hooks[0])
	if strings.Contains(hooksData, "${CLAUDE_PLUGIN_ROOT}") {
		t.Error("CLAUDE_PLUGIN_ROOT was not resolved in hook command")
	}
	if !strings.Contains(hooksData, tmp) {
		t.Errorf("expected plugin dir %q in resolved hook, got %q", tmp, hooksData)
	}
}
