package update

import (
	"encoding/json"

	tea "charm.land/bubbletea/v2"
	"goc/gou/app/config"
	"goc/gou/conversation"
	state "goc/gou/app/state"
)

func (d *Dispatcher) handleStreamEvent(msg config.StreamEventMsg) (tea.Model, tea.Cmd) {
	var wrap struct {
		Delta struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			Thinking string `json:"thinking"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(msg.Raw, &wrap); err != nil {
		return d.deps.Model(), nil
	}
	switch wrap.Delta.Type {
	case "text_delta":
		if wrap.Delta.Text != "" {
			d.deps.GetConversation().Store.AppendStreamingChunk(wrap.Delta.Text)
		}
	case "thinking_delta":
		if wrap.Delta.Thinking != "" {
			d.deps.GetConversation().Store.AppendStreamingThinkingChunk(wrap.Delta.Thinking)
		}
	}
	if (wrap.Delta.Text != "" || wrap.Delta.Thinking != "") && d.deps.GetScreen().Mode != state.ScreenTranscript {
		d.deps.GetScroll().Sticky = true
		d.deps.GetScroll().Top = 1 << 30
	}
	return d.deps.Model(), nil
}

func (d *Dispatcher) handleStreamingToolUses(msg config.StreamingToolUsesMsg) (tea.Model, tea.Cmd) {
	if msg.Uses == nil {
		d.deps.GetConversation().Store.ClearStreamingToolUses()
	} else {
		d.deps.GetConversation().Store.ClearStreamingToolUses()
		for _, u := range msg.Uses {
			d.deps.GetConversation().Store.AppendStreamingToolUse(conversation.StreamingToolUse{
				Index:         u.Index,
				ToolUseID:     u.ToolUseID,
				Name:          u.Name,
				UnparsedInput: u.UnparsedInput,
			})
		}
	}
	if d.deps.GetScreen().Mode == state.ScreenTranscript {
		d.deps.RebuildHeightCache()
	}
	if d.deps.GetScreen().Mode != state.ScreenTranscript {
		d.deps.GetScroll().Sticky = true
		d.deps.GetScroll().Top = 1 << 30
	}

	return d.deps.Model(), nil
}
