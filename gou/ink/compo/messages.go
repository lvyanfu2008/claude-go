package compo

import "goc/gou/ink"

// Messages renders all messages with streaming integration as a single Box.
func Messages(ctx *ink.Context, p ink.Props) ink.VNode {
	return ink.VNode{
		Type:     "Box",
		Props:    ink.Props{"direction": "column"},
		Children: MessageRows(ctx),
	}
}

// MessageRows returns individual VNode rows for each message (including
// streaming placeholders). Each row is a direct child for VirtualList so
// virtual scrolling can track per-row heights.
func MessageRows(ctx *ink.Context) []ink.VNode {
	store := ctx.Store
	rawMsgs := store.GetMessages()

	// Inject streaming as in-progress messages
	if store.StreamingText() != "" {
		rawMsgs = append(rawMsgs, ink.Message{
			UUID: "__streaming_text__", Type: "assistant",
			ContentBlocks: []ink.ContentBlock{{Type: "text", Content: store.StreamingText()}},
		})
	}
	for _, st := range store.StreamingTools() {
		rawMsgs = append(rawMsgs, ink.Message{
			UUID: "__streaming_tool_" + st.UUID + "__", Type: "assistant",
			ContentBlocks: []ink.ContentBlock{{Type: "tool_use", Name: st.Name, Input: st.Input, State: "running"}},
		})
	}

	msgs := ProcessMessages(rawMsgs)

	children := make([]ink.VNode, len(msgs))
	for i, msg := range msgs {
		children[i] = MessageRow(ctx, msg)
	}
	return children
}

// MessageRow dispatches to the appropriate message renderer based on type.
func MessageRow(ctx *ink.Context, msg ink.Message) ink.VNode {
	var content ink.VNode
	switch msg.Type {
	case "user":
		content = UserMessage(ctx, msg)
	case "assistant":
		content = AssistantMessage(ctx, msg)
	case "system":
		content = SystemMessage(ctx, msg)
	case "collapsed_read_search":
		content = CollapsedReadSearch(ctx, msg)
	case "grouped_tool_use":
		content = GroupedToolUse(ctx, msg)
	default:
		content = ink.VNode{Type: "Text"}
	}
	return ink.VNode{
		Type: "Box", Key: msg.UUID,
		Props:    ink.Props{"direction": "column"},
		Children: []ink.VNode{content},
	}
}
