// Package iface defines interfaces shared between gou/app sub-packages.
// Sub-packages depend on these interfaces, never on the root app package.
package iface

import (
	"goc/gou/conversation"
	"goc/types"
)

// StoreReader provides read access to conversation state.
type StoreReader interface {
	Messages() []types.Message
	ConversationID() string
	StreamingToolUses() []conversation.StreamingToolUse
	HasStreaming() bool
	StreamingText() string
	StreamingThinkingText() string
}

// ScrollState controls the message list scroll position.
type ScrollState interface {
	ScrollTop() int
	SetScrollTop(v int)
	Sticky() bool
	SetSticky(v bool)
}

// LayoutInfo provides terminal dimensions and layout parameters.
type LayoutInfo interface {
	Width() int
	Height() int
	Cols() int
	BodyCols() int
	ScrollbarW() int
}

// ScreenState reports the current screen mode for rendering decisions.
type ScreenState interface {
	IsTranscript() bool
	ShowAll() bool
	DumpMode() bool
	SearchOpen() bool
}
