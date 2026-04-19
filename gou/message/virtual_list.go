package message

import (
	"goc/gou/virtualscroll"
	"goc/types"
)

// VirtualList implements virtual scrolling for messages.
// Similar to TS VirtualMessageList component.
type VirtualList struct {
	dispatcher *Dispatcher
	heightCache map[string]int // Cache of message heights by UUID
}

// NewVirtualList creates a new virtual list.
func NewVirtualList() *VirtualList {
	return &VirtualList{
		dispatcher: NewDispatcher(),
		heightCache: make(map[string]int),
	}
}

// RenderRange renders a range of messages for virtual scrolling.
func (vl *VirtualList) RenderRange(messages []*types.Message, startIdx, endIdx int, ctx *RenderContext) ([]string, error) {
	var result []string

	for i := startIdx; i < endIdx && i < len(messages); i++ {
		msg := messages[i]
		lines, err := vl.dispatcher.Render(msg, ctx)
		if err != nil {
			// Add error line
			result = append(result, "[Error rendering message]")
			continue
		}

		result = append(result, lines...)

		// Add separator between messages if needed
		if i < endIdx-1 && i < len(messages)-1 {
			// Add appropriate spacing based on message types
			if shouldAddSpacing(messages[i], messages[i+1]) {
				result = append(result, "") // Empty line between messages
			}
		}
	}

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
		ItemKeys: keys,
		HeightCache: heightCache,
		ScrollTop: scrollTop,
		ViewportH: viewportHeight,
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

// ProcessMessagesForDisplay processes messages for display (collapsing, grouping, etc.).
func (vl *VirtualList) ProcessMessagesForDisplay(messages []*types.Message, ctx *RenderContext) []*types.Message {
	// Message processing pipeline:
	// 1. Group consecutive tool uses by tool name
	// 2. Collapse consecutive read/search operations
	// 3. Apply other transformations

	processed := make([]*types.Message, len(messages))
	copy(processed, messages)

	// Apply grouping
	processed = GroupConsecutiveToolUses(processed)

	// Apply collapsing for read/search operations
	// Note: This is a simplified version - full implementation would
	// check timing and other factors
	processed = collapseReadSearchOperations(processed)

	return processed
}

// collapseReadSearchOperations collapses consecutive read/search operations.
func collapseReadSearchOperations(messages []*types.Message) []*types.Message {
	var result []*types.Message
	var currentBatch []*types.Message

	for i, msg := range messages {
		if isReadSearchMessage(msg) {
			currentBatch = append(currentBatch, msg)
			// If this is the last message or next message is not read/search, flush batch
			if i == len(messages)-1 || !isReadSearchMessage(messages[i+1]) {
				if len(currentBatch) > 1 {
					// Create collapsed group
					group := CreateCollapsedGroup(currentBatch, generateUUID())
					if group != nil {
						result = append(result, group)
					}
				} else if len(currentBatch) == 1 {
					result = append(result, currentBatch[0])
				}
				currentBatch = nil
			}
		} else {
			// Not a read/search operation
			if len(currentBatch) > 0 {
				// Flush any pending batch
				if len(currentBatch) > 1 {
					group := CreateCollapsedGroup(currentBatch, generateUUID())
					if group != nil {
						result = append(result, group)
					}
				} else if len(currentBatch) == 1 {
					result = append(result, currentBatch[0])
				}
				currentBatch = nil
			}
			result = append(result, msg)
		}
	}

	// Handle any remaining batch
	if len(currentBatch) > 0 {
		if len(currentBatch) > 1 {
			group := CreateCollapsedGroup(currentBatch, generateUUID())
			if group != nil {
				result = append(result, group)
			}
		} else if len(currentBatch) == 1 {
			result = append(result, currentBatch[0])
		}
	}

	return result
}

// BuildDisplayList builds the display list with proper spacing and separators.
func (vl *VirtualList) BuildDisplayList(messages []*types.Message, ctx *RenderContext) ([]*DisplayItem, error) {
	var items []*DisplayItem

	for i, msg := range messages {
		// Create display item
		item := &DisplayItem{
			Message: msg,
			Index:   i,
		}

		// Determine spacing
		item.SpacingBefore = vl.determineSpacingBefore(msg, i, messages, ctx)
		item.SpacingAfter = vl.determineSpacingAfter(msg, i, messages, ctx)

		items = append(items, item)
	}

	return items, nil
}

// DisplayItem represents a message in the display list.
type DisplayItem struct {
	Message        *types.Message
	Index          int
	SpacingBefore  int // Lines before this message
	SpacingAfter   int // Lines after this message
}

// determineSpacingBefore determines spacing before a message.
func (vl *VirtualList) determineSpacingBefore(msg *types.Message, idx int, messages []*types.Message, ctx *RenderContext) int {
	if idx == 0 {
		return 0 // No spacing before first message
	}

	prevMsg := messages[idx-1]

	// Add spacing between different message types
	if prevMsg.Type != msg.Type {
		return 1 // One empty line between different message types
	}

	// Add more sophisticated spacing rules
	// Based on message content, timing, etc.

	// Add spacing after long messages (more than 5 lines)
	prevHeight := vl.measureMessageHeight(prevMsg, ctx)
	if prevHeight > 5 {
		return 1
	}

	// Add spacing before system messages
	if msg.Type == types.MessageTypeSystem {
		return 1
	}

	// Add spacing before grouped tool uses
	if msg.Type == types.MessageTypeGroupedToolUse {
		return 1
	}

	return 0
}

// determineSpacingAfter determines spacing after a message.
func (vl *VirtualList) determineSpacingAfter(msg *types.Message, idx int, messages []*types.Message, ctx *RenderContext) int {
	if idx == len(messages)-1 {
		return 0 // No spacing after last message
	}

	nextMsg := messages[idx+1]

	// Add spacing between different message types
	if msg.Type != nextMsg.Type {
		return 1 // One empty line between different message types
	}

	return 0
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

// measureMessageHeight measures the height of a message.
func (vl *VirtualList) measureMessageHeight(msg *types.Message, ctx *RenderContext) int {
	// Check cache first
	if height, ok := vl.heightCache[msg.UUID]; ok {
		return height
	}

	// Measure using dispatcher
	height, err := vl.dispatcher.Measure(msg, ctx)
	if err != nil {
		height = 1 // Default height
	}

	// Cache the result
	if vl.heightCache == nil {
		vl.heightCache = make(map[string]int)
	}
	vl.heightCache[msg.UUID] = height

	return height
}