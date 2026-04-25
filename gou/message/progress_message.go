package message

import "goc/types"

// ProgressMessageRenderer implements TS behavior where type "progress" is not a top-level transcript row.
// Messages.tsx filters .filter(msg => msg.type !== 'progress') before reorderMessagesInUI / VirtualMessageList;
// progress is shown in tool UI via progressMessagesForMessage. If a progress row still reaches the
// dispatcher, render nothing (zero height) rather than "Unknown message type".
type ProgressMessageRenderer struct{}

func (r *ProgressMessageRenderer) CanRender(msg *types.Message) bool {
	return msg.Type == types.MessageTypeProgress
}

func (r *ProgressMessageRenderer) Render(_ *types.Message, _ *RenderContext) ([]string, error) {
	return nil, nil
}

func (r *ProgressMessageRenderer) Measure(_ *types.Message, _ *RenderContext) (int, error) {
	return 0, nil
}
