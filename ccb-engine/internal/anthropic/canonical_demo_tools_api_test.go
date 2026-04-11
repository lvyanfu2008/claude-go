package anthropic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type toolsAPITool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type toolsAPIFile struct {
	Tools []toolsAPITool `json:"tools"`
}

// TestGouParityToolsIntersectToolsAPIExport ensures GouParityToolList stays in sync with the
// TS export (commands/data/tools_api.json). [GouParityToolsJSON] uses export descriptions; this list
// still intersects export names (plus echo_stub / optional DiscoverSkills).
func TestGouParityToolsIntersectToolsAPIExport(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..")
	apiPath := filepath.Join(root, "commands", "data", "tools_api.json")
	raw, err := os.ReadFile(apiPath)
	if err != nil {
		t.Skipf("tools_api.json not found at %s: %v", apiPath, err)
	}
	var doc toolsAPIFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	export := make(map[string]struct{}, len(doc.Tools))
	for _, e := range doc.Tools {
		n := strings.TrimSpace(e.Name)
		if n != "" {
			export[n] = struct{}{}
		}
	}
	var intersect int
	var missing []string
	for _, def := range GouParityToolList() {
		n := strings.TrimSpace(def.Name)
		if n == "" {
			continue
		}
		if _, ok := export[n]; ok {
			intersect++
		} else {
			// Optional head tools (DiscoverSkills) and stubs may be absent from a given export.
			if n != "echo_stub" && n != discoverSkillsToolNameOrEmpty() {
				missing = append(missing, n)
			}
		}
	}
	if intersect < 12 {
		t.Fatalf("expected at least 12 parity tools to appear in tools_api.json export, got %d (missing non-stub: %v)", intersect, missing)
	}
}

func discoverSkillsToolNameOrEmpty() string {
	return strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME"))
}
