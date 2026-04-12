package main

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"goc/commands"
	"goc/toolexecution"
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

type slashPickerOverlay struct {
	allNames []string
	filter   string
	idx      int
}

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

func (o *slashPickerOverlay) filtered() []string {
	if o == nil {
		return nil
	}
	f := strings.ToLower(strings.TrimSpace(o.filter))
	if f == "" {
		return slices.Clone(o.allNames)
	}
	var out []string
	for _, n := range o.allNames {
		if strings.Contains(strings.ToLower(n), f) {
			out = append(out, n)
		}
	}
	return out
}

func (o *slashPickerOverlay) clampIdx() {
	vis := o.filtered()
	if len(vis) == 0 {
		o.idx = 0
		return
	}
	if o.idx >= len(vis) {
		o.idx = len(vis) - 1
	}
	if o.idx < 0 {
		o.idx = 0
	}
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
func (m *model) handlePermissionKey(msg tea.KeyMsg) bool {
	if m.permAsk == nil {
		return false
	}
	switch msg.String() {
	case "y", "Y", "enter", " ":
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

func (m *model) toggleSlashPicker() {
	if m.slashPick != nil {
		m.slashPick = nil
		return
	}
	m.loadSlashCommandsOnce()
	seen := map[string]struct{}{}
	var names []string
	for _, c := range m.slashCommands {
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
	m.slashPick = &slashPickerOverlay{
		allNames: names,
		filter:   slashFilterFromPrompt(m.pr.Value()),
		idx:      0,
	}
	m.slashPick.clampIdx()
}

// handleSlashPickerKey returns true when consumed by the slash picker.
func (m *model) handleSlashPickerKey(msg tea.KeyMsg) bool {
	if m.slashPick == nil {
		return false
	}
	switch msg.String() {
	case "esc", "f2":
		m.slashPick = nil
		return true
	case "up":
		if m.slashPick.idx > 0 {
			m.slashPick.idx--
		}
		return true
	case "down":
		vis := m.slashPick.filtered()
		if m.slashPick.idx+1 < len(vis) {
			m.slashPick.idx++
		}
		return true
	case "enter":
		vis := m.slashPick.filtered()
		if len(vis) == 0 {
			m.slashPick = nil
			return true
		}
		sel := vis[m.slashPick.idx]
		m.pr.SetValue(sel + " ")
		m.slashPick = nil
		return true
	case "backspace":
		if m.slashPick.filter == "" {
			return true
		}
		_, size := utf8.DecodeLastRuneInString(m.slashPick.filter)
		if size > 0 {
			m.slashPick.filter = m.slashPick.filter[:len(m.slashPick.filter)-size]
		}
		m.slashPick.clampIdx()
		return true
	}
	if msg.Type == tea.KeyRunes && !msg.Paste && len(msg.Runes) > 0 {
		m.slashPick.filter += string(msg.Runes)
		m.slashPick.clampIdx()
		return true
	}
	return false
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
	if m.slashPick == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Slash (F2) filter: "))
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("/" + m.slashPick.filter))
	b.WriteByte('\n')
	vis := m.slashPick.filtered()
	start := 0
	if m.slashPick.idx >= maxH-2 {
		start = m.slashPick.idx - (maxH - 3)
		if start < 0 {
			start = 0
		}
	}
	for i := start; i < len(vis) && i < start+maxH-2; i++ {
		line := vis[i]
		if i == m.slashPick.idx {
			b.WriteString(lipgloss.NewStyle().Reverse(true).Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	box := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1).Width(min(width-4, 50)).Render(strings.TrimSuffix(b.String(), "\n"))
	return lipgloss.Place(width, max(6, maxH), lipgloss.Right, lipgloss.Bottom, box)
}
