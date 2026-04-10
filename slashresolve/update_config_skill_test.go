package slashresolve

import (
	"strings"
	"testing"

	"goc/types"
)

func TestResolveUpdateConfig_fullBody(t *testing.T) {
	res, err := resolveUpdateConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != types.SlashResolveBundledEmbed {
		t.Fatalf("source: %q", res.Source)
	}
	if !strings.Contains(res.UserText, "# Update Config Skill") {
		t.Fatal("missing title")
	}
	if !strings.Contains(res.UserText, "## Full Settings JSON Schema") {
		t.Fatal("missing generated schema section (keep bundleddata/update-config.md in sync with TS)")
	}
}

func TestResolveUpdateConfig_userRequestSuffix(t *testing.T) {
	res, err := resolveUpdateConfig("allow npm")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.UserText, "## User Request") || !strings.Contains(res.UserText, "allow npm") {
		t.Fatalf("expected User Request section: tail %q", res.UserText[len(res.UserText)-min(200, len(res.UserText)):])
	}
}

func TestResolveUpdateConfig_hooksOnly(t *testing.T) {
	res, err := resolveUpdateConfig("[hooks-only] add formatter")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.UserText, "## Hooks Configuration") {
		t.Fatal("expected hooks doc body")
	}
	if !strings.Contains(res.UserText, "## User Request") || !strings.Contains(res.UserText, "add formatter") {
		t.Fatal("expected task suffix")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
