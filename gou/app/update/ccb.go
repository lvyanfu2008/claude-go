package update

import (
	tea "charm.land/bubbletea/v2"
	"goc/gou/app/config"
	"goc/gou/ccbstream"
	state "goc/gou/app/state"
)

func (d *Dispatcher) handleCCBStream(msg ccbstream.Msg) (tea.Model, tea.Cmd) {
	ev := ccbstream.StreamEvent(msg)
	switch ev.Type {
	case "assistant_delta":
		config.Tracef("ui ccbstream.Msg assistant_delta textLen=%d", len(ev.Text))
	case "error":
		config.Tracef("ui ccbstream.Msg error code=%q message=%q", ev.Code, ev.Message)
	default:
		config.Tracef("ui ccbstream.Msg type=%s", ev.Type)
	}
	ccbstream.Apply(d.deps.GetConversation().Store, ev)
	if ccbStreamEventNeedsFullHeightRebuild(ev) {
		d.deps.RebuildHeightCache()
	}
	if d.deps.GetConversation().Transcript != nil && (ev.Type == "turn_complete" || ev.Type == "response_end") {
		d.deps.MaybeRecordTranscript()
	}
	if d.deps.GetScreen().Mode != state.ScreenTranscript {
		switch ev.Type {
		case "assistant_delta", "tool_use", "tool_result", "turn_complete", "error":
			d.deps.GetScroll().Sticky = true
			d.deps.GetScroll().Top = 1 << 30
		}
	}
	return d.deps.Model(), nil
}

// ccbStreamEventNeedsFullHeightRebuild mirrors the app-level helper.
func ccbStreamEventNeedsFullHeightRebuild(ev ccbstream.StreamEvent) bool {
	switch ev.Type {
	case "tool_use", "tool_result":
		return true
	}
	return false
}
