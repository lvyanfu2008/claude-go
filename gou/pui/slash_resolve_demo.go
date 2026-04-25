package pui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"goc/commands/handlers"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/gou/conversation"
	"goc/slashresolve"
	"goc/tools/localtools"
	"goc/types"
)

// SlashResolveHandlerOptions configures [NewSlashResolveProcessSlashCommand] for gou-demo.
type SlashResolveHandlerOptions struct {
	// SessionID substitutes ${CLAUDE_SESSION_ID} in disk skills.
	SessionID string
	// Store is the conversation store, needed for state-mutating local commands
	// like /clear. When nil, those commands return a notice instead.
	Store *conversation.Store
	// ReadFileState is the session-scoped read file state, needed by /files.
	// May be nil (no files tracked yet).
	ReadFileState *localtools.ReadFileState
	// Cwd is the current working directory, used by /files to relativize paths.
	Cwd string
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
	rfs := opt.ReadFileState
	cwd := strings.TrimSpace(opt.Cwd)

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
				return unknownSkillSlashResult(parsed, attachmentMessages, suggestAvailableCommands(parsed.CommandName)), nil
			}
			return slashResultToBase(types.SlashResolveResult{
				UserText: strings.TrimSpace(inputString),
				Source:   types.SlashResolveUnknown,
			}, attachmentMessages, uuid, p), nil
		}

		switch cmd.Type {
		case "local", "local-jsx":
			return handleLocalCommand(cmd.Name, parsed.Args, cmd, opt.Store, attachmentMessages, uuid, p, rfs, cwd)
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
					Messages:    []types.Message{SystemNotice(fmt.Sprintf("Slash resolve (disk) for /%s: %v", cmd.Name, err))},
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
					Messages:    []types.Message{SystemNotice(fmt.Sprintf("Slash resolve (bundled) for /%s: %v", cmd.Name, err))},
					ShouldQuery: false,
				}, nil
			}
			return slashResultToBase(res, attachmentMessages, uuid, p), nil
		}

		return &processuserinput.ProcessUserInputBaseResult{
			Messages: []types.Message{SystemNotice(fmt.Sprintf(
				"gou-demo: /%s could not be resolved (type=%q). "+
					"Disk skills need SkillRoot pointing at a directory with SKILL.md. "+
					"Bundled prompts need a Go-side resolver or embedded .md. "+
					"Add a project skill under .claude/skills/SKILL.md or implement a resolver in slashresolve/.",
				cmd.Name, cmd.Type))},
			ShouldQuery: false,
		}, nil
	}
}

// handleLocalCommand dispatches a local or local-jsx command.
// Pure-text handlers go through the handlers registry; state-mutating commands
// like /clear are handled inline with store access.
func handleLocalCommand(
	name string,
	args string,
	cmd *types.Command,
	store *conversation.Store,
	attachmentMessages []types.Message,
	uuid *string,
	p *processuserinput.ProcessUserInputParams,
	rfs *localtools.ReadFileState,
	cwd string,
) (*processuserinput.ProcessUserInputBaseResult, error) {
	// State-mutating commands that need store access.
	if name == "clear" || name == "reset" || name == "new" {
		return handleClearCommand(store)
	}
	// compact is complex (needs API call) — not yet implemented in Go TUI.
	if name == "compact" {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{SystemNotice("/compact is not yet implemented in gou-demo. Use the TS CLI instead.")},
			ShouldQuery: false,
		}, nil
	}

	if name == "files" {
		return handleFilesCommand(rfs, cwd)
	}
	if name == "advisor" {
		return handleAdvisorCommand(args)
	}

	// Pure-text local commands: try the handler registry.
	result, err := handlers.HandleLocalCommand(name, args)
	if err == nil {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{localTextResultNotice(result)},
			ShouldQuery: false,
		}, nil
	}

	// Unknown local command or local-jsx (React-rendered commands).
	return &processuserinput.ProcessUserInputBaseResult{
		Messages:    []types.Message{SystemNotice(fmt.Sprintf("/%s is a %s command — not executed in gou-demo TUI (use TS CLI for interactive handling).", name, cmd.Type))},
		ShouldQuery: false,
	}, nil
}

// handleClearCommand builds the result for /clear, /reset, /new.
func handleClearCommand(store *conversation.Store) (*processuserinput.ProcessUserInputBaseResult, error) {
	if store != nil {
		store.ClearMessages()
	}
	return &processuserinput.ProcessUserInputBaseResult{
		Messages:    []types.Message{SystemNotice("Cleared conversation history.")},
		ShouldQuery: false,
	}, nil
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

func unknownSkillSlashResult(parsed *processuserinput.ParsedSlashCommand, attachmentMessages []types.Message, suggestion string) *processuserinput.ProcessUserInputBaseResult {
	msgs := append([]types.Message(nil), attachmentMessages...)
	msg := fmt.Sprintf("Unknown skill: %s", parsed.CommandName)
	if suggestion != "" {
		msg += ". " + suggestion
	}
	msgs = append(msgs, SystemNotice(msg))
	if a := strings.TrimSpace(parsed.Args); a != "" {
		msgs = append(msgs, SystemNotice(fmt.Sprintf("Args from unknown skill: %s", a)))
	}
	return &processuserinput.ProcessUserInputBaseResult{
		Messages:    msgs,
		ShouldQuery: false,
	}
}

// suggestAvailableCommands returns a fuzzy suggestion for a command name that was not found.
// It scans the available commands and suggests the closest match or a general tip.
func suggestAvailableCommands(name string) string {
	// Check if the name matches any known local command alias.
	aliases := map[string]string{
		"new":     "clear",
		"reset":   "clear",
		"fork":    "branch",
		"quit":    "exit",
		"remote":  "session",
		"ios":     "mobile",
		"android": "mobile",
	}
	if canonical, ok := aliases[name]; ok {
		return fmt.Sprintf("Did you mean /%s?", canonical)
	}
	return ""
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

// handleFilesCommand builds a system notice from handler/files (ReadFileState + cwd).
func handleFilesCommand(rfs *localtools.ReadFileState, cwd string) (*processuserinput.ProcessUserInputBaseResult, error) {
	b, err := handlers.HandleFilesCommand(rfs, cwd)
	if err != nil {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{SystemNotice(fmt.Sprintf("/files: %v", err))},
			ShouldQuery: false,
		}, nil
	}
	return &processuserinput.ProcessUserInputBaseResult{
		Messages:    []types.Message{localTextResultNotice(b)},
		ShouldQuery: false,
	}, nil
}

// handleAdvisorCommand builds a system notice from the advisor handler.
func handleAdvisorCommand(args string) (*processuserinput.ProcessUserInputBaseResult, error) {
	b, err := handlers.HandleAdvisorCommand(args)
	if err != nil {
		return &processuserinput.ProcessUserInputBaseResult{
			Messages:    []types.Message{SystemNotice(fmt.Sprintf("/advisor: %v", err))},
			ShouldQuery: false,
		}, nil
	}
	return &processuserinput.ProcessUserInputBaseResult{
		Messages:    []types.Message{localTextResultNotice(b)},
		ShouldQuery: false,
	}, nil
}

// localTextResultNotice turns handler JSON payloads like {"type":"text","value":"…"} into a readable system row.
func localTextResultNotice(raw []byte) types.Message {
	var v struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(raw, &v) == nil && v.Value != "" {
		return SystemNotice(v.Value)
	}
	return SystemNotice(string(raw))
}
