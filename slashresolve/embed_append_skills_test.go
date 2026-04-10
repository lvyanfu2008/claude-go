package slashresolve

import (
	"strings"
	"testing"
)

func TestResolveRemember_argsSection(t *testing.T) {
	res, err := resolveRemember("promote to CLAUDE.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.UserText, "## Additional context from user") || !strings.Contains(res.UserText, "promote") {
		t.Fatal(res.UserText[len(res.UserText)-200:])
	}
}

func TestResolveStuck_argsSection(t *testing.T) {
	res, err := resolveStuck("pid 123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.UserText, "## User-provided context") || !strings.Contains(res.UserText, "pid 123") {
		t.Fatal(res.UserText[len(res.UserText)-200:])
	}
}

func TestResolveKeybindingsHelp_userRequest(t *testing.T) {
	res, err := resolveKeybindingsHelp("rebind ctrl+s")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.UserText, "## User Request") || !strings.Contains(res.UserText, "ctrl+s") {
		t.Fatal(res.UserText[len(res.UserText)-120:])
	}
}

func TestResolveDream_dynamicPaths(t *testing.T) {
	res, err := resolveDream("", &BundledResolveOptions{Cwd: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.UserText, "# Dream: Memory Consolidation (manual run)") {
		t.Fatal("missing manual prefix")
	}
	if !strings.Contains(res.UserText, "## Phase 1 — Orient") {
		t.Fatal("missing consolidation phases")
	}
	if !strings.Contains(res.UserText, "Memory directory:") {
		t.Fatal("missing memory dir line")
	}
}
