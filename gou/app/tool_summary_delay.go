// Tool summary delay: before merged "Searched for … / Read …" lines, show full Search/Grep/Read rows for a configurable time (GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS).

package app

import (
	"os"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"goc/types"
	state "goc/gou/app/state"
)

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
	if m.MessageTracking.FirstShownAt == nil {
		m.MessageTracking.FirstShownAt = make(map[string]time.Time)
	}
	if m.MessageTracking.LastAssistantContentLen == nil {
		m.MessageTracking.LastAssistantContentLen = make(map[string]int)
	}
	seen := make(map[string]struct{})
	for i := range m.Conversation.Store.Messages {
		msg := m.Conversation.Store.Messages[i]
		id := strings.TrimSpace(msg.UUID)
		if id == "" {
			continue
		}
		seen[id] = struct{}{}
		if msg.Type == types.MessageTypeAssistant {
			n := len(msg.Content)
			prev := m.MessageTracking.LastAssistantContentLen[id]
			if n > prev {
				m.MessageTracking.FirstShownAt[id] = time.Now()
				m.MessageTracking.LastAssistantContentLen[id] = n
			} else if _, ok := m.MessageTracking.FirstShownAt[id]; !ok {
				m.MessageTracking.FirstShownAt[id] = time.Now()
				m.MessageTracking.LastAssistantContentLen[id] = n
			}
		} else {
			if _, ok := m.MessageTracking.FirstShownAt[id]; !ok {
				m.MessageTracking.FirstShownAt[id] = time.Now()
			}
		}
	}
	for k := range m.MessageTracking.FirstShownAt {
		if _, ok := seen[k]; !ok {
			delete(m.MessageTracking.FirstShownAt, k)
			delete(m.MessageTracking.LastAssistantContentLen, k)
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
	t0, ok := m.MessageTracking.FirstShownAt[id]
	if !ok {
		return false
	}
	return time.Since(t0) < d
}

func (m *model) anyToolSummaryDelayPending() bool {
	d := gouDemoToolUseSummaryDelay()
	if d <= 0 || m.Screen.Mode != state.ScreenPrompt {
		return false
	}
	now := time.Now()
	for i := range m.Conversation.Store.Messages {
		msg := m.Conversation.Store.Messages[i]
		if msg.Type != types.MessageTypeAssistant {
			continue
		}
		id := strings.TrimSpace(msg.UUID)
		t0, ok := m.MessageTracking.FirstShownAt[id]
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
	if m.Screen.Mode == state.ScreenPrompt && m.anyToolSummaryDelayPending() {
		m.rebuildHeightCache()
	}

	//return m, nil
	return m, tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return gouToolSummaryDelayTickMsg{} })
}
