package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"goc/commands"
	processuserinput "goc/conversation-runtime/process-user-input"
)

// envProcessUserInputCwd overrides cwd when goCommandsLoad.cwd is empty (after trim).
const envProcessUserInputCwd = "CLAUDE_CODE_PROCESS_USER_INPUT_CWD"

// goCommandsLoad is an optional stdin envelope: extra options for
// [commands.LoadAndGetCommandsWithFilePathsDynamic]. This binary always builds the
// slash/skill list in Go (see main: args.context.options.commands cleared before load).
type goCommandsLoad struct {
	// Cwd is the session project root for LoadAllCommands / discover.
	// If empty, see [resolveCwdForGoCommands].
	Cwd string `json:"cwd,omitempty"`
	// TouchedFiles optional paths for dynamic .claude/skills discovery (see commands.DiscoverSkillDirsForPaths).
	TouchedFiles                 []string `json:"touchedFiles,omitempty"`
	SessionProjectRoot           string   `json:"sessionProjectRoot,omitempty"`
	IsClaudeAISubscriber         *bool    `json:"isClaudeAISubscriber,omitempty"`
	IsUsing3PServices            *bool    `json:"isUsing3PServices,omitempty"`
	IsFirstPartyAnthropicBaseURL *bool    `json:"isFirstPartyAnthropicBaseURL,omitempty"`
	// IsRemoteMode when true includes /session in the filtered list (TS: getIsRemoteMode() for src/commands/session).
	IsRemoteMode *bool `json:"isRemoteMode,omitempty"`
	// IsNonInteractiveSession toggles /context and /extra-usage local-jsx vs local variants (TS: getIsNonInteractiveSession).
	IsNonInteractiveSession *bool `json:"isNonInteractiveSession,omitempty"`
	// ExtraUsageAllowed mirrors isOverageProvisioningAllowed for /extra-usage (default false in DefaultConsoleAPIAuth).
	ExtraUsageAllowed *bool `json:"extraUsageAllowed,omitempty"`
	// IsConsumerSubscriber enables /privacy-settings (TS: isConsumerSubscriber).
	IsConsumerSubscriber *bool `json:"isConsumerSubscriber,omitempty"`
	// BlockRemoteSessions when true hides /remote-env (TS: !isPolicyAllowed('allow_remote_sessions')).
	BlockRemoteSessions *bool `json:"blockRemoteSessions,omitempty"`
	// DenyProductFeedback when true hides /feedback (TS: !isPolicyAllowed('allow_product_feedback')).
	DenyProductFeedback *bool `json:"denyProductFeedback,omitempty"`
}

func resolveCwdForGoCommands(load *goCommandsLoad) (string, error) {
	if load != nil {
		if s := strings.TrimSpace(load.Cwd); s != "" {
			return s, nil
		}
	}
	if s := strings.TrimSpace(os.Getenv(envProcessUserInputCwd)); s != "" {
		return s, nil
	}
	if s := strings.TrimSpace(os.Getenv("PWD")); s != "" {
		return s, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(wd), nil
}

func applyGoCommandsLoad(ctx context.Context, p *processuserinput.ProcessUserInputParams, load *goCommandsLoad) error {
	cwd, err := resolveCwdForGoCommands(load)
	if err != nil {
		return fmt.Errorf("resolve cwd: %w", err)
	}
	if cwd == "" {
		return fmt.Errorf(
			"missing cwd for Go command list (set goCommandsLoad.cwd, %s, PWD, or run from a directory)",
			envProcessUserInputCwd,
		)
	}
	auth := commands.DefaultConsoleAPIAuth()
	var sessionRoot string
	var touched []string
	if load != nil {
		if load.IsClaudeAISubscriber != nil {
			auth.IsClaudeAISubscriber = *load.IsClaudeAISubscriber
		}
		if load.IsUsing3PServices != nil {
			auth.IsUsing3PServices = *load.IsUsing3PServices
		}
		if load.IsFirstPartyAnthropicBaseURL != nil {
			auth.IsFirstPartyAnthropicBaseURL = *load.IsFirstPartyAnthropicBaseURL
		}
		if load.IsRemoteMode != nil {
			auth.IsRemoteMode = *load.IsRemoteMode
		}
		if load.IsNonInteractiveSession != nil {
			auth.IsNonInteractiveSession = *load.IsNonInteractiveSession
		}
		if load.ExtraUsageAllowed != nil {
			auth.ExtraUsageAllowed = *load.ExtraUsageAllowed
		}
		if load.IsConsumerSubscriber != nil {
			auth.IsConsumerSubscriber = *load.IsConsumerSubscriber
		}
		if load.BlockRemoteSessions != nil {
			auth.BlockRemoteSessions = *load.BlockRemoteSessions
		}
		if load.DenyProductFeedback != nil {
			auth.DenyProductFeedback = *load.DenyProductFeedback
		}
		sessionRoot = strings.TrimSpace(load.SessionProjectRoot)
		touched = load.TouchedFiles
	}
	opts := commands.LoadOptions{
		SessionProjectRoot: sessionRoot,
		// P6 workflow listing deferred — do not enable WorkflowScripts by default (see docs/plans/goc-load-all-commands.md).
	}
	if strings.TrimSpace(os.Getenv("CLAUDE_CODE_BARE")) == "1" {
		t := true
		opts.BareMode = &t
	}
	if len(touched) == 0 {
		touched = nil
	}
	cmds, err := commands.LoadAndGetCommandsWithFilePathsDynamic(ctx, cwd, opts, auth, touched, nil)
	if err != nil {
		return err
	}
	p.Commands = cmds
	if p.RuntimeContext != nil {
		o := p.RuntimeContext.Options
		o.Commands = cmds
		p.RuntimeContext.Options = o
	}
	return nil
}
