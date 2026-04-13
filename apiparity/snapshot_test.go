package apiparity

import (
	"testing"

	"goc/modelenv"
)

// Golden digests for the default gou-demo API slice (no DiscoverSkills, no MCP merge, empty skill listing).
// System is built via [querycontext.FetchSystemPromptParts] (same as gou-demo); user_context_reminder is the prepend-only block.
// Tools include [toolpool.PatchAgentToolDescriptionWithBuiltins] (inline agent whenToUse lines; TS getPrompt non-attachment branch).
// Embedded tools_api.json includes AskUserQuestion (TS default getTools); system digest reflects enabled-tool names.
// Omit AskUserQuestion the TS way: FEATURE_KAIROS or FEATURE_KAIROS_CHANNELS plus non-empty CLAUDE_CODE_GO_ALLOWED_CHANNELS.
// When tools or system assembly intentionally changes, update these constants and document in the PR.
const (
	goldenToolsSHA256Default        = "c24fd5e9b02cf80fd35557776a4bac239bce0ac2175b9042d7999e356cb5be9e"
	goldenSystemSHA256Default       = "6db65686769ee3cb0de2a0c9ca909741035353a3e3625ebfb906c98a953a9f33"
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
	t.Setenv("CLAUDE_CODE_GO_OS_VERSION", "Linux test")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "1")
	t.Setenv("CLAUDE_CODE_SYSTEM_PROMPT_MODEL_ID", "")
	for _, k := range modelenv.LookupKeys {
		t.Setenv(k, "")
	}

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

func TestGouDemo_kairosChannelsOmitsAskUserQuestion(t *testing.T) {
	t.Setenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME", "")
	t.Setenv("GOU_DEMO_NON_INTERACTIVE", "")
	t.Setenv("FEATURE_MCP_SKILLS", "")
	t.Setenv("CLAUDE_CODE_LANGUAGE", "")
	t.Setenv("CLAUDE_CODE_OUTPUT_STYLE_NAME", "")
	t.Setenv("CLAUDE_CODE_OUTPUT_STYLE_PROMPT", "")
	t.Setenv("CLAUDE_CODE_REMOTE", "1")
	t.Setenv("CLAUDE_CODE_OVERRIDE_DATE", "2030-06-15")
	t.Setenv("CLAUDE_CODE_GO_OS_VERSION", "Linux test")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "1")
	t.Setenv("CLAUDE_CODE_SYSTEM_PROMPT_MODEL_ID", "")
	t.Setenv("FEATURE_KAIROS_CHANNELS", "1")
	t.Setenv("CLAUDE_CODE_GO_ALLOWED_CHANNELS", "discord")
	for _, k := range modelenv.LookupKeys {
		t.Setenv(k, "")
	}

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
	for _, n := range out.ToolNames {
		if n == "AskUserQuestion" {
			t.Fatalf("tool_names should omit AskUserQuestion under KAIROS_CHANNELS + channels: %v", out.ToolNames)
		}
	}
	if out.ToolsSHA256 == goldenToolsSHA256Default {
		t.Fatal("expected tools digest to differ when AskUserQuestion is filtered")
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
	t.Setenv("CLAUDE_CODE_GO_OS_VERSION", "Linux test")
	t.Setenv("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS", "1")
	t.Setenv("CLAUDE_CODE_SYSTEM_PROMPT_MODEL_ID", "")
	for _, k := range modelenv.LookupKeys {
		t.Setenv(k, "")
	}

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
