// Package iface defines interfaces for cross-component dependencies in the gou app,
// enabling gou/app/components to depend on abstractions rather than the full model.
package iface

import (
	"goc/gou/messagerow"
	"goc/types"
)

// MessageRenderDeps provides model-level state to the messages component renderer
// without exposing the full model struct.
type MessageRenderDeps interface {
	// MessagerowOpts returns rendering options for a given message.
	MessagerowOpts(msg types.Message) *messagerow.RenderOpts

	// ShowToolUseCtrlOExpandHint returns true when the "(ctrl+o to expand)" hint
	// should be appended to tool_use rows (prompt mode, not dump).
	ShowToolUseCtrlOExpandHint() bool

	// ResolvedToolIDs returns the set of tool_use IDs that already have results.
	ResolvedToolIDs() map[string]struct{}

	// ScreenIsTranscript reports whether the current screen mode is transcript.
	ScreenIsTranscript() bool

	// ScreenShowAll reports whether "show all" is active (ctrl+e).
	ScreenShowAll() bool

	// ScreenDumpMode reports whether dump mode is active.
	ScreenDumpMode() bool
}
