package main

import (
	"context"
	"os"
	"strings"

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/hookexec"
	"goc/sessiontranscript"
	"goc/types"
)

func buildBaseHookInputForPUI(p *processuserinput.ProcessUserInputParams, cwd string) hookexec.BaseHookInput {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		} else {
			cwd = "."
		}
	}
	var sessionID, agentID, agentType string
	if p != nil && p.RuntimeContext != nil {
		rc := p.RuntimeContext
		if rc.ConversationID != nil {
			sessionID = strings.TrimSpace(*rc.ConversationID)
		}
		if rc.AgentID != nil {
			agentID = strings.TrimSpace(*rc.AgentID)
		}
		if rc.AgentType != nil {
			agentType = strings.TrimSpace(*rc.AgentType)
		}
	}
	transcriptPath := ""
	if sessionID != "" {
		transcriptPath = sessiontranscript.TranscriptPath(sessionID, cwd, "", sessiontranscript.ConfigHomeDir())
	}
	pm := ""
	if p != nil {
		pm = string(p.PermissionMode)
	}
	return hookexec.BaseHookInput{
		SessionID:       sessionID,
		TranscriptPath:  transcriptPath,
		Cwd:             cwd,
		PermissionMode:  pm,
		AgentID:         agentID,
		AgentType:       agentType,
		HookEventName:   "UserPromptSubmit",
	}
}

func wireUserPromptSubmitHooks(p *processuserinput.ProcessUserInputParams, merged hookexec.HooksTable, cwd string) {
	if p == nil || len(merged["UserPromptSubmit"]) == 0 {
		return
	}
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		} else {
			cwd = "."
		}
	}
	cwdFinal := cwd
	p.ExecuteUserPromptSubmitHooks = func(ctx context.Context, pp *processuserinput.ProcessUserInputParams, inputMessage string) ([]types.AggregatedHookResult, error) {
		base := buildBaseHookInputForPUI(pp, cwdFinal)
		return hookexec.RunUserPromptSubmitHooks(ctx, merged, cwdFinal, base, inputMessage, hookexec.DefaultHookTimeoutMs)
	}
}
