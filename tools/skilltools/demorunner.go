package skilltools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"goc/tools/toolstub"
	"goc/commands"
	"goc/slashresolve"
	"goc/types"
)

// DemoToolRunner handles Skill tool calls like TS SkillTool (validate + disk/bundled expand); other tools delegate to [toolstub.StubRunner].
type DemoToolRunner struct {
	Commands  []types.Command
	SessionID string
}

// Run implements [toolstub.ToolRunner].
func (r DemoToolRunner) Run(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
	if name != SkillToolName() {
		return toolstub.StubRunner{}.Run(ctx, name, toolUseID, input)
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
	cwd := "."
	if wd, err := os.Getwd(); err == nil {
		cwd = wd
	}

	if found.SkillRoot != nil && strings.TrimSpace(*found.SkillRoot) != "" {
		res, err := slashresolve.ResolveDiskSkill(*found, in.Args, sid)
		if err != nil {
			return err.Error(), true, nil
		}
		return res.UserText, false, nil
	}

	if slashresolve.IsBundledPrompt(*found) {
		res, err := slashresolve.ResolveBundledSkill(*found, in.Args, sid, &slashresolve.BundledResolveOptions{
			Cwd: cwd,
		})
		if err != nil {
			return err.Error(), true, nil
		}
		return res.UserText, false, nil
	}

	return fmt.Sprintf("Skill %s is not a disk skill (SkillRoot) or embedded bundled prompt", normalized), true, nil
}
