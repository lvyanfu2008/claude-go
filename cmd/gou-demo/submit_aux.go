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

func filterSlashCommandNames(all []string, filter string) []string {
	if len(all) == 0 {
		return nil
	}
	f := strings.ToLower(strings.TrimSpace(filter))
	if f == "" {
		return append([]string(nil), all...)
	}
	var out []string
	lf := strings.ToLower(f)
	for _, n := range all {
		if strings.Contains(strings.ToLower(n), lf) {
			out = append(out, n)
		}
	}
	return out
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

// slashListVisible is true when the command list should show (TS auto-suggest from "/" or F2).
func (m *model) slashListVisible() bool {
	m.loadSlashCommandsOnce()
	if len(sortedSlashDisplayNames(m.slashCommands)) == 0 {
		return false
	}
	if m.uiScreen != gouDemoScreenPrompt {
		return false
	}
	if m.slashListUser {
		return true
	}
	return shouldShowTSSlashList(m.pr.Value(), m.pr.CursorRuneIndex())
}

func (m *model) syncSlashListAfterPrompt() {
	if m.uiScreen != gouDemoScreenPrompt {
		return
	}
	m.loadSlashCommandsOnce()
	names := sortedSlashDisplayNames(m.slashCommands)
	if len(names) == 0 {
		m.slashListSel = 0
		return
	}
	should := shouldShowTSSlashList(m.pr.Value(), m.pr.CursorRuneIndex())
	visible := should || m.slashListUser
	if !visible {
		m.slashListSel = 0
		return
	}
	vis := filterSlashCommandNames(names, slashFilterFromPrompt(m.pr.Value()))
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

// handleSlashListNavKey handles ↑/↓ for the inline slash list; text keys always go to the prompt.
func (m *model) handleSlashListNavKey(msg tea.KeyPressMsg) bool {
	if m.uiScreen != gouDemoScreenPrompt || !m.slashListVisible() {
		return false
	}
	switch msg.String() {
	case "up":
		if m.slashListSel > 0 {
			m.slashListSel--
		}
		return true
	case "down":
		names := sortedSlashDisplayNames(m.slashCommands)
		vis := filterSlashCommandNames(names, slashFilterFromPrompt(m.pr.Value()))
		if m.slashListSel+1 < len(vis) {
			m.slashListSel++
		}
		return true
	default:
		return false
	}
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

func (m *model) renderSlashPicker(width, maxH int) string {
	if !m.slashListVisible() {
		return ""
	}
	names := sortedSlashDisplayNames(m.slashCommands)
	filter := slashFilterFromPrompt(m.pr.Value())
	vis := filterSlashCommandNames(names, filter)
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Slash commands  "))
	b.WriteString(lipgloss.NewStyle().Faint(true).Render(prSlashFilterLabel(filter) + "  ↑/↓  F2  Esc closes F2-only"))
	b.WriteByte('\n')
	start := 0
	idx := m.slashListSel
	if len(vis) > 0 && idx >= len(vis) {
		idx = len(vis) - 1
	}
	if len(vis) > 0 && idx >= maxH-2 {
		start = idx - (maxH - 3)
		if start < 0 {
			start = 0
		}
	}
	for i := start; i < len(vis) && i < start+maxH-2; i++ {
		line := vis[i]
		if i == idx {
			b.WriteString(lipgloss.NewStyle().Reverse(true).Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	if len(vis) == 0 {
		b.WriteString(lipgloss.NewStyle().Faint(true).Render("(no matches)"))
	}
	box := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1).Width(min(width-4, 50)).Render(strings.TrimSuffix(b.String(), "\n"))
	return lipgloss.Place(width, max(6, maxH), lipgloss.Right, lipgloss.Bottom, box)
}

func prSlashFilterLabel(filter string) string {
	if strings.TrimSpace(filter) == "" {
		return "/…"
	}
	return "/" + filter
}
