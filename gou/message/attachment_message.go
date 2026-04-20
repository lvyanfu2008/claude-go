package message

import (
	"goc/types"
)

// AttachmentMessageRenderer renders attachment messages (silently, without showing anything).
type AttachmentMessageRenderer struct{}

// CanRender returns true for attachment messages.
func (r *AttachmentMessageRenderer) CanRender(msg *types.Message) bool {
	return msg.Type == types.MessageTypeAttachment
}

// Render renders an attachment message - returns empty slice to show nothing.
func (r *AttachmentMessageRenderer) Render(msg *types.Message, ctx *RenderContext) ([]string, error) {
	// Return empty slice to show nothing in terminal
	return []string{}, nil
}

// Measure returns 0 lines for attachment messages (they don't take up space).
func (r *AttachmentMessageRenderer) Measure(msg *types.Message, ctx *RenderContext) (int, error) {
	return 0, nil
}