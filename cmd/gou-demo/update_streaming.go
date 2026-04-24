package main

import (
	"fmt"

	"goc/gou/ccbstream"
	"goc/gou/conversation"
	"goc/gou/pui"

	tea "charm.land/bubbletea/v2"
)

// Streaming / query-parity / NDJSON stream UI updates (extracted from [model.Update] for navigation).

func (m *model) handleUpdateGouQueryYield(msg gouQueryYieldMsg) (tea.Model, tea.Cmd) {
	m.store.AppendMessage(msg.Message)
	m.rebuildHeightCache()
	if m.uiScreen != gouDemoScreenTranscript {
		m.sticky = true
		m.scrollTop = 1 << 30
	}
	return m, nil
}

func (m *model) handleUpdateGouStreamingToolUses(msg gouStreamingToolUsesMsg) (tea.Model, tea.Cmd) {
	if msg.Uses == nil {
		m.store.ClearStreamingToolUses()
	} else {
		m.store.ClearStreamingToolUses()
		for _, u := range msg.Uses {
			m.store.AppendStreamingToolUse(conversation.StreamingToolUse{
				Index:         u.Index,
				ToolUseID:     u.ToolUseID,
				Name:          u.Name,
				UnparsedInput: u.UnparsedInput,
			})
		}
	}
	if m.uiScreen == gouDemoScreenTranscript {
		m.rebuildHeightCache()
	}
	if m.uiScreen != gouDemoScreenTranscript {
		m.sticky = true
		m.scrollTop = 1 << 30
	}

	return m, nil
}

func (m *model) handleUpdateGouSpinnerTick(_ gouSpinnerTickMsg) (tea.Model, tea.Cmd) {
	if !m.queryBusy {
		return m, nil
	}
	m.spinnerFrame++
	return m, spinnerTickCmd()
}

func (m *model) handleUpdateGouQueryDone(msg gouQueryDoneMsg) (tea.Model, tea.Cmd) {
	m.queryBusy = false
	m.endQuerySpinner()
	m.store.ClearStreamingToolUses()
	if msg.Err != nil {
		m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: query streaming: %v", msg.Err)))
		m.rebuildHeightCache()
	} else if gouDemoEnvTruthy("GOU_DEMO_BELL") {
		fmt.Print("\a")
	}
	gouDemoLogStoreMessages("after_query_stream", m.store)
	if m.transcript != nil {
		m.maybeRecordTranscript()
	}
	m.rebuildHeightCache()
	if m.uiScreen != gouDemoScreenTranscript {
		m.sticky = true
		m.scrollTop = 1 << 30
	}
	return m, nil
}

func (m *model) handleUpdateCCBStream(msg ccbstream.Msg) (tea.Model, tea.Cmd) {
	ev := ccbstream.StreamEvent(msg)
	if gouDemoTrace != nil {
		switch ev.Type {
		case "assistant_delta":
			gouDemoTracef("ui ccbstream.Msg assistant_delta textLen=%d", len(ev.Text))
		case "error":
			gouDemoTracef("ui ccbstream.Msg error code=%q message=%q", ev.Code, ev.Message)
		default:
			gouDemoTracef("ui ccbstream.Msg type=%s", ev.Type)
		}
	}
	ccbstream.Apply(m.store, ev)
	if ev.Type == "turn_complete" || ev.Type == "response_end" {
		gouDemoLogStoreMessages("after_stream_"+ev.Type, m.store)
	}
	if ccbStreamEventNeedsFullHeightRebuild(ev) {
		m.rebuildHeightCache()
	}
	if m.transcript != nil && (ev.Type == "turn_complete" || ev.Type == "response_end") {
		m.maybeRecordTranscript()
	}
	if m.uiScreen != gouDemoScreenTranscript {
		switch ev.Type {
		case "assistant_delta", "tool_use", "tool_result", "turn_complete", "error":
			m.sticky = true
			m.scrollTop = 1 << 30
		}
	}
	return m, nil
}
