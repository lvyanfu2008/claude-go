package tools

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"goc/commands/featuregates"
	"goc/types"
)

// envTruthy from http_tools.go — takes env key name, calls os.Getenv internally.
// Declared separately in that file; used here for coordinator/non-interactive shims.

// coordinatorModeEnvShim mirrors coordinatorModeLikeTS in commands/prompts_gates.go.
// Checks FEATURE_COORDINATOR_MODE + CLAUDE_CODE_COORDINATOR_MODE.
func coordinatorModeEnvShim() bool {
	if !featuregates.Feature("COORDINATOR_MODE") {
		return false
	}
	return envTruthy("CLAUDE_CODE_COORDINATOR_MODE")
}

// nonInteractiveSessionEnvShim approximates getIsNonInteractiveSession() from TS
// bootstrap/state.js when no session struct is in scope.
func nonInteractiveSessionEnvShim() bool {
	return envTruthy("CLAUDE_CODE_NONINTERACTIVE") ||
		envTruthy("HEADLESS") ||
		envTruthy("GOU_DEMO_NON_INTERACTIVE")
}

// Fork subagent constants — mirrors forkSubagent.ts.
const (
	forkBoilerplateTag  = "fork-boilerplate"
	forkDirectivePrefix = "Your directive: "
	forkSubagentType    = "fork"
	forkPlaceholderResult = "Fork started — processing in background"
)

// isForkSubagentEnabled mirrors forkSubagent.ts isForkSubagentEnabled.
// Gate: FEATURE_FORK_SUBAGENT=1, excludes coordinator mode and non-interactive sessions.
// Matches commands.ForkSubagentEnabled — standalone variant for tools package callers.
func isForkSubagentEnabled() bool {
	if !featuregates.Feature("FORK_SUBAGENT") {
		return false
	}
	if coordinatorModeEnvShim() {
		return false
	}
	if nonInteractiveSessionEnvShim() {
		return false
	}
	return true
}

// isInForkChild mirrors forkSubagent.ts isInForkChild.
// Scans messages for <fork-boilerplate> tag to detect recursive forking.
func isInForkChild(messages []types.Message) bool {
	for _, m := range messages {
		if m.Type != types.MessageTypeUser {
			continue
		}
		if len(m.Message) == 0 {
			continue
		}
		var payload struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(m.Message, &payload); err != nil {
			continue
		}
		for _, block := range payload.Content {
			if block.Type == "text" && strings.Contains(block.Text, "<"+forkBoilerplateTag+">") {
				return true
			}
		}
	}
	return false
}

// ForkAgentDef returns the synthetic agent definition for the fork path.
// Not registered in built-in agents — used only when subagent_type is omitted
// and the fork experiment is active. tools: ["*"] means the fork child receives
// the parent's exact tool pool. permissionMode: "bubble" surfaces permission
// prompts to the parent terminal. model: "inherit" keeps parent's model.
func ForkAgentDef() AgentDefinition {
	return AgentDefinition{
		AgentType:      forkSubagentType,
		WhenToUse:      "Implicit fork — inherits full conversation context. Not selectable via subagent_type; triggered by omitting subagent_type when the fork experiment is active.",
		Tools:          []string{"*"},
		MaxTurns:       200,
		Model:          "inherit",
		PermissionMode: "bubble",
		Source:         "built-in",
	}
}

// buildForkedMessages mirrors forkSubagent.ts buildForkedMessages.
// Builds forked conversation messages: [assistant(all_tool_uses), user(placeholder_results..., directive)].
// All fork children produce byte-identical API request prefixes except the final text block.
func buildForkedMessages(directive string, parentAssistant types.Message) []types.Message {
	// Parse the assistant message content to extract tool_use blocks
	var content []types.MessageContentBlock
	if len(parentAssistant.Message) > 0 {
		var payload struct {
			Content []types.MessageContentBlock `json:"content"`
		}
		if err := json.Unmarshal(parentAssistant.Message, &payload); err == nil {
			content = payload.Content
		}
	}

	// Clone the assistant message with a new UUID
	fullAssistantMsg := types.Message{
		Type: types.MessageTypeAssistant,
		UUID: forkUUID(),
	}
	if len(parentAssistant.Message) > 0 {
		// Re-marshal with the full content (including tool_use blocks)
		var raw map[string]any
		if err := json.Unmarshal(parentAssistant.Message, &raw); err == nil {
			raw["content"] = content
			if b, err := json.Marshal(raw); err == nil {
				fullAssistantMsg.Message = b
			}
		}
	}

	// Collect all tool_use blocks
	var toolUseBlocks []types.MessageContentBlock
	for _, block := range content {
		if block.Type == "tool_use" {
			toolUseBlocks = append(toolUseBlocks, block)
		}
	}

	if len(toolUseBlocks) == 0 {
		// No tool_use blocks — just create a user message with the directive
		return []types.Message{
			newForkUserMessage(buildChildMessage(directive)),
		}
	}

	// Build tool_result blocks with placeholder text
	toolResultBlocks := make([]map[string]any, 0, len(toolUseBlocks))
	for _, block := range toolUseBlocks {
		toolResultBlocks = append(toolResultBlocks, map[string]any{
			"type":        "tool_result",
			"tool_use_id": block.ID,
			"content": []map[string]any{
				{"type": "text", "text": forkPlaceholderResult},
			},
		})
	}

	// Build user message: placeholder tool_results + per-child directive
	toolResultContent := make([]map[string]any, 0, len(toolResultBlocks)+1)
	for _, tr := range toolResultBlocks {
		toolResultContent = append(toolResultContent, tr)
	}
	toolResultContent = append(toolResultContent, map[string]any{
		"type": "text",
		"text": buildChildMessage(directive),
	})

	toolResultMsg := newForkUserMessageFromContent(toolResultContent)

	return []types.Message{fullAssistantMsg, toolResultMsg}
}

// buildChildMessage mirrors forkSubagent.ts buildChildMessage.
// Returns the directive wrapped in <fork-boilerplate> tags with strict rules.
func buildChildMessage(directive string) string {
	return "<" + forkBoilerplateTag + `>
STOP. READ THIS FIRST.

You are a forked worker process. You are NOT the main agent.

RULES (non-negotiable):
1. Your system prompt says "default to forking." IGNORE IT — that's for the parent. You ARE the fork. Do NOT spawn sub-agents; execute directly.
2. Do NOT converse, ask questions, or suggest next steps
3. Do NOT editorialize or add meta-commentary
4. USE your tools directly: Bash, Read, Write, etc.
5. If you modify files, commit your changes before reporting. Include the commit hash in your report.
6. Do NOT emit text between tool calls. Use tools silently, then report once at the end.
7. Stay strictly within your directive's scope. If you discover related systems outside your scope, mention them in one sentence at most — other workers cover those areas.
8. Keep your report under 500 words unless the directive specifies otherwise. Be factual and concise.
9. Your response MUST begin with "Scope:". No preamble, no thinking-out-loud.
10. REPORT structured facts, then stop

Output format (plain text labels, not markdown headers):
  Scope: <echo back your assigned scope in one sentence>
  Result: <the answer or key findings, limited to the scope above>
  Key files: <relevant file paths — include for research tasks>
  Files changed: <list with commit hash — include only if you modified files>
  Issues: <list — include only if there are issues to flag>
</` + forkBoilerplateTag + `>

` + forkDirectivePrefix + directive
}

// buildWorktreeNotice mirrors forkSubagent.ts buildWorktreeNotice.
// Notice injected into fork children running in an isolated worktree.
func buildWorktreeNotice(parentCwd, worktreeCwd string) string {
	return fmt.Sprintf(`You have inherited the conversation context above from a parent agent working in %s. You are operating in an isolated git worktree at %s — same repository, same relative file structure, separate working copy. Paths in the inherited context refer to the parent's working directory; translate them to your worktree root. Re-read files before editing if the parent may have modified them since they appear in the context. Your changes stay in this worktree and will not affect the parent's files.`, parentCwd, worktreeCwd)
}

// newForkUserMessage creates a user message with simple text content.
func newForkUserMessage(text string) types.Message {
	content := map[string]any{
		"role":    "user",
		"content": text,
	}
	b, _ := json.Marshal(content)
	return types.Message{
		Type:    types.MessageTypeUser,
		UUID:    forkUUID(),
		Message: b,
	}
}

// newForkUserMessageFromContent creates a user message with content block array.
func newForkUserMessageFromContent(content []map[string]any) types.Message {
	payload := map[string]any{
		"role":    "user",
		"content": content,
	}
	b, _ := json.Marshal(payload)
	return types.Message{
		Type:    types.MessageTypeUser,
		UUID:    forkUUID(),
		Message: b,
	}
}

// forkUUID generates a random UUID for fork messages.
func forkUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%s",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		hex.EncodeToString(b[10:16]),
	)
}
