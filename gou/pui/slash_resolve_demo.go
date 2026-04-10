package pui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/slashresolve"
	"goc/types"
)

// SlashResolveHandlerOptions configures [NewSlashResolveProcessSlashCommand] for gou-demo.
type SlashResolveHandlerOptions struct {
	// RepoRoot is the monorepo root (directory containing scripts/slash-resolve-bridge.ts).
	// Empty skips bundled resolution via bridge (disk skills still work).
	RepoRoot string
	// SessionID substitutes ${CLAUDE_SESSION_ID} in disk skills.
	SessionID string
}

// FindRepoRootForBridge walks upward from startDir looking for scripts/slash-resolve-bridge.ts.
func FindRepoRootForBridge(startDir string) string {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		dir = startDir
	}
	for i := 0; i < 32; i++ {
		bridge := filepath.Join(dir, "scripts", "slash-resolve-bridge.ts")
		if st, err := os.Stat(bridge); err == nil && !st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// NewSlashResolveProcessSlashCommand returns a [processuserinput.ProcessUserInputParams.ProcessSlashCommand]
// that resolves disk skills in Go and bundled skills via the optional Bun bridge.
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
	repoRoot := strings.TrimSpace(opt.RepoRoot)

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
			return &processuserinput.ProcessUserInputBaseResult{
				Messages: []types.Message{SystemNotice(fmt.Sprintf("Unknown command: /%s", parsed.CommandName))},
				ShouldQuery: false,
			}, nil
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
					Messages: []types.Message{SystemNotice(fmt.Sprintf("Slash resolve (disk): %v", err))},
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

		// Optional TS bridge for non-embedded commands
		if repoRoot != "" {
			cmdJSON, err := json.Marshal(cmd)
			if err != nil {
				return &processuserinput.ProcessUserInputBaseResult{
					Messages:    []types.Message{SystemNotice(fmt.Sprintf("slash bridge: marshal command: %v", err))},
					ShouldQuery: false,
				}, nil
			}
			cwd, _ := os.Getwd()
			res, err := slashresolve.ResolveViaBridge(repoRoot, slashresolve.BridgeRequest{
				CommandName: parsed.CommandName,
				Cwd:         cwd,
				Args:        parsed.Args,
				CommandJSON: cmdJSON,
			})
			if err != nil {
				return &processuserinput.ProcessUserInputBaseResult{
					Messages: []types.Message{SystemNotice(fmt.Sprintf("Slash bridge (bundled): %v", err))},
					ShouldQuery: false,
				}, nil
			}
			return slashResultToBase(res, attachmentMessages, uuid, p), nil
		}

		return &processuserinput.ProcessUserInputBaseResult{
			Messages: []types.Message{SystemNotice(fmt.Sprintf(
				"gou-demo: /%s could not be resolved (not disk, not bundled); set cwd so repo root contains scripts/slash-resolve-bridge.ts, or add a project skill under .claude/skills.",
				cmd.Name))},
			ShouldQuery: false,
		}, nil
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
		Messages:    tp.Messages,
		ShouldQuery: tp.ShouldQuery,
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
