package apiparity

import (
	"testing"
)

// Golden digests for the default gou-demo API slice (no DiscoverSkills, no MCP merge, empty skill listing).
// System is built via [querycontext.FetchSystemPromptParts] (same as gou-demo); user_context_reminder is the prepend-only block.
// When tools or system assembly intentionally changes, update these constants and document in the PR.
const (
	goldenToolsSHA256Default        = "42639e2db923ed5c66ee4bfbbbb60d05833f5bc72a25f3168dc5cb2bfe4c3353"
	goldenSystemSHA256Default       = "c3b08e0cd428cfc09570de38fe0405ac0055a165bb3ef0d95656a6d701757467"
	goldenUserContextReminderSHA256 = "83ae35d35803cc9ec3e35280018a91b78af1e71190e68001e395c5bb7ca15f7a"
)

func TestGouDemo_snapshotGolden_defaultSlice(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME", "")
	t.Setenv("GOU_DEMO_NON_INTERACTIVE", "")
	t.Setenv("FEATURE_MCP_SKILLS", "")
	t.Setenv("CLAUDE_CODE_LANGUAGE", "")
	t.Setenv("CLAUDE_CODE_OUTPUT_STYLE_NAME", "")
	t.Setenv("CLAUDE_CODE_OUTPUT_STYLE_PROMPT", "")
	t.Setenv("CLAUDE_CODE_REMOTE", "1")
	t.Setenv("CLAUDE_CODE_OVERRIDE_DATE", "2030-06-15")

	out, err := GouDemo(SnapshotInput{
		Cwd:            "/tmp/gou-parity-golden",
		MainLoopModel:  DefaultMainLoopModel,
		ParityGOOS:     "linux",
		ParityGOARCH:   "amd64",
		LoadedCommands: nil,
		MCPCommands:    nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ToolsSHA256 != goldenToolsSHA256Default {
		t.Fatalf("tools_sha256\ngot  %s\nwant %s", out.ToolsSHA256, goldenToolsSHA256Default)
	}
	if out.SystemSHA256 != goldenSystemSHA256Default {
		t.Fatalf("system_sha256\ngot  %s\nwant %s", out.SystemSHA256, goldenSystemSHA256Default)
	}
	if out.UserContextReminderSHA256 != goldenUserContextReminderSHA256 {
		t.Fatalf("user_context_reminder_sha256\ngot  %s\nwant %s", out.UserContextReminderSHA256, goldenUserContextReminderSHA256)
	}
}

func TestGouDemo_discoverSkills_changesToolsDigest(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME", "DiscoverSkills")
	t.Setenv("GOU_DEMO_NON_INTERACTIVE", "")
	t.Setenv("FEATURE_MCP_SKILLS", "")
	t.Setenv("CLAUDE_CODE_LANGUAGE", "")
	t.Setenv("CLAUDE_CODE_OUTPUT_STYLE_NAME", "")
	t.Setenv("CLAUDE_CODE_OUTPUT_STYLE_PROMPT", "")
	t.Setenv("CLAUDE_CODE_REMOTE", "1")
	t.Setenv("CLAUDE_CODE_OVERRIDE_DATE", "2030-06-15")

	out, err := GouDemo(SnapshotInput{
		Cwd:           "/tmp/gou-parity-golden",
		MainLoopModel: DefaultMainLoopModel,
		ParityGOOS:    "linux",
		ParityGOARCH:  "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ToolsSHA256 == goldenToolsSHA256Default {
		t.Fatal("expected tools digest to differ when DiscoverSkills is registered")
	}
	found := false
	for _, n := range out.ToolNames {
		if n == "DiscoverSkills" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("tool_names %v missing DiscoverSkills", out.ToolNames)
	}
}
