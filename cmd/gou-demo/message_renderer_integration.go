package main

import (
	"encoding/json"
	"strings"

	"goc/ccb-engine/diaglog"
	"goc/gou/markdown"
	"goc/gou/message"
	"goc/gou/messagerow"
	"goc/gou/theme"
	"goc/types"

	"charm.land/lipgloss/v2"
)

// MessageRendererIntegration integrates the new message rendering system into gou-demo.
type MessageRendererIntegration struct {
	dispatcher   *message.Dispatcher
	processor    *message.Processor
	virtualList  *message.VirtualList
	currentTheme *theme.Palette
	highlighter  *markdown.Highlighter
}

// NewMessageRendererIntegration creates a new integration instance.
func NewMessageRendererIntegration(highlighter *markdown.Highlighter) *MessageRendererIntegration {
	return &MessageRendererIntegration{
		dispatcher:   message.NewDispatcher(),
		processor:    message.NewProcessor(),
		virtualList:  message.NewVirtualList(),
		currentTheme: theme.ActivePalette(),
		highlighter:  highlighter,
	}
}

// UpdateTheme updates the current theme.
func (mri *MessageRendererIntegration) UpdateTheme(themeName string) {
	theme.InitFromThemeName(themeName)
	mri.currentTheme = theme.ActivePalette()
}

// ProcessMessages processes messages for display.
// Note: This method is kept for compatibility but messages are now processed
// by the VirtualList during rendering.
func (mri *MessageRendererIntegration) ProcessMessages(messages []*types.Message, isTranscript bool) []*types.Message {
	// Return messages as-is - processing is now done by VirtualList
	return messages
}

// RenderMessage renders a single message.
func (mri *MessageRendererIntegration) RenderMessage(msg *types.Message, width int, isTranscript, verbose, shouldAnimate, shouldShowDot bool) string {
	ctx := &message.RenderContext{
		Width:         width,
		Theme:         mri.currentTheme,
		IsTranscript:  isTranscript,
		IsStatic:      isTranscript, // Transcript mode is static
		ShouldAnimate: shouldAnimate,
		ShouldShowDot: shouldShowDot,
		AddMargin:     true,
		Verbose:       verbose,
		Highlighter:   mri.highlighter,
	}

	// Process the single message (though processing typically works on message sequences)
	// For single message rendering, we just render it as-is
	lines, err := mri.dispatcher.Render(msg, ctx)
	if err != nil {
		return "[Error rendering message]"
	}

	return strings.Join(lines, "\n")
}

// ComputeVisibleRange computes the visible range for virtual scrolling.
func (mri *MessageRendererIntegration) ComputeVisibleRange(messages []*types.Message, scrollTop, viewportHeight int, isTranscript, verbose bool, width int) (startIdx, endIdx int, totalHeight int) {
	ctx := &message.RenderContext{
		Width:         width, // Use actual width for measurement
		Theme:         mri.currentTheme,
		IsTranscript:  isTranscript,
		IsStatic:      isTranscript,
		ShouldAnimate: false, // Measurement doesn't need animation
		ShouldShowDot: false,
		AddMargin:     true,
		Verbose:       verbose,
		Highlighter:   mri.highlighter,
	}

	return mri.virtualList.ComputeVisibleRange(messages, scrollTop, viewportHeight, ctx)
}

// ComputeTotalHeight computes the total height of all messages.
func (mri *MessageRendererIntegration) ComputeTotalHeight(messages []*types.Message, isTranscript, verbose bool, width int) int {
	if len(messages) == 0 {
		return 0
	}
	_, _, totalHeight := mri.ComputeVisibleRange(messages, 0, 1, isTranscript, verbose, width)
	return totalHeight
}

// RenderVisibleRange renders the visible range of messages.
func (mri *MessageRendererIntegration) RenderVisibleRange(messages []*types.Message, startIdx, endIdx int, width int, isTranscript, verbose, shouldAnimate, shouldShowDot bool) string {
	diaglog.Line("[new-renderer] RenderVisibleRange: messages=%d, range=[%d,%d), width=%d, isTranscript=%v, verbose=%v, shouldAnimate=%v",
		len(messages), startIdx, endIdx, width, isTranscript, verbose, shouldAnimate)

	ctx := &message.RenderContext{
		Width:         width,
		Theme:         mri.currentTheme,
		IsTranscript:  isTranscript,
		IsStatic:      isTranscript,
		ShouldAnimate: shouldAnimate,
		ShouldShowDot: shouldShowDot,
		AddMargin:     true,
		Verbose:       verbose,
		Highlighter:   mri.highlighter,
	}

	lines, err := mri.virtualList.RenderRange(messages, startIdx, endIdx, ctx)
	if err != nil {
		diaglog.Line("[new-renderer] RenderRange error: %v", err)
		return "[Error rendering message range]"
	}

	diaglog.Line("[new-renderer] RenderVisibleRange result: %d lines", len(lines))
	return strings.Join(lines, "\n")
}

// Palette returns the palette used for Render / Measure (kept in sync with [theme.InitFromThemeName] via [UpdateTheme]).
func (mri *MessageRendererIntegration) Palette() *theme.Palette {
	return mri.currentTheme
}

// MeasureMessage returns the line count for one message using the same [message.Dispatcher]
// stack as [RenderVisibleRange] / [ComputeVisibleRange] (not the legacy messagerow + [layout.WrappedRowCount] path).
func (mri *MessageRendererIntegration) MeasureMessage(msg *types.Message, ctx *message.RenderContext) (int, error) {
	return mri.dispatcher.Measure(msg, ctx)
}

// InvalidateCache invalidates the cache for a message.
func (mri *MessageRendererIntegration) InvalidateCache(msgUUID string) {
	mri.virtualList.InvalidateCache(msgUUID)
}

// InvalidateAllCache invalidates all caches.
func (mri *MessageRendererIntegration) InvalidateAllCache() {
	mri.virtualList.InvalidateAllCache()
}

// Integration with existing model

// integrateMessageRenderer integrates the new renderer into the existing model.
func (m *model) integrateMessageRenderer() {
	if m.msgRenderer == nil {
		m.msgRenderer = NewMessageRendererIntegration(markdownHighlighter)
		// Set default theme
		m.msgRenderer.UpdateTheme("default")
	}
}

// renderMessagesWithNewRenderer renders messages using the new renderer.
func (m *model) renderMessagesWithNewRenderer() string {
	m.integrateMessageRenderer()

	// Get messages from store
	messages := m.store.Messages

	// Convert []types.Message to []*types.Message
	var messagesPtr []*types.Message
	for i := range messages {
		messagesPtr = append(messagesPtr, &messages[i])
	}

	// Determine rendering parameters
	width := m.messageBodyColsForLayout()
	isTranscript := m.uiScreen == gouDemoScreenTranscript
	verbose := m.transcriptShowAll || (m.uiScreen == gouDemoScreenTranscript && m.transcriptSearchOpen)
	shouldAnimate := m.uiScreen == gouDemoScreenPrompt && m.store.StreamingText != ""
	shouldShowDot := m.uiScreen == gouDemoScreenPrompt && len(m.store.StreamingText) > 0

	// Use RenderVisibleRange to render all messages with proper processing
	// This ensures messages are processed (grouped, collapsed) before rendering
	content := m.msgRenderer.RenderVisibleRange(
		messagesPtr,
		0,                // startIdx
		len(messagesPtr), // endIdx
		width,
		isTranscript,
		verbose,
		shouldAnimate,
		shouldShowDot,
	)

	// Note: This function doesn't handle streaming elements
	// For consistency with TS side, streaming should be integrated into message flow
	// But this function is named "renderMessages" not "renderMessagePane"
	// So it might be intentional to only render messages
	return content
}

// Update model struct to include renderer
func (m *model) initMessageRenderer() {
	m.integrateMessageRenderer()
}

// Hook into existing update and view methods

// In update methods, invalidate cache when messages change
func (m *model) invalidateMessageCache() {
	if m.msgRenderer != nil {
		m.msgRenderer.InvalidateAllCache()
	}
}

// In view method, use new renderer
func (m *model) renderMessagePaneWithNewRenderer() string {
	m.integrateMessageRenderer()

	// Get messages from store
	messages := m.store.Messages
	diaglog.Line("[new-renderer] renderMessagePaneWithNewRenderer: messages count=%d, streamingTools=%d, streamingText='%s'",
		len(messages), len(m.store.StreamingToolUses), m.store.StreamingText)

	// Convert []types.Message to []*types.Message
	var messagesPtr []*types.Message
	for i := range messages {
		messagesPtr = append(messagesPtr, &messages[i])
	}

	// Determine rendering parameters
	width := m.messageBodyColsForLayout()
	isTranscript := m.uiScreen == gouDemoScreenTranscript
	verbose := m.transcriptShowAll || (m.uiScreen == gouDemoScreenTranscript && m.transcriptSearchOpen)
	shouldAnimate := m.uiScreen == gouDemoScreenPrompt && m.store.StreamingText != ""
	shouldShowDot := m.uiScreen == gouDemoScreenPrompt && len(m.store.StreamingText) > 0

	// Get viewport height
	vpH := listViewportH(m)
	scrollTop := m.scrollTop

	// Compute visible range using virtual list
	startIdx, endIdx, _ := m.msgRenderer.ComputeVisibleRange(
		messagesPtr,
		scrollTop,
		vpH,
		isTranscript,
		verbose,
		width,
	)

	// Render only visible range
	content := m.msgRenderer.RenderVisibleRange(
		messagesPtr,
		startIdx,
		endIdx,
		width,
		isTranscript,
		verbose,
		shouldAnimate,
		shouldShowDot,
	)

	// Add streaming tools and streaming text if needed
	// In TS side, streaming elements appear as part of the ongoing assistant response
	if m.uiScreen != gouDemoScreenTranscript {
		hasStreamingElements := false

		// Add streaming tools
		streamingToolUses := m.store.StreamingToolUses

		if len(streamingToolUses) > 0 {
			hasStreamingElements = true
			grouped := groupStreamingTools(streamingToolUses)
			for _, group := range grouped {
				if content != "" {
					content += "\n"
				}
				var sb strings.Builder

				if !group.IsGroup {
					tu := group.Single
					// 对于单个搜索/读取工具，也显示活动状态
					name := strings.TrimSpace(tu.Name)
					if name == "Grep" || name == "Glob" || name == "Read" || name == "View" || name == "LS" || name == "SemanticSearch" {
						// 当作单个项目的分组处理
						var searchCount, readCount, listCount int
						switch name {
						case "Grep", "Glob", "SemanticSearch":
							searchCount = 1
						case "Read", "View":
							readCount = 1
						case "LS":
							listCount = 1
						}
						summary := messagerow.SearchReadSummaryText(true, searchCount, readCount, listCount, 0, 0, 0, 0, 0, nil, nil, nil) + "…"
						toolTitle := toolRowLeadPrefix(false) + lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render(summary) + lipgloss.NewStyle().Faint(true).Render(messagerow.CtrlOToExpandHint)
						sb.WriteString(toolTitle)
						// 添加路径提示
						path := extractPartialJSONField(tu.UnparsedInput, "file_path")
						if path == "" {
							path = extractPartialJSONField(tu.UnparsedInput, "path")
						}
						if path == "" {
							path = extractPartialJSONField(tu.UnparsedInput, "pattern")
						}
						if path == "" {
							path = extractPartialJSONField(tu.UnparsedInput, "glob")
						}
						if path == "" {
							path = "..."
						}
						sb.WriteByte('\n')
						sb.WriteString(lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render("  ⎿  " + path))
					} else {
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
					}
				} else {
					summary := messagerow.SearchReadSummaryText(true, group.SearchCount, group.ReadCount, group.ListCount, 0, 0, 0, 0, 0, nil, nil, nil) + "…"
					toolTitle := toolRowLeadPrefix(false) + lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render(summary) + lipgloss.NewStyle().Faint(true).Render(messagerow.CtrlOToExpandHint)
					sb.WriteString(toolTitle)
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
				content += applyMessagePaneGutter(sb.String(), m.messageBodyColsForLayout())
			}
		}

		// Add streaming text
		if strings.TrimSpace(m.store.StreamingText) != "" {
			hasStreamingElements = true
			if content != "" {
				content += "\n"
			}
			// Simpler approach: just add the streaming text as assistant content
			// TS side integrates this as part of the ongoing assistant message
			md := styleMarkdownTokens(markdown.CachedLexerStreaming(m.store.StreamingText), m.messageBodyColsForLayout(), false)
			content += applyMessagePaneGutter(md, m.messageBodyColsForLayout())
		}

		// If we added streaming elements but there's no assistant message yet,
		// we might need to add an assistant header
		if hasStreamingElements && content != "" {
			// Check if we need to add assistant header
			// Simplified logic: if last message is user or no messages, add header
			// Actually, looking at TS side behavior: streaming elements
			// appear as continuation of assistant response, not with separate header
			// So we might NOT need to add header here
			// For now, keep it simple - no automatic header
		}
	}

	return content
}

// tryBuildFullMessagePaneContentWithNewRenderer builds the full scrollable document for bubbles/viewport using the new renderer.
func (m *model) tryBuildFullMessagePaneContentWithNewRenderer() (string, bool) {
	m.integrateMessageRenderer()

	// Get messages from store
	messages := m.store.Messages
	diaglog.Line("[new-renderer] tryBuildFullMessagePaneContentWithNewRenderer: messages count=%d, streamingTools=%d, streamingText='%s', uiScreen=%v, msgViewportWanted=%v",
		len(messages), len(m.store.StreamingToolUses), m.store.StreamingText, m.uiScreen, m.msgViewportWanted())

	// Convert []types.Message to []*types.Message
	var messagesPtr []*types.Message
	for i := range messages {
		messagesPtr = append(messagesPtr, &messages[i])
	}

	// Determine rendering parameters
	width := m.messageBodyColsForLayout()
	isTranscript := m.uiScreen == gouDemoScreenTranscript
	verbose := m.transcriptShowAll || (m.uiScreen == gouDemoScreenTranscript && m.transcriptSearchOpen)
	shouldAnimate := m.uiScreen == gouDemoScreenPrompt && m.store.StreamingText != ""
	shouldShowDot := m.uiScreen == gouDemoScreenPrompt && len(m.store.StreamingText) > 0

	// Render all messages using the new renderer
	content := m.msgRenderer.RenderVisibleRange(
		messagesPtr,
		0,                // startIdx
		len(messagesPtr), // endIdx
		width,
		isTranscript,
		verbose,
		shouldAnimate,
		shouldShowDot,
	)
	diaglog.Line("[new-renderer] tryBuildFullMessagePaneContentWithNewRenderer: RenderVisibleRange returned, content length=%d", len(content))

	// Add streaming tools and streaming text if needed (similar to renderMessagePaneWithNewRenderer)
	if m.uiScreen != gouDemoScreenTranscript {
		hasStreamingElements := false

		// Add streaming tools
		streamingToolUses := m.store.StreamingToolUses

		if len(streamingToolUses) > 0 {
			hasStreamingElements = true
			grouped := groupStreamingTools(streamingToolUses)
			for _, group := range grouped {
				if content != "" {
					content += "\n"
				}
				var sb strings.Builder

				if !group.IsGroup {
					tu := group.Single
					// 对于单个搜索/读取工具，也显示活动状态
					name := strings.TrimSpace(tu.Name)
					if name == "Grep" || name == "Glob" || name == "Read" || name == "View" || name == "LS" || name == "SemanticSearch" {
						// 当作单个项目的分组处理
						var searchCount, readCount, listCount int
						switch name {
						case "Grep", "Glob", "SemanticSearch":
							searchCount = 1
						case "Read", "View":
							readCount = 1
						case "LS":
							listCount = 1
						}
						summary := messagerow.SearchReadSummaryText(true, searchCount, readCount, listCount, 0, 0, 0, 0, 0, nil, nil, nil) + "…"
						toolTitle := toolRowLeadPrefix(false) + lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render(summary) + lipgloss.NewStyle().Faint(true).Render(messagerow.CtrlOToExpandHint)
						sb.WriteString(toolTitle)
						// 添加路径提示
						path := extractPartialJSONField(tu.UnparsedInput, "file_path")
						if path == "" {
							path = extractPartialJSONField(tu.UnparsedInput, "path")
						}
						if path == "" {
							path = extractPartialJSONField(tu.UnparsedInput, "pattern")
						}
						if path == "" {
							path = extractPartialJSONField(tu.UnparsedInput, "glob")
						}
						if path == "" {
							path = "..."
						}
						sb.WriteByte('\n')
						sb.WriteString(lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render("  ⎿  " + path))
					} else {
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
					}
				} else {
					summary := messagerow.SearchReadSummaryText(true, group.SearchCount, group.ReadCount, group.ListCount, 0, 0, 0, 0, 0, nil, nil, nil) + "…"
					toolTitle := toolRowLeadPrefix(false) + lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render(summary) + lipgloss.NewStyle().Faint(true).Render(messagerow.CtrlOToExpandHint)
					sb.WriteString(toolTitle)
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
				content += applyMessagePaneGutter(sb.String(), m.messageBodyColsForLayout())
			}
		}

		// Add streaming text
		if strings.TrimSpace(m.store.StreamingText) != "" {
			hasStreamingElements = true
			if content != "" {
				content += "\n"
			}
			// Simpler approach: just add the streaming text as assistant content
			// TS side integrates this as part of the ongoing assistant message
			md := styleMarkdownTokens(markdown.CachedLexerStreaming(m.store.StreamingText), m.messageBodyColsForLayout(), false)
			content += applyMessagePaneGutter(md, m.messageBodyColsForLayout())
		}

		// If we added streaming elements but there's no assistant message yet,
		// we might need to add an assistant header
		if hasStreamingElements && content != "" {
			// Check if we need to add assistant header
			// Simplified logic: if last message is user or no messages, add header
			// Actually, looking at TS side behavior: streaming elements
			// appear as continuation of assistant response, not with separate header
			// So we might NOT need to add header here
			// For now, keep it simple - no automatic header
		}
	}

	diaglog.Line("[new-renderer] tryBuildFullMessagePaneContentWithNewRenderer returning: content length=%d, lines≈%d", len(content), strings.Count(content, "\n")+1)

	return content, true
}
