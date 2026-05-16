package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"goc/gou/suggestions"
	"goc/types"
)

// syncAtSuggestions runs after every prompt Update to refresh the @ suggestion list.
func (m *model) syncAtSuggestions() {
	if m.suggestionEngine == nil {
		return
	}
	prevVisible := m.suggVisible
	value := m.pr.Value()
	cursor := m.pr.CursorRuneIndex()
	result := m.suggestionEngine.Update(value, cursor)
	if result == nil || !result.HasResults {
		m.suggVisible = false
		m.suggestions = nil
		m.selectedSuggIdx = 0
	} else {
		m.suggestions = result.Items
		if m.selectedSuggIdx >= len(m.suggestions) {
			m.selectedSuggIdx = 0
		}
		m.suggVisible = true
	}
	if prevVisible != m.suggVisible {
		m.rebuildHeightCache()
	}
}

// handleAtSuggestKeys handles keyboard input when the @ suggestion list is visible.
// Returns: 0 = not handled, 1 = handled.
func (m *model) handleAtSuggestKeys(msg tea.KeyPressMsg) int {
	if m.uiScreen != gouDemoScreenPrompt || !m.suggVisible || len(m.suggestions) == 0 {
		return 0
	}
	k := msg.Key()
	switch msg.String() {
	case "tab":
		m.applySuggestion(m.suggestions[m.selectedSuggIdx])
		return 1
	case "up":
		if m.selectedSuggIdx > 0 {
			m.selectedSuggIdx--
		} else {
			m.selectedSuggIdx = len(m.suggestions) - 1
		}
		return 1
	case "down":
		if m.selectedSuggIdx+1 < len(m.suggestions) {
			m.selectedSuggIdx++
		} else {
			m.selectedSuggIdx = 0
		}
		return 1
	case "esc":
		m.suggVisible = false
		m.suggestionEngine.Dismiss()
		return 1
	}
	// Arrow key codes (non-string forms from some terminals)
	if !k.Mod.Contains(tea.ModShift) {
		if k.Code == tea.KeyUp {
			if m.selectedSuggIdx > 0 {
				m.selectedSuggIdx--
			} else {
				m.selectedSuggIdx = len(m.suggestions) - 1
			}
			return 1
		}
		if k.Code == tea.KeyDown {
			if m.selectedSuggIdx+1 < len(m.suggestions) {
				m.selectedSuggIdx++
			} else {
				m.selectedSuggIdx = 0
			}
			return 1
		}
	}
	// Enter: apply suggestion without submitting (user continues typing)
	if isPromptEnterKey(msg) {
		m.applySuggestion(m.suggestions[m.selectedSuggIdx])
		return 1
	}
	return 0
}

// applySuggestion replaces the @token at the cursor with the selected suggestion value.
func (m *model) applySuggestion(item suggestions.ScoredItem) {
	value := m.pr.Value()
	cursor := m.pr.CursorRuneIndex()
	token, rng := extractCompletionTokenForApply(value, cursor)
	if token == "" {
		return
	}
	rs := []rune(value)
	rep := item.Value + " "
	var b strings.Builder
	b.WriteString(string(rs[:rng.Start]))
	b.WriteString(rep)
	b.WriteString(string(rs[rng.End:]))
	m.pr.SetValue(b.String())
	m.suggVisible = false
	m.selectedSuggIdx = 0
}

// extractCompletionTokenForApply finds the @token to replace when applying a suggestion.
// Scans runes backward from cursor to find @ preceded by space or line start.
// Returns the token text (without @) and the range covering [@, tokenEnd).
func extractCompletionTokenForApply(value string, cursor int) (string, suggestions.CompletionRange) {
	rs := []rune(value)
	if cursor < 0 || cursor > len(rs) {
		return "", suggestions.CompletionRange{}
	}
	// Find @ before cursor by scanning runes backward from cursor
	atIdx := -1
	for i := cursor - 1; i >= 0; i-- {
		if rs[i] == '@' {
			// Check that @ is preceded by space or line start
			if i == 0 || rs[i-1] == ' ' || rs[i-1] == '\n' {
				atIdx = i
				break
			}
		}
	}
	if atIdx < 0 {
		return "", suggestions.CompletionRange{}
	}
	token := string(rs[atIdx+1 : cursor])
	return token, suggestions.CompletionRange{Start: atIdx, End: cursor}
}

// renderAtSuggestions renders the suggestion list above the prompt input (footer area).
func (m *model) renderAtSuggestions() string {
	if !m.suggVisible || len(m.suggestions) == 0 || m.uiScreen != gouDemoScreenPrompt {
		return ""
	}
	width := m.cols
	if width < 40 {
		width = 40
	}
	maxVisible := 6
	if len(m.suggestions) < maxVisible {
		maxVisible = len(m.suggestions)
	}
	// Center the visible window on selectedSuggIdx
	start := m.selectedSuggIdx - maxVisible/2
	if start < 0 {
		start = 0
	}
	if start+maxVisible > len(m.suggestions) {
		start = len(m.suggestions) - maxVisible
	}
	if start < 0 {
		start = 0
	}

	var b strings.Builder
	// Title line
	title := lipgloss.NewStyle().Bold(true).Render("Suggestions  ") +
		lipgloss.NewStyle().Faint(true).Render("Tab/Enter accept  Esc dismiss")
	b.WriteString(lipgloss.NewStyle().Width(width).MaxWidth(width).Render(title))
	b.WriteByte('\n')

	for i := start; i < len(m.suggestions) && i < start+maxVisible; i++ {
		item := m.suggestions[i]
		icon := item.Icon
		if icon == "" {
			switch item.Type {
			case suggestions.SuggestionTypeFile:
				icon = "F"
			case suggestions.SuggestionTypeDirectory:
				icon = "D"
			case suggestions.SuggestionTypeAgent:
				icon = "*"
			case suggestions.SuggestionTypeMcpResource:
				icon = "◇"
			}
		}
		line := "  " + icon + " " + item.Label
		if i == m.selectedSuggIdx {
			line = lipgloss.NewStyle().Reverse(true).Render("  " + icon + " " + item.Label)
		}
		b.WriteString(lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line))
		b.WriteByte('\n')
	}
	return b.String()
}

// refreshAgentSuggestions updates the suggestion engine's agent list from the loaded slash commands.
func (m *model) refreshAgentSuggestions() {
	if m.suggestionEngine == nil {
		return
	}
	var agents []suggestions.AgentDef
	for _, cmd := range m.slashCommands {
		if cmd.Agent != nil {
			agents = append(agents, suggestions.AgentDef{
				Name:        types.GetCommandName(cmd),
				DisplayName: types.GetCommandName(cmd),
				Description: cmd.Description,
			})
		}
	}
	m.suggestionEngine.SetAgents(agents)
}
