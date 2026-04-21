package hookexec

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"

	"goc/compactservice"
	"goc/types"
)

const hookEventSessionStart = "SessionStart"

// SessionStartExtra is optional stdin fields for SessionStart.
type SessionStartExtra struct {
	Source    string `json:"source"`
	AgentType string `json:"agent_type,omitempty"`
	Model     string `json:"model,omitempty"`
}

type sessionStartInput struct {
	BaseHookInput
	SessionStartExtra
}

// RunSessionStartHooks runs SessionStart command hooks synchronously and returns hook_additional_context attachment messages (TS processSessionStartHooks).
func RunSessionStartHooks(ctx context.Context, table HooksTable, workDir string, base BaseHookInput, extra SessionStartExtra, batchTimeoutMs int) ([]types.Message, error) {
	if HooksDisabled() {
		return nil, nil
	}
	hooks := matchingCommandHooks(table, hookEventSessionStart, extra.Source)
	if len(hooks) == 0 {
		return nil, nil
	}
	wd := trimOrDot(workDir)
	var contexts []string
	for _, h := range hooks {
		in := sessionStartInput{
			BaseHookInput: base,
			SessionStartExtra: SessionStartExtra{
				Source:    extra.Source,
				AgentType: extra.AgentType,
				Model:     extra.Model,
			},
		}
		in.HookEventName = hookEventSessionStart
		jsonIn, err := marshalHookInput(in)
		if err != nil {
			continue
		}
		ms := hookTimeoutMS(h, batchTimeoutMs)
		stdout, _, _, err := RunCommandHook(ctx, wd, h.Command, jsonIn, ms)
		if err != nil {
			continue
		}
		add, _ := ParseHookJSONOutput(stdout, hookEventSessionStart)
		if add != "" {
			contexts = append(contexts, add)
		}
	}
	if len(contexts) == 0 {
		return nil, nil
	}
	return []types.Message{newHookAdditionalContextMessage(contexts, hookEventSessionStart)}, nil
}

func newHookAdditionalContextMessage(parts []string, hookEvent string) types.Message {
	toolUseID := hookEvent
	hookName := hookEvent
	att := map[string]any{
		"type":      "hook_additional_context",
		"content":   parts,
		"hookName":  hookName,
		"toolUseID": toolUseID,
		"hookEvent": hookEvent,
	}
	raw, err := json.Marshal(att)
	if err != nil {
		return types.Message{}
	}
	return types.Message{
		Type:       types.MessageTypeAttachment,
		UUID:       randomUUID(),
		Attachment: raw,
	}
}

func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// SessionStartHookRunner returns a compactservice.SessionStartHookRunner backed by merged settings hooks.
func SessionStartHookRunner(projectRoot, cwd, sessionID, transcriptPath string) compactservice.SessionStartHookRunner {
	return func(ctx context.Context, trigger compactservice.SessionStartHookTrigger, in compactservice.SessionStartHookInput) ([]compactservice.HookResultMessage, error) {
		table, err := MergedHooksFromPaths(strings.TrimSpace(projectRoot))
		if err != nil {
			return nil, err
		}
		base := BaseHookInput{
			SessionID:      strings.TrimSpace(sessionID),
			TranscriptPath: strings.TrimSpace(transcriptPath),
			Cwd:            trimOrDot(cwd),
		}
		if base.SessionID == "" {
			base.SessionID = "local"
		}
		msgs, err := RunSessionStartHooks(ctx, table, cwd, base, SessionStartExtra{
			Source: string(trigger),
			Model:  strings.TrimSpace(in.Model),
		}, DefaultHookTimeoutMs)
		if err != nil {
			return nil, err
		}
		out := make([]compactservice.HookResultMessage, len(msgs))
		for i := range msgs {
			out[i] = msgs[i]
		}
		return out, nil
	}
}
