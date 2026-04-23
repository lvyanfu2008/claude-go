package slashresolve

import (
	"errors"
	"fmt"
	"strings"

	"goc/types"
)

var errNotBundledPrompt = errors.New("slashresolve: command is not a bundled prompt")

// BundledResolveOptions carries optional session context for skills that read transcript/memory in TS.
type BundledResolveOptions struct {
	Cwd           string
	SessionMemory string
	UserMessages  []string
}

// IsBundledPrompt mirrors listing metadata: prompt command with source or loadedFrom "bundled".
func IsBundledPrompt(cmd types.Command) bool {
	if cmd.Type != "prompt" {
		return false
	}
	if cmd.Source != nil && *cmd.Source == "bundled" {
		return true
	}
	if cmd.LoadedFrom != nil && *cmd.LoadedFrom == "bundled" {
		return true
	}
	return false
}

// ResolveBundledSkill expands a bundled skill to user text (TS getPromptForCommand parity).
func ResolveBundledSkill(cmd types.Command, args, sessionID string, opt *BundledResolveOptions) (types.SlashResolveResult, error) {
	if !IsBundledPrompt(cmd) {
		return types.SlashResolveResult{}, errNotBundledPrompt
	}
	if opt == nil {
		opt = &BundledResolveOptions{}
	}

	var res types.SlashResolveResult
	var err error

	switch cmd.Name {
	case "update-config":
		res, err = resolveUpdateConfig(args)
	case "remember":
		res, err = resolveRemember(args)
	case "simplify":
		res, err = resolveSimplify(args)
	case "stuck":
		res, err = resolveStuck(args)
	case "dream":
		res, err = resolveDream(args, opt)
	case "keybindings-help":
		res, err = resolveKeybindingsHelp(args)
	case "hunter":
		res, err = resolveBundledMarkdownUserRequest("hunter.md", args)
	case "schedule":
		res, err = resolveSchedule(args, opt)
	case "claude-api":
		res, err = resolveClaudeAPI(args, opt.Cwd)
	case "run-skill-generator":
		res, err = resolveBundledMarkdownUserRequest("run-skill-generator.md", args)
	case "loop":
		res, err = resolveLoop(args)
	case "batch":
		res, err = resolveBatch(args, opt.Cwd)
	case "lorem-ipsum":
		res, err = resolveLoremIpsum(args)
	case "debug":
		res = resolveDebugBundled(args, sessionID)
	case "verify":
		res, err = resolveVerifyBundled(args)
	case "claude-in-chrome":
		res = resolveClaudeInChrome(args)
	case "skillify":
		res, err = resolveSkillifyBundled(args, opt.SessionMemory, opt.UserMessages)
	case "cron-list":
		res = resolveCronList(args)
	case "cron-delete":
		res, err = resolveCronDelete(args)
	default:
		res, err = resolveDefaultBundledEmbed(cmd.Name, args)
	}
	if err != nil {
		return types.SlashResolveResult{}, err
	}

	if len(cmd.AllowedTools) > 0 {
		res.AllowedTools = append([]string(nil), cmd.AllowedTools...)
	}
	if cmd.Model != nil && strings.TrimSpace(*cmd.Model) != "" {
		m := *cmd.Model
		res.Model = &m
	}
	if cmd.Effort != nil {
		ev := *cmd.Effort
		res.Effort = &ev
	}
	return res, nil
}

func appendUserSection(base, args string) string {
	a := strings.TrimSpace(args)
	if a == "" {
		return base
	}
	return base + "\n\n## User Request\n\n" + a
}

func bundledMarkdownName(cmdName string) string {
	if cmdName == "" {
		return ""
	}
	return cmdName + ".md"
}

func resolveDefaultBundledEmbed(cmdName, args string) (types.SlashResolveResult, error) {
	rel := bundledMarkdownName(cmdName)
	body, err := readBundledText(rel)
	if err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("bundled embed %s: %w", rel, err)
	}
	return types.SlashResolveResult{UserText: body, Source: types.SlashResolveBundledEmbed}, nil
}

func resolveVerifyBundled(args string) (types.SlashResolveResult, error) {
	dir, err := materializeVerifySkillDir()
	if err != nil {
		return types.SlashResolveResult{}, err
	}
	body, err := readVerifySkillBody()
	if err != nil {
		return types.SlashResolveResult{}, err
	}
	text := fmt.Sprintf("Base directory for this skill: %s\n\n%s", dir, body)
	text = appendUserSection(text, args)
	return types.SlashResolveResult{
		UserText:          text,
		Source:            types.SlashResolveBundledEmbed,
		MaterializedPaths: []string{dir},
	}, nil
}

// replaceTaggedBlock swaps inner text between <tag>\n and </tag> (TS skillify session placeholders).
func replaceTaggedBlock(s, tag, inner string) string {
	open := "<" + tag + ">\n"
	close := "</" + tag + ">"
	i := strings.Index(s, open)
	if i < 0 {
		return s
	}
	start := i + len(open)
	j := strings.Index(s[start:], close)
	if j < 0 {
		return s
	}
	j += start
	return s[:start] + inner + s[j:]
}

func resolveSkillifyBundled(args, sessionMem string, userMsgs []string) (types.SlashResolveResult, error) {
	raw, err := readBundledText("skillify.md")
	if err != nil {
		return types.SlashResolveResult{}, err
	}
	mem := strings.TrimSpace(sessionMem)
	if mem == "" {
		mem = "No session memory available."
	}
	msgs := strings.Join(userMsgs, "\n\n---\n\n")
	out := replaceTaggedBlock(raw, "session_memory", mem+"\n")
	out = replaceTaggedBlock(out, "user_messages", msgs+"\n")
	if ub := strings.TrimSpace(args); ub != "" {
		out += "\n\n## User description\n\n" + ub
	}
	return types.SlashResolveResult{UserText: out, Source: types.SlashResolveBundledEmbed}, nil
}

// resolveCronList mirrors registerCronListSkill getPromptForCommand in src/skills/bundled/cronManage.ts
func resolveCronList(args string) types.SlashResolveResult {
	text := "Call CronList to list all scheduled cron jobs. Display the results in a table with columns: ID, Schedule, Prompt, Recurring, Durable. If no jobs exist, say \"No scheduled tasks.\""
	if argsText := strings.TrimSpace(args); argsText != "" {
		text = appendUserSection(text, argsText)
	}
	return types.SlashResolveResult{
		UserText: text,
		Source:   types.SlashResolveBundledEmbed,
	}
}

// resolveCronDelete mirrors registerCronDeleteSkill getPromptForCommand in src/skills/bundled/cronManage.ts
func resolveCronDelete(args string) (types.SlashResolveResult, error) {
	id := strings.TrimSpace(args)
	if id == "" {
		text := "Usage: /cron-delete <job-id>\n\nProvide the job ID to cancel. Use /cron-list to see active jobs and their IDs."
		return types.SlashResolveResult{
			UserText: text,
			Source:   types.SlashResolveBundledEmbed,
		}, nil
	}
	text := fmt.Sprintf("Call CronDelete with id \"%s\" to cancel that scheduled job. Confirm the result to the user.", id)
	return types.SlashResolveResult{
		UserText: text,
		Source:   types.SlashResolveBundledEmbed,
	}, nil
}
