package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// renderMessagePane renders the message list area using the bubbles/viewport path.
func (m *Model) renderMessagePane(b *strings.Builder, vpH, bodyCols int, useVp bool) {
	b.WriteString(m.messagePaneViewportBlock(vpH, bodyCols))
	b.WriteByte('\n')
}

// promptAreaLayout renders everything below the message pane.
func (m *Model) promptAreaLayout(b *strings.Builder) (promptLineOffset int) {
	if m.uiScreen == gouDemoScreenTranscript {
		foot := joinFooterLines(transcriptChromeFootLines(m, m.cols > 0 && m.cols < 80), m.cols)
		b.WriteString(lipgloss.NewStyle().Faint(true).Width(m.cols).Render(foot))
		return 0
	}
	// @-mention autocomplete suggestions (above input footer area)
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
	// Track cursor line offset before the prompt for Windows IME positioning.
	// CJK characters are double-width; ConPTY needs explicit cursor coords.
	promptLineOffset = strings.Count(b.String(), "\n")
	promptView := userInputViewWithPromptPrefix(m)
	b.WriteString(promptView)
	// Agent footer (only when sub-agents exist)
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
		vis := m.visibleSlashList()
		hint := m.slashListFooterHint()
		if sp := m.slashPicker.View(vis, m.cols, m.height, hint); sp != "" {
			b.WriteByte('\n')
			b.WriteString(sp)
		}
	}
	return
}
