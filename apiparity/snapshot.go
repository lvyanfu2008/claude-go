// Package apiparity builds deterministic snapshots of gou-demo API fields (tools[], system)
// without importing goc/commands from goc/ccb-engine/skilltools (avoids an import cycle).
package apiparity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"slices"
	"strings"

	"goc/ccb-engine/settingsfile"
	"goc/ccb-engine/skilltools"
	"goc/commands"
	"goc/querycontext"
	"goc/types"
)

// DefaultMainLoopModel matches [pui.DefaultMainLoopModelForDemo] / gou-demo default.
const DefaultMainLoopModel = "claude-sonnet-4-20250514"

// SnapshotInput mirrors gou-demo inputs used to build system + tools[] (phase 2 slice).
type SnapshotInput struct {
	Cwd            string
	MainLoopModel  string
	LoadedCommands []types.Command
	MCPCommands    []types.Command
	ParityGOOS     string
	ParityGOARCH   string
	// ExtraClaudeMdRoots matches [querycontext.FetchOpts.ExtraClaudeMdRoots] (with CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD).
	ExtraClaudeMdRoots []string
}

// SnapshotOutput is JSON-serializable for diffing and sha256 checks (plan milestone C).
type SnapshotOutput struct {
	ToolsJSON    json.RawMessage `json:"tools"`
	System       string          `json:"system"`
	ToolNames    []string        `json:"tool_names"`
	ToolsSHA256  string          `json:"tools_sha256"`
	SystemSHA256 string          `json:"system_sha256"`
	// UserContextReminder is the string gou-demo prepends as an extra user message (FormatUserContextReminder).
	UserContextReminder       string `json:"user_context_reminder,omitempty"`
	UserContextReminderSHA256 string `json:"user_context_reminder_sha256,omitempty"`
	Cwd                       string `json:"cwd"`
	MainLoopModel             string `json:"main_loop_model"`
	DiscoverSkills            string `json:"discover_skills_tool_name,omitempty"`
}

func envTruthy(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// GouDemo builds the same tools JSON and system string gou-demo sends on a model turn
// (subset of full TS getSystemPrompt). Reads CLAUDE_CODE_* / GOU_DEMO_NON_INTERACTIVE from the environment.
func GouDemo(in SnapshotInput) (SnapshotOutput, error) {
	cwd := strings.TrimSpace(in.Cwd)
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	model := strings.TrimSpace(in.MainLoopModel)
	if model == "" {
		model = DefaultMainLoopModel
	}

	toolsRaw, err := skilltools.GouDemoParityToolsJSON()
	if err != nil {
		return SnapshotOutput{}, err
	}

	var defs []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(toolsRaw, &defs); err != nil {
		return SnapshotOutput{}, err
	}
	names := make([]string, 0, len(defs))
	for _, d := range defs {
		n := strings.TrimSpace(d.Name)
		if n != "" {
			names = append(names, n)
		}
	}
	enabled := commands.EnabledToolNames(names)

	loaded := in.LoadedCommands
	mcp := in.MCPCommands
	listing := commands.SkillListingCommandsForAPI(loaded, mcp, commands.FeatureMcpSkills())
	discover := skilltools.DiscoverSkillsToolNameFromEnv()

	projRoot, errRoot := settingsfile.FindClaudeProjectRoot(cwd)
	if errRoot != nil {
		return SnapshotOutput{}, errRoot
	}
	locLang, locStyleKey, errLoc := settingsfile.MergeGouDemoLocalePrefs(projRoot, false)
	if errLoc != nil {
		return SnapshotOutput{}, errLoc
	}
	lang := strings.TrimSpace(os.Getenv("CLAUDE_CODE_LANGUAGE"))
	if lang == "" {
		lang = locLang
	}
	outName, outPrompt := commands.ResolveGouDemoOutputStyle(
		os.Getenv("CLAUDE_CODE_OUTPUT_STYLE_NAME"),
		os.Getenv("CLAUDE_CODE_OUTPUT_STYLE_PROMPT"),
		locStyleKey,
	)

	gouOpts := commands.GouDemoSystemOpts{
		EnabledToolNames:       enabled,
		SkillToolCommands:      listing,
		ModelID:                model,
		Cwd:                    cwd,
		Language:               lang,
		OutputStyleName:        outName,
		OutputStylePrompt:      outPrompt,
		DiscoverSkillsToolName: discover,
		NonInteractiveSession:  envTruthy("GOU_DEMO_NON_INTERACTIVE"),
		ParityGOOS:             in.ParityGOOS,
		ParityGOARCH:           in.ParityGOARCH,
		AdditionalWorkingDirs:  slices.Clone(in.ExtraClaudeMdRoots),
		SkipPromptGitDetect:    true,
	}
	extraRoots := slices.Clone(in.ExtraClaudeMdRoots)
	partsRes, errFetch := querycontext.FetchSystemPromptParts(context.Background(), querycontext.FetchOpts{
		Gou:                  gouOpts,
		ExtraClaudeMdRoots:   extraRoots,
		SessionStartSource:   "startup",
		HooksSessionID:       "apiparity",
		HooksTranscriptPath:  "",
	})
	var sys string
	var reminder string
	if errFetch != nil {
		sys = commands.BuildGouDemoSystemPrompt(gouOpts)
	} else {
		reminder = querycontext.FormatUserContextReminder(partsRes.UserContext)
		base := slices.Clone(partsRes.DefaultSystemPrompt)
		fullParts := querycontext.AppendSystemContextParts(base, partsRes.SystemContext)
		sys = strings.Join(fullParts, "\n\n")
	}

	th := sha256.Sum256(toolsRaw)
	sh := sha256.Sum256([]byte(sys))
	out := SnapshotOutput{
		ToolsJSON:      toolsRaw,
		System:         sys,
		ToolNames:      names,
		ToolsSHA256:    hex.EncodeToString(th[:]),
		SystemSHA256:   hex.EncodeToString(sh[:]),
		Cwd:            cwd,
		MainLoopModel:  model,
		DiscoverSkills: discover,
	}
	if strings.TrimSpace(reminder) != "" {
		out.UserContextReminder = reminder
		rh := sha256.Sum256([]byte(reminder))
		out.UserContextReminderSHA256 = hex.EncodeToString(rh[:])
	}
	return out, nil
}
