# Message Rendering Refactor Design

**Date**: 2026-05-31
**Goal**: Refactor claude-go message display to match claude-code (TS) rendering approach — unify to a single rendering path, add MessageRow layer, enhance context, delete old code.

## 1. Architecture Overview

Three-layer architecture matching TS:

```
┌─────────────────────────────┐
│   MessagesForScrollList      │  ← preprocess (collapse/group/filter, unchanged)
└─────────────┬───────────────┘
              │ []types.Message
┌─────────────▼───────────────┐
│   MessageRow (NEW)           │  ← mirrors MessageRow.tsx
│   Compute per-message ctx:   │
│   isStatic, shouldAnimate,   │
│   inProgress, isContinuation │
│   Chrome (⎿ prefix, spacing) │
└─────────────┬───────────────┘
              │ RenderContext (enhanced)
┌─────────────▼───────────────┐
│   Dispatcher → Renderer      │  ← mirrors Message.tsx + sub-components
│   Dispatch by message type   │
└─────────────────────────────┘
```

## 2. MessageRow Layer (NEW: `gou/message/message_row.go`)

Mirrors `src/components/MessageRow.tsx`. For each message in the list, computes rendering context then delegates to Dispatcher.

### Responsibilities

| Logic | Description |
|-------|-------------|
| `isUserContinuation` | Previous message also user → adjust spacing/prefix |
| `isActiveCollapsedGroup` | Whether collapsed group tools are still executing → grey spinner vs completed |
| `shouldAnimate` | Whether message content is still changing → affects "Reading…" vs "Read" |
| `isInProgress` | Whether tool_use is executing or text is streaming |
| Tool chrome | `⎿` prefix inserted when assistant text transitions to tool_use (mirrors `MessageResponse`) |
| Spacing | Row gaps between different message types |

### BuildMessageRowContexts

```go
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

func BuildMessageRowContexts(
    messages []*types.Message,
    opts MessageRowBuildOpts,
) []*MessageRowContext
```

## 3. RenderContext Enhancement

### New fields added to `RenderContext`

```go
// Per-message context (computed by MessageRow)
IsUserContinuation     bool
IsActiveCollapsedGroup bool
ShouldAnimate          bool
IsInProgress           bool

// Shared state (same across all messages)
InProgressToolUseIDs   map[string]struct{}
StreamingToolUseIDs    map[string]struct{}
ResolvedToolUseIDs     map[string]struct{}

// Transcript features (from old formatMessageSegments)
SearchHighlight        string
ShowToolUseCtrlOHint   bool
ShowResolvedToolStats  bool
```

## 4. Renderer Changes

### 4.1 UserMessageRenderer (`user_message.go`)

- `isUserContinuation`: when true, suppress duplicate `❯` prefix or reduce spacing
- `SearchHighlight`: highlight matching text in transcript search mode
- `ShowResolvedToolStats`: show resolved hint (`⎿` + stats) in transcript mode
- New `UserBashInputMessage` rendering: `$ command` format (mirrors TS `UserBashInputMessage.tsx`)
- `renderToolResultBlock`: inline prompt mode OK, keep existing behavior

### 4.2 AssistantMessageRenderer (`assistant_message.go`)

- **`isInProgress`**: read from `ctx.InProgressToolUseIDs` instead of hardcoded `false`
- **`⎿` (MessageResponse) prefix**: insert `⎿` prefix line when switching from text to tool_use block (mirrors TS `MessageResponse` component)
- **Markdown style rendering (CRITICAL FIX)**: refactor `renderMarkdown` to apply lipgloss styles matching TS `styleMarkdownTokens`:
  - Headings: bold + heading color, remove `#` prefix
  - Code blocks: syntax highlighting via chroma, remove backtick fences
  - Blockquotes: italic style, keep `│` prefix
  - Horizontal rules: faint style
  - Lists: proper indentation, no style removal
- `SearchHighlight`: text content search highlighting

### 4.3 ToolUseMessageRenderer (`tool_use.go`)

- `isInProgress`: from ctx, show activity description ("Reading…", "Searching…")
- `ShowToolUseCtrlOHint`: control `(ctrl+o to expand)` display
- `ShowResolvedToolStats`: control `⎿` + stats display

### 4.4 CollapsedGroupRenderer / GroupedToolUseRenderer

- `shouldAnimate`: check `InProgressToolUseIDs` for animation state
- `isActiveCollapsedGroup`: active = grey dot + present tense, inactive = completed state

### 4.5 Other Renderers

- `SystemMessageRenderer`: `SearchHighlight` support
- `AttachmentMessageRenderer`, `ProgressMessageRenderer`: minimal changes

## 5. Deletions

| File / Code | Reason |
|-------------|--------|
| `gou/message/processor.go` (entire file) | Dead code, `ProcessMessages` is pass-through |
| `virtual_list.go`: `BuildDisplayList`, `DisplayItem`, `determineSpacingBefore/After`, `shouldAddSpacing`, `measureMessageHeight`, `collapseReadSearchOperations`, `ProcessMessagesForDisplay` | Spacing moves to MessageRow; collapse handled by pipeline.go |
| `gou/app/view.go`: `formatMessageSegments` + old render path | Replaced by new renderer |
| `message_renderer_integration.go`: `ProcessMessages`, old path branches | Simplified |
| All `GOU_DEMO_USE_NEW_RENDERER` references | New renderer becomes only path |

## 6. Result File Structure

```
gou/message/
├── renderer.go              # Dispatcher + Renderer interface + RenderContext
├── message_row.go           # [NEW] MessageRow context computation + orchestration
├── user_message.go          # UserMessageRenderer
├── assistant_message.go     # AssistantMessageRenderer
├── tool_use.go              # ToolUseMessageRenderer
├── collapsed_group.go       # CollapsedGroupRenderer
├── grouped_tool_use.go      # GroupedToolUseRenderer
├── system_message.go        # SystemMessageRenderer
├── attachment_message.go    # AttachmentMessageRenderer
├── progress_message.go      # ProgressMessageRenderer (no-op)
├── file_edit_result.go      # FileEditResultRenderer + diff preview (unchanged)
├── tool_result_write_edit.go # write_edit diff helpers (unchanged)
├── unified_diff_style.go    # Diff styling (unchanged)
├── virtual_list.go          # Slimmed: ComputeVisibleRange + RenderRange + cache only
└── *_test.go                # Tests
```

## 7. Implementation Order

1. Enhance `RenderContext` — add new fields (no behavior change)
2. Implement `MessageRow` — new `message_row.go`, per-message context computation
3. Fix `AssistantMessageRenderer` — markdown styles + isInProgress + `⎿` prefix
4. Fix `UserMessageRenderer` — searchHL, bash input/output, tool_result resolved stats
5. Fix `ToolUseMessageRenderer` / `CollapsedGroupRenderer` — ctrl+o hint, animation states
6. Slim `VirtualList` — remove dead code
7. Switch render path — new renderer as only path, delete `formatMessageSegments` + `processor.go`
8. Remove `GOU_DEMO_USE_NEW_RENDERER` env var gate
9. Write tests — unit + integration
10. Verify — `go build` + `go test ./gou/message/...` all green

## 8. Testing

| Level | Content |
|-------|---------|
| Unit | Each renderer input/output, edge cases (empty, long text, special chars) |
| MessageRow | Context computation correctness: isUserContinuation, shouldAnimate, isActiveCollapsedGroup |
| Markdown | Heading/bold/code/blockquote produce ANSI styles, not raw markdown syntax |
| Integration | Full message sequence → rendered output, prompt + transcript modes |
| Regression | Compare old vs new renderer output before deleting old code |
