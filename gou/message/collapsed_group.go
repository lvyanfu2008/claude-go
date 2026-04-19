package message

import (
	"fmt"
	"strings"

	"goc/types"
)

// CollapsedGroupRenderer renders collapsed read/search groups.
// Similar to TS CollapsedReadSearchContent component.
type CollapsedGroupRenderer struct{}

// CanRender returns true for collapsed read/search groups.
func (r *CollapsedGroupRenderer) CanRender(msg *types.Message) bool {
	return msg.Type == types.MessageTypeCollapsedReadSearch
}

// Render renders a collapsed group.
func (r *CollapsedGroupRenderer) Render(msg *types.Message, ctx *RenderContext) ([]string, error) {
	var lines []string
	width := getContainerWidth(ctx)

	// Build summary line
	summary := r.buildSummary(msg)
	if len(summary) > width && width > 10 {
		summary = summary[:width-3] + "..."
	}
	lines = append(lines, summary)

	// Add hint line if available
	if msg.LatestDisplayHint != nil && *msg.LatestDisplayHint != "" && ctx.ShouldAnimate {
		hint := fmt.Sprintf("  ⎿ %s", *msg.LatestDisplayHint)
		if len(hint) > width && width > 10 {
			hint = hint[:width-3] + "..."
		}
		lines = append(lines, hint)
	}

	return lines, nil
}

// Measure measures a collapsed group.
func (r *CollapsedGroupRenderer) Measure(msg *types.Message, ctx *RenderContext) (int, error) {
	// Collapsed groups are 1 line when completed, 2 lines when active with hint
	if msg.LatestDisplayHint != nil && *msg.LatestDisplayHint != "" && ctx.ShouldAnimate {
		return 2, nil
	}
	return 1, nil
}

// buildSummary builds the summary line for a collapsed group.
func (r *CollapsedGroupRenderer) buildSummary(msg *types.Message) string {
	var parts []string

	// Add counts
	if msg.ReadCount > 0 {
		parts = append(parts, fmt.Sprintf("%d read", msg.ReadCount))
	}
	if msg.SearchCount > 0 {
		parts = append(parts, fmt.Sprintf("%d search", msg.SearchCount))
	}
	if msg.ListCount > 0 {
		parts = append(parts, fmt.Sprintf("%d list", msg.ListCount))
	}
	if msg.ReplCount > 0 {
		parts = append(parts, fmt.Sprintf("%d repl", msg.ReplCount))
	}
	if msg.BashCount != nil && *msg.BashCount > 0 {
		parts = append(parts, fmt.Sprintf("%d bash", *msg.BashCount))
	}

	// Add MCP calls if any
	if msg.McpCallCount != nil && *msg.McpCallCount > 0 {
		parts = append(parts, fmt.Sprintf("%d MCP", *msg.McpCallCount))
	}

	// Add memory operations
	if msg.MemoryReadCount > 0 {
		parts = append(parts, fmt.Sprintf("%d memory read", msg.MemoryReadCount))
	}
	if msg.MemoryWriteCount > 0 {
		parts = append(parts, fmt.Sprintf("%d memory write", msg.MemoryWriteCount))
	}

	// Add Git operations
	gitParts := r.buildGitSummary(msg)
	if gitParts != "" {
		parts = append(parts, gitParts)
	}

	// Add hook info
	if msg.HookCount != nil && *msg.HookCount > 0 {
		parts = append(parts, fmt.Sprintf("%d hook", *msg.HookCount))
	}

	if len(parts) == 0 {
		return "⟳ Working..."
	}

	return fmt.Sprintf("⤿ %s", strings.Join(parts, ", "))
}

// buildGitSummary builds Git operation summary.
func (r *CollapsedGroupRenderer) buildGitSummary(msg *types.Message) string {
	var gitParts []string

	// Commits
	if len(msg.Commits) > 0 {
		gitParts = append(gitParts, fmt.Sprintf("%d commit", len(msg.Commits)))
	}

	// Pushes
	if len(msg.Pushes) > 0 {
		gitParts = append(gitParts, fmt.Sprintf("%d push", len(msg.Pushes)))
	}

	// Branches
	if len(msg.Branches) > 0 {
		gitParts = append(gitParts, fmt.Sprintf("%d branch", len(msg.Branches)))
	}

	// PRs
	if len(msg.Prs) > 0 {
		gitParts = append(gitParts, fmt.Sprintf("%d PR", len(msg.Prs)))
	}

	if len(gitParts) == 0 {
		return ""
	}

	return fmt.Sprintf("Git: %s", strings.Join(gitParts, ", "))
}

// ShouldCollapseMessages checks if messages should be collapsed into a group.
func ShouldCollapseMessages(messages []*types.Message) bool {
	// TODO: Implement proper collapse logic
	// Similar to TS shouldCollapseReadSearch function
	// Based on message types and timing

	if len(messages) < 2 {
		return false
	}

	// Check if we have consecutive read/search operations
	readSearchCount := 0
	for _, msg := range messages {
		if isReadSearchMessage(msg) {
			readSearchCount++
		} else {
			break
		}
	}

	return readSearchCount >= 2
}

// CreateCollapsedGroup creates a collapsed group from messages.
func CreateCollapsedGroup(messages []*types.Message, groupUUID string) *types.Message {
	// TODO: Implement proper group creation
	// Similar to TS createCollapsedReadSearchGroup function

	// Convert []*types.Message to []types.Message
	var msgSlice []types.Message
	for _, msg := range messages {
		msgSlice = append(msgSlice, *msg)
	}

	group := &types.Message{
		Type:     types.MessageTypeCollapsedReadSearch,
		UUID:     groupUUID,
		Messages: msgSlice,
	}

	// Count operations
	for _, msg := range messages {
		// TODO: Count different types of operations
		_ = msg // Use variable to avoid unused error
	}

	return group
}

// Helper function to check if a message is a read/search operation
func isReadSearchMessage(msg *types.Message) bool {
	if msg.Type != types.MessageTypeAssistant {
		return false
	}

	// TODO: Check if message contains Read, Grep, or Glob tool uses
	// This requires parsing the message content
	return false
}