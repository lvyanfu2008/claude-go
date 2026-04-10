// Package localturn runs one ccb-engine SubmitUserTurn in-process (no Unix socket; contrast with socketserve / gou-demo).
// Same protocol events as socketserve / gou-demo embedded listener. Default [engine.StubRunner]; gou-demo passes [skilltools.ParityToolRunner] for real Read/Write/Edit/Glob/Grep (no execute_tool wire).
package localturn

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goc/ccb-engine/internal/anthropic"
	"goc/ccb-engine/internal/engine"
	"goc/ccb-engine/internal/llm"
	"goc/ccb-engine/internal/protocol"
	"goc/ccb-engine/settingsfile"
	"goc/ccb-engine/submitfill"
)

// Params matches SubmitUserTurn payload (text and/or messages, optional tools JSON array).
type Params struct {
	RequestID string
	Messages  json.RawMessage
	Text      string
	Tools     json.RawMessage
	System    string
	// SkillExpandUserFollowUp when true appends Skill tool expanded text as an extra user message (see [engine.Session.RunTurn]).
	SkillExpandUserFollowUp bool
	// Runner optional tool execution (nil → [engine.StubRunner]).
	Runner engine.ToolRunner `json:"-"`

	// FetchSystemPromptIfEmpty when true (or env CCB_ENGINE_FETCH_SYSTEM_PROMPT_IF_EMPTY), and System is empty,
	// builds system via [querycontext.FetchSystemPromptParts] and may prepend a user-context reminder. See [submitfill] package doc.
	FetchSystemPromptIfEmpty bool     `json:"fetch_system_prompt_if_empty,omitempty"`
	Cwd                      string   `json:"cwd,omitempty"`
	ExtraClaudeMdRoots       []string `json:"extra_claude_md_roots,omitempty"`
	CustomSystemPrompt       string   `json:"custom_system_prompt,omitempty"`
	AppendSystemPrompt       string   `json:"append_system_prompt,omitempty"`
	ModelID                  string   `json:"model,omitempty"`
}

func validateParams(p Params) error {
	hasMsgs := len(p.Messages) > 0
	hasText := strings.TrimSpace(p.Text) != ""
	if !hasMsgs && !hasText {
		return fmt.Errorf("need non-empty text and/or messages")
	}
	return nil
}

func fromProto(ev protocol.StreamEvent) StreamEvent {
	return StreamEvent{
		Type:         ev.Type,
		Text:         ev.Text,
		ID:           ev.ID,
		Name:         ev.Name,
		Input:        ev.Input,
		ToolUseID:    ev.ToolUseID,
		CallID:       ev.CallID,
		Content:      ev.Content,
		StateRev:     ev.StateRev,
		StopReason:   ev.StopReason,
		Code:         ev.Code,
		Message:      ev.Message,
		InputTokens:  ev.InputTokens,
		OutputTokens: ev.OutputTokens,
		IsError:      ev.IsError,
	}
}

// RunSubmitUserTurn runs sess.RunTurn with Params.Runner or StubRunner, emitting StreamEvent for each protocol event,
// then ResponseEnd(RequestID). On validation errors, emits error + response_end and returns the error.
func RunSubmitUserTurn(ctx context.Context, p Params, emit func(StreamEvent)) error {
	if emit == nil {
		emit = func(StreamEvent) {}
	}
	send := func(ev protocol.StreamEvent) { emit(fromProto(ev)) }

	ridEarly := p.RequestID
	if ridEarly == "" {
		ridEarly = "local"
	}
	if err := settingsfile.EnsureProjectClaudeEnvOnce(); err != nil {
		send(protocol.ErrEvent("config", err.Error()))
		send(protocol.ResponseEnd(ridEarly))
		return err
	}

	if err := validateParams(p); err != nil {
		send(protocol.ErrEvent("invalid_request", err.Error()))
		send(protocol.ResponseEnd(p.RequestID))
		return err
	}
	if p.RequestID == "" {
		p.RequestID = "local"
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	completer := llm.NewFromEnv()
	tools := anthropic.DefaultStubTools()
	toolsSource := "default_stub"
	if len(p.Tools) > 0 {
		var parsed []anthropic.ToolDefinition
		if err := json.Unmarshal(p.Tools, &parsed); err != nil {
			send(protocol.ErrEvent("invalid_request", "tools: "+err.Error()))
			send(protocol.ResponseEnd(p.RequestID))
			return err
		}
		if len(parsed) > 0 {
			tools = parsed
			toolsSource = "payload_json"
		}
	}
	anthropic.LogToolsLoaded("localturn", p.RequestID, toolsSource, tools)

	msgsRaw := p.Messages
	systemStr := p.System
	fillOpts := submitfill.Options{
		FetchIfEmpty:       submitfill.FetchDesired(p.FetchSystemPromptIfEmpty),
		Cwd:                p.Cwd,
		ToolsJSON:          p.Tools,
		ExtraClaudeMdRoots: p.ExtraClaudeMdRoots,
		CustomSystemPrompt: p.CustomSystemPrompt,
		AppendSystemPrompt: p.AppendSystemPrompt,
		ModelID:            p.ModelID,
	}
	var errFill error
	systemStr, msgsRaw, errFill = submitfill.ApplyIfEmpty(systemStr, msgsRaw, fillOpts)
	if errFill != nil {
		send(protocol.ErrEvent("invalid_request", "system_context: "+errFill.Error()))
		send(protocol.ResponseEnd(p.RequestID))
		return errFill
	}

	sess := engine.NewSession(send)

	runner := engine.ToolRunner(p.Runner)
	if runner == nil {
		runner = engine.StubRunner{}
	}

	if len(msgsRaw) > 0 {
		var msgs []anthropic.Message
		if err := json.Unmarshal(msgsRaw, &msgs); err != nil {
			send(protocol.ErrEvent("invalid_request", "messages: "+err.Error()))
			send(protocol.ResponseEnd(p.RequestID))
			return err
		}
		sess.HydrateFromMessages(msgs)
	}
	if strings.TrimSpace(p.Text) != "" {
		sess.AppendUserText(p.Text)
	}

	err := sess.RunTurn(ctx, completer, tools, systemStr, runner, p.SkillExpandUserFollowUp)
	send(protocol.ResponseEnd(p.RequestID))
	return err
}
