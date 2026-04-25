package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"goc/ccb-engine/diaglog"
)

// gouDemoBubblesViewport defaults on (bubbles/viewport for the prompt message pane, same scrolling style as go-tui).
// Disable with GOU_DEMO_BUBBLES_VIEWPORT=0|false|off|no to render the new renderer's visible slice directly on top of m.scrollTop instead of a full-document viewport.
func gouDemoBubblesViewport() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_BUBBLES_VIEWPORT")))
	if v == "0" || v == "false" || v == "off" || v == "no" {
		return false
	}
	return true
}

// msgViewportWanted is true when the bubbles/viewport message pane is available (new renderer drives both prompt and transcript).
func (m *model) msgViewportWanted() bool {
	result := m.useMsgViewport && !m.msgViewportFallback
	//diaglog.Line("[viewport] msgViewportWanted: useMsgViewport=%v, msgViewportFallback=%v, returning %v", m.useMsgViewport, m.msgViewportFallback, result)
	return result
}

// messagePaneContentSig changes when the message list body should be rebuilt for the viewport pane.
// msgFoldRev bumps on ctrl+y so fold toggles always rebuild even if other fields unchanged.
func (m *model) messagePaneContentSig() string {
	chunk := len(m.store.StreamingText) / 32
	return fmt.Sprintf("%d|%d|%d|%v|%d", len(m.store.Messages), len(m.store.StreamingToolUses), chunk, m.msgFoldAll, m.msgFoldRev)
}

// gouDemoMsgViewportKeyMap aligns bubbles/viewport keybindings with handleMsgViewportScrollKey (pager keys, not h/l).
func gouDemoMsgViewportKeyMap() viewport.KeyMap {
	def := viewport.DefaultKeyMap()
	return viewport.KeyMap{
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+f", "ctrl+n"),
			key.WithHelp("ctrl+f", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+b", "ctrl+p"),
			key.WithHelp("ctrl+b", "page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("pgdown", "space", "ctrl+d"),
			key.WithHelp("pgdn", "½ page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("pgup", "b", "ctrl+u"),
			key.WithHelp("pgup", "½ page up"),
		),
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "down"),
		),
		Left:  def.Left,
		Right: def.Right,
	}
}

func (m *model) msgViewportSyncGeometry() {
	if !m.msgViewportWanted() {
		diaglog.Line("[viewport] msgViewportSyncGeometry: msgViewportWanted=false, returning")
		return
	}
	w := m.messageBodyColsForLayout()
	h := listViewportH(m)
	if w < 1 {
		w = 40
	}
	if h < 3 {
		h = 3
	}
	sig := fmt.Sprintf("%d,%d", w, h)
	//diaglog.Line("[viewport] msgViewportSyncGeometry: w=%d, h=%d, sig=%s, lastVpGeom=%s", w, h, sig, m.lastVpGeom)
	if sig != m.lastVpGeom {
		if m.msgViewport.Width() == 0 || m.msgViewport.Height() == 0 {
			diaglog.Line("[viewport] msgViewportSyncGeometry: creating new viewport")
			m.msgViewport = viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
		} else {
			diaglog.Line("[viewport] msgViewportSyncGeometry: resizing existing viewport")
			m.msgViewport.SetWidth(w)
			m.msgViewport.SetHeight(h)
		}
		m.msgViewport.KeyMap = gouDemoMsgViewportKeyMap()
		m.msgViewport.MouseWheelEnabled = true
		m.lastVpGeom = sig
		m.vpNeedResizeContent = true
		diaglog.Line("[viewport] msgViewportSyncGeometry: viewport created/resized, width=%d, height=%d, listViewportH=%d", m.msgViewport.Width(), m.msgViewport.Height(), h)
	}
}

// applyMsgViewportContentFromView rebuilds bubbles/viewport content from the new renderer.
// It skips rebuild if the content signature is unchanged (unless vpNeedResizeContent forces it).
// On rebuild failure, it sets msgViewportFallback=true so the caller falls back to the old renderer.
// When sticky (auto-scroll), it calls GotoBottom after a no-op or successful content refresh.
func (m *model) applyMsgViewportContentFromView() {
	// Viewport 不可用时直接返回（例如 fallback 模式）
	if !m.msgViewportWanted() {
		diaglog.Line("[viewport] applyMsgViewportContentFromView: msgViewportWanted=false, returning")
		return
	}

	// 计算内容签名，与上次对比判断是否需要重建
	sig := m.messagePaneContentSig()
	if sig == m.lastVpContentSig && !m.vpNeedResizeContent {
		// 内容未变化：仅 sticky 模式下滚动到底部保持跟随
		if m.sticky {
			m.msgViewport.GotoBottom()
		}
		//diaglog.Line("[viewport] applyMsgViewportContentFromView: content unchanged, sig=%s", sig)
		return
	}

	// 内容有变化：用新渲染器生成完整文档内容
	s, ok := m.tryBuildFullMessagePaneContentWithNewRenderer()
	if !ok {
		// 新渲染器失败：切回 fallback 模式，清除签名以便下次完整重建
		diaglog.Line("[viewport] applyMsgViewportContentFromView: build failed, setting fallback")
		m.msgViewportFallback = true
		m.lastVpContentSig = ""
		m.vpNeedResizeContent = false
		return
	}

	// 将构建好的内容设置到 viewport 中，更新签名标记
	m.msgViewport.SetContent(s)
	m.lastVpContentSig = sig
	m.vpNeedResizeContent = false

	// sticky 模式下自动滚动到底部
	if m.sticky {
		m.msgViewport.GotoBottom()
	}
}

// maybeTeaResetHistoryBrowseMouse clears go-tui/test.go history-browse mode and re-enables SGR mouse if needed.
func (m *model) maybeTeaResetHistoryBrowseMouse() tea.Cmd {
	if !m.msgHistoryBrowseMouseOff {
		return nil
	}
	m.msgHistoryBrowseMouseOff = false
	return nil
}

// handleMsgViewportScrollKey forwards list keys through bubbles/viewport.Update (go-tui/main pattern) plus
// GotoTop/GotoBottom bindings not in the default viewport keymap.
func (m *model) handleMsgViewportScrollKey(msg tea.KeyPressMsg) tea.Cmd {
	diaglog.Line("[viewport] handleMsgViewportScrollKey: key=%s, viewport width=%d, height=%d", msg.String(), m.msgViewport.Width(), m.msgViewport.Height())
	var cmd tea.Cmd
	m.msgViewport, cmd = m.msgViewport.Update(msg)
	diaglog.Line("[viewport] handleMsgViewportScrollKey: after Update, yOffset=%d, totalLines=%d, AtTop=%v, AtBottom=%v",
		m.msgViewport.YOffset(), m.msgViewport.TotalLineCount(), m.msgViewport.AtTop(), m.msgViewport.AtBottom())
	switch msg.String() {
	case "end", "G", "shift+g", "ctrl+end":
		m.sticky = true
		m.msgViewport.GotoBottom()
		return cmd
	case "home", "ctrl+home":
		m.msgViewport.GotoTop()
		m.sticky = false
		return cmd
	}
	if !m.msgViewport.AtBottom() {
		m.sticky = false
	}
	return cmd
}

// messagePaneViewportBlock renders the message list using bubbles/viewport.
// Caller must run msgViewportSyncGeometry + applyMsgViewportContentFromView first.
func (m *model) messagePaneViewportBlock(vpH, bodyCols int) string {
	msgArea := m.msgViewport.View()
	lines := strings.Split(msgArea, "\n")
	for len(lines) < vpH {
		lines = append(lines, "")
	}
	if len(lines) > vpH {
		lines = lines[:vpH]
	}
	totalH := m.msgViewport.TotalLineCount()
	if totalH < vpH {
		totalH = vpH
	}
	return joinMessagePaneLinesWithScrollbar(lines, bodyCols, vpH, totalH, m.msgViewport.YOffset(), m.msgScrollbarW)
}

func (m *model) handleMsgViewportMouseWheel(delta int) {
	if delta == 0 {
		return
	}
	n := messageListMouseWheelStep(listViewportH(m))
	if delta < 0 {
		m.msgViewport.ScrollDown(n)
	} else {
		m.msgViewport.ScrollUp(n)
	}
	if !m.msgViewport.AtBottom() {
		m.sticky = false
	}
}
