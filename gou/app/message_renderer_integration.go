package app

import (
	"strings"

	"goc/ccb-engine/diaglog"
	"goc/gou/markdown"
	"goc/gou/message"
	"goc/gou/theme"
	"goc/types"
)

// MessageRendererIntegration integrates the new message rendering system into gou-demo.
type MessageRendererIntegration struct {
	dispatcher   *message.Dispatcher
	virtualList  *message.VirtualList
	currentTheme *theme.Palette
	highlighter  *markdown.Highlighter
}

// NewMessageRendererIntegration creates a new integration instance.
func NewMessageRendererIntegration(highlighter *markdown.Highlighter) *MessageRendererIntegration {
	return &MessageRendererIntegration{
		dispatcher:   message.NewDispatcher(),
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
	//	len(messages), startIdx, endIdx, width, isTranscript, verbose, shouldAnimate)

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
func (m *Model) integrateMessageRenderer() {
	if m.msgRenderer == nil {
		m.msgRenderer = NewMessageRendererIntegration(markdownHighlighter)
		// Set default theme
		m.msgRenderer.UpdateTheme("default")
	}
}

// renderMessagesWithNewRenderer renders messages using the new renderer.
func (m *Model) renderMessagesWithNewRenderer() string {
	m.integrateMessageRenderer()

	// UI-ordered messages (same as Messages.tsx / messagesForScroll: drops progress, etc.)
	messagesPtr := m.messagePtrSliceForNewRenderer()

	// Determine rendering parameters
	width := m.messageBodyColsForLayout()
	isTranscript := m.uiScreen == gouDemoScreenTranscript
	verbose := m.transcriptShowAll || (m.uiScreen == gouDemoScreenTranscript && m.transcriptSearchOpen)
	shouldAnimate := m.uiScreen == gouDemoScreenPrompt && m.store.HasStreaming()
	shouldShowDot := m.uiScreen == gouDemoScreenPrompt && m.store.HasStreaming()

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
func (m *Model) initMessageRenderer() {
	m.integrateMessageRenderer()
}

// Hook into existing update and view methods

// In update methods, invalidate cache when messages change
func (m *Model) invalidateMessageCache() {
	if m.msgRenderer != nil {
		m.msgRenderer.InvalidateAllCache()
	}
}

// In view method, use new renderer
func (m *Model) renderMessagePaneWithNewRenderer() string {
	m.integrateMessageRenderer()

	messagesPtr := m.messagePtrSliceForNewRenderer()
	//	len(messagesPtr), len(m.store.StreamingToolUses), m.store.StreamingText)

	// Determine rendering parameters
	width := m.messageBodyColsForLayout()
	isTranscript := m.uiScreen == gouDemoScreenTranscript
	verbose := m.transcriptShowAll || (m.uiScreen == gouDemoScreenTranscript && m.transcriptSearchOpen)
	shouldAnimate := m.uiScreen == gouDemoScreenPrompt && m.store.HasStreaming()
	shouldShowDot := m.uiScreen == gouDemoScreenPrompt && m.store.HasStreaming()

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

	// Add streaming tail using unified renderer
	if m.uiScreen != gouDemoScreenTranscript {
		streamingCtx := &message.RenderContext{
			Width:                width,
			Theme:                m.msgRenderer.currentTheme,
			IsTranscript:         false,
			Verbose:              verbose,
			ShouldAnimate:        shouldAnimate,
			ShouldShowDot:        shouldShowDot,
			Highlighter:          m.msgRenderer.highlighter,
			ShowToolUseCtrlOHint: true,
		}
		streamingToolUses := messageStreamingToolUses(m)
		tailLines := message.RenderStreamingTail(
			m.store.StreamingText,
			m.store.StreamingThinkingText,
			streamingToolUses,
			streamingCtx,
		)
		if len(tailLines) > 0 {
			if content != "" {
				content += "\n"
			}
			content += strings.Join(tailLines, "\n")
		}
	}

	return content
}

// tryBuildFullMessagePaneContentWithNewRenderer builds the full scrollable document for bubbles/viewport using the new renderer.
func (m *Model) tryBuildFullMessagePaneContentWithNewRenderer() (string, bool) {
	m.integrateMessageRenderer()

	messagesPtr := m.messagePtrSliceForNewRenderer()
	//	len(messagesPtr), len(m.store.StreamingToolUses), m.store.StreamingText, m.uiScreen, m.msgViewportWanted())

	// Determine rendering parameters
	width := m.messageBodyColsForLayout()
	isTranscript := m.uiScreen == gouDemoScreenTranscript
	verbose := m.transcriptShowAll || (m.uiScreen == gouDemoScreenTranscript && m.transcriptSearchOpen)
	shouldAnimate := m.uiScreen == gouDemoScreenPrompt && m.store.HasStreaming()
	shouldShowDot := m.uiScreen == gouDemoScreenPrompt && m.store.HasStreaming()

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

	// Add streaming tail using unified renderer
	if m.uiScreen != gouDemoScreenTranscript {
		streamingCtx := &message.RenderContext{
			Width:                width,
			Theme:                m.msgRenderer.currentTheme,
			IsTranscript:         false,
			Verbose:              verbose,
			ShouldAnimate:        shouldAnimate,
			ShouldShowDot:        shouldShowDot,
			Highlighter:          m.msgRenderer.highlighter,
			ShowToolUseCtrlOHint: true,
		}
		streamingToolUses := messageStreamingToolUses(m)
		tailLines := message.RenderStreamingTail(
			m.store.StreamingText,
			m.store.StreamingThinkingText,
			streamingToolUses,
			streamingCtx,
		)
		if len(tailLines) > 0 {
			if content != "" {
				content += "\n"
			}
			content += strings.Join(tailLines, "\n")
		}
	}

	return content, true
}

// messageStreamingToolUses converts store streaming tool uses to the message package format.
func messageStreamingToolUses(m *Model) []message.StreamingToolUse {
	var result []message.StreamingToolUse
	for _, tu := range m.store.StreamingToolUses {
		result = append(result, message.StreamingToolUse{
			Name:  tu.Name,
			Input: tu.UnparsedInput,
		})
	}
	return result
}
