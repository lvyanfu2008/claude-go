package message

import (
	"goc/ccb-engine/diaglog"
	"goc/gou/virtualscroll"
	"goc/types"
)

// VirtualList implements virtual scrolling for messages.
// Similar to TS VirtualMessageList component.
type VirtualList struct {
	dispatcher  *Dispatcher
	heightCache map[string]int // Cache of message heights by UUID
}

// NewVirtualList creates a new virtual list.
func NewVirtualList() *VirtualList {
	return &VirtualList{
		dispatcher:  NewDispatcher(),
		heightCache: make(map[string]int),
	}
}

// RenderRange renders a range of messages for virtual scrolling.
func (vl *VirtualList) RenderRange(messages []*types.Message, startIdx, endIdx int, ctx *RenderContext) ([]string, error) {
	var result []string

	diaglog.Line("[virtual-list] RenderRange: messages=%d, range=[%d,%d), isTranscript=%v, verbose=%v",
		len(messages), startIdx, endIdx, ctx.IsTranscript, ctx.Verbose)

	for i := startIdx; i < endIdx && i < len(messages); i++ {
		msg := messages[i]
		diaglog.Line("[virtual-list] Rendering message %d: type=%s, uuid=%s", i, msg.Type, msg.UUID)

		lines, err := vl.dispatcher.Render(msg, ctx)
		if err != nil {
			diaglog.Line("[virtual-list] Error rendering message %d: %v", i, err)
			// Add error line
			result = append(result, "[Error rendering message]")
			continue
		}

		diaglog.Line("[virtual-list] Message %d rendered %d lines", i, len(lines))
		result = append(result, lines...)

		// Add separator between messages if needed
		if i < endIdx-1 && i < len(messages)-1 {
			// Add appropriate spacing based on message types
			if shouldAddSpacing(messages[i], messages[i+1]) {
				result = append(result, "") // Empty line between messages
			}
		}
	}

	diaglog.Line("[virtual-list] RenderRange complete: total %d lines", len(result))
	return result, nil
}

// ComputeVisibleRange computes the visible range for virtual scrolling.
func (vl *VirtualList) ComputeVisibleRange(messages []*types.Message, scrollTop, viewportHeight int, ctx *RenderContext) (startIdx, endIdx int, totalHeight int) {
	if len(messages) == 0 {
		return 0, 0, 0
	}

	// Build item keys and heights
	keys := make([]string, len(messages))
	heights := make([]int, len(messages))

	for i, msg := range messages {
		keys[i] = msg.UUID
		height, ok := vl.heightCache[msg.UUID]
		if !ok {
			// Measure message height
			height, _ = vl.dispatcher.Measure(msg, ctx)
			vl.heightCache[msg.UUID] = height
		}
		heights[i] = height
	}

	// Populate height cache
	heightCache := make(map[string]int)
	for i, key := range keys {
		heightCache[key] = heights[i]
	}

	// Use virtualscroll to compute range
	input := virtualscroll.RangeInput{
		ItemKeys:                keys,
		HeightCache:             heightCache,
		ScrollTop:               scrollTop,
		ViewportH:               viewportHeight,
		MaxMountedItemsOverride: 50, // Reasonable default
	}

	output := virtualscroll.ComputeRange(input)

	return output.Range.Start, output.Range.End, output.TotalHeight
}

// InvalidateCache invalidates the height cache for a message.
func (vl *VirtualList) InvalidateCache(msgUUID string) {
	delete(vl.heightCache, msgUUID)
}

// InvalidateAllCache invalidates the entire height cache.
func (vl *VirtualList) InvalidateAllCache() {
	vl.heightCache = make(map[string]int)
}

// GetMessageHeight gets the cached height of a message.
func (vl *VirtualList) GetMessageHeight(msg *types.Message, ctx *RenderContext) int {
	height, ok := vl.heightCache[msg.UUID]
	if !ok {
		height, _ = vl.dispatcher.Measure(msg, ctx)
		vl.heightCache[msg.UUID] = height
	}
	return height
}

// shouldAddSpacing checks if spacing should be added between two messages.
func shouldAddSpacing(msg1, msg2 *types.Message) bool {
	// Add spacing between different message types
	if msg1.Type != msg2.Type {
		return true
	}

	// Special cases: add spacing after system messages
	if msg1.Type == types.MessageTypeSystem {
		return true
	}

	// Add spacing after grouped tool uses
	if msg1.Type == types.MessageTypeGroupedToolUse {
		return true
	}

	// Add spacing after collapsed groups
	if msg1.Type == types.MessageTypeCollapsedReadSearch {
		return true
	}

	return false
}
