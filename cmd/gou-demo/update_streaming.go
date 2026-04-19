package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goc/gou/ccbstream"
	"goc/gou/conversation"
	"goc/gou/pui"
	"goc/types"

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

	if len(m.store.StreamingToolUses) > 0 {
		//return m, tea.Tick(20*time.Millisecond, func(time.Time) tea.Msg { return streamToolRevealTickMsg{} })
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

func (m *model) handleUpdateStreamTick(_ streamTick) (tea.Model, tea.Cmd) {
	if len(m.streamChunks) == 0 || m.streamIdx >= len(m.streamChunks) {
		return m, nil
	}
	m.store.AppendStreamingChunk(m.streamChunks[m.streamIdx])
	m.streamIdx++
	if m.streamIdx < len(m.streamChunks) {
		return m, tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg { return streamTick{} })
	}
	raw, _ := json.Marshal([]map[string]string{{"type": "text", "text": strings.TrimSpace(m.store.StreamingText)}})
	m.store.AppendMessage(types.Message{
		UUID:    fmt.Sprintf("a-%d", time.Now().UnixNano()),
		Type:    types.MessageTypeAssistant,
		Content: raw,
	})
	m.store.ClearStreaming()
	m.store.ClearStreamingToolUses()
	m.streamChunks = nil
	m.streamIdx = 0
	m.queryBusy = false
	m.endQuerySpinner()
	m.rebuildHeightCache()
	gouDemoTracef("fake streamTick finished storeMessages=%d", len(m.store.Messages))
	if m.transcript != nil {
		m.maybeRecordTranscript()
	}
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
	if ev.Type == "tool_use" {
		return m, tea.Tick(20*time.Millisecond, func(time.Time) tea.Msg { return streamToolRevealTickMsg{} })
	}
	return m, nil
}
