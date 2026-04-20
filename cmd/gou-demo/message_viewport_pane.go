package main

import (
	"encoding/json"
	"fmt"
	"goc/ccb-engine/diaglog"
	"goc/gou/conversation"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"goc/gou/markdown"
	"goc/gou/messagerow"
	"goc/gou/theme"
	"goc/types"

	"github.com/samber/lo"
)

// gouDemoBubblesViewport defaults on (bubbles/viewport for the prompt message pane, same scrolling style as go-tui).
// Use legacy virtual list only with GOU_DEMO_LEGACY_VIRTUAL_MESSAGE_SCROLL=1, or disable viewport with
// GOU_DEMO_BUBBLES_VIEWPORT=0|false|off|no.
func gouDemoBubblesViewport() bool {
	if gouDemoEnvTruthy("GOU_DEMO_LEGACY_VIRTUAL_MESSAGE_SCROLL") {
		return false
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_BUBBLES_VIEWPORT")))
	if v == "0" || v == "false" || v == "off" || v == "no" {
		return false
	}
	return true
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
	// New renderer also uses viewport
	useNewRenderer := os.Getenv("GOU_DEMO_USE_NEW_RENDERER") == "1"
	if useNewRenderer {
		// 新渲染器在所有屏幕都使用 viewport
		result := !m.msgViewportFallback
		diaglog.Line("[viewport] msgViewportWanted: useNewRenderer=true, msgViewportFallback=%v, returning %v", m.msgViewportFallback, result)
		return result
	}
	result := m.useMsgViewport && m.uiScreen == gouDemoScreenPrompt && !m.msgViewportFallback
	diaglog.Line("[viewport] msgViewportWanted: useNewRenderer=false, useMsgViewport=%v, uiScreen=%v, msgViewportFallback=%v, returning %v", m.useMsgViewport, m.uiScreen, m.msgViewportFallback, result)
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
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
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
	diaglog.Line("[viewport] msgViewportSyncGeometry: w=%d, h=%d, sig=%s, lastVpGeom=%s", w, h, sig, m.lastVpGeom)
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

	//检测是否最新工具已经在视图中了,不在则使用上次的结果
	//if !m.isViewMsgComplete() {
	//	return m.lastB, true
	//}

	for i := 0; i < n; i++ {
		if i < len(msgView) {
			msg := msgView[i]
			if m.isStreamToolUsing(msg.Message) {
				msg = m.filterStreamingToolsFromMessage(msg)
				if len(msg.Content) == 0 || string(msg.Content) == "[]" {
					continue
				}
			}
			if m.msgFoldAll {
				u := msg.UUID
				if len(u) > 12 {
					u = u[:12] + "…"
				}
				var line string
				switch msg.Type {
				case types.MessageTypeUser:
					line = fmt.Sprintf(">  %s  [folded]", u)
				case types.MessageTypeAssistant:
					line = fmt.Sprintf("%s  [folded]", u)
				default:
					line = fmt.Sprintf("%s  %s  [folded]", msg.Type, u)
				}
				if i > 0 && (userAssistantPairBlankLine(msgView[i-1], msg) || transcriptAssistantPairBlankLine(m, msgView[i-1], msg)) {
					if lineCnt > 0 {
						if lineCnt+1 > maxL {
							return "", false
						}
						b.WriteByte('\n')
						lineCnt++
					}
				}
				if !addBlock(line) {
					return "", false
				}
				continue
			}
			if i > 0 && (userAssistantPairBlankLine(msgView[i-1], msg) || transcriptAssistantPairBlankLine(m, msgView[i-1], msg)) {
				if lineCnt > 0 {
					if lineCnt+1 > maxL {
						return "", false
					}
					b.WriteByte('\n')
					lineCnt++
				}
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
		if ti == 0 && len(msgView) > 0 && msgView[len(msgView)-1].Type == types.MessageTypeUser {
			if lineCnt > 0 {
				if lineCnt+1 > maxL {
					return "", false
				}
				b.WriteByte('\n')
				lineCnt++
			}
		}
		if !addBlock(block) {
			return "", false
		}
	}

	//需要兼容一下，这里有个问题，已经有工具了但是msgView却没有同步过来
	if m.uiScreen != gouDemoScreenTranscript &&
		len(m.store.StreamingToolUses) > 0 &&
		(m.isStreamToolUsing(msgView[len(msgView)-1].Message) || m.isStreamToolUsing(msgView[len(msgView)-2].Message)) {
		// Same breathing room as user↔assistant rows and StreamingText: last scroll message is user
		// but no assistant row yet — only a single \n from addBlock would sit the tool chrome too close.
		// || streamGapAfterUserMessage(msgView)
		if lineCnt > 0 && (m.isStreamToolUsing(msgView[len(msgView)-1].Message)) {
			if lineCnt+1 > maxL {
				return "", false
			}
			b.WriteByte('\n')
			lineCnt++
		}

		now := time.Now()
		if m.streamToolFirstSeen == nil {
			m.streamToolFirstSeen = make(map[string]time.Time)
		}
		for _, tu := range m.store.StreamingToolUses {
			if _, ok := m.streamToolFirstSeen[tu.ToolUseID]; !ok {
				m.streamToolFirstSeen[tu.ToolUseID] = now
			}
		}

		grouped := groupStreamingTools(m.store.StreamingToolUses)
		for _, group := range grouped {
			var firstSeen time.Time
			if len(group.Items) > 0 {
				firstSeen = m.streamToolFirstSeen[group.Items[0].ToolUseID]
			} else if group.Single.ToolUseID != "" {
				firstSeen = m.streamToolFirstSeen[group.Single.ToolUseID]
			}

			elapsed := time.Since(firstSeen)

			titleDelayMs := 50
			detailDelayMs := 100
			if v := os.Getenv("GOU_DEMO_STREAM_TOOL_TITLE_DELAY_MS"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n >= 0 {
					titleDelayMs = n
				}
			}
			if v := os.Getenv("GOU_DEMO_STREAM_TOOL_DETAIL_DELAY_MS"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n >= 0 {
					detailDelayMs = n
				}
			}

			if elapsed < time.Duration(titleDelayMs)*time.Millisecond {
				continue
			}

			var sb strings.Builder
			if len(msgView) == 0 || msgView[len(msgView)-1].Type != types.MessageTypeAssistant {
				head := lipglossStyleAssistantHead()
				sb.WriteString(head)
				sb.WriteByte('\n')
			}

			if !group.IsGroup {
				tu := group.Single
				// 所有工具都显示活动状态
				activityLine := messagerow.ActivityLineForToolUse(tu.Name, json.RawMessage(tu.UnparsedInput))
				if activityLine == "" {
					// 如果没有活动描述，使用工具名
					facing, paren, _ := messagerow.ToolChromeParts(tu.Name, json.RawMessage(tu.UnparsedInput))
					if facing == "" {
						facing = tu.Name
					}
					activityLine = facing
					if p := strings.TrimSpace(paren); p != "" {
						activityLine += " " + p
					}
				}
				// 添加省略号表示正在执行
				activityLine += "…"
				// 添加交互提示
				toolTitle := toolRowLeadPrefix(false) + lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render(activityLine) + lipgloss.NewStyle().Faint(true).Render(messagerow.CtrlOToExpandHint)
				sb.WriteString(toolTitle)
			} else {
				summary := messagerow.SearchReadSummaryText(true, group.SearchCount, group.ReadCount, group.ListCount, 0, 0, 0, 0, 0, nil, nil, nil)
				toolTitle := toolRowLeadPrefix(false) + lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render(summary) + lipgloss.NewStyle().Faint(true).Render(messagerow.CtrlOToExpandHint)
				diaglog.Line("tryBuildFullMessagePaneContent|toolSummary %s, toolTitle %s", summary, toolTitle)
				sb.WriteString(toolTitle)
				if elapsed >= time.Duration(detailDelayMs)*time.Millisecond || true {
					for _, item := range group.Items {
						path := extractPartialJSONField(item.UnparsedInput, "file_path")
						if path == "" {
							path = extractPartialJSONField(item.UnparsedInput, "path")
						}
						if path == "" {
							path = extractPartialJSONField(item.UnparsedInput, "pattern")
						}
						if path == "" {
							path = "..."
						}
						sb.WriteByte('\n')
						sb.WriteString(lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render("  ⎿  " + path))
					}
				}
			}

			if !addBlock(applyMessagePaneGutter(sb.String(), bodyCols)) {
				return "", false
			}
		}
	}

	if m.uiScreen != gouDemoScreenTranscript && strings.TrimSpace(m.store.StreamingText) != "" {
		if lineCnt > 0 && streamGapAfterUserMessage(msgView) {
			if lineCnt+1 > maxL {
				return "", false
			}
			b.WriteByte('\n')
			lineCnt++
		}
		var sb strings.Builder
		md := styleMarkdownTokens(markdown.CachedLexerStreaming(m.store.StreamingText), bodyCols, false)
		sb.WriteString(applyMessagePaneGutter(lipglossStyleAssistantHead()+"\n"+md, bodyCols))
		if !addBlock(sb.String()) {
			return "", false
		}
	}

	//m.lastB = b.String()

	content := b.String()

	// 确保内容足够多，可以滚动
	// 如果内容行数少于视口高度，添加更多行
	lines := strings.Count(content, "\n") + 1
	vpHeight := listViewportH(m)
	if lines <= vpHeight {
		diaglog.Line("[viewport] tryBuildFullMessagePaneContent: content has %d lines, viewport height is %d, adding more lines", lines, vpHeight)
		var sb strings.Builder
		sb.WriteString(content)
		for i := lines; i <= vpHeight + 10; i++ {
			sb.WriteString(fmt.Sprintf("Additional line %d to enable scrolling\n", i+1))
		}
		content = sb.String()
		diaglog.Line("[viewport] tryBuildFullMessagePaneContent: added lines, now has %d lines", strings.Count(content, "\n")+1)
	}

	return content, true
}

func (m *model) isViewMsgComplete() bool {
	if len(m.store.StreamingToolUses) == 0 {
		return true
	}

	for _, msg := range m.store.Messages {
		if m.isStreamToolUsing(msg.Message) {
			return true
		}
	}

	return false
}

func (m *model) isAnyToolUsing() bool {

	for _, msg := range m.store.Messages {
		if m.isStreamToolUsing(msg.Message) {
			return true
		}
	}

	return false
}

// msg.Message
func (m *model) isStreamToolUsing(c json.RawMessage) bool {
	var contentBlocks []types.MessageContentBlock
	var inner struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	isToolUsing := false
	if err := json.Unmarshal(c, &inner); err != nil {
		return false
	}

	if err := json.Unmarshal(inner.Content, &contentBlocks); err != nil {
		return false
	}

	toolUseIds := lo.Map(m.store.StreamingToolUses, func(m conversation.StreamingToolUse, _ int) string {
		return m.ToolUseID
	})

	for _, c := range contentBlocks {
		if lo.Contains(toolUseIds, c.ID) {
			isToolUsing = true
			break
		}
	}
	return isToolUsing
}

func (m *model) filterStreamingToolsFromMessage(msg types.Message) types.Message {
	if !m.isStreamToolUsing(msg.Message) {
		return msg
	}
	var contentBlocks []types.MessageContentBlock
	var inner struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(msg.Message, &inner); err != nil {
		return msg
	}
	if err := json.Unmarshal(inner.Content, &contentBlocks); err != nil {
		return msg
	}

	toolUseIds := lo.Map(m.store.StreamingToolUses, func(tu conversation.StreamingToolUse, _ int) string {
		return tu.ToolUseID
	})

	var filtered []types.MessageContentBlock
	for _, c := range contentBlocks {
		if c.Type == "tool_use" && lo.Contains(toolUseIds, c.ID) {
			continue
		}
		filtered = append(filtered, c)
	}

	newContent, _ := json.Marshal(filtered)
	inner.Content = newContent
	newMessage, _ := json.Marshal(inner)
	msg.Message = newMessage
	msg.Content = newContent
	return msg
}

func lipglossStyleAssistantHead() string {
	return ""
	//return lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
}

func lipglossStyleStreamingToolTitle(name string) string {
	// 改为显示活动状态
	activityLine := messagerow.ActivityLineForToolUse(name, json.RawMessage("{}"))
	if activityLine == "" {
		activityLine = name
	}
	activityLine += "…"
	return lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render(activityLine) + lipgloss.NewStyle().Faint(true).Render(messagerow.CtrlOToExpandHint)
}

func lipglossStyleFaintPreview(s string) string {
	return lipgloss.NewStyle().Faint(true).Render(s)
}

func (m *model) applyMsgViewportContentFromView() {
	if !m.msgViewportWanted() {
		diaglog.Line("[viewport] applyMsgViewportContentFromView: msgViewportWanted=false, returning")
		return
	}
	sig := m.messagePaneContentSig()
	if sig == m.lastVpContentSig && !m.vpNeedResizeContent {
		if m.sticky {
			m.msgViewport.GotoBottom()
		}
		diaglog.Line("[viewport] applyMsgViewportContentFromView: content unchanged, sig=%s", sig)
		return
	}

	useNewRenderer := os.Getenv("GOU_DEMO_USE_NEW_RENDERER") == "1"
	var s string
	var ok bool

	diaglog.Line("[viewport] applyMsgViewportContentFromView: building content, useNewRenderer=%v, sig=%s", useNewRenderer, sig)
	if useNewRenderer {
		s, ok = m.tryBuildFullMessagePaneContentWithNewRenderer()
	} else {
		s, ok = m.tryBuildFullMessagePaneContent()
	}

	if !ok {
		diaglog.Line("[viewport] applyMsgViewportContentFromView: build failed, setting fallback")
		m.msgViewportFallback = true
		m.lastVpContentSig = ""
		m.vpNeedResizeContent = false
		return
	}
	diaglog.Line("[viewport] applyMsgViewportContentFromView: setting content, length=%d, lines≈%d", len(s), strings.Count(s, "\n")+1)
	m.msgViewport.SetContent(s)
	diaglog.Line("[viewport] applyMsgViewportContentFromView: after SetContent, totalLines=%d, height=%d, AtTop=%v, AtBottom=%v",
		m.msgViewport.TotalLineCount(), m.msgViewport.Height(), m.msgViewport.AtTop(), m.msgViewport.AtBottom())
	m.lastVpContentSig = sig
	m.vpNeedResizeContent = false
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

// messagePaneViewportBlock renders the message list using bubbles/viewport (prompt screen when viewport mode is on).
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
