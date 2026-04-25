package main

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"goc/commands"
	"goc/tools/toolexecution"
	"goc/types"
)

type permissionAskReply struct {
	dec toolexecution.PermissionDecision
	err error
}

type permissionAskOverlay struct {
	toolName  string
	toolUseID string
	input     json.RawMessage
	prompt    string
	replyCh   chan permissionAskReply
}

// gouPermissionAskMsg is sent from [toolexecution.ExecutionDeps.AskResolver] to the Bubble Tea Update loop.
type gouPermissionAskMsg struct {
	toolName  string
	toolUseID string
	input     json.RawMessage
	prompt    string
	replyCh   chan permissionAskReply
}

// slashFilterFromPrompt returns the text after the leading "/" on the first line (for filtering),
// or "" if the first line is not a slash command.
func slashFilterFromPrompt(prompt string) string {
	line := prompt
	if i := strings.IndexByte(prompt, '\n'); i >= 0 {
		line = prompt[:i]
	}
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "/") {
		return ""
	}
	return strings.TrimPrefix(line, "/")
}

func cursorOnFirstLine(value string, cursorRune int) bool {
	if cursorRune < 0 {
		return false
	}
	rs := []rune(value)
	if cursorRune > len(rs) {
		return false
	}
	for i := 0; i < cursorRune; i++ {
		if rs[i] == '\n' {
			return false
		}
	}
	return true
}

func isAtEndWithWhitespaceRune(value string, cursorRune int) bool {
	rs := []rune(value)
	if len(rs) == 0 || cursorRune != len(rs) {
		return false
	}
	return rs[cursorRune-1] == ' '
}

// hasCommandWithArgumentsTS matches useTypeahead.tsx hasCommandWithArguments.
func hasCommandWithArgumentsTS(value string, isAtEndWithWhitespace bool) bool {
	return !isAtEndWithWhitespace && strings.Contains(value, " ") && !strings.HasSuffix(value, " ")
}

// shouldShowTSSlashList mirrors src/hooks/useTypeahead slash command list visibility
// (isCommandInput + cursor position + not in "command with real arguments" state).
func shouldShowTSSlashList(value string, cursorRune int) bool {
	if !strings.HasPrefix(value, "/") {
		return false
	}
	if !cursorOnFirstLine(value, cursorRune) {
		return false
	}
	if cursorRune <= 0 {
		return false
	}
	isAt := isAtEndWithWhitespaceRune(value, cursorRune)
	if hasCommandWithArgumentsTS(value, isAt) {
		return false
	}
	return true
}

func sortedSlashDisplayNames(commands []types.Command) []string {
	if len(commands) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var names []string
	for _, c := range commands {
		nm := strings.TrimSpace(c.Name)
		if nm == "" {
			continue
		}
		if !strings.HasPrefix(nm, "/") {
			nm = "/" + nm
		}
		if _, ok := seen[nm]; ok {
			continue
		}
		seen[nm] = struct{}{}
		names = append(names, nm)
	}
	sort.Strings(names)
	if len(names) > 200 {
		names = names[:200]
	}
	return names
}

func (m *model) installAskResolver(te *toolexecution.ExecutionDeps) {
	if m.ccbSend == nil {
		return
	}
	switch strings.TrimSpace(strings.ToLower(os.Getenv("GOU_QUERY_ASK_STRATEGY"))) {
	case "allow":
		te.AskResolver = func(ctx context.Context, toolName, toolUseID string, input json.RawMessage, prompt string) (toolexecution.PermissionDecision, error) {
			return toolexecution.AllowDecision(), nil
		}
		return
	}
	send := m.ccbSend
	te.AskResolver = func(ctx context.Context, toolName, toolUseID string, input json.RawMessage, prompt string) (toolexecution.PermissionDecision, error) {
		ch := make(chan permissionAskReply, 1)
		send(gouPermissionAskMsg{
			toolName:  toolName,
			toolUseID: toolUseID,
			input:     input,
			prompt:    prompt,
			replyCh:   ch,
		})
		select {
		case r := <-ch:
			return r.dec, r.err
		case <-ctx.Done():
			return toolexecution.DenyDecision("cancelled"), ctx.Err()
		}
	}
}

func (m *model) finishPermissionAsk(r permissionAskReply) {
	if m.permAsk != nil && m.permAsk.replyCh != nil {
		select {
		case m.permAsk.replyCh <- r:
		default:
		}
	}
	m.permAsk = nil
}

// handlePermissionKey returns true when the permission modal is showing (all keys are swallowed).
func (m *model) handlePermissionKey(msg tea.KeyPressMsg) bool {
	if m.permAsk == nil {
		return false
	}
	switch msg.String() {
	case "y", "Y", "enter", "space":
		m.finishPermissionAsk(permissionAskReply{dec: toolexecution.AllowDecision(), err: nil})
	case "n", "N", "esc", "q":
		m.finishPermissionAsk(permissionAskReply{dec: toolexecution.DenyDecision("denied by user"), err: nil})
	}
	return true
}

func (m *model) loadSlashCommandsOnce() {
	if m.slashCommandsOnce {
		return
	}
	m.slashCommandsOnce = true
	cwd, _ := os.Getwd()
	lc, err := commands.GetCommandsWithDefaults(context.Background(), cwd)
	if err != nil {
		gouDemoTracef("slash picker: GetCommands: %v", err)
		m.slashCommands = nil
		return
	}
	m.slashCommands = lc
}

// slashListVisible is true when the command list should show: leading "/" (TS), mid-input
// whitespace+"/token", or F2.
func (m *model) slashListVisible() bool {
	m.loadSlashCommandsOnce()
	if len(m.slashCommands) == 0 {
		return false
	}
	if m.uiScreen != gouDemoScreenPrompt {
		return false
	}
	if m.slashListUser {
		return true
	}
	v := m.pr.Value()
	cur := m.pr.CursorRuneIndex()
	if shouldShowTSSlashList(v, cur) {
		return true
	}
	return findMidInputSlashCommand(v, cur) != nil
}

func (m *model) syncSlashListAfterPrompt() {
	if m.uiScreen != gouDemoScreenPrompt {
		return
	}
	m.loadSlashCommandsOnce()
	if !m.slashListVisible() {
		m.slashListSel = 0
		return
	}
	vis := m.visibleSlashList()
	if m.slashListSel >= len(vis) {
		if len(vis) == 0 {
			m.slashListSel = 0
		} else {
			m.slashListSel = len(vis) - 1
		}
	}
	if m.slashListSel < 0 {
		m.slashListSel = 0
	}
}

// toggleSlashListUser toggles F2 manual list (when input is empty or not in /… mode).
func (m *model) toggleSlashListUser() {
	m.loadSlashCommandsOnce()
	if len(sortedSlashDisplayNames(m.slashCommands)) == 0 {
		m.slashListUser = false
		return
	}
	m.slashListUser = !m.slashListUser
	m.slashListSel = 0
	m.syncSlashListAfterPrompt()
}

// isPromptEnterKey is true for a normal Enter (submit) but not Alt+Enter (insert newline in REPL).
func isPromptEnterKey(msg tea.KeyPressMsg) bool {
	if msg.String() == "enter" {
		return true
	}
	k := msg.Key()
	if k.Mod.Contains(tea.ModAlt) {
		return false
	}
	return k.Code == tea.KeyEnter
}

// handleSlashListNavKey handles ↑/↓/Tab for the inline slash list. Must run before message
// viewport scroll so ↑/↓ change selection instead of the transcript (see main.handleKeyMsgPreserving).
func (m *model) handleSlashListNavKey(msg tea.KeyPressMsg) bool {
	if m.uiScreen != gouDemoScreenPrompt || !m.slashListVisible() {
		return false
	}
	if msg.String() == "tab" || msg.Key().Code == tea.KeyTab {
		m.applySlashTab()
		return true
	}
	dir := 0
	switch msg.String() {
	case "up":
		dir = -1
	case "down":
		dir = 1
	default:
		k := msg.Key()
		// Multiline: Shift+↑/↓ moves lines; do not hijack for slash.
		if k.Mod.Contains(tea.ModShift) {
			return false
		}
		if k.Code == tea.KeyUp {
			dir = -1
		} else if k.Code == tea.KeyDown {
			dir = 1
		}
	}
	if dir == 0 {
		return false
	}
	if dir < 0 {
		if m.slashListSel > 0 {
			m.slashListSel--
		}
	} else {
		vis := m.visibleSlashList()
		if m.slashListSel+1 < len(vis) {
			m.slashListSel++
		}
	}
	return true
}

func (m *model) renderPermissionModal(width int) string {
	if m.permAsk == nil {
		return ""
	}
	inPreview := string(m.permAsk.input)
	if len(inPreview) > 400 {
		inPreview = inPreview[:400] + "…"
	}
	body := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Tool permission"),
		"",
		"Tool: "+m.permAsk.toolName+"  id: "+m.permAsk.toolUseID,
		"",
		lipgloss.NewStyle().Faint(true).Render("Input (preview):"),
		inPreview,
		"",
		m.permAsk.prompt,
		"",
		lipgloss.NewStyle().Bold(true).Render("[Y] allow   [N] deny   [Esc] deny"),
	)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(min(width-4, 72)).
		Render(body)
}

// slashPickerListRows returns the number of list body lines shown (0 if none, 1 for empty hint).
func slashPickerListRows(vis []string, maxListRows int) int {
	if maxListRows < 1 {
		maxListRows = 1
	}
	if len(vis) == 0 {
		return 1
	}
	if len(vis) < maxListRows {
		return len(vis)
	}
	return maxListRows
}

// slashListChromeExtra is the terminal row count for the slash list below the input (rule + list block).
func (m *model) slashListChromeExtra() int {
	if !m.slashListVisible() {
		return 0
	}
	rows := slashPickerListRows(m.visibleSlashList(), slashPickerMaxListRows(m.height))
	// 1 faint rule, 1 title line, then list rows
	return 1 + 1 + rows
}

func slashPickerMaxListRows(termHeight int) int {
	if termHeight < 1 {
		termHeight = 24
	}
	// keep modest so message pane stays primary
	return min(12, max(3, termHeight/4))
}

// renderSlashPicker draws a full-width block directly below the input: separator rule, then
// title + left-aligned list (not a corner overlay).
func (m *model) renderSlashPicker(width, termHeight int) string {
	if !m.slashListVisible() {
		return ""
	}
	if width < 1 {
		width = 40
	}
	vis := m.visibleSlashList()
	maxList := slashPickerMaxListRows(termHeight)
	var b strings.Builder
	rule := strings.Repeat("─", max(1, width))
	b.WriteString(lipgloss.NewStyle().Faint(true).Width(width).Render(rule))
	b.WriteByte('\n')
	title := lipgloss.NewStyle().Bold(true).Render("Slash commands  ") +
		lipgloss.NewStyle().Faint(true).Render(m.slashListFooterHint()+"  F2  Esc  Tab  Enter run")
	b.WriteString(lipgloss.NewStyle().Width(width).MaxWidth(width).Render(title))
	b.WriteByte('\n')
	start := 0
	idx := m.slashListSel
	if len(vis) > 0 && idx >= len(vis) {
		idx = len(vis) - 1
	}
	if len(vis) > 0 && idx >= maxList {
		start = idx - (maxList - 1)
		if start < 0 {
			start = 0
		}
	}
	indent := "  "
	for i := start; i < len(vis) && i < start+maxList; i++ {
		line := vis[i]
		if i == idx {
			b.WriteString(indent)
			b.WriteString(lipgloss.NewStyle().Reverse(true).Render(line))
		} else {
			b.WriteString(indent)
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	if len(vis) == 0 {
		b.WriteString(indent)
		b.WriteString(lipgloss.NewStyle().Faint(true).Render("(no matches)"))
	}
	return b.String()
}

// slashListFooterHint is a short filter hint (leading "/" vs mid-input …/q).
func (m *model) slashListFooterHint() string {
	q, start := m.currentSlashQuery()
	v := m.pr.Value()
	cur := m.pr.CursorRuneIndex()
	if !start && findMidInputSlashCommand(v, cur) != nil {
		if strings.TrimSpace(q) == "" {
			return "…/…  ↑/↓"
		}
		return "…/" + q + "  ↑/↓"
	}
	if strings.TrimSpace(q) == "" {
		return "/…  ↑/↓"
	}
	return "/" + q + "  ↑/↓"
}
