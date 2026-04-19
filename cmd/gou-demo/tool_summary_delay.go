// Tool summary delay: before merged "Searched for … / Read …" lines, show full Search/Grep/Read rows for a configurable time (GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS).

package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"goc/types"
)

// streamToolRevealTickMsg drives periodic height rebuild while a streaming tool is in the progressive reveal phase.
type streamToolRevealTickMsg struct{}

// gouToolSummaryDelayTickMsg drives periodic height rebuild while any assistant row is still in the "detail" phase.
type gouToolSummaryDelayTickMsg struct{}

// gouDemoToolUseSummaryDelay returns how long to show full Grep/Glob/Read chrome before merged summary lines (prompt only).
// Empty/unset env defaults to 2s; 0 disables. Negative or invalid values are treated as 0.
func gouDemoToolUseSummaryDelay() time.Duration {
	v := strings.TrimSpace(os.Getenv("GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS"))
	if v == "" {
		return 2 * time.Second
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Millisecond
}

func (m *model) syncMsgFirstShownAt() {
	if m.msgFirstShownAt == nil {
		m.msgFirstShownAt = make(map[string]time.Time)
	}
	if m.msgLastAssistantContentLen == nil {
		m.msgLastAssistantContentLen = make(map[string]int)
	}
	seen := make(map[string]struct{})
	for i := range m.store.Messages {
		msg := m.store.Messages[i]
		id := strings.TrimSpace(msg.UUID)
		if id == "" {
			continue
		}
		seen[id] = struct{}{}
		if msg.Type == types.MessageTypeAssistant {
			n := len(msg.Content)
			prev := m.msgLastAssistantContentLen[id]
			if n > prev {
				m.msgFirstShownAt[id] = time.Now()
				m.msgLastAssistantContentLen[id] = n
			} else if _, ok := m.msgFirstShownAt[id]; !ok {
				m.msgFirstShownAt[id] = time.Now()
				m.msgLastAssistantContentLen[id] = n
			}
		} else {
			if _, ok := m.msgFirstShownAt[id]; !ok {
				m.msgFirstShownAt[id] = time.Now()
			}
		}
	}
	for k := range m.msgFirstShownAt {
		if _, ok := seen[k]; !ok {
			delete(m.msgFirstShownAt, k)
			delete(m.msgLastAssistantContentLen, k)
		}
	}
}

func (m *model) suppressToolUseSummaryLine(msg types.Message) bool {
	if msg.Type != types.MessageTypeAssistant {
		return false
	}
	d := gouDemoToolUseSummaryDelay()
	if d <= 0 {
		return false
	}
	id := strings.TrimSpace(msg.UUID)
	t0, ok := m.msgFirstShownAt[id]
	if !ok {
		return false
	}
	return time.Since(t0) < d
}

func (m *model) anyToolSummaryDelayPending() bool {
	d := gouDemoToolUseSummaryDelay()
	if d <= 0 || m.uiScreen != gouDemoScreenPrompt {
		return false
	}
	now := time.Now()
	for i := range m.store.Messages {
		msg := m.store.Messages[i]
		if msg.Type != types.MessageTypeAssistant {
			continue
		}
		id := strings.TrimSpace(msg.UUID)
		t0, ok := m.msgFirstShownAt[id]
		if !ok {
			continue
		}
		if now.Sub(t0) < d {
			return true
		}
	}
	return false
}

func (m *model) handleUpdateToolSummaryDelayTick(_ gouToolSummaryDelayTickMsg) (tea.Model, tea.Cmd) {
	d := gouDemoToolUseSummaryDelay()
	if d <= 0 {
		return m, nil
	}
	if m.uiScreen == gouDemoScreenPrompt && m.anyToolSummaryDelayPending() {
		m.rebuildHeightCache()
	}
	return m, tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return gouToolSummaryDelayTickMsg{} })
}

func (m *model) handleUpdateStreamToolTick(_ streamToolRevealTickMsg) (tea.Model, tea.Cmd) {
	if len(m.store.StreamingToolUses) > 0 {
		m.vpNeedResizeContent = true
		m.rebuildHeightCache()
		//return m, tea.Tick(20*time.Millisecond, func(time.Time) tea.Msg { return streamToolRevealTickMsg{} })
	}
	return m, nil
}
