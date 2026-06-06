package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"goc/gou/app/components/input"
	state "goc/gou/app/state"
	"goc/gou/suggestions"
	"goc/types"
)

// syncAtSuggestions runs after every prompt Update to refresh the @ suggestion list.
func (m *model) syncAtSuggestions() {
	if m.Input.SuggestionEngine == nil {
		return
	}
	prevVisible := m.Input.SuggVisible
	value := m.Input.PR.Value()
	cursor := m.Input.PR.CursorRuneIndex()
	result := m.Input.SuggestionEngine.Update(value, cursor)
	if result == nil || !result.HasResults {
		m.Input.SuggVisible = false
		m.Input.Suggestions = nil
		m.Input.SelectedSuggIdx = 0
	} else {
		m.Input.Suggestions = result.Items
		if m.Input.SelectedSuggIdx >= len(m.Input.Suggestions) {
			m.Input.SelectedSuggIdx = 0
		}
		m.Input.SuggVisible = true
	}
	if prevVisible != m.Input.SuggVisible {
		m.rebuildHeightCache()
	}
}

// handleAtSuggestKeys handles keyboard input when the @ suggestion list is visible.
// Returns: 0 = not handled, 1 = handled.
func (m *model) handleAtSuggestKeys(msg tea.KeyPressMsg) int {
	if m.Screen.Mode != state.ScreenPrompt || !m.Input.SuggVisible || len(m.Input.Suggestions) == 0 {
		return 0
	}
	k := msg.Key()
	switch msg.String() {
	case "tab":
		m.applySuggestion(m.Input.Suggestions[m.Input.SelectedSuggIdx])
		return 1
	case "up":
		if m.Input.SelectedSuggIdx > 0 {
			m.Input.SelectedSuggIdx--
		} else {
			m.Input.SelectedSuggIdx = len(m.Input.Suggestions) - 1
		}
		return 1
	case "down":
		if m.Input.SelectedSuggIdx+1 < len(m.Input.Suggestions) {
			m.Input.SelectedSuggIdx++
		} else {
			m.Input.SelectedSuggIdx = 0
		}
		return 1
	case "esc":
		m.Input.SuggVisible = false
		m.Input.SuggestionEngine.Dismiss()
		return 1
	}
	// Arrow key codes (non-string forms from some terminals)
	if !k.Mod.Contains(tea.ModShift) {
		if k.Code == tea.KeyUp {
			if m.Input.SelectedSuggIdx > 0 {
				m.Input.SelectedSuggIdx--
			} else {
				m.Input.SelectedSuggIdx = len(m.Input.Suggestions) - 1
			}
			return 1
		}
		if k.Code == tea.KeyDown {
			if m.Input.SelectedSuggIdx+1 < len(m.Input.Suggestions) {
				m.Input.SelectedSuggIdx++
			} else {
				m.Input.SelectedSuggIdx = 0
			}
			return 1
		}
	}
	// Enter: apply suggestion without submitting (user continues typing)
	if isPromptEnterKey(msg) {
		m.applySuggestion(m.Input.Suggestions[m.Input.SelectedSuggIdx])
		return 1
	}
	return 0
}

// applySuggestion replaces the @token at the cursor with the selected suggestion value.
func (m *model) applySuggestion(item suggestions.ScoredItem) {
	value := m.Input.PR.Value()
	cursor := m.Input.PR.CursorRuneIndex()
	_, rng := extractCompletionTokenForApply(value, cursor)
	rs := []rune(value)
	rep := "@" + item.Value + " "
	var b strings.Builder
	b.WriteString(string(rs[:rng.Start]))
	b.WriteString(rep)
	b.WriteString(string(rs[rng.End:]))
	m.Input.PR.SetValue(b.String())
	m.Input.SuggVisible = false
	m.Input.SelectedSuggIdx = 0
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

// renderAtSuggestions delegates to input.RenderSuggestions.
func (m *model) renderAtSuggestions() string {
	return input.RenderSuggestions(inputDeps{m})
}

// refreshAgentSuggestions updates the suggestion engine's agent list from the loaded slash commands.
func (m *model) refreshAgentSuggestions() {
	if m.Input.SuggestionEngine == nil {
		return
	}
	var agents []suggestions.AgentDef
	for _, cmd := range m.Input.SlashCommands {
		if cmd.Agent != nil {
			agents = append(agents, suggestions.AgentDef{
				Name:        types.GetCommandName(cmd),
				DisplayName: types.GetCommandName(cmd),
				Description: cmd.Description,
			})
		}
	}
	m.Input.SuggestionEngine.SetAgents(agents)
}
