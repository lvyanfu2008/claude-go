package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"goc/gou/markdown"
	"goc/gou/theme"
	"goc/types"
)

func gouDemoBubblesViewport() bool {
	return gouDemoEnvTruthy("GOU_DEMO_BUBBLES_VIEWPORT")
}

func gouDemoViewportMaxLines() int {
	const def = 20000
	s := strings.TrimSpace(os.Getenv("GOU_DEMO_VIEWPORT_MAX_LINES"))
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 100 {
		return def
	}
	return n
}

func (m *model) msgViewportWanted() bool {
	return m.useMsgViewport && m.uiScreen == gouDemoScreenPrompt && !m.msgViewportFallback
}

// messagePaneContentSig changes when the message list body should be rebuilt for the viewport pane.
// msgFoldRev bumps on ctrl+y so fold toggles always rebuild even if other fields unchanged.
func (m *model) messagePaneContentSig() string {
	chunk := len(m.store.StreamingText) / 32
	return fmt.Sprintf("%d|%d|%d|%v|%d", len(m.store.Messages), len(m.store.StreamingToolUses), chunk, m.msgFoldAll, m.msgFoldRev)
}

func (m *model) msgViewportSyncGeometry() {
	if !m.msgViewportWanted() {
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
	if sig != m.lastVpGeom {
		if m.msgViewport.Width == 0 || m.msgViewport.Height == 0 {
			m.msgViewport = viewport.New(w, h)
		} else {
			m.msgViewport.Width = w
			m.msgViewport.Height = h
		}
		m.lastVpGeom = sig
		m.vpNeedResizeContent = true
	}
}

// tryBuildFullMessagePaneContent builds the full scrollable document for bubbles/viewport.
func (m *model) tryBuildFullMessagePaneContent() (string, bool) {
	maxL := gouDemoViewportMaxLines()
	keys := m.scrollItemKeys()
	n := len(keys)
	msgView := m.messagesForScroll()
	bodyCols := m.messageBodyColsForLayout()
	hl := m.transcriptSearchHighlightNeedle()

	var b strings.Builder
	lineCnt := 0

	addBlock := func(block string) bool {
		if strings.TrimSpace(block) == "" {
			return true
		}
		nl := strings.Count(block, "\n") + 1
		if lineCnt > 0 {
			if lineCnt+1 > maxL {
				return false
			}
			b.WriteByte('\n')
			lineCnt++
		}
		if lineCnt+nl > maxL {
			return false
		}
		b.WriteString(block)
		lineCnt += nl
		return true
	}

	for i := 0; i < n; i++ {
		if i < len(msgView) {
			msg := msgView[i]
			if m.msgFoldAll {
				u := msg.UUID
				if len(u) > 12 {
					u = u[:12] + "…"
				}
				line := fmt.Sprintf("%s  %s  [folded]", msg.Type, u)
				if !addBlock(line) {
					return "", false
				}
				continue
			}
			h := m.measureMessageRows(msg, bodyCols, hl)
			block := m.renderMessageRow(msg, bodyCols, h, hl)
			if !addBlock(block) {
				return "", false
			}
			continue
		}
		ti := i - len(msgView)
		st := m.transcriptStreamingToolsForView()
		if ti < 0 || ti >= len(st) {
			continue
		}
		h := m.measureTranscriptStreamingToolRow(st[ti], bodyCols, hl)
		block := m.renderTranscriptStreamingToolRow(st[ti], bodyCols, h, hl)
		if !addBlock(block) {
			return "", false
		}
	}

	if m.uiScreen != gouDemoScreenTranscript && len(m.store.StreamingToolUses) > 0 {
		for _, tu := range m.store.StreamingToolUses {
			var sb strings.Builder
			head := lipglossStyleAssistantHead()
			sb.WriteString(head)
			sb.WriteByte('\n')
			toolTitle := lipglossStyleStreamingToolTitle(tu.Name)
			sb.WriteString(toolTitle)
			if s := strings.TrimSpace(tu.UnparsedInput); s != "" {
				sb.WriteByte('\n')
				maxW := bodyCols * 4
				if maxW < 80 {
					maxW = 80
				}
				sb.WriteString(lipglossStyleFaintPreview(previewForTrace(s, maxW)))
			}
			if !addBlock(sb.String()) {
				return "", false
			}
		}
	}

	if m.uiScreen != gouDemoScreenTranscript && strings.TrimSpace(m.store.StreamingText) != "" {
		var sb strings.Builder
		sb.WriteString(lipglossStyleAssistantHead())
		sb.WriteByte('\n')
		sb.WriteString(styleMarkdownTokens(markdown.CachedLexerStreaming(m.store.StreamingText), bodyCols))
		if !addBlock(sb.String()) {
			return "", false
		}
	}

	return b.String(), true
}

func lipglossStyleAssistantHead() string {
	return lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
}

func lipglossStyleStreamingToolTitle(name string) string {
	return lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Bold(true).Render("⚙ "+name) + lipgloss.NewStyle().Faint(true).Render(" · streaming")
}

func lipglossStyleFaintPreview(s string) string {
	return lipgloss.NewStyle().Faint(true).Render(s)
}

func (m *model) applyMsgViewportContentFromView() {
	if !m.msgViewportWanted() {
		return
	}
	sig := m.messagePaneContentSig()
	if sig == m.lastVpContentSig && !m.vpNeedResizeContent {
		if m.sticky {
			m.msgViewport.GotoBottom()
		}
		return
	}
	s, ok := m.tryBuildFullMessagePaneContent()
	if !ok {
		m.msgViewportFallback = true
		m.lastVpContentSig = ""
		m.vpNeedResizeContent = false
		return
	}
	m.msgViewport.SetContent(s)
	m.lastVpContentSig = sig
	m.vpNeedResizeContent = false
	if m.sticky {
		m.msgViewport.GotoBottom()
	}
}

func (m *model) handleMsgViewportScrollKey(s string) {
	switch s {
	case "end", "G", "shift+g", "ctrl+end":
		m.sticky = true
		m.msgViewport.GotoBottom()
		return
	case "up", "k":
		m.msgViewport.ScrollUp(1)
	case "down", "j":
		m.msgViewport.ScrollDown(1)
	case "pgup", "b", "ctrl+u":
		m.msgViewport.HalfPageUp()
	case "pgdown", " ", "ctrl+d":
		m.msgViewport.HalfPageDown()
	case "ctrl+b", "ctrl+p":
		m.msgViewport.PageUp()
	case "ctrl+f", "ctrl+n":
		m.msgViewport.PageDown()
	case "home", "g", "ctrl+home":
		m.msgViewport.GotoTop()
	}
	if !m.msgViewport.AtBottom() {
		m.sticky = false
	}
}

// messagePaneViewportBlock renders the message list using bubbles/viewport (prompt + GOU_DEMO_BUBBLES_VIEWPORT only).
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
	m.cachePaneLinesForSelection(lines, bodyCols)
	if m.selDragging || m.msgSelectionActive() {
		lines = applyMsgSelectionVisualHighlight(lines, bodyCols, vpH, m.selAnchorR, m.selAnchorC, m.selFocusR, m.selFocusC)
	}
	totalH := m.msgViewport.TotalLineCount()
	if totalH < vpH {
		totalH = vpH
	}
	return joinMessagePaneLinesWithScrollbar(lines, bodyCols, vpH, totalH, m.msgViewport.YOffset, m.msgScrollbarW)
}

func (m *model) handleMsgViewportMouseWheel(delta int) {
	if delta == 0 {
		return
	}
	n := max(1, listViewportH(m)/6)
	if delta < 0 {
		m.msgViewport.ScrollDown(n)
	} else {
		m.msgViewport.ScrollUp(n)
	}
	if !m.msgViewport.AtBottom() {
		m.sticky = false
	}
}
