package message

import (
	"fmt"
	"strings"

	"goc/gou/textutil"
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

	// in-progress when hint is available (tool_use had pending work).
	// Don't depend on ShouldAnimate which can flicker false when results arrive.
	inProgress := msg.LatestDisplayHint != nil && *msg.LatestDisplayHint != ""
	summary := r.buildSummary(msg, inProgress)
	if inProgress {
		summary += "…" // …
	}
	if len(summary) > width && width > 10 {
		summary = summary[:width-3] + "..."
	}
	lines = append(lines, summary)

	// Add hint line if available
	if inProgress {
		hint := fmt.Sprintf("  ⎿ %s", *msg.LatestDisplayHint) // ⎿
		if len(hint) > width && width > 10 {
			hint = hint[:width-3] + "..."
		}
		lines = append(lines, hint)
	}

	return lines, nil
}

// Measure measures a collapsed group.
func (r *CollapsedGroupRenderer) Measure(msg *types.Message, ctx *RenderContext) (int, error) {
	if msg.LatestDisplayHint != nil && *msg.LatestDisplayHint != "" {
		return 2, nil
	}
	return 1, nil
}

// buildSummary builds the summary line for a collapsed group.
func (r *CollapsedGroupRenderer) buildSummary(msg *types.Message, inProgress bool) string {
	var parts []string

	// Add counts — present progressive when active, past tense when complete
	if msg.ReadCount > 0 {
		if inProgress {
			parts = append(parts, fmt.Sprintf("Reading %d files", msg.ReadCount))
		} else {
			parts = append(parts, fmt.Sprintf("Read %d files", msg.ReadCount))
		}
	}
	if msg.SearchCount > 0 {
		if inProgress {
			parts = append(parts, fmt.Sprintf("Searching for %d patterns", msg.SearchCount))
		} else {
			parts = append(parts, fmt.Sprintf("Searched for %d patterns", msg.SearchCount))
		}
	}
	if msg.ListCount > 0 {
		if inProgress {
			parts = append(parts, fmt.Sprintf("Listing %d items", msg.ListCount))
		} else {
			parts = append(parts, fmt.Sprintf("Listed %d items", msg.ListCount))
		}
	}
	if msg.ReplCount > 0 {
		parts = append(parts, fmt.Sprintf("Ran %d repl", msg.ReplCount))
	}
	if msg.BashCount != nil && *msg.BashCount > 0 {
		parts = append(parts, fmt.Sprintf("Ran %d commands", *msg.BashCount))
	}

	// Add MCP calls if any
	if msg.McpCallCount != nil && *msg.McpCallCount > 0 {
		parts = append(parts, fmt.Sprintf("%d MCP calls", *msg.McpCallCount))
	}

	// Add memory operations
	if msg.MemoryReadCount > 0 {
		parts = append(parts, fmt.Sprintf("Read %d memories", msg.MemoryReadCount))
	}
	if msg.MemoryWriteCount > 0 {
		parts = append(parts, fmt.Sprintf("Saved %d memories", msg.MemoryWriteCount))
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
		return "● Working..." // ●
	}

	return textutil.AssistantBullet() + strings.Join(parts, ", ")
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
	if len(messages) < 2 {
		return false
	}

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
	var msgSlice []types.Message
	for _, msg := range messages {
		msgSlice = append(msgSlice, *msg)
	}

	group := &types.Message{
		Type:     types.MessageTypeCollapsedReadSearch,
		UUID:     groupUUID,
		Messages: msgSlice,
	}

	for _, msg := range messages {
		if isReadSearchMessage(msg) {
			group.ReadCount++
		}
	}

	return group
}

// Helper function to check if a message is a read/search operation
func isReadSearchMessage(msg *types.Message) bool {
	if msg.Type != types.MessageTypeAssistant {
		return false
	}

	content := string(msg.Content)
	if len(content) == 0 && msg.Message != nil {
		content = string(msg.Message)
	}

	return strings.Contains(content, `"name":"Read"`) ||
		strings.Contains(content, `"name":"Grep"`) ||
		strings.Contains(content, `"name":"Glob"`)
}
