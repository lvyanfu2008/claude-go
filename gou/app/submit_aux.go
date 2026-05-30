package app

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"goc/commands"
	"goc/tools/toolexecution"
	"goc/types"
)

type permissionAskReply struct {
	dec toolexecution.PermissionDecision
	err error
}

// gouPermissionAskMsg is sent from [toolexecution.ExecutionDeps.AskResolver] to the Bubble Tea Update loop.
type gouPermissionAskMsg struct {
	toolName     string
	toolUseID    string
	input        json.RawMessage
	prompt       string
	replyCh      chan permissionAskReply
	questions    []ParsedQuestion // set for AskUserQuestion
	askAutoFirst bool             // if true, auto-allow AskUserQuestion instead of showing UI
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

func (m *model) installAskResolver(te *toolexecution.ExecutionDeps, askAutoFirst bool) {
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

		// AskUserQuestion: parse questions and either auto-allow or show interactive UI.
		var questions []ParsedQuestion
		if toolName == "AskUserQuestion" {
			qs, err := parseAskUserQuestionInput(input)
			if err != nil {
				return toolexecution.DenyDecision("invalid AskUserQuestion input: " + err.Error()), nil
			}
			questions = qs
			if askAutoFirst {
				// Auto-select first option for each question, matching TS AskAutoFirst behavior.
				answers := make(map[string]string)
				for _, q := range questions {
					answers[q.Question] = q.Options[0].Label
				}
				// Build updatedInput with auto-answers so the tool handler returns them.
				updated := buildUpdatedInputWithAnswers(input, answers, nil)
				return toolexecution.PermissionDecision{
					Behavior:     toolexecution.PermissionAllow,
					UpdatedInput: updated,
				}, nil
			}
		}

		send(gouPermissionAskMsg{
			toolName:     toolName,
			toolUseID:    toolUseID,
			input:        input,
			prompt:       prompt,
			replyCh:      ch,
			questions:    questions,
			askAutoFirst: askAutoFirst,
		})
		select {
		case r := <-ch:
			return r.dec, r.err
		case <-ctx.Done():
			return toolexecution.DenyDecision("cancelled"), ctx.Err()
		}
	}
}

// buildUpdatedInputWithAnswers merges user answers into the original tool input JSON.
func buildUpdatedInputWithAnswers(original json.RawMessage, answers map[string]string, annotations map[string]any) json.RawMessage {
	var in map[string]any
	if err := json.Unmarshal(original, &in); err != nil {
		in = make(map[string]any)
	}
	in["answers"] = answers
	if len(annotations) > 0 {
		in["annotations"] = annotations
	}
	b, _ := json.Marshal(in)
	return b
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
	m.refreshAgentSuggestions()
}

// slashListVisible is true when the command list should show: leading "/" (TS), mid-input
// whitespace+"/token", or F2.
func (m *model) slashListVisible() bool {
	m.loadSlashCommandsOnce()
	m.slashPicker.SetCommands(m.slashCommands)
	v := m.pr.Value()
	cur := m.pr.CursorRuneIndex()
	return m.slashPicker.Visible(v, cur, m.uiScreen == gouDemoScreenPrompt)
}

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
		m.slashPicker.NavUp()
	} else {
		m.slashPicker.NavDown(m.visibleSlashList())
	}
	return true
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
