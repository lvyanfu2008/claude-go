package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	"goc/tools/toolexecution"
	"goc/types"
)

const summaryInterval = 30 * time.Second

// sharedMessageBuffer accumulates messages during agent execution so the
// summarization goroutine can read the latest conversation state without
// waiting for sidechain persistence (which only happens after execution).
type sharedMessageBuffer struct {
	mu       sync.Mutex
	messages []types.Message
}

func (b *sharedMessageBuffer) Snapshot() []types.Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]types.Message, len(b.messages))
	copy(out, b.messages)
	return out
}

func (b *sharedMessageBuffer) Append(msgs ...types.Message) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = append(b.messages, msgs...)
}

// summaryManager manages the periodic agent summarization lifecycle.
// TS parity: src/services/AgentSummary/agentSummary.ts
type summaryManager struct {
	session         *AgentSession
	cfg             AgentRuntimeConfig
	msgBuf          *sharedMessageBuffer
	previousSummary string
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

// buildSummaryPrompt mirrors agentSummary.ts buildSummaryPrompt.
func buildSummaryPrompt(previousSummary string) string {
	prevLine := ""
	if previousSummary != "" {
		prevLine = fmt.Sprintf("\nPrevious: %q — say something NEW.\n", previousSummary)
	}

	return fmt.Sprintf(`Describe your most recent action in 3-5 words using present tense (-ing). Name the file or function, not the branch. Do not use tools.
%s
Good: "Reading runAgent.ts"
Good: "Fixing null check in validate.ts"
Good: "Running auth module tests"
Good: "Adding retry logic to fetchUser"

Bad (past tense): "Analyzed the branch diff"
Bad (too vague): "Investigating the issue"
Bad (too long): "Reviewing full branch diff and AgentTool.tsx integration"
Bad (branch name): "Analyzed adam/background-summary branch diff"`, prevLine)
}

// filterIncompleteToolCalls mirrors runAgent.ts filterIncompleteToolCalls.
// Removes assistant messages with tool_use blocks lacking corresponding tool_result blocks.
func filterIncompleteToolCalls(messages []types.Message) []types.Message {
	toolUseIDsWithResults := make(map[string]struct{})
	for _, m := range messages {
		if m.Type != types.MessageTypeUser || len(m.Message) == 0 {
			continue
		}
		var payload struct {
			Content []struct {
				Type      string `json:"type"`
				ToolUseID string `json:"tool_use_id"`
			} `json:"content"`
		}
		if err := json.Unmarshal(m.Message, &payload); err != nil {
			continue
		}
		for _, block := range payload.Content {
			if block.Type == "tool_result" && block.ToolUseID != "" {
				toolUseIDsWithResults[block.ToolUseID] = struct{}{}
			}
		}
	}

	out := make([]types.Message, 0, len(messages))
	for _, m := range messages {
		if m.Type != types.MessageTypeAssistant || len(m.Message) == 0 {
			out = append(out, m)
			continue
		}
		var payload struct {
			Content []struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			} `json:"content"`
		}
		if err := json.Unmarshal(m.Message, &payload); err != nil {
			out = append(out, m)
			continue
		}
		hasIncomplete := false
		for _, block := range payload.Content {
			if block.Type == "tool_use" && block.ID != "" {
				if _, ok := toolUseIDsWithResults[block.ID]; !ok {
					hasIncomplete = true
					break
				}
			}
		}
		if !hasIncomplete {
			out = append(out, m)
		}
	}
	return out
}

// startAgentSummarization starts a background goroutine that periodically
// summarizes the agent's recent activity. Returns a stop function.
// TS parity: agentSummary.ts startAgentSummarization
func startAgentSummarization(session *AgentSession, cfg AgentRuntimeConfig, msgBuf *sharedMessageBuffer) func() {
	m := &summaryManager{
		session: session,
		cfg:     cfg,
		msgBuf:  msgBuf,
		stopCh:  make(chan struct{}),
	}
	m.wg.Add(1)
	go m.runLoop()
	return func() {
		close(m.stopCh)
		m.wg.Wait()
	}
}

func (m *summaryManager) runLoop() {
	defer m.wg.Done()

	// Initial delay before first summary so the agent has produced some output.
	timer := time.NewTimer(summaryInterval)
	defer timer.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-timer.C:
			m.runSummary()
			timer.Reset(summaryInterval)
		}
	}
}

func (m *summaryManager) runSummary() {
	// Snapshot current accumulated messages.
	history := m.msgBuf.Snapshot()
	if len(history) < 3 {
		// Not enough context yet.
		return
	}

	// Filter to clean message state (drop assistant turns with incomplete tool calls).
	cleanMessages := filterIncompleteToolCalls(history)

	prompt := buildSummaryPrompt(m.previousSummary)

	pr, err := processuserinput.ProcessTextPrompt(prompt, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil || len(pr.Messages) == 0 {
		return
	}

	msgs := make([]types.Message, 0, len(cleanMessages)+len(pr.Messages))
	msgs = append(msgs, cleanMessages...)
	msgs = append(msgs, pr.Messages...)

	summaryText := m.querySummary(msgs)
	if summaryText == "" {
		return
	}

	m.previousSummary = summaryText

	// Forward summary via progress callback.
	if m.cfg.ProgressCallback != nil {
		payload := map[string]any{
			"type":    "agent_summary",
			"summary": summaryText,
		}
		payloadJSON, _ := json.Marshal(payload)
		progMsg := types.Message{
			Type: types.MessageTypeProgress,
			UUID: fmt.Sprintf("summary-%d", time.Now().UnixNano()),
			Data: payloadJSON,
		}
		if m.session.ID != "" {
			id := m.session.ID
			progMsg.ParentToolUseID = &id
		}
		m.cfg.ProgressCallback(&progMsg)
	}
}

// querySummary runs a single-turn query to generate a summary of the agent's
// recent activity. No tools are available — the model is instructed to only
// produce a 3-5 word present-tense summary.
func (m *summaryManager) querySummary(msgs []types.Message) string {
	ctx := context.Background()

	qdeps := query.ProductionDeps()
	qdeps.ToolexecutionDeps = toolexecution.ExecutionDeps{
		InvokeTool: func(ctx context.Context, name, _ string, input json.RawMessage) (string, bool, error) {
			return "", true, fmt.Errorf("summary agent cannot use tools")
		},
		// TS parity: canUseTool always returns {behavior: 'deny'}.
	}

	// Minimal system prompt — the summary instruction is in the user message.
	systemPromptParts := []string{
		"Summarize the agent's most recent action.",
	}

	qp := query.QueryParams{
		Messages:        msgs,
		SystemPrompt:    query.AsSystemPrompt(systemPromptParts),
		QuerySource:     types.QuerySource("agent_summary"),
		StreamingParity: true,
		Deps:            &qdeps,
	}
	processuserinput.ApplyQueryHostEnvGates(&qp)

	// Single turn only — no tool execution.
	mt := 1
	qp.MaxTurns = &mt

	// Set agent identity on ToolUseContext for the streaming parity loop.
	qp.ToolUseContext = types.ToolUseContext{}
	if m.session.ID != "" {
		id := m.session.ID
		qp.ToolUseContext.AgentID = &id
	}
	if m.session.AgentType != "" {
		at := m.session.AgentType
		qp.ToolUseContext.AgentType = &at
	}

	var assistantChunks []string
	for y, qerr := range query.Query(ctx, qp) {
		if qerr != nil {
			return ""
		}
		if y.Message == nil || y.Message.Type != types.MessageTypeAssistant {
			continue
		}
		if text := assistantMessageText(*y.Message); strings.TrimSpace(text) != "" {
			assistantChunks = append(assistantChunks, text)
		}
	}

	return strings.Join(assistantChunks, "\n")
}
