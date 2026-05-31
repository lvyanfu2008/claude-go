# Message Rendering Refactor Implementation Plan

> **For agentic workers:** Use TaskCreate + TaskUpdate to track progress step-by-step. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Refactor claude-go message display to match claude-code (TS) rendering — unify rendering paths, add MessageRow layer, enhance RenderContext, fix markdown rendering, delete dead code.

**Architecture:** Three layers — MessagesForScrollList (preprocess, unchanged) → MessageRow (NEW, per-message context) → Dispatcher/Renderer (enhanced with new RenderContext fields). Streaming moves from post-hoc append into first-class rendering.

**Tech Stack:** Go, lipgloss v2, chroma (syntax highlighting), Bubble Tea v2 viewport

---

### Task 1: Enhance RenderContext with message-level fields

**Files:**
- Modify: `gou/message/renderer.go`

- [ ] **Step 1: Add new fields to RenderContext**

In `gou/message/renderer.go`, replace the current `RenderContext` struct:

```go
// RenderContext contains rendering context information.
type RenderContext struct {
	Width           int
	Verbose         bool
	Theme           *theme.Palette
	IsTranscript    bool
	IsStatic        bool
	ShouldAnimate   bool
	ShouldShowDot   bool
	AddMargin       bool
	ContainerWidth  *int
	Style           string // "condensed" or empty
	IsUserContinuation bool
	Highlighter     *markdown.Highlighter

	// NEW: per-message context (computed by MessageRow)
	IsActiveCollapsedGroup bool
	IsInProgress           bool

	// NEW: shared state (same across all messages in a render pass)
	InProgressToolUseIDs  map[string]struct{}
	StreamingToolUseIDs   map[string]struct{}
	ResolvedToolUseIDs    map[string]struct{}

	// NEW: transcript features
	SearchHighlight        string
	ShowToolUseCtrlOHint   bool
	ShowResolvedToolStats  bool

	// NEW: streaming state
	StreamingText         string
	StreamingThinkingText string
}
```

- [ ] **Step 2: Build to verify no breakage**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/message/...
```
Expected: PASS (new fields are only additions, no callers use them yet)

- [ ] **Step 3: Commit**

```bash
git add gou/message/renderer.go
git commit -m "feat: add message-level fields to RenderContext"
```

---

### Task 2: Move markdown styling into gou/markdown package

**Files:**
- Modify: `gou/markdown/render.go` (add styled renderer)
- Reference: `gou/app/view.go:643` (styleMarkdownTokens — the target behavior)

The new renderer's `renderMarkdown` (`gou/message/renderer.go:102`) calls `RenderTokensWithHighlight` which outputs raw syntax. We need a styled equivalent that matches `styleMarkdownTokens` behavior.

- [ ] **Step 1: Add RenderTokensStyled function to gou/markdown/render.go**

```go
// RenderTokensStyled renders markdown tokens with lipgloss styles matching TS behavior.
// It removes raw markdown syntax (#, ```, >, ---) and applies styles:
// - headings: bold + heading color
// - code blocks: syntax highlighting via highlighter, or faint fallback
// - blockquotes: italic
// - horizontal rules: faint
// - lists: proper indentation
func RenderTokensStyled(
	toks []Token,
	highlighter *Highlighter,
	cols int,
	baseStyle lipgloss.Style,
	codeStyle lipgloss.Style,
	boldStyle lipgloss.Style,
	italicStyle lipgloss.Style,
	inlineCodeStyle lipgloss.Style,
) string {
	if len(toks) == 0 {
		return ""
	}
	var parts []string
	for _, t := range toks {
		switch t.Type {
		case "heading":
			lv := min(max(t.Level, 1), 6)
			levelPad := strings.Repeat(" ", (lv-1)*2)
			hst := boldStyle
			if len(t.Segments) > 0 {
				inner := renderInlineSegmentsStyled(t.Segments, baseStyle, boldStyle, italicStyle, inlineCodeStyle)
				parts = append(parts, wrapHeading(inner, levelPad, cols, hst))
			} else {
				parts = append(parts, wrapHeading(hst.Render(t.Text), levelPad, cols, hst))
			}
		case "code":
			var highlighted string
			if highlighter != nil {
				h, err := highlighter.HighlightCode(t.Text, t.Lang)
				if err == nil && h != "" {
					highlighted = h
				}
			}
			if highlighted != "" {
				parts = append(parts, baseStyle.Render(highlighted))
			} else {
				cb := "```" + t.Lang + "\n" + t.Text
				if t.Text != "" && !strings.HasSuffix(t.Text, "\n") {
					cb += "\n"
				}
				cb += "```"
				parts = append(parts, codeStyle.Render(cb))
			}
		case "list_item":
			indent := strings.Repeat(" ", t.ListIndent)
			var prefix string
			if t.ListContinuation {
				prefix = indent + "   "
			} else if t.ListOrdered && t.ListIndex > 0 {
				prefix = indent + fmt.Sprintf("%d. ", t.ListIndex)
			} else {
				prefix = indent + "- "
			}
			if len(t.Segments) > 0 {
				parts = append(parts, renderInlineSegmentsStyled(t.Segments, baseStyle, boldStyle, italicStyle, inlineCodeStyle).withPrefix(prefix))
			} else {
				parts = append(parts, baseStyle.Render(prefix+t.Text))
			}
		case "blockquote":
			pref := "> "
			if len(t.Segments) > 0 {
				inner := renderInlineSegmentsStyled(t.Segments, baseStyle, boldStyle, italicStyle, inlineCodeStyle)
				parts = append(parts, italicStyle.Render(pref+strings.ReplaceAll(inner.plain, "\n", "\n"+pref)))
			} else {
				parts = append(parts, italicStyle.Render(pref+strings.ReplaceAll(t.Text, "\n", "\n"+pref)))
			}
		case "hr":
			parts = append(parts, baseStyle.Faint(true).Render("---"))
		case "paragraph":
			if len(t.Segments) > 0 {
				parts = append(parts, renderInlineSegmentsStyled(t.Segments, baseStyle, boldStyle, italicStyle, inlineCodeStyle).plain)
			} else {
				parts = append(parts, baseStyle.Render(t.Text))
			}
		default:
			parts = append(parts, baseStyle.Render(t.Text))
		}
	}
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			if toks[i-1].Type == "list_item" {
				b.WriteByte('\n')
			} else {
				b.WriteString("\n\n")
			}
		}
		b.WriteString(part)
	}
	return strings.TrimSpace(b.String())
}
```

- [ ] **Step 2: Build to verify**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/markdown/...
```
Expected: PASS (or fix compilation errors)

- [ ] **Step 3: Commit**

```bash
git add gou/markdown/render.go
git commit -m "feat: add RenderTokensStyled for styled markdown output"
```

---

### Task 3: Rewrite renderMarkdown in gou/message to use styled renderer

**Files:**
- Modify: `gou/message/renderer.go`

- [ ] **Step 1: Replace renderMarkdown function**

In `gou/message/renderer.go`, replace the current `renderMarkdown` and `paletteToLipglossStyle` functions:

```go
// renderMarkdown renders markdown text with proper theme styles (no raw syntax).
func renderMarkdown(text string, width int, palette *theme.Palette, highlighter *markdown.Highlighter) []string {
	if text == "" {
		return []string{""}
	}
	tokens := markdown.ParseWithGoldmark(text)

	baseStyle := lipgloss.NewStyle().Foreground(palette.Text)
	codeStyle := lipgloss.NewStyle().Faint(true)
	boldStyle := lipgloss.NewStyle().Foreground(palette.Heading).Bold(true)
	italicStyle := lipgloss.NewStyle().Italic(true)
	inlineCodeStyle := lipgloss.NewStyle().Foreground(palette.InlineCode)

	rendered := markdown.RenderTokensStyled(
		tokens, highlighter, width,
		baseStyle, codeStyle, boldStyle, italicStyle, inlineCodeStyle,
	)

	lines := strings.Split(rendered, "\n")
	var result []string
	for _, line := range lines {
		hasAnsi := strings.Contains(line, "\x1b[")
		if hasAnsi {
			result = append(result, line)
		} else {
			visibleLen := len(line)
			if width > 0 && visibleLen > width {
				wrapped := wrapText(line, width)
				result = append(result, wrapped...)
			} else {
				result = append(result, line)
			}
		}
	}
	return result
}
```

Delete `paletteToLipglossStyle` (no longer needed).

- [ ] **Step 2: Build**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/message/...
```
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add gou/message/renderer.go
git commit -m "fix: use styled markdown rendering in renderMarkdown"
```

---

### Task 4: Create MessageRow layer

**Files:**
- Create: `gou/message/message_row.go`

- [ ] **Step 1: Create message_row.go**

```go
package message

import (
	"goc/types"
)

// MessageRowContext carries per-message context computed by MessageRow.
type MessageRowContext struct {
	// IsUserContinuation is true when the previous message is also a user message.
	IsUserContinuation bool
	// IsActiveCollapsedGroup is true when a collapsed group's tools are still executing.
	IsActiveCollapsedGroup bool
	// ShouldAnimate is true when the message content may still be changing.
	ShouldAnimate bool
	// IsInProgress is true when a tool_use has not yet received its result.
	IsInProgress bool
}

// MessageRowBuildOpts are the inputs for building per-message row contexts.
type MessageRowBuildOpts struct {
	TranscriptMode       bool
	Verbose              bool
	InProgressToolUseIDs map[string]struct{}
	StreamingToolUseIDs  map[string]struct{}
	ResolvedToolUseIDs   map[string]struct{}
	SearchHighlight      string
	Columns              int
	Loading              bool
}

// BuildMessageRowContexts computes per-message rendering contexts.
// Mirrors MessageRow.tsx context computation.
func BuildMessageRowContexts(
	messages []*types.Message,
	opts MessageRowBuildOpts,
) []*MessageRowContext {
	if len(messages) == 0 {
		return nil
	}

	contexts := make([]*MessageRowContext, len(messages))
	for i, msg := range messages {
		ctx := &MessageRowContext{}

		// isUserContinuation: previous message also user
		if i > 0 && messages[i-1].Type == types.MessageTypeUser {
			ctx.IsUserContinuation = msg.Type == types.MessageTypeUser
		}

		// isInProgress: tool_use IDs for this assistant message vs resolved set
		ctx.IsInProgress = !allToolUsesResolved(msg, opts.ResolvedToolUseIDs)

		// shouldAnimate: streaming or in-progress
		ctx.ShouldAnimate = opts.Loading && (hasStreamingTools(msg, opts.StreamingToolUseIDs) || ctx.IsInProgress)

		// isActiveCollapsedGroup (see TS hasAnyToolInProgress + isLoading + !hasContentAfter)
		if msg.Type == types.MessageTypeCollapsedReadSearch {
			ctx.IsActiveCollapsedGroup = hasAnyToolInProgress(msg, opts.InProgressToolUseIDs) ||
				(opts.Loading && !hasContentAfterIndex(messages, i))
		}

		contexts[i] = ctx
	}
	return contexts
}

// hasContentAfterIndex checks if any non-progress content follows the given index.
func hasContentAfterIndex(messages []*types.Message, idx int) bool {
	for i := idx + 1; i < len(messages); i++ {
		msg := messages[i]
		switch msg.Type {
		case types.MessageTypeProgress, types.MessageTypeAttachment:
			continue
		case types.MessageTypeSystem:
			continue
		case types.MessageTypeUser:
			continue // tool_results follow collapsed groups
		default:
			return true
		}
	}
	return false
}

// allToolUsesResolved checks whether every tool_use in the message has a resolved result.
func allToolUsesResolved(msg *types.Message, resolved map[string]struct{}) bool {
	if resolved == nil || len(resolved) == 0 {
		return false
	}
	content, err := parseMessageContent(msg)
	if err != nil {
		return true // can't determine, assume resolved
	}
	for _, block := range content {
		if blockType, _ := block["type"].(string); blockType == "tool_use" {
			if id, _ := block["id"].(string); id != "" {
				if _, ok := resolved[id]; !ok {
					return false
				}
			}
		}
	}
	return true
}

// hasAnyToolInProgress checks if any tool_use in the message is in the in-progress set.
func hasAnyToolInProgress(msg *types.Message, inProgress map[string]struct{}) bool {
	if inProgress == nil || len(inProgress) == 0 {
		return false
	}
	content, err := parseMessageContent(msg)
	if err != nil {
		return false
	}
	for _, block := range content {
		if blockType, _ := block["type"].(string); blockType == "tool_use" {
			if id, _ := block["id"].(string); id != "" {
				if _, ok := inProgress[id]; ok {
					return true
				}
			}
		}
	}
	return false
}

// hasStreamingTools checks if any tool_use in the message is in the streaming set.
func hasStreamingTools(msg *types.Message, streaming map[string]struct{}) bool {
	if streaming == nil || len(streaming) == 0 {
		return false
	}
	content, err := parseMessageContent(msg)
	if err != nil {
		return false
	}
	for _, block := range content {
		if blockType, _ := block["type"].(string); blockType == "tool_use" {
			if id, _ := block["id"].(string); id != "" {
				if _, ok := streaming[id]; ok {
					return true
				}
			}
		}
	}
	return false
}
```

- [ ] **Step 2: Build**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/message/...
```
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add gou/message/message_row.go
git commit -m "feat: add MessageRow layer for per-message context computation"
```

---

### Task 5: Fix AssistantMessageRenderer — isInProgress + markdown

**Files:**
- Modify: `gou/message/assistant_message.go`

- [ ] **Step 1: Use ctx.IsInProgress instead of hardcoded false**

In `renderContentBlock` (line 79) and `measureContentBlock` (line 109), replace `isInProgress := false` with `isInProgress := ctx.IsInProgress`.

- [ ] **Step 2: Add ⎿ (MessageResponse) prefix for tool_use after text**

In `renderContentBlock`, when rendering a `tool_use` block after a `text` block, prefix with `⎿`:

```go
case "tool_use":
	if r.toolUseRenderer == nil {
		r.toolUseRenderer = &ToolUseMessageRenderer{}
	}
	isInProgress := ctx.IsInProgress
	blockLines, err := r.toolUseRenderer.RenderToolUseBlock(block, ctx, isInProgress)
	if err != nil {
		return nil, err
	}
	// Insert ⎿ prefix when switching from text to tool_use (TS MessageResponse)
	if index > 0 {
		if prevBlock, ok := content[index-1]["type"].(string); ok && prevBlock == "text" {
			blockLines = append([]string{"  ⎿"}, blockLines...)
		}
	}
	return blockLines, nil
```

- [ ] **Step 3: Add SearchHighlight to text rendering**

In `renderTextBlock`, apply search highlighting if `ctx.SearchHighlight` is non-empty:

```go
func (r *AssistantMessageRenderer) renderTextBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	text, _ := block["text"].(string)
	trimmed := strings.TrimSpace(text)

	if compactservice.IsRateLimitErrorMessage(text) {
		return r.renderRateLimitError(text, ctx)
	}
	if compactservice.StartsWithApiErrorPrefix(trimmed) {
		return r.renderApiError(text, ctx)
	}

	// Apply search highlight before markdown rendering
	if ctx.SearchHighlight != "" {
		text = highlightSearchPlain(text, ctx.SearchHighlight)
	}

	contentWidth := getContainerWidth(ctx) - 3
	if contentWidth < 20 {
		contentWidth = 20
	}
	lines := renderMarkdown(text, contentWidth, ctx.Theme, ctx.Highlighter)
	// ... rest unchanged
```

- [ ] **Step 4: Build + verify**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/message/...
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gou/message/assistant_message.go
git commit -m "fix: use ctx.IsInProgress, add ⎿ prefix, search highlight in AssistantMessageRenderer"
```

---

### Task 6: Fix UserMessageRenderer

**Files:**
- Modify: `gou/message/user_message.go`

- [ ] **Step 1: Add isUserContinuation behavior**

In `styleUserLines`, suppress `❯` prefix when `ctx.IsUserContinuation`:

```go
func (r *UserMessageRenderer) styleUserLines(lines []string, ctx *RenderContext) []string {
	containerWidth := getContainerWidth(ctx)
	userStyle := lipgloss.NewStyle().
		Background(ctx.Theme.UserMessageBackground).
		Foreground(ctx.Theme.UserMessageText).
		Bold(true).
		Width(containerWidth)
	for i, line := range lines {
		if i == 0 {
			if ctx.IsUserContinuation {
				line = "  " + line // continuation: no ❯
			} else {
				line = "❯ " + line
			}
		} else {
			line = "  " + line
		}
		lines[i] = userStyle.Render(line)
	}
	return lines
}
```

- [ ] **Step 2: Add SearchHighlight to text blocks**

In `renderTextBlock`, apply highlight before markdown:

```go
func (r *UserMessageRenderer) renderTextBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	text, _ := block["text"].(string)
	// Apply search highlight
	if ctx.SearchHighlight != "" {
		text = highlightSearchPlain(text, ctx.SearchHighlight)
	}
	// ... rest unchanged
```

- [ ] **Step 3: Add ShowResolvedToolStats to tool_result rendering**

In `renderToolResultBlock`, when `ctx.ShowResolvedToolStats` is true and the result is available, display the `⎿` resolved hint:

```go
func (r *UserMessageRenderer) renderToolResultBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	// ... existing diff/WIP handling ...

	if ctx.ShowResolvedToolStats {
		toolUseID, _ := block["tool_use_id"].(string)
		if _, resolved := ctx.ResolvedToolUseIDs[toolUseID]; resolved {
			// Show resolved tool stats
			toolName := extractToolNameFromToolUseID(toolUseID)
			hint := toolName + " completed"
			return []string{"  ⎿  " + hint}, nil
		}
	}

	// ... rest unchanged
```

- [ ] **Step 4: Build + verify**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/message/...
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gou/message/user_message.go
git commit -m "feat: add isUserContinuation, search highlight, resolved stats to UserMessageRenderer"
```

---

### Task 7: Fix ToolUseMessageRenderer, CollapsedGroupRenderer, GroupedToolUseRenderer

**Files:**
- Modify: `gou/message/tool_use.go`
- Modify: `gou/message/collapsed_group.go`
- Modify: `gou/message/grouped_tool_use.go`

- [ ] **Step 1: Add ctrl+o hint and resolved stats to ToolUseMessageRenderer**

In `RenderToolUseBlock`, after building the base line, append ctrl+o hint when `ctx.ShowToolUseCtrlOHint`:

```go
func (r *ToolUseMessageRenderer) RenderToolUseBlock(block map[string]interface{}, ctx *RenderContext, isInProgress bool) ([]string, error) {
	// ... existing chrome logic ...

	// After building the tool line, check resolved state
	toolUseID, _ := block["id"].(string)
	resolved := false
	if ctx.ResolvedToolUseIDs != nil {
		_, resolved = ctx.ResolvedToolUseIDs[toolUseID]
	}

	if ctx.ShowResolvedToolStats && resolved {
		// Show resolved hint
		facing, _, _ := messagerow.ToolChromeParts(name, inputJSON)
		hint, extra := messagerow.TranscriptResolvedHintExtra(facing, inputJSON)
		if hint != "" {
			lines = append(lines, "  ⎿  "+hint)
			if extra != "" {
				lines = append(lines, "     "+extra)
			}
		}
	} else if !resolved && ctx.ShowToolUseCtrlOHint {
		// Show ctrl+o hint line
		lines[len(lines)-1] += " (ctrl+o to expand)"
	}

	return lines, nil
}
```

- [ ] **Step 2: Use ctx.IsActiveCollapsedGroup in CollapsedGroupRenderer**

```go
func (r *CollapsedGroupRenderer) Render(msg *types.Message, ctx *RenderContext) ([]string, error) {
	// Use ctx.IsActiveCollapsedGroup instead of local heuristic
	inProgress := ctx.IsActiveCollapsedGroup
	
	// If no explicit active flag, fall back to hint presence
	if !inProgress {
		inProgress = msg.LatestDisplayHint != nil && *msg.LatestDisplayHint != ""
	}
	// ... rest unchanged
}
```

- [ ] **Step 3: Use ctx.ShouldAnimate in GroupedToolUseRenderer**

In `Render`, conditionally show animation indicator based on `ctx.ShouldAnimate`:

```go
func (r *GroupedToolUseRenderer) Render(msg *types.Message, ctx *RenderContext) ([]string, error) {
	// ... existing rendering ...
	// When shouldAnimate, show in-progress indicator
	if ctx.ShouldAnimate {
		lines[0] = "● " + lines[0] // animate dot
	}
	return lines, nil
}
```

- [ ] **Step 4: Build + verify**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/message/...
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gou/message/tool_use.go gou/message/collapsed_group.go gou/message/grouped_tool_use.go
git commit -m "fix: ctrl+o hint, resolved stats, active collapsed group in tool renderers"
```

---

### Task 8: Slim VirtualList — remove dead code

**Files:**
- Modify: `gou/message/virtual_list.go`

- [ ] **Step 1: Remove dead code from virtual_list.go**

Delete these items:
- `BuildDisplayList` function (entire body + type)
- `DisplayItem` struct
- `determineSpacingBefore` function
- `determineSpacingAfter` function
- `shouldAddSpacing` function
- `measureMessageHeight` function
- `collapseReadSearchOperations` function (if exists)
- `ProcessMessagesForDisplay` function (if exists)

Keep:
- `VirtualList` struct
- `NewVirtualList`
- `RenderRange`
- `ComputeVisibleRange`
- `InvalidateCache` / `InvalidateAllCache`
- `GetMessageHeight` (but simplified)

The resulting file keeps only the core virtual scrolling functions.

- [ ] **Step 2: Build**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/message/...
```
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add gou/message/virtual_list.go
git commit -m "refactor: remove dead code from VirtualList"
```

---

### Task 9: Delete processor.go

**Files:**
- Delete: `gou/message/processor.go`

- [ ] **Step 1: Remove processor reference from MessageRendererIntegration**

In `gou/app/message_renderer_integration.go`, remove:
1. The `processor` field from `MessageRendererIntegration` struct
2. `message.NewProcessor()` call in `NewMessageRendererIntegration`
3. The `ProcessMessages` method (dead pass-through)

- [ ] **Step 2: Delete the file**

```bash
rm gou/message/processor.go
```

- [ ] **Step 3: Build full project**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./...
```
Expected: PASS (no references remain)

- [ ] **Step 4: Commit**

```bash
git rm gou/message/processor.go
git add gou/app/message_renderer_integration.go
git commit -m "refactor: delete dead processor.go and its references"
```

---

### Task 10: Integrate streaming into render pipeline

**Files:**
- Modify: `gou/message/message_row.go` (add streaming render)
- Modify: `gou/app/message_renderer_integration.go` (remove separate streaming append)

Streaming content (StreamingText, StreamingThinkingText, StreamingToolUses) currently rendered separately after the message list. We move this into the render pipeline so it's part of the regular message rendering flow.

- [ ] **Step 1: Add RenderStreamingTail to message_row.go**

```go
// RenderStreamingTail renders streaming content (text, thinking, tool uses) that
// hasn't been committed to messages yet. Mirrors TS behavior where streaming
// appears as continuation of the last assistant message.
func RenderStreamingTail(
	streamingText string,
	streamingThinking string,
	streamingToolUses []StreamingToolUse,
	ctx *RenderContext,
) []string {
	var lines []string

	// Streaming thinking
	if strings.TrimSpace(streamingThinking) != "" {
		lines = append(lines, "\x1b[2;3m∴ Thinking\x1b[0m")
	}

	// Streaming text
	if strings.TrimSpace(streamingText) != "" {
		textLines := renderMarkdown(streamingText, getContainerWidth(ctx), ctx.Theme, ctx.Highlighter)
		for _, l := range textLines {
			lines = append(lines, "⏺ "+l)
		}
	}

	// Streaming tool uses
	for _, tu := range streamingToolUses {
		facing, paren, hint := messagerow.ToolChromeParts(tu.Name, json.RawMessage(tu.Input))
		line := "  ⎿ " + facing
		if paren != "" {
			line += " (" + paren + ")"
		}
		if hint != "" {
			line += "\n     " + hint
		}
		line += "…"
		if ctx.ShowToolUseCtrlOHint {
			line += " (ctrl+o to expand)"
		}
		lines = append(lines, line)
	}

	return lines
}

// StreamingToolUse represents an in-flight tool_use from the store.
type StreamingToolUse struct {
	Name  string
	Input string
}
```

- [ ] **Step 2: Build**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/message/...
```
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add gou/message/message_row.go
git commit -m "feat: add RenderStreamingTail for unified streaming rendering"
```

---

### Task 11: Simplify message_renderer_integration.go

**Files:**
- Modify: `gou/app/message_renderer_integration.go`

Remove the duplicated streaming logic (lines ~246-524) from both `renderMessagePaneWithNewRenderer` and `tryBuildFullMessagePaneContentWithNewRenderer`. Replace with a call to `message.RenderStreamingTail`.

- [ ] **Step 1: Simplify renderMessagePaneWithNewRenderer**

Remove the entire streaming append block (lines 248-370). Replace with:

```go
// Add streaming tail using unified renderer
if m.uiScreen != gouDemoScreenTranscript {
	streamingCtx := &message.RenderContext{
		Width:         width,
		Theme:         m.msgRenderer.currentTheme,
		IsTranscript:  false,
		Verbose:       verbose,
		ShouldAnimate: shouldAnimate,
		ShouldShowDot: shouldShowDot,
		Highlighter:   m.msgRenderer.highlighter,
		ShowToolUseCtrlOHint: true,
	}
	tailLines := message.RenderStreamingTail(
		m.store.StreamingText,
		m.store.StreamingThinkingText,
		convertStreamingToolUses(m.store.StreamingToolUses),
		streamingCtx,
	)
	if len(tailLines) > 0 {
		if content != "" {
			content += "\n"
		}
		content += strings.Join(tailLines, "\n")
	}
}
```

- [ ] **Step 2: Same simplification for tryBuildFullMessagePaneContentWithNewRenderer**

Remove lines 401-524, replace with the same call to `RenderStreamingTail`.

- [ ] **Step 3: Remove dead helper functions**

Delete these dead functions from the file:
- `groupStreamingTools` (if only used in the deleted streaming code)
- `extractPartialJSONField` (if only used in the deleted streaming code)
- `applyMessagePaneGutter` (if only used in the deleted streaming code)
- `applyAssistantStreamingGutter` (if only used in the deleted streaming code)

- [ ] **Step 4: Build full project**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./...
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gou/app/message_renderer_integration.go
git commit -m "refactor: simplify streaming rendering in integration layer"
```

---

### Task 12: Simplify message_pane.go — remove fallback to old renderer

**Files:**
- Modify: `gou/app/message_pane.go`
- Modify: `gou/app/message_viewport_pane.go` (remove fallback flag)

- [ ] **Step 1: Simplify renderMessagePane**

In `gou/app/message_pane.go`, remove the condition that falls back to old renderer. The new renderer is the only path:

```go
func (m *Model) renderMessagePane(b *strings.Builder, vpH, bodyCols int, useVp bool) {
	// Always use viewport path (new renderer)
	b.WriteString(m.messagePaneViewportBlock(vpH, bodyCols))
	b.WriteByte('\n')
}
```

- [ ] **Step 2: Remove msgViewportFallback**

In `gou/app/message_viewport_pane.go`, remove `msgViewportFallback` field references from:
- `msgViewportWanted` (simplify to just `m.useMsgViewport`)
- `applyMsgViewportContentFromView` (remove fallback logic)

- [ ] **Step 3: Build**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/app/...
```
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add gou/app/message_pane.go gou/app/message_viewport_pane.go
git commit -m "refactor: remove old renderer fallback, new renderer is only path"
```

---

### Task 13: Delete formatMessageSegments from view.go

**Files:**
- Modify: `gou/app/view.go`

The old `formatMessageSegments` function and its helper `segmentJoinSeparator` are no longer called. The `styleMarkdownTokens` function is preserved — it's still used by the new renderer's markdown layer and streaming.

- [ ] **Step 1: Delete formatMessageSegments**

Delete `formatMessageSegments` (lines 406-589) and `segmentJoinSeparator` (line 365).

- [ ] **Step 2: Verify no callers remain**

```bash
grep -rn "formatMessageSegments\|segmentJoinSeparator" gou/app/ --include="*.go"
```
Expected: No results (or only in deleted code)

- [ ] **Step 3: Build**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./...
```
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add gou/app/view.go
git commit -m "refactor: delete formatMessageSegments old render path"
```

---

### Task 14: Remove model.go old renderer fields

**Files:**
- Modify: `gou/app/model.go`

Remove `msgViewportFallback` field from the Model struct. Update any remaining references.

- [ ] **Step 1: Remove field from Model struct**

- [ ] **Step 2: Build full project**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./...
```
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add gou/app/model.go
git commit -m "refactor: remove msgViewportFallback from Model"
```

---

### Task 15: Add highlightSearchPlain helper to gou/message

**Files:**
- Modify: `gou/message/renderer.go` (or new `gou/message/search.go`)

The `highlightSearchPlain` function lives in `gou/app/view.go`. Move a simplified version into `gou/message/` so renderers can use it without importing the app package.

- [ ] **Step 1: Add highlightSearchPlain to gou/message/renderer.go**

```go
// highlightSearchPlain wraps all occurrences of needle in haystack with ANSI reverse-video.
func highlightSearchPlain(haystack, needle string) string {
	if strings.TrimSpace(needle) == "" {
		return haystack
	}
	hlStyle := "\x1b[7m" // reverse video
	reset := "\x1b[0m"
	lower := strings.ToLower(haystack)
	needleLower := strings.ToLower(needle)
	var b strings.Builder
	idx := 0
	for {
		pos := strings.Index(lower[idx:], needleLower)
		if pos < 0 {
			b.WriteString(haystack[idx:])
			break
		}
		b.WriteString(haystack[idx : idx+pos])
		b.WriteString(hlStyle)
		b.WriteString(haystack[idx+pos : idx+pos+len(needle)])
		b.WriteString(reset)
		idx += pos + len(needle)
	}
	return b.String()
}
```

- [ ] **Step 2: Update renderers to use the local version**

In `assistant_message.go` and `user_message.go`, call `highlightSearchPlain` directly (no import change needed if defined in same package).

- [ ] **Step 3: Build + verify**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/message/...
```
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add gou/message/renderer.go
git commit -m "feat: add highlightSearchPlain helper to message package"
```

---

### Task 16: Write unit tests

**Files:**
- Create: `gou/message/renderer_test.go`
- Create: `gou/message/message_row_test.go`

- [ ] **Step 1: Write message_row_test.go**

```go
package message

import (
	"testing"
	"goc/types"
)

func TestBuildMessageRowContexts_Empty(t *testing.T) {
	ctxs := BuildMessageRowContexts(nil, MessageRowBuildOpts{})
	if len(ctxs) != 0 {
		t.Fatalf("expected 0 contexts, got %d", len(ctxs))
	}
}

func TestBuildMessageRowContexts_UserContinuation(t *testing.T) {
	msgs := []*types.Message{
		{Type: types.MessageTypeUser, UUID: "1"},
		{Type: types.MessageTypeUser, UUID: "2"},
		{Type: types.MessageTypeAssistant, UUID: "3"},
	}
	ctxs := BuildMessageRowContexts(msgs, MessageRowBuildOpts{})
	if !ctxs[1].IsUserContinuation {
		t.Error("expected second user message to be continuation")
	}
	if ctxs[2].IsUserContinuation {
		t.Error("assistant after user should not be continuation")
	}
}

func TestBuildMessageRowContexts_IsInProgress(t *testing.T) {
	msgs := []*types.Message{
		{Type: types.MessageTypeAssistant, UUID: "1", Content: []byte(
			`[{"type":"tool_use","id":"tu_1","name":"Read","input":{}}]`,
		)},
	}
	resolved := map[string]struct{}{"tu_1": {}}
	ctxs := BuildMessageRowContexts(msgs, MessageRowBuildOpts{
		ResolvedToolUseIDs: resolved,
	})
	if ctxs[0].IsInProgress {
		t.Error("tool_use with resolved result should not be in progress")
	}

	ctxs2 := BuildMessageRowContexts(msgs, MessageRowBuildOpts{
		ResolvedToolUseIDs: map[string]struct{}{},
	})
	if !ctxs2[0].IsInProgress {
		t.Error("tool_use without resolved result should be in progress")
	}
}
```

- [ ] **Step 2: Write renderer_test.go**

```go
package message

import (
	"strings"
	"testing"
	"goc/gou/theme"
	"goc/types"
)

func TestUserMessageRenderer_Basic(t *testing.T) {
	r := &UserMessageRenderer{}
	msg := &types.Message{
		Type: types.MessageTypeUser,
		Content: []byte(`[{"type":"text","text":"hello"}]`),
	}
	ctx := &RenderContext{Width: 80, Theme: theme.ActivePalette()}
	lines, err := r.Render(msg, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) == 0 {
		t.Fatal("expected at least one line")
	}
	// First line should have ❯ prefix
	if !strings.Contains(lines[0], "❯") && !strings.Contains(lines[0], "hello") {
		t.Errorf("unexpected first line: %s", lines[0])
	}
}

func TestUserMessageRenderer_Continuation(t *testing.T) {
	r := &UserMessageRenderer{}
	msg := &types.Message{
		Type: types.MessageTypeUser,
		Content: []byte(`[{"type":"text","text":"hello"}]`),
	}
	ctx := &RenderContext{
		Width: 80,
		Theme: theme.ActivePalette(),
		IsUserContinuation: true,
	}
	lines, err := r.Render(msg, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(lines[0], "❯") {
		t.Error("continuation should not have ❯ prefix")
	}
}

func TestAssistantMessageRenderer_Basic(t *testing.T) {
	r := &AssistantMessageRenderer{}
	msg := &types.Message{
		Type: types.MessageTypeAssistant,
		Content: []byte(`[{"type":"text","text":"Hello **world**"}]`),
	}
	ctx := &RenderContext{Width: 80, Theme: theme.ActivePalette()}
	lines, err := r.Render(msg, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) == 0 {
		t.Fatal("expected at least one line")
	}
}

func TestSystemMessageRenderer_Informational(t *testing.T) {
	r := &SystemMessageRenderer{}
	msg := &types.Message{
		Type:    types.MessageTypeSystem,
		Subtype: strPtr("informational"),
		Content: []byte(`"test info"`),
	}
	ctx := &RenderContext{Width: 80, Theme: theme.ActivePalette()}
	lines, err := r.Render(msg, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lines[0], "ℹ") {
		t.Errorf("expected ℹ prefix, got: %s", lines[0])
	}
}

func strPtr(s string) *string { return &s }
```

- [ ] **Step 3: Run tests**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go test ./gou/message/... -v
```
Expected: All new tests PASS

- [ ] **Step 4: Commit**

```bash
git add gou/message/renderer_test.go gou/message/message_row_test.go
git commit -m "test: add unit tests for renderers and MessageRow"
```

---

### Task 17: Full build + verify

- [ ] **Step 1: Full project build**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./...
```
Expected: PASS with zero errors

- [ ] **Step 2: Run all tests**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go test ./gou/message/... ./gou/app/... -count=1
```
Expected: All tests PASS

- [ ] **Step 3: Run gou-demo build**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./cmd/gou-demo/...
```
Expected: PASS

- [ ] **Step 4: Final manual check of cleaned files**

```bash
grep -rn "GOU_DEMO_USE_NEW_RENDERER" gou/ --include="*.go"
grep -rn "formatMessageSegments" gou/ --include="*.go"
grep -rn "processor\." gou/app/ --include="*.go" | grep -v "_test.go"
```
Expected: No results from any of these

- [ ] **Step 5: Commit final verification**

```bash
git add -A
git commit -m "chore: final cleanup and verification after render refactor"
```
