package update

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"goc/gou/app/config"
	"goc/gou/pui"
	state "goc/gou/app/state"
)

func (d *Dispatcher) handleQueryYield(msg config.QueryYieldMsg) (tea.Model, tea.Cmd) {
	// Clear streaming text when a complete message arrives (mirrors TS handleMessageFromStream:
	// onStreamingText(() => null) for all non-stream_event message types).
	d.deps.GetConversation().Store.ClearStreaming()
	d.deps.GetConversation().Store.AppendMessage(msg.Message)
	d.deps.RebuildHeightCache()
	if d.deps.GetScreen().Mode != state.ScreenTranscript {
		d.deps.GetScroll().Sticky = true
		d.deps.GetScroll().Top = 1 << 30
	}
	return d.deps.Model(), nil
}

func (d *Dispatcher) handleQueryDone(msg config.QueryDoneMsg) (tea.Model, tea.Cmd) {
	d.deps.GetQuery().Busy = false
	d.deps.GetQuery().Cancel = nil
	d.deps.GetQuery().CtrlCPending = false
	d.deps.EndQuerySpinner()
	d.deps.GetConversation().Store.ClearStreamingToolUses()
	if msg.Err != nil {
		d.deps.GetConversation().Store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: query streaming: %v", msg.Err)))
		d.deps.RebuildHeightCache()
	} else if config.EnvTruthy("GOU_DEMO_BELL") {
		fmt.Print("\a")
	}
	if d.deps.GetConversation().Transcript != nil {
		d.deps.MaybeRecordTranscript()
	}
	d.deps.RebuildHeightCache()
	if d.deps.GetScreen().Mode != state.ScreenTranscript {
		d.deps.GetScroll().Sticky = true
		d.deps.GetScroll().Top = 1 << 30
	}
	return d.deps.Model(), nil
}

func (d *Dispatcher) handleMemoryAppend(msg config.MemoryAppendMsg) (tea.Model, tea.Cmd) {
	d.deps.GetConversation().Store.AppendMessage(msg.Msg)
	d.deps.RebuildHeightCache()
	if d.deps.GetScreen().Mode != state.ScreenTranscript {
		d.deps.GetScroll().Sticky = true
		d.deps.GetScroll().Top = 1 << 30
	}
	return d.deps.Model(), nil
}

func (d *Dispatcher) handleCompactPhase(msg config.CompactPhaseMsg) (tea.Model, tea.Cmd) {
	switch msg.Phase {
	case "started":
		d.deps.GetQuery().PreCompactVerb = d.deps.GetQuery().SpinnerVerb
		d.deps.GetQuery().SpinnerVerb = "Compacting"
	case "done":
		if d.deps.GetQuery().PreCompactVerb != "" {
			d.deps.GetQuery().SpinnerVerb = d.deps.GetQuery().PreCompactVerb
			d.deps.GetQuery().PreCompactVerb = ""
		}
	}
	return d.deps.Model(), nil
}
