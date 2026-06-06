package app

import (
	"encoding/json"
	"fmt"

	"goc/gou/ccbstream"
	"goc/gou/conversation"
	"goc/gou/pui"
	state "goc/gou/app/state"

	tea "charm.land/bubbletea/v2"
)

// Streaming / query-parity / NDJSON stream UI updates (extracted from [model.Update] for navigation).

func (m *model) handleUpdateGouQueryYield(msg gouQueryYieldMsg) (tea.Model, tea.Cmd) {
	// Clear streaming text when a complete message arrives (mirrors TS handleMessageFromStream:
	// onStreamingText(() => null) for all non-stream_event message types).
	m.Conversation.Store.ClearStreaming()
	m.Conversation.Store.AppendMessage(msg.Message)
	m.rebuildHeightCache()
	if m.Screen.Mode != state.ScreenTranscript {
		m.Scroll.Sticky = true
		m.Scroll.Top = 1 << 30
	}
	return m, nil
}

// handleUpdateGouStreamEvent processes raw SSE stream events (content_block_delta) for incremental
// streaming text display. Mirrors TS handleMessageFromStream content_block_delta → text_delta /
// thinking_delta paths where onStreamingText appends delta text character by character.
func (m *model) handleUpdateGouStreamEvent(msg gouStreamEventMsg) (tea.Model, tea.Cmd) {
	var wrap struct {
		Delta struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			Thinking string `json:"thinking"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(msg.Raw, &wrap); err != nil {
		return m, nil
	}
	switch wrap.Delta.Type {
	case "text_delta":
		if wrap.Delta.Text != "" {
			m.Conversation.Store.AppendStreamingChunk(wrap.Delta.Text)
		}
	case "thinking_delta":
		if wrap.Delta.Thinking != "" {
			m.Conversation.Store.AppendStreamingThinkingChunk(wrap.Delta.Thinking)
		}
	}
	if (wrap.Delta.Text != "" || wrap.Delta.Thinking != "") && m.Screen.Mode != state.ScreenTranscript {
		m.Scroll.Sticky = true
		m.Scroll.Top = 1 << 30
	}
	return m, nil
}

func (m *model) handleUpdateGouStreamingToolUses(msg gouStreamingToolUsesMsg) (tea.Model, tea.Cmd) {
	if msg.Uses == nil {
		m.Conversation.Store.ClearStreamingToolUses()
	} else {
		m.Conversation.Store.ClearStreamingToolUses()
		for _, u := range msg.Uses {
			m.Conversation.Store.AppendStreamingToolUse(conversation.StreamingToolUse{
				Index:         u.Index,
				ToolUseID:     u.ToolUseID,
				Name:          u.Name,
				UnparsedInput: u.UnparsedInput,
			})
		}
	}
	if m.Screen.Mode == state.ScreenTranscript {
		m.rebuildHeightCache()
	}
	if m.Screen.Mode != state.ScreenTranscript {
		m.Scroll.Sticky = true
		m.Scroll.Top = 1 << 30
	}

	return m, nil
}

func (m *model) handleUpdateGouSpinnerTick(_ gouSpinnerTickMsg) (tea.Model, tea.Cmd) {
	if !m.Query.Busy {
		return m, nil
	}
	m.Query.SpinnerFrame++
	return m, spinnerTickCmd()
}

func (m *model) handleUpdateGouMemoryAppend(msg gouMemoryAppendMsg) (tea.Model, tea.Cmd) {
	m.Conversation.Store.AppendMessage(msg.Msg)
	m.rebuildHeightCache()
	if m.Screen.Mode != state.ScreenTranscript {
		m.Scroll.Sticky = true
		m.Scroll.Top = 1 << 30
	}
	return m, nil
}

func (m *model) handleUpdateGouQueryDone(msg gouQueryDoneMsg) (tea.Model, tea.Cmd) {
	m.Query.Busy = false
	m.Query.Cancel = nil
	m.Query.CtrlCPending = false
	m.endQuerySpinner()
	m.Conversation.Store.ClearStreamingToolUses()
	if msg.Err != nil {
		m.Conversation.Store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: query streaming: %v", msg.Err)))
		m.rebuildHeightCache()
	} else if gouDemoEnvTruthy("GOU_DEMO_BELL") {
		fmt.Print("\a")
	}
	if m.Conversation.Transcript != nil {
		m.maybeRecordTranscript()
	}
	m.rebuildHeightCache()
	if m.Screen.Mode != state.ScreenTranscript {
		m.Scroll.Sticky = true
		m.Scroll.Top = 1 << 30
	}
	return m, nil
}

func (m *model) handleUpdateCompactPhase(msg compactPhaseMsg) (tea.Model, tea.Cmd) {
	switch msg.Phase {
	case "started":
		m.Query.PreCompactVerb = m.Query.SpinnerVerb
		m.Query.SpinnerVerb = "Compacting"
	case "done":
		if m.Query.PreCompactVerb != "" {
			m.Query.SpinnerVerb = m.Query.PreCompactVerb
			m.Query.PreCompactVerb = ""
		}
	}
	return m, nil
}

func (m *model) handleUpdateCCBStream(msg ccbstream.Msg) (tea.Model, tea.Cmd) {
	ev := ccbstream.StreamEvent(msg)
	switch ev.Type {
	case "assistant_delta":
		gouDemoTracef("ui ccbstream.Msg assistant_delta textLen=%d", len(ev.Text))
	case "error":
		gouDemoTracef("ui ccbstream.Msg error code=%q message=%q", ev.Code, ev.Message)
	default:
		gouDemoTracef("ui ccbstream.Msg type=%s", ev.Type)
	}
	ccbstream.Apply(m.Conversation.Store, ev)
	if ccbStreamEventNeedsFullHeightRebuild(ev) {
		m.rebuildHeightCache()
	}
	if m.Conversation.Transcript != nil && (ev.Type == "turn_complete" || ev.Type == "response_end") {
		m.maybeRecordTranscript()
	}
	if m.Screen.Mode != state.ScreenTranscript {
		switch ev.Type {
		case "assistant_delta", "tool_use", "tool_result", "turn_complete", "error":
			m.Scroll.Sticky = true
			m.Scroll.Top = 1 << 30
		}
	}
	return m, nil
}
