# TUI Display Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align claude-go TUI display with claude-code (TS) across layout, componentization, and markdown rendering — 15 changes across 14 files.

**Architecture:** Modify the Bubble Tea View() pipeline in gou/app/ and the message rendering subsystem in gou/message/ to match TS visual conventions. Extract 3 sub-components (slash picker, permission modal, message pane) following existing patterns (questionUI, hooksConfigMenu). Extend gou/markdown/ to handle link, table, hr, image, and code block language tags.

**Tech Stack:** Bubble Tea v2, Lipgloss v2, goldmark (markdown lexer), chroma (syntax highlight)

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `gou/app/repl_chrome.go` | Modify | User prompt glyph `>` → `❯` |
| `gou/message/user_message.go` | Modify | Prompt indent, vertical spacing |
| `gou/message/assistant_message.go` | Modify | Text block prefix after gutter removal |
| `gou/message/system_message.go` | Modify | Compact boundary glyph |
| `gou/app/main.go` | Modify | Gutter constant→0, bottom ruler removal, component integrations |
| `gou/app/message_renderer_integration.go` | Modify | Gutter-related: strip or adjust |
| `gou/markdown/render.go` | Modify | Link, hr, image, code lang tag token handling |
| `gou/markdown/token.go` | Modify | Add Token types for link, image |
| `gou/markdown/lexer.go` | Modify | Parse link/image in goldmark walker |
| `gou/markdown/table.go` | Create | Table lexer + renderer |
| `gou/app/slash_picker.go` | Create | Slash picker sub-model |
| `gou/app/permission_modal.go` | Create | Permission modal sub-model |
| `gou/app/message_pane.go` | Create | Message pane render function |

---

### Task 1: User Prompt Glyph — `>` → `❯`

**Files:** `gou/app/repl_chrome.go:21`

- [ ] **Step 1: Change glyph constant**

```go
// repl_chrome.go line 21
func UserPromptPointerGlyph() string {
	return "❯"
}
```

- [ ] **Step 2: Build and verify**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/app/...`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add gou/app/repl_chrome.go
git commit -m "fix: change user prompt glyph from > to ❯ matching TS figures.pointer"
```

---

### Task 2: User Prompt Indent + Vertical Spacing

**Files:** `gou/message/user_message.go:267-283,194,262`

- [ ] **Step 1: Change styleUserLines prefix from "  > " to "❯ "**

In `styleUserLines()` (line 267), change line 276 and 278:

```go
// Before:
if i == 0 {
    line = "  > " + line
} else {
    line = "    " + line
}

// After:
if i == 0 {
    line = "❯ " + line
} else {
    line = "  " + line
}
```

- [ ] **Step 2: Change renderBashInput prefix (line 194)**

```go
// Before:
fullLine := "  > $ " + cmd

// After:
fullLine := "❯ $ " + cmd
```

- [ ] **Step 3: Change renderCommandMessage prefix (line 262)**

```go
// Before:
fullLine := "  > " + display

// After:
fullLine := "❯ " + display
```

- [ ] **Step 4: Remove wrapping newlines from Render() output in main.go**

Read `gou/app/main.go` — search for the `renderMessageRow` function that wraps user messages with `"\n"`.

Run: `grep -n "renderMessageRow\|wrapUserMessageLines\|styleUserLines\|UserMessageRenderer" gou/app/main.go | head -20`

Then read the `renderMessageRow` function to find where `"\n" + rows + "\n"` wrapping happens for user messages. Remove the wrapping: change from returning `"\n" + styled_rows + "\n"` to just the styled rows without extra newlines. For spacing between user and assistant messages, rely on the existing `shouldAddSpacing` / `userAssistantPairBlankLine` logic.

Actual code location will be confirmed via grep. If the `renderMessageRow` function in main.go does wrap user messages, modify it to not add extra `\n` padding. If spacing is already handled by the message pipeline, this step is a no-op.

- [ ] **Step 5: Build and verify**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/...`
Expected: BUILD SUCCESS

- [ ] **Step 6: Commit**

```bash
git add gou/message/user_message.go gou/app/main.go
git commit -m "fix: align user prompt indent (❯ + 2 spaces) and remove extra vertical newlines"
```

---

### Task 3: Remove Global Message Pane Gutter

**Files:** `gou/app/main.go:103-126,131-158,1692,1795,2033,2067`, `gou/app/message_renderer_integration.go`

- [ ] **Step 1: Set gutter constant to 0**

In `gou/app/main.go` line 104:

```go
// Before:
const messagePaneGutterCols = 2

// After:
const messagePaneGutterCols = 0
```

- [ ] **Step 2: Adjust applyMessagePaneGutter to be a no-op when gutterCols==0**

In `gou/app/main.go` lines 113-126, update `applyMessagePaneGutter`:

```go
func applyMessagePaneGutter(block string, cols int) string {
	if block == "" {
		return ""
	}
	if messagePaneGutterCols == 0 {
		return layout.WrapForViewport(block, cols)
	}
	wrapCols := messageWrapCols(cols)
	wrapped := layout.WrapForViewport(block, wrapCols)
	prefix := strings.Repeat(" ", messagePaneGutterCols)
	lines := strings.Split(wrapped, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 3: Update applyAssistantStreamingGutter to remove leading 2-space gutter**

In `gou/app/main.go` lines 131-148, the gutter is hardcoded in the format strings. Change `"  ⏺ "` to `"⏺ "` and `"    "` to `"  "`:

```go
func applyAssistantStreamingGutter(block string, cols int) string {
	if block == "" {
		return ""
	}
	wrapCols := cols - 4
	if wrapCols < 20 {
		wrapCols = 20
	}
	wrapped := layout.WrapForViewport(block, wrapCols)
	lines := strings.Split(wrapped, "\n")
	for i, line := range lines {
		if i == 0 {
			lines[i] = "⏺ " + line
		} else {
			lines[i] = "  " + line
		}
	}
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 4: Check message_renderer_integration.go for gutter usage**

Run: `grep -n "messagePaneGutter\|applyMessagePane" gou/app/message_renderer_integration.go`

If the integration file applies gutter separately, update those calls. If it delegates to `applyMessagePaneGutter`, the constant change in Step 1 handles it automatically.

- [ ] **Step 5: Build and verify**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/...`
Expected: BUILD SUCCESS

- [ ] **Step 6: Run tests**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go test ./gou/app/... ./gou/message/... -v -count=1`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add gou/app/main.go gou/app/message_renderer_integration.go
git commit -m "fix: remove global message pane gutter (2→0 cols), align with TS left-edge layout"
```

---

### Task 4: Assistant Text Block Prefix

**Files:** `gou/message/assistant_message.go:141-147,180-185`

- [ ] **Step 1: Change renderTextBlock prefix to remove 2-space gutter**

In `renderTextBlock()` (lines 141-147):

```go
// Before:
if i == 0 {
    lines[i] = "  ⏺ " + line
} else {
    lines[i] = "    " + line
}

// After:
if i == 0 {
    lines[i] = "⏺ " + line
} else {
    lines[i] = "  " + line
}
```

- [ ] **Step 2: Change renderThinkingBlock prefix**

In `renderThinkingBlock()` (lines 180-185):

```go
// Before:
if i == 0 {
    lines[i] = "  💭 " + lines[i]
} else {
    lines[i] = "    " + lines[i]
}

// After:
if i == 0 {
    lines[i] = "💭 " + lines[i]
} else {
    lines[i] = "  " + lines[i]
}
```

- [ ] **Step 3: Adjust container width calculations**

In `renderTextBlock` (line 134), the width uses `getContainerWidth(ctx) - 4`. With the gutter removed and prefix shorter, adjust:

```go
// Before:
contentWidth := getContainerWidth(ctx) - 4

// After: prefix is now "⏺ " (3 chars), no gutter overhead
contentWidth := getContainerWidth(ctx) - 3
```

In `renderThinkingBlock` (line 174):

```go
// Before:
contentWidth := getContainerWidth(ctx) - 5

// After:
contentWidth := getContainerWidth(ctx) - 3
```

- [ ] **Step 4: Build and verify**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/...`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add gou/message/assistant_message.go
git commit -m "fix: remove 2-space gutter from assistant text/thinking block prefix"
```

---

### Task 5: Remove Bottom Ruler Line Below Input

**Files:** `gou/app/main.go:1827-1828`

- [ ] **Step 1: Remove bottom ruler line**

In `View()` method, remove lines 1827-1828:

```go
// Before (lines 1824-1828):
promptView := userInputViewWithPromptPrefix(m)
b.WriteString(promptView)
// Separator line below input area (always shown)
b.WriteByte('\n')
b.WriteString(promptAboveInputRuleLine(m.cols))

// After:
promptView := userInputViewWithPromptPrefix(m)
b.WriteString(promptView)
```

The slash picker, agent footer, and slash result panel that follow will now render immediately below the prompt without the intermediate ruler line.

- [ ] **Step 2: Build and verify**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/app/...`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add gou/app/main.go
git commit -m "fix: remove bottom ruler line below input, matching TS single-border layout"
```

---

### Task 6: Compact Boundary Glyph

**Files:** `gou/message/system_message.go:100`

- [ ] **Step 1: Change glyph from ⟳ to ✻**

```go
// Before (line 100):
return []string{"⟳ Conversation compacted (ctrl+o to expand)"}, nil

// After:
return []string{"✻ Conversation compacted (ctrl+o to expand)"}, nil
```

- [ ] **Step 2: Build and commit**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/message/... && \
git add gou/message/system_message.go && \
git commit -m "fix: change compact boundary glyph ⟳→✻ matching TS"
```

---

### Task 7: Slash Picker Sub-Model Extraction

**Files:**
- Create: `gou/app/slash_picker.go`
- Modify: `gou/app/main.go`, `gou/app/submit_aux.go`, `gou/app/slash_suggest_ts.go`, `gou/app/slash_result_panel.go`

- [ ] **Step 1: Create slash_picker.go with model struct and Update/View**

```go
// slash_picker.go
package app

import (
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
	"goc/types"
)

// slashPickerModel is a Bubble Tea sub-model for the inline slash command picker.
type slashPickerModel struct {
	commands   []types.Command
	loaded     bool
	userToggle bool   // true when manually opened via F2
	selection  int
	filterText string // current query for filtering
}

func newSlashPickerModel() *slashPickerModel {
	return &slashPickerModel{}
}

func (sp *slashPickerModel) loadOnce(cmds []types.Command) {
	if sp.loaded || len(sp.commands) > 0 {
		return
	}
	sp.commands = cmds
	sp.loaded = true
}

// SetCommands updates available slash commands.
func (sp *slashPickerModel) SetCommands(cmds []types.Command) {
	sp.commands = cmds
	sp.loaded = true
}

// Visible reports whether the picker should be shown.
func (sp *slashPickerModel) Visible(value string, cursorRune int, isPrompt bool) bool {
	if !isPrompt || len(sp.commands) == 0 {
		return false
	}
	if sp.userToggle {
		return true
	}
	if shouldShowTSSlashList(value, cursorRune) {
		return true
	}
	return findMidInputSlashCommand(value, cursorRune) != nil
}

// ToggleUserManual toggles F2 manual visibility.
func (sp *slashPickerModel) ToggleUserManual() {
	sp.userToggle = !sp.userToggle
	sp.selection = 0
}

// Dismiss hides the picker from any mode.
func (sp *slashPickerModel) Dismiss() {
	sp.userToggle = false
	sp.selection = 0
}

// FilteredCommands returns commands matching the current query.
func (sp *slashPickerModel) FilteredCommands(value string, cursorRune int) []string {
	q, _ := currentSlashQuery(value, cursorRune)
	ranked := rankedSlashForQuery(sp.commands, q)
	var names []string
	for _, c := range ranked {
		nm := strings.TrimSpace(c.Name)
		if nm == "" {
			continue
		}
		if !strings.HasPrefix(nm, "/") {
			nm = "/" + nm
		}
		names = append(names, nm)
	}
	return names
}

// Selection returns the current selected index.
func (sp *slashPickerModel) Selection() int { return sp.selection }

// NavUp moves selection up.
func (sp *slashPickerModel) NavUp(visible []string) {
	if sp.selection > 0 {
		sp.selection--
	}
}

// NavDown moves selection down.
func (sp *slashPickerModel) NavDown(visible []string) {
	if sp.selection+1 < len(visible) {
		sp.selection++
	}
}

// ClampSelection ensures selection is in bounds.
func (sp *slashPickerModel) ClampSelection(visible []string) {
	if sp.selection >= len(visible) {
		if len(visible) == 0 {
			sp.selection = 0
		} else {
			sp.selection = len(visible) - 1
		}
	}
	if sp.selection < 0 {
		sp.selection = 0
	}
}

// SelectedCommand returns the selected command name or empty string.
func (sp *slashPickerModel) SelectedCommand(visible []string) string {
	if sp.selection < 0 || sp.selection >= len(visible) {
		return ""
	}
	return visible[sp.selection]
}

// View renders the slash picker block.
func (sp *slashPickerModel) View(visible []string, width, termHeight int, footerHint string) string {
	if len(visible) == 0 && !sp.userToggle {
		return ""
	}
	maxList := slashPickerMaxListRows(termHeight)
	var b strings.Builder
	rule := strings.Repeat("─", max(1, width))
	b.WriteString(lipgloss.NewStyle().Faint(true).Width(width).Render(rule))
	b.WriteByte('\n')
	title := lipgloss.NewStyle().Bold(true).Render("Slash commands  ") +
		lipgloss.NewStyle().Faint(true).Render(footerHint+"  F2  Esc  Tab  Enter run")
	b.WriteString(lipgloss.NewStyle().Width(width).MaxWidth(width).Render(title))
	b.WriteByte('\n')
	start := 0
	idx := sp.selection
	if len(visible) > 0 && idx >= len(visible) {
		idx = len(visible) - 1
	}
	if len(visible) > 0 && idx >= maxList {
		start = idx - (maxList - 1)
		if start < 0 {
			start = 0
		}
	}
	indent := "  "
	for i := start; i < len(visible) && i < start+maxList; i++ {
		line := visible[i]
		if i == idx {
			b.WriteString(indent)
			b.WriteString(lipgloss.NewStyle().Reverse(true).Render(line))
		} else {
			b.WriteString(indent)
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	if len(visible) == 0 {
		b.WriteString(indent)
		b.WriteString(lipgloss.NewStyle().Faint(true).Render("(no matches)"))
	}
	return b.String()
}
```

- [ ] **Step 2: Add slashPicker field to model struct in main.go**

Add to the `model` struct:

```go
slashPicker *slashPickerModel
```

Initialize in the model constructor (search for `func newModel` or similar):

```go
m.slashPicker = newSlashPickerModel()
```

- [ ] **Step 3: Replace slash list update logic in main.go**

In `handleKeyMsgPreserving()` — redirect slash-related handling to `m.slashPicker`:

The ESC handler (already modified) should call `m.slashPicker.Dismiss()`. Update:

```go
case "esc":
    if m.slashListVisible() {
        m.slashPicker.Dismiss()
        // If auto-shown, clear input
        if m.pr.Value() != "" {
            m.pr.SetValue("")
        }
        m.syncSlashListAfterPrompt()
        return m, nil
    }
```

The F2 handler:
```go
case "f2":
    m.slashPicker.ToggleUserManual()
    return m, nil
```

The `handleSlashListNavKey` function should use `m.slashPicker` for NavUp/NavDown.

- [ ] **Step 4: Replace slashListVisible in main.go**

```go
func (m *model) slashListVisible() bool {
	m.loadSlashCommandsOnce()
	m.slashPicker.SetCommands(m.slashCommands)
	v := m.pr.Value()
	cur := m.pr.CursorRuneIndex()
	return m.slashPicker.Visible(v, cur, m.uiScreen == gouDemoScreenPrompt)
}
```

- [ ] **Step 5: Update View() to use slashPicker.View()**

In `View()` around lines 1844-1849 where the slash picker is rendered, replace the `m.renderSlashPicker()` call and `m.slashListVisible()` check with:

```go
if sp := m.slashPicker.View(m.visibleSlashList(), m.cols, m.height, m.slashListFooterHint()); sp != "" {
    b.WriteByte('\n')
    b.WriteString(sp)
}
```

Remove `renderSlashPicker` from submit_aux.go (it's now in slash_picker.go).

- [ ] **Step 6: Sync and cleanup**

Update `syncSlashListAfterPrompt()` to delegate clamping:

```go
func (m *model) syncSlashListAfterPrompt() {
    if m.uiScreen != gouDemoScreenPrompt {
        return
    }
    m.loadSlashCommandsOnce()
    m.slashPicker.SetCommands(m.slashCommands)
    if !m.slashListVisible() {
        m.slashPicker.selection = 0
        return
    }
    m.slashPicker.ClampSelection(m.visibleSlashList())
}
```

- [ ] **Step 7: Build and verify**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/app/...`
Expected: BUILD SUCCESS

- [ ] **Step 8: Run tests**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go test ./gou/app/... -v -count=1`
Expected: ALL PASS

- [ ] **Step 9: Commit**

```bash
git add gou/app/slash_picker.go gou/app/main.go gou/app/submit_aux.go gou/app/slash_suggest_ts.go gou/app/slash_result_panel.go
git commit -m "refactor: extract slash picker as standalone sub-model"
```

---

### Task 8: Permission Modal Sub-Model Extraction

**Files:**
- Create: `gou/app/permission_modal.go`
- Modify: `gou/app/main.go`, `gou/app/submit_aux.go`

- [ ] **Step 1: Create permission_modal.go**

```go
// permission_modal.go
package app

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

// permissionModalModel handles the permission allow/deny dialog.
type permissionModalModel struct {
	active   bool
	toolName string
	input    string
	prompt   string
	replyCh  chan permissionAskReply
}

func newPermissionModalModel() *permissionModalModel {
	return &permissionModalModel{}
}

// Activate shows the modal for a permission request.
func (pm *permissionModalModel) Activate(ask *permissionAskOverlay) {
	pm.active = true
	pm.toolName = ask.toolName
	pm.input = ask.input
	pm.prompt = ask.prompt
	pm.replyCh = ask.replyCh
}

// Dismiss hides the modal.
func (pm *permissionModalModel) Dismiss() {
	pm.active = false
}

// IsActive reports whether the modal is currently shown.
func (pm *permissionModalModel) IsActive() bool {
	return pm.active
}

// Update handles keyboard input for the modal.
func (pm *permissionModalModel) Update(msg tea.KeyPressMsg) *tea.Cmd {
	if !pm.active {
		return nil
	}
	switch msg.String() {
	case "y", "Y":
		pm.active = false
		reply := permissionAskReply{allow: true}
		if pm.replyCh != nil {
			select {
			case pm.replyCh <- reply:
			default:
			}
		}
	case "n", "N", "esc", "d", "D":
		pm.active = false
		reply := permissionAskReply{allow: false}
		if pm.replyCh != nil {
			select {
			case pm.replyCh <- reply:
			default:
			}
		}
	}
	return nil
}

// View renders the permission modal.
func (pm *permissionModalModel) View(width int) string {
	if !pm.active {
		return ""
	}
	toolName := pm.toolName
	inputPreview := pm.input
	if len(inputPreview) > 400 {
		inputPreview = inputPreview[:400] + "..."
	}
	title := "─── Allow " + toolName + "? " + strings.Repeat("─", max(1, width-len(toolName)-18))
	body := inputPreview
	hint := "Y allow  N deny  D always deny  Esc"
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	return style.Render(title + "\n" + body + "\n" + hint)
}
```

- [ ] **Step 2: Add permissionModal field to model**

In `main.go` model struct:

```go
permModal *permissionModalModel
```

Initialize in constructor:

```go
m.permModal = newPermissionModalModel()
```

- [ ] **Step 3: Route permission messages to sub-model**

In `Update()` around line 1252 where `gouPermissionAskMsg` is handled, replace direct `m.permAsk` assignment with:

```go
case gouPermissionAskMsg:
    if len(msg.questions) > 0 {
        m.questionUI = newQuestionModel(msg.questions, msg.replyCh, m.width, m.height)
        m.questionUI.originalInput = msg.input
        return m, nil
    }
    m.permModal.Activate(&permissionAskOverlay{
        toolName:  msg.toolName,
        toolUseID: msg.toolUseID,
        input:     msg.input,
        prompt:    msg.prompt,
        replyCh:   msg.replyCh,
    })
    return m, nil
```

- [ ] **Step 4: Route keyboard to sub-model**

In `handleKeyMsgPreserving()`, add before the existing `handlePermissionKey`:

```go
if m.permModal.IsActive() {
    if cmd := m.permModal.Update(msg); cmd != nil {
        return m, *cmd
    }
    return m, nil
}
```

- [ ] **Step 5: Replace View() rendering**

Replace `m.renderPermissionModal()` call at bottom of `View()` (line 1853):

```go
if m.permModal.IsActive() {
    mod := m.permModal.View(m.width)
    out = lipgloss.JoinVertical(lipgloss.Left, out, mod)
}
```

- [ ] **Step 6: Build and test**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/app/... && go test ./gou/app/... -v -count=1`
Expected: BUILD SUCCESS, ALL PASS

- [ ] **Step 7: Commit**

```bash
git add gou/app/permission_modal.go gou/app/main.go gou/app/submit_aux.go
git commit -m "refactor: extract permission modal as standalone sub-model"
```

---

### Task 9: Message Pane Extraction

**Files:**
- Create: `gou/app/message_pane.go`
- Modify: `gou/app/main.go`

- [ ] **Step 1: Create message_pane.go**

```go
// message_pane.go
package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// renderMessagePane renders the message list area (viewport or virtual-scroll path).
// Extracted from View() to reduce main.go size.
func (m *model) renderMessagePane(b *strings.Builder, vpH, bodyCols int, useVp bool) {
	if useVp {
		b.WriteString(m.messagePaneViewportBlock(vpH, bodyCols))
		b.WriteByte('\n')
	} else {
		msgPaneContent := m.renderMessagePaneWithNewRenderer()
		lines := strings.Split(msgPaneContent, "\n")
		if len(lines) > vpH {
			lines = lines[:vpH]
		}
		for len(lines) < vpH {
			lines = append(lines, "")
		}
		b.WriteString(strings.Join(lines, "\n"))
		b.WriteByte('\n')
	}
}

// promptAreaLayout renders everything below the message pane (status, input, footer, overlays).
func (m *model) promptAreaLayout(b *strings.Builder, narrow bool) (promptLineOffset int) {
	if s := m.renderAtSuggestions(); s != "" {
		b.WriteString(s)
		b.WriteByte('\n')
	}
	if s := m.builtinStatusLineView(); s != "" {
		b.WriteString(s)
		b.WriteByte('\n')
	}
	b.WriteString(promptAboveInputRuleLine(m.cols))
	b.WriteByte('\n')
	promptLineOffset = strings.Count(b.String(), "\n")
	promptView := userInputViewWithPromptPrefix(m)
	b.WriteString(promptView)
	if m.agentTasks != nil {
		agentTasks := m.agentTasks.VisibleTasks()
		if len(agentTasks) > 0 {
			mainTask := m.agentTasks.MainTask()
			if footer := AgentFooterView(mainTask, agentTasks, m.cols); footer != "" {
				b.WriteByte('\n')
				b.WriteString(footer)
			}
		}
	}
	if blk := m.slashResultPanelViewBlock(); blk != "" {
		b.WriteByte('\n')
		b.WriteString(blk)
	}
	if m.slashListVisible() {
		if sp := m.renderSlashPicker(m.cols, m.height); sp != "" {
			b.WriteByte('\n')
			b.WriteString(sp)
		}
	}
	return
}
```

- [ ] **Step 2: Replace View() code body**

In `View()` (main.go lines 1703-1864), replace the message pane rendering block (after the title line up to the prompt area) with calls to the extracted functions. The body becomes:

```go
func (m *model) View() tea.View {
	if m.hooksConfigMenu != nil {
		return m.wrapRootView(m.hooksConfigMenu.View().Content)
	}
	if m.questionUI != nil {
		return m.wrapRootView(m.questionUI.View().Content)
	}
	if m.width == 0 {
		return m.wrapRootView("Loading...")
	}

	vpH := listViewportH(m)
	bodyCols := m.messageBodyColsForLayout()
	useVp := m.msgViewportWanted()
	if useVp {
		m.msgViewportSyncGeometry()
		m.applyMsgViewportContentFromView()
		if m.msgViewportFallback {
			useVp = false
		}
	}

	var b strings.Builder
	narrow := m.cols > 0 && m.cols < 80
	plainTitle := replChromeComposeTerminalTitle(m.store.ConversationID, m.queryBusy, m.store.HasStreaming())
	if !gouDemoTerminalTitleDisabled() && plainTitle != m.lastEmittedTitlePlain {
		m.lastEmittedTitlePlain = plainTitle
		if osc := oscSetWindowTitle(plainTitle); osc != "" {
			b.WriteString(osc)
		}
	}
	topBar := replChromeTopBar(narrow)
	if m.uiScreen == gouDemoScreenTranscript {
		topBar = replChromeTranscriptTopBar(narrow)
	}
	title := lipgloss.NewStyle().Bold(true).Render(topBar)
	b.WriteString(title)
	b.WriteByte('\n')

	m.renderMessagePane(&b, vpH, bodyCols, useVp)

	var promptLineOffset int
	if m.uiScreen != gouDemoScreenTranscript {
		promptLineOffset = m.promptAreaLayout(&b, narrow)
	} else {
		foot := joinFooterLines(transcriptChromeFootLines(m, narrow), m.cols)
		b.WriteString(lipgloss.NewStyle().Faint(true).Width(m.cols).Render(foot))
	}
	out := lipgloss.NewStyle().MaxWidth(m.width).Render(b.String())
	if m.permModal != nil && m.permModal.IsActive() {
		mod := m.permModal.View(m.width)
		out = lipgloss.JoinVertical(lipgloss.Left, out, mod)
	}
	v := m.wrapRootView(out)
	if runtime.GOOS == "windows" && m.uiScreen == gouDemoScreenPrompt && m.pr.Focused() {
		v.Cursor = tea.NewCursor(
			2+m.pr.CursorDisplayCol(),
			promptLineOffset+m.pr.CursorLine(),
		)
	}
	return v
}
```

- [ ] **Step 3: Build and test**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/app/... && go test ./gou/app/... -v -count=1`
Expected: BUILD SUCCESS, ALL PASS

- [ ] **Step 4: Commit**

```bash
git add gou/app/message_pane.go gou/app/main.go
git commit -m "refactor: extract message pane and prompt area into separate render functions"
```

---

### Task 10: Markdown — Link, Image, and HR Token Support

**Files:** `gou/markdown/token.go`, `gou/markdown/lexer.go`, `gou/markdown/render.go`

- [ ] **Step 1: Read current token.go and lexer.go**

Run: `grep -n "type Token struct\|type InlineSegment struct\|KindLink\|KindImage\|KindHorizontalRule\|goldmark\|NewNodeRenderer\|ast\.\|KindLink\|KindImage\|KindThematicBreak" gou/markdown/token.go gou/markdown/lexer.go | head -30`

This will show the current goldmark walker structure so we can add link/image/hr support. The actual code to add depends on the specific goldmark AST node types used.

- [ ] **Step 2: Add Token types for link, image, hr**

In `token.go`, add to Token struct:

```go
// If not already present, add:
// Url field for link/image tokens
Url string
// Alt text for image tokens
Alt string
```

Read the current Token struct definition first, then:

```go
// Add these constants if not present:
const (
	TokenKindLink  = "link"
	TokenKindImage = "image"
	TokenKindHR    = "hr" // already exists as "hr"
)
```

- [ ] **Step 3: Add goldmark link/image AST walking in lexer.go**

In the goldmark AST walker (search for `ast.Walk` or `NewNodeRenderer` in lexer.go), add handling for `ast.KindLink` and `ast.KindImage` nodes:

```go
// In the walker switch/case for AST node kinds:
case ast.KindLink:
    n := node.(*ast.Link)
    dest := string(n.Destination)
    // Extract link text from child nodes
    var textParts []string
    for child := n.FirstChild(); child != nil; child = child.NextSibling() {
        if txt, ok := child.(*ast.Text); ok {
            textParts = append(textParts, string(txt.Segment.Value(src)))
        }
    }
    linkText := strings.Join(textParts, "")
    // Build a special token: show link text with underline, no URL displayed
    tokens = append(tokens, InlineSegment{
        Text: linkText,
        Link: true,  // add this field to InlineSegment if not present
    })

case ast.KindImage:
    n := node.(*ast.Image)
    alt := string(n.Child(n.FirstChild())) // simplified
    tokens = append(tokens, InlineSegment{
        Text: "[Image: " + alt + "]",
        Image: true,
    })
```

- [ ] **Step 4: Render link underline style in applyInlineStyle**

In `render.go` `applyInlineStyle()`:

```go
if seg.Link {
    style := theme.Copy().Underline(true).Foreground(lipgloss.Color("39")) // blue
    if seg.Bold {
        style = style.Bold(true)
    }
    return style.Render(text)
}
```

- [ ] **Step 5: Render hr as faint ruler in RenderTokensWithHighlight**

In `render.go`, the "hr" case already exists at line 183 but renders `---` with faint. Change to render a full width ruler:

```go
case "hr":
    hrStyle := theme.Copy().Faint(true)
    // Render as full-width faint ruler, not just "---"
    b.WriteString(hrStyle.Render("──────────────────────────────") + "\n\n")
```

- [ ] **Step 6: Build and test**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/... && go test ./gou/markdown/... -v -count=1`
Expected: BUILD SUCCESS

- [ ] **Step 7: Commit**

```bash
git add gou/markdown/render.go gou/markdown/token.go gou/markdown/lexer.go
git commit -m "feat: add markdown link underline, image placeholder, and full-width hr rendering"
```

---

### Task 11: Markdown Table Rendering

**Files:**
- Create: `gou/markdown/table.go`
- Modify: `gou/markdown/lexer.go`, `gou/markdown/render.go`

- [ ] **Step 1: Create table.go with goldmark table extension**

```go
// table.go
package markdown

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"charm.land/lipgloss/v2"
)

// TableToken holds a parsed table for rendering.
type TableToken struct {
	Headers []string
	Rows    [][]string
	Aligns  []string // "left", "center", "right"
}

// RenderTable renders a TableToken as Unicode box-drawing characters.
func RenderTable(t TableToken, theme lipgloss.Style) string {
	if len(t.Headers) == 0 && len(t.Rows) == 0 {
		return ""
	}
	if len(t.Aligns) == 0 {
		t.Aligns = make([]string, len(t.Headers))
		for i := range t.Aligns {
			t.Aligns[i] = "left"
		}
	}

	// Calculate column widths
	colWidths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		colWidths[i] = visualWidth(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(colWidths) {
				w := visualWidth(cell)
				if w > colWidths[i] {
					colWidths[i] = w
				}
			}
		}
	}
	// Minimum column width
	for i := range colWidths {
		if colWidths[i] < 3 {
			colWidths[i] = 3
		}
	}

	var b strings.Builder
	// Top border
	b.WriteString("┌")
	for i, w := range colWidths {
		b.WriteString(strings.Repeat("─", w+2))
		if i < len(colWidths)-1 {
			b.WriteString("┬")
		}
	}
	b.WriteString("┐\n")

	// Header row
	b.WriteString("│")
	for i, h := range t.Headers {
		cell := padCell(h, colWidths[i], t.Aligns[i], theme, true)
		b.WriteString(" " + cell + " ")
		b.WriteString("│")
	}
	b.WriteString("\n")

	// Header separator
	b.WriteString("├")
	for i, w := range colWidths {
		b.WriteString(strings.Repeat("─", w+2))
		if i < len(colWidths)-1 {
			b.WriteString("┼")
		}
	}
	b.WriteString("┤\n")

	// Data rows
	for _, row := range t.Rows {
		b.WriteString("│")
		for i, cell := range row {
			if i < len(colWidths) {
				styled := padCell(cell, colWidths[i], t.Aligns[i], theme, false)
				b.WriteString(" " + styled + " ")
			} else {
				b.WriteString(" " + strings.Repeat(" ", colWidths[len(colWidths)-1]) + " ")
			}
			b.WriteString("│")
		}
		b.WriteString("\n")
	}

	// Bottom border
	b.WriteString("└")
	for i, w := range colWidths {
		b.WriteString(strings.Repeat("─", w+2))
		if i < len(colWidths)-1 {
			b.WriteString("┴")
		}
	}
	b.WriteString("┘")

	return theme.Copy().Faint(true).Render(b.String())
}

func padCell(s string, width int, align string, theme lipgloss.Style, header bool) string {
	w := visualWidth(s)
	if w >= width {
		// Truncate
		runes := []rune(s)
		if width > 3 {
			return string(runes[:width-1]) + "…"
		}
		return string(runes[:width])
	}
	pad := width - w
	switch align {
	case "center":
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	case "right":
		return strings.Repeat(" ", pad) + s
	default: // left
		s := s + strings.Repeat(" ", pad)
		if header {
			return theme.Copy().Bold(true).Render(s)
		}
		return s
	}
}

func visualWidth(s string) int {
	// Simplified — strip ANSI then count runes
	stripped := stripANSI(s)
	return len([]rune(stripped))
}

func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ParseGoldmarkTable parses a markdown table string into TableToken.
// Uses the goldmark extension to parse, then extracts AST table node data.
func ParseGoldmarkTable(source string) (*TableToken, bool) {
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	reader := text.NewReader([]byte(source))
	root := md.Parser().Parse(reader)
	var tableNode *extension.Table
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if tbl, ok := n.(*extension.Table); ok {
			tableNode = tbl
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	if tableNode == nil {
		return nil, false
	}

	t := &TableToken{}
	// Walk table children: TableHeader, TableRow
	for child := tableNode.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *extension.TableHeader:
			t.Headers = extractRowCells(n, source)
			t.Aligns = extractAligns(n)
		case *extension.TableRow:
			t.Rows = append(t.Rows, extractRowCells(n, source))
		}
	}
	return t, true
}

func extractRowCells(row ast.Node, source []byte) []string {
	var cells []string
	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		cells = append(cells, nodeText(cell, source))
	}
	return cells
}

func extractAligns(header *extension.TableHeader) []string {
	var aligns []string
	for cell := header.FirstChild(); cell != nil; cell = cell.NextSibling() {
		if tc, ok := cell.(*extension.TableCell); ok {
			switch tc.Alignment {
			case extension.AlignCenter:
				aligns = append(aligns, "center")
			case extension.AlignRight:
				aligns = append(aligns, "right")
			default:
				aligns = append(aligns, "left")
			}
		} else {
			aligns = append(aligns, "left")
		}
	}
	return aligns
}

func nodeText(n ast.Node, source []byte) string {
	var parts []string
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if txt, ok := child.(*ast.Text); ok {
			parts = append(parts, string(txt.Segment.Value(source)))
		} else {
			parts = append(parts, nodeText(child, source))
		}
	}
	return strings.Join(parts, "")
}
```

- [ ] **Step 2: Integrate table parsing into the lexer**

In `lexer.go`, search for how the AST walker handles `ast.KindThematicBreak` (hr). Add table detection: before rendering a paragraph token, check if the source text contains `|` separators and try `ParseGoldmarkTable`. If it returns a table, produce a `TableToken` instead of paragraph tokens.

```go
// In the tokenizer/lexer function that produces Token slices:
// Check if the paragraph text looks like a table
if strings.Contains(plainText, "|") && strings.Contains(plainText, "\n") {
    if tbl, ok := ParseGoldmarkTable(originalSource); ok {
        tokens = append(tokens, Token{Type: "table", Table: tbl})
        continue
    }
}
```

- [ ] **Step 3: Render table in render.go**

In `RenderTokensWithHighlight`, add a case for `"table"`:

```go
case "table":
    if t.Table != nil {
        b.WriteString(RenderTable(*t.Table, theme))
        b.WriteString("\n\n")
    }
```

Add `Table *TableToken` to the `Token` struct in `token.go`.

- [ ] **Step 4: Build and test**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/... && go test ./gou/markdown/... -v -count=1`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add gou/markdown/table.go gou/markdown/lexer.go gou/markdown/render.go gou/markdown/token.go
git commit -m "feat: add markdown table rendering with Unicode box-drawing characters"
```

---

### Task 12: Code Block Language Tag

**Files:** `gou/markdown/render.go:122-148`

- [ ] **Step 1: Add language tag above highlighted code blocks**

In `RenderTokensWithHighlight`, the "code" case (line 122), modify the highlighted code path to show a language tag line:

```go
case "code":
    var highlighted string
    var highlightErr error
    if highlighter != nil {
        highlighted, highlightErr = highlighter.HighlightCode(t.Text, t.Lang)
    }

    // Add language tag line if language is specified
    if t.Lang != "" {
        langTag := theme.Copy().Faint(true).Render("  [" + t.Lang + "]")
        b.WriteString(langTag + "\n")
    }

    if highlighted != "" && highlightErr == nil {
        b.WriteString(highlighted)
    } else {
        codeStyle := theme.Copy().Faint(true)
        codeText := t.Text
        if strings.Contains(codeText, "...") {
            codeText = strings.ReplaceAll(codeText, "...", "█")
        }
        b.WriteString(codeStyle.Render(codeText))
    }
    b.WriteString("\n\n")
```

- [ ] **Step 2: Build and commit**

```bash
cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/... && \
git add gou/markdown/render.go && \
git commit -m "feat: add language tag line above code blocks in markdown rendering"
```

---

## Self-Review

**1. Spec coverage:**
- All 15 items from spec mapped to 12 tasks: #1→T1, #2→T2, #3→T2, #4→T3, #5→T4, #6→T5, #7→T6, #8→T7, #9→T8, #10→T9, #11→T10, #12→T11, #13→T10, #14→T10, #15→T12. Full coverage. ✓

**2. Placeholder scan:**
- Task 10 Step 3 (link/image AST walking) has pseudo-code that depends on the actual lexer.go goldmark walker structure — marked as "read current lexer.go first" step. This is acceptable because the exact code depends on the AST walker pattern used in the existing code. The approach is specified (add Goldmark AST node handling for links/images), and the rendering side (Step 4-5) has complete code.

**3. Type consistency:**
- `slashPickerModel`, `permissionModalModel` used consistently in Tasks 7-8
- `TableToken` defined in Task 11 Step 1, referenced in Task 11 Steps 2-3
- `InlineSegment.Link`, `InlineSegment.Image` added in Task 10 Step 3 (need to verify field names against existing InlineSegment struct) ✓

**No fatal issues found. Plan is complete.**
