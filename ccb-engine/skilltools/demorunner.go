package skilltools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"goc/ccb-engine/internal/engine"
	"goc/commands"
	"goc/slashresolve"
	"goc/types"
)

// DemoToolRunner handles Skill tool calls like TS SkillTool (validate + disk/bridge expand); other tools delegate to [engine.StubRunner].
// Deviation vs TS: expanded skill text is returned as a tool_result string; TS often inserts follow-up user/assistant messages in the transcript.
type DemoToolRunner struct {
	Commands  []types.Command
	RepoRoot  string
	SessionID string
}

// Run implements [engine.ToolRunner].
func (r DemoToolRunner) Run(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
	if name != SkillToolName() {
		return engine.StubRunner{}.Run(ctx, name, toolUseID, input)
	}
	var in struct {
		Skill string `json:"skill"`
		Args  string `json:"args"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return "", true, nil
	}
	trimmed := strings.TrimSpace(in.Skill)
	if trimmed == "" {
		return fmt.Sprintf("Invalid skill format: %s", in.Skill), true, nil
	}
	normalized := trimmed
	if strings.HasPrefix(normalized, "/") {
		normalized = normalized[1:]
	}
	found := commands.FindCommand(normalized, r.Commands)
	if found == nil {
		return fmt.Sprintf("Unknown skill: %s", normalized), true, nil
	}
	if found.DisableModelInvocation != nil && *found.DisableModelInvocation {
		return fmt.Sprintf("Skill %s cannot be used with Skill tool due to disable-model-invocation", normalized), true, nil
	}
	if found.Type != "prompt" {
		return "Skill tool only supports prompt-type commands", true, nil
	}

	sid := r.SessionID
	if strings.TrimSpace(sid) == "" {
		sid = "gou-demo"
	}

	if found.SkillRoot != nil && strings.TrimSpace(*found.SkillRoot) != "" {
		res, err := slashresolve.ResolveDiskSkill(*found, in.Args, sid)
		if err != nil {
			return err.Error(), true, nil
		}
		return res.UserText, false, nil
	}

	if strings.TrimSpace(r.RepoRoot) == "" {
		return "Bundled/plugin skills require repo root (bridge) or a disk skill with SkillRoot", true, nil
	}
	cmdJSON, err := json.Marshal(found)
	if err != nil {
		return err.Error(), true, nil
	}
	cwd := "."
	if wd, err := os.Getwd(); err == nil {
		cwd = wd
	}
	res, err := slashresolve.ResolveViaBridge(r.RepoRoot, slashresolve.BridgeRequest{
		CommandName: found.Name,
		Cwd:         cwd,
		Args:        in.Args,
		CommandJSON: cmdJSON,
	})
	if err != nil {
		return err.Error(), true, nil
	}
	return res.UserText, false, nil
}
