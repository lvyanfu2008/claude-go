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
	"goc/ccb-engine/internal/toolsearch"
	"goc/ccb-engine/localtools"
	"goc/modelenv"
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
	permissionContext     json.RawMessage
	modelID               string
	hasPendingMcpServers  bool
	pendingMcpServerNames []string
}

// WithPermissionContext passes TS-provided JSON for Go-side allowlisting (see toolpolicy).
func WithPermissionContext(raw json.RawMessage) RunTurnOption {
	return func(o *runTurnOptions) {
		o.permissionContext = raw
	}
}

// WithModelID passes the request model id for tool-search gating (mirrors TS modelSupportsToolReference / isToolSearchEnabled model checks).
func WithModelID(id string) RunTurnOption {
	return func(o *runTurnOptions) {
		o.modelID = strings.TrimSpace(id)
	}
}

// WithPendingMcpServers mirrors options.hasPendingMcpServers in src/services/api/claude.ts (keeps ToolSearch when MCP still connecting).
func WithPendingMcpServers(pending bool) RunTurnOption {
	return func(o *runTurnOptions) {
		o.hasPendingMcpServers = pending
	}
}

// WithPendingMcpServerNames optional display names for connecting MCP servers (TS ToolSearch empty-result copy).
func WithPendingMcpServerNames(names []string) RunTurnOption {
	return func(o *runTurnOptions) {
		if len(names) == 0 {
			o.pendingMcpServerNames = nil
			return
		}
		o.pendingMcpServerNames = append([]string(nil), names...)
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

	if ctx == nil {
		ctx = context.Background()
	}
	ctx = ContextWithToolTurn(ctx, tools, rtOpts.hasPendingMcpServers, rtOpts.pendingMcpServerNames)

	for round := 0; round < maxToolRounds; round++ {
		msgs := func() []anthropic.Message {
			s.mu.Lock()
			defer s.mu.Unlock()
			return s.cloneMessagesLocked()
		}()

		modelID := rtOpts.modelID
		if modelID == "" {
			modelID = modelenv.ResolveWithFallback("")
		}
		openAI := llm.UseOpenAICompat()
		wireCfg := toolsearch.BuildWireConfig(modelID, tools, rtOpts.hasPendingMcpServers, openAI)
		wiredTools := toolsearch.ApplyWire(tools, msgs, wireCfg)
		apiMsgs := toolsearch.PrepareAnthropicMessages(msgs, tools, wireCfg)
		toolsearch.LogWireRound(round, modelID, msgs, wireCfg, openAI, tools, wiredTools)

		result, err := completer.Complete(ctx, apiMsgs, wiredTools, system)
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
				} else if !isErr && b.Name == "Read" {
					content = mapReadToolResultForModel(content, b.Input, modelID, runner)
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

// mapReadToolResultForModel mirrors toolexecution syntheticToolMessageAfterInvoke Read branch:
// ParityToolRunner returns raw Read JSON; the model sees mapToolResultToToolResultBlockParam output.
func mapReadToolResultForModel(content string, input json.RawMessage, modelID string, runner ToolRunner) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" || !json.Valid([]byte(trimmed)) {
		return content
	}
	var probe struct {
		Type string `json:"type"`
	}
	if json.Unmarshal([]byte(trimmed), &probe) != nil {
		return content
	}
	if probe.Type != "text" && probe.Type != "file_unchanged" {
		return content
	}
	var roots []string
	var memCwd string
	if v, ok := runner.(interface {
		ToolReadMappingRoots() []string
		ToolReadMappingMemCWD() string
	}); ok {
		roots = v.ToolReadMappingRoots()
		memCwd = v.ToolReadMappingMemCWD()
	}
	opts := localtools.ReadToolResultMapOptsForToolInput(input, roots, memCwd, modelID)
	mapped, err := localtools.MapReadToolResultToAssistantText(trimmed, opts)
	if err != nil {
		return content
	}
	return mapped
}
