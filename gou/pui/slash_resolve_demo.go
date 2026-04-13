package pui

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/slashresolve"
	"goc/types"
)

// SlashResolveHandlerOptions configures [NewSlashResolveProcessSlashCommand] for gou-demo.
type SlashResolveHandlerOptions struct {
	// SessionID substitutes ${CLAUDE_SESSION_ID} in disk skills.
	SessionID string
}

// NewSlashResolveProcessSlashCommand returns a [processuserinput.ProcessUserInputParams.ProcessSlashCommand]
// that resolves disk skills and embedded bundled prompts in Go (no external TS process).
func NewSlashResolveProcessSlashCommand(opt SlashResolveHandlerOptions) func(
	ctx context.Context,
	inputString string,
	precedingBlocks []types.ContentBlockParam,
	imageContentBlocks []types.ContentBlockParam,
	attachmentMessages []types.Message,
	uuid *string,
	isAlreadyProcessing *bool,
	p *processuserinput.ProcessUserInputParams,
) (*processuserinput.ProcessUserInputBaseResult, error) {
	sid := strings.TrimSpace(opt.SessionID)
	if sid == "" {
		sid = "gou-demo"
	}

	return func(
		ctx context.Context,
		inputString string,
		precedingBlocks []types.ContentBlockParam,
		imageContentBlocks []types.ContentBlockParam,
		attachmentMessages []types.Message,
		uuid *string,
		isAlreadyProcessing *bool,
		p *processuserinput.ProcessUserInputParams,
	) (*processuserinput.ProcessUserInputBaseResult, error) {
		_ = ctx
		_ = precedingBlocks
		_ = imageContentBlocks
		_ = isAlreadyProcessing

		parsed := processuserinput.ParseSlashCommand(inputString)
		if parsed == nil {
			return &processuserinput.ProcessUserInputBaseResult{
				Messages:    []types.Message{SystemNotice("Invalid slash command.")},
				ShouldQuery: false,
			}, nil
		}
		if parsed.IsMcp {
			return &processuserinput.ProcessUserInputBaseResult{
				Messages:    []types.Message{SystemNotice("gou-demo: MCP slash commands are not supported here.")},
				ShouldQuery: false,
			}, nil
		}

		cmd := processuserinput.FindCommand(parsed.CommandName, p.Commands)
		if cmd == nil {
			// TS processSlashCommand: if looksLikeCommand && !isFilePath → Unknown skill, shouldQuery false.
			// Default gou-demo: fall through to a normal prompt (strict mode: GOU_DEMO_SLASH_STRICT_UNKNOWN=1).
			if strictSlashUnknown() &&
				processuserinput.LooksLikeSlashCommandName(parsed.CommandName) &&
				!rootSlashPathExists(parsed.CommandName) {
				return unknownSkillSlashResult(parsed, attachmentMessages), nil
			}
			return slashResultToBase(types.SlashResolveResult{
				UserText: strings.TrimSpace(inputString),
				Source:   types.SlashResolveUnknown,
			}, attachmentMessages, uuid, p), nil
		}

		switch cmd.Type {
		case "local", "local-jsx":
			return &processuserinput.ProcessUserInputBaseResult{
				Messages: []types.Message{SystemNotice(fmt.Sprintf(
					"gou-demo: /%s is a local command — not executed in this TUI (use TS CLI).", cmd.Name))},
				ShouldQuery: false,
			}, nil
		case "prompt":
			// handled below
		default:
			return &processuserinput.ProcessUserInputBaseResult{
				Messages:    []types.Message{SystemNotice(fmt.Sprintf("Unsupported command type %q for /%s", cmd.Type, cmd.Name))},
				ShouldQuery: false,
			}, nil
		}

		// Disk skill: SkillRoot points at directory containing SKILL.md
		if cmd.SkillRoot != nil && strings.TrimSpace(*cmd.SkillRoot) != "" {
			res, err := slashresolve.ResolveDiskSkill(*cmd, parsed.Args, sid)
			if err != nil {
				return &processuserinput.ProcessUserInputBaseResult{
					Messages:    []types.Message{SystemNotice(fmt.Sprintf("Slash resolve (disk): %v", err))},
					ShouldQuery: false,
				}, nil
			}
			return slashResultToBase(res, attachmentMessages, uuid, p), nil
		}

		if slashresolve.IsBundledPrompt(*cmd) {
			cwd, _ := os.Getwd()
			res, err := slashresolve.ResolveBundledSkill(*cmd, parsed.Args, sid, &slashresolve.BundledResolveOptions{Cwd: cwd})
			if err != nil {
				return &processuserinput.ProcessUserInputBaseResult{
					Messages:    []types.Message{SystemNotice(fmt.Sprintf("Slash resolve (bundled): %v", err))},
					ShouldQuery: false,
				}, nil
			}
			return slashResultToBase(res, attachmentMessages, uuid, p), nil
		}

		return &processuserinput.ProcessUserInputBaseResult{
			Messages: []types.Message{SystemNotice(fmt.Sprintf(
				"gou-demo: /%s could not be resolved (not a disk skill and not an embedded bundled prompt); add a project skill under .claude/skills with SKILL.md, or use a bundled command supported by Go.",
				cmd.Name))},
			ShouldQuery: false,
		}, nil
	}
}

func strictSlashUnknown() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_SLASH_STRICT_UNKNOWN")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// rootSlashPathExists mirrors TS getFsImplementation().stat(`/${commandName}`) for unknown-command handling.
func rootSlashPathExists(commandName string) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	_, err := os.Stat("/" + commandName)
	return err == nil
}

func unknownSkillSlashResult(parsed *processuserinput.ParsedSlashCommand, attachmentMessages []types.Message) *processuserinput.ProcessUserInputBaseResult {
	msgs := append([]types.Message(nil), attachmentMessages...)
	msgs = append(msgs, SystemNotice(fmt.Sprintf("Unknown skill: %s", parsed.CommandName)))
	if a := strings.TrimSpace(parsed.Args); a != "" {
		msgs = append(msgs, SystemNotice(fmt.Sprintf("Args from unknown skill: %s", a)))
	}
	return &processuserinput.ProcessUserInputBaseResult{
		Messages:    msgs,
		ShouldQuery: false,
	}
}

func permissionModePtrPI(p *processuserinput.ProcessUserInputParams) *types.PermissionMode {
	if p == nil || p.PermissionMode == "" {
		return nil
	}
	pm := p.PermissionMode
	return &pm
}

func slashResultToBase(
	res types.SlashResolveResult,
	attachmentMessages []types.Message,
	uuid *string,
	p *processuserinput.ProcessUserInputParams,
) *processuserinput.ProcessUserInputBaseResult {
	tp, err := processuserinput.ProcessTextPrompt(
		res.UserText,
		nil, nil, nil,
		attachmentMessages,
		uuid,
		permissionModePtrPI(p),
		p.IsMeta,
		nil,
	)
	if err != nil {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{SystemNotice(fmt.Sprintf("build user message: %v", err))},
			ShouldQuery: false,
		}
	}
	out := &processuserinput.ProcessUserInputBaseResult{
		Messages:     tp.Messages,
		ShouldQuery:  tp.ShouldQuery,
		AllowedTools: append([]string(nil), res.AllowedTools...),
	}
	if res.Model != nil {
		out.Model = *res.Model
	}
	if res.Effort != nil {
		out.Effort = res.Effort
	}
	return out
}
