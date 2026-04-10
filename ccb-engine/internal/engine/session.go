package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"goc/ccb-engine/internal/anthropic"
	"goc/ccb-engine/internal/llm"
	"goc/ccb-engine/internal/protocol"
	"goc/ccb-engine/internal/toolinput"
	"goc/ccb-engine/internal/toolpolicy"
)

const maxToolRounds = 24

// EventSink receives protocol stream events (logging, future socket).
type EventSink func(protocol.StreamEvent)

// Session holds canonical API-shaped messages and a monotonic stateRev.
type Session struct {
	mu       sync.Mutex
	stateRev atomic.Uint64
	messages []anthropic.Message
	sink     EventSink
}

func NewSession(sink EventSink) *Session {
	return &Session{sink: sink}
}

// StateRev returns the current monotonic revision (lock-free; safe while bridge calls it during RunTurn).
func (s *Session) StateRev() uint64 {
	return s.stateRev.Load()
}

func (s *Session) bumpLocked() uint64 {
	return s.stateRev.Add(1)
}

func (s *Session) emit(ev protocol.StreamEvent) {
	if s.sink != nil {
		s.sink(ev)
	}
}

// AppendUserText appends a user message with string content.
func (s *Session) AppendUserText(text string) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, anthropic.Message{Role: "user", Content: text})
	return s.bumpLocked()
}

func (s *Session) cloneMessagesLocked() []anthropic.Message {
	out := make([]anthropic.Message, len(s.messages))
	copy(out, s.messages)
	return out
}

// HydrateFromMessages replaces the session transcript with msgs (API-shaped JSON messages)
// and bumps stateRev once. Used before AppendUserText for TS-provided history snapshots.
func (s *Session) HydrateFromMessages(msgs []anthropic.Message) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append([]anthropic.Message(nil), msgs...)
	return s.bumpLocked()
}

// RunTurnOption configures a single RunTurn (e.g. permission_context for toolpolicy).
type RunTurnOption func(*runTurnOptions)

type runTurnOptions struct {
	permissionContext json.RawMessage
}

// WithPermissionContext passes TS-provided JSON for Go-side allowlisting (see toolpolicy).
func WithPermissionContext(raw json.RawMessage) RunTurnOption {
	return func(o *runTurnOptions) {
		o.permissionContext = raw
	}
}

// RunTurn calls the model until end_turn or max rounds; uses runner for tool_result content.
// Lock is not held during HTTP calls to the API. system is forwarded to the Messages API (Anthropic) or prepended as a system message (OpenAI compat).
// When skillUserFollowUp is true, successful Skill tool results also append a plain-text user message (TS often inserts expanded skill content as follow-up transcript).
func (s *Session) RunTurn(ctx context.Context, completer llm.TurnCompleter, tools []anthropic.ToolDefinition, system string, runner ToolRunner, skillUserFollowUp bool, opts ...RunTurnOption) error {
	if runner == nil {
		runner = StubRunner{}
	}
	var rtOpts runTurnOptions
	for _, fn := range opts {
		fn(&rtOpts)
	}

	for round := 0; round < maxToolRounds; round++ {
		msgs := func() []anthropic.Message {
			s.mu.Lock()
			defer s.mu.Unlock()
			return s.cloneMessagesLocked()
		}()

		result, err := completer.Complete(ctx, msgs, tools, system)
		if err != nil {
			s.emit(protocol.ErrEvent("api_error", err.Error()))
			return err
		}

		s.emit(protocol.Usage(result.InputTokens, result.OutputTokens))

		blocks := result.Blocks
		for _, b := range blocks {
			if b.Type == "text" {
				s.emit(protocol.AssistantDelta(b.Text))
			}
			if b.Type == "tool_use" {
				var inputObj map[string]any
				_ = json.Unmarshal(b.Input, &inputObj)
				s.emit(protocol.ToolUse(b.ID, b.Name, inputObj))
			}
		}

		s.mu.Lock()
		s.messages = append(s.messages, anthropic.Message{
			Role:    "assistant",
			Content: blocks,
		})
		s.bumpLocked()
		s.mu.Unlock()

		if result.StopReason != "tool_use" {
			rev := s.StateRev()
			s.emit(protocol.TurnComplete(rev, result.StopReason))
			return nil
		}

		var toolBlocks []anthropic.ContentBlock
		var skillFollowUps []string
		for _, b := range blocks {
			if b.Type != "tool_use" {
				continue
			}
			var content string
			var isErr bool
			var err error
			if verr := toolinput.ValidateAgainstTools(tools, b.Name, b.Input); verr != nil {
				content = fmt.Sprintf("tool input_schema validation (ccb-engine): %v", verr)
				isErr = true
			} else if deny := toolpolicy.DenyReason(rtOpts.permissionContext, b.Name); deny != "" {
				content = "ccb-engine policy: " + deny
				isErr = true
			} else {
				content, isErr, err = runner.Run(ctx, b.Name, b.ID, b.Input)
				if err != nil {
					content = err.Error()
					isErr = true
				}
			}
			s.emit(protocol.ToolResult(b.ID, content, isErr))
			toolBlocks = append(toolBlocks, anthropic.ContentBlock{
				Type:      "tool_result",
				ToolUseID: b.ID,
				Content:   content,
				IsError:   isErr,
			})
			if skillUserFollowUp && b.Name == anthropic.SkillToolName && !isErr && strings.TrimSpace(content) != "" {
				skillFollowUps = append(skillFollowUps, content)
			}
		}
		if len(toolBlocks) == 0 {
			s.emit(protocol.ErrEvent("invalid_request", "stop_reason tool_use but no tool_use blocks"))
			return fmt.Errorf("tool_use without blocks")
		}

		s.mu.Lock()
		s.messages = append(s.messages, anthropic.Message{
			Role:    "user",
			Content: toolBlocks,
		})
		for _, text := range skillFollowUps {
			s.messages = append(s.messages, anthropic.Message{
				Role:    "user",
				Content: text,
			})
		}
		s.bumpLocked()
		s.mu.Unlock()
	}

	s.emit(protocol.ErrEvent("invalid_request", "max tool rounds exceeded"))
	return fmt.Errorf("max tool rounds (%d) exceeded", maxToolRounds)
}
