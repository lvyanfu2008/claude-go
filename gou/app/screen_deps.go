package app

import (
	"goc/gou/app/screen"
	state "goc/gou/app/state"
	"goc/types"
)

// screenDeps adapts *model to screen.Deps.
type screenDeps struct {
	m *model
}

func (d screenDeps) ModalQuestionViewContent() string {
	if d.m.Modal.Question == nil {
		return ""
	}
	return d.m.Modal.Question.(*questionModel).View().Content
}

func (d screenDeps) ModalPermission() interface{} { return d.m.Modal.Permission }

func (d screenDeps) Width() int              { return d.m.Layout.Width }
func (d screenDeps) Cols() int               { return d.m.Layout.Cols }
func (d screenDeps) Height() int             { return d.m.Layout.Height }
func (d screenDeps) MsgBodyCols() int        { return d.m.Layout.MsgBodyCols }
func (d screenDeps) MsgScrollbarW() int      { return d.m.Layout.MsgScrollbarW }
func (d screenDeps) ScreenMode() state.ScreenMode { return d.m.Screen.Mode }
func (d screenDeps) ShowAll() bool           { return d.m.Screen.ShowAll }
func (d screenDeps) DumpMode() bool          { return d.m.Screen.DumpMode }
func (d screenDeps) TranscriptSearchOpen() bool { return d.m.Screen.SearchOpen }

func (d screenDeps) LastEmittedTitlePlain() string    { return d.m.Chrome.LastEmittedTitlePlain }
func (d screenDeps) SetLastEmittedTitlePlain(v string) { d.m.Chrome.LastEmittedTitlePlain = v }
func (d screenDeps) ConversationID() string            { return d.m.Conversation.Store.ConversationID }
func (d screenDeps) IsBusy() bool                      { return d.m.Query.Busy }
func (d screenDeps) HasStreaming() bool                { return d.m.Conversation.Store.HasStreaming() }
func (d screenDeps) PermissionMode() types.PermissionMode { return d.m.Chrome.PermissionMode }

func (d screenDeps) ListViewportH() int           { return listViewportH(d.m) }
func (d screenDeps) ViewportWanted() bool         { return d.m.msgViewportWanted() }
func (d screenDeps) SyncViewportGeometry()        { d.m.msgViewportSyncGeometry() }
func (d screenDeps) ApplyViewportContent()        { d.m.applyMsgViewportContentFromView() }
func (d screenDeps) ViewportFallback() bool       { return d.m.Viewport.Fallback }
func (d screenDeps) ViewportBlock(vpH, bodyCols int) string {
	return d.m.messagePaneViewportBlock(vpH, bodyCols)
}

func (d screenDeps) RenderMessagePane() string { return d.m.renderMessagePaneWithNewRenderer() }
func (d screenDeps) IntegrateRenderer()        { d.m.integrateMessageRenderer() }

func (d screenDeps) MessagePtrCount() int {
	return len(d.m.messagePtrSliceForNewRenderer())
}
func (d screenDeps) MessagePtrSlice() []*types.Message { return d.m.messagePtrSliceForNewRenderer() }

func (d screenDeps) ComputeVisibleRange(msgs []*types.Message, scrollTop, vpHeight int, isTranscript, verbose bool, width int) (int, int, int) {
	return d.m.msgRenderer.ComputeVisibleRange(msgs, scrollTop, vpHeight, isTranscript, verbose, width)
}

func (d screenDeps) ScrollTop() int { return d.m.Scroll.Top }

func (d screenDeps) PromptStreamRows() []string { return d.m.promptBottomStreamRows() }

func (d screenDeps) TaskListVisible() bool {
	if d.m.Agent.TaskList == nil {
		return false
	}
	return d.m.Agent.TaskList.(*taskListModel).isVisible()
}

func (d screenDeps) TaskListView(maxDisplay, cols int) string {
	if d.m.Agent.TaskList == nil {
		return ""
	}
	return d.m.Agent.TaskList.(*taskListModel).view(maxDisplay, cols)
}

func (d screenDeps) TaskListViewMaxDisplay() int    { return d.m.taskListViewMaxDisplay() }
func (d screenDeps) TaskListViewReservedRows() int   { return d.m.taskListViewReservedRows() }
func (d screenDeps) AgentCoordinatorView() string   { return d.m.agentCoordinatorView() }
func (d screenDeps) StatusLine() string             { return d.m.statusLineString() }

func (d screenDeps) TranscriptFootLines(narrow bool) string {
	return joinFooterLines(transcriptChromeFootLines(d.m, narrow), d.m.Layout.Cols)
}

func (d screenDeps) TranscriptEditorStatus() string {
	return d.m.Screen.EditorStatus
}

func (d screenDeps) AtSuggestionView() string  { return d.m.renderAtSuggestions() }
func (d screenDeps) BuiltinStatusView() string  { return d.m.builtinStatusLineView() }
func (d screenDeps) UserInputView() string      { return userInputViewWithPromptPrefix(d.m) }
func (d screenDeps) SlashPanelBlock() string    { return d.m.slashResultPanelViewBlock() }
func (d screenDeps) SlashListVisible() bool     { return d.m.slashListVisible() }
func (d screenDeps) SlashPicker(width, height int) string {
	return d.m.renderSlashPicker(width, height)
}

func (d screenDeps) PermissionModalView(width int) string {
	return d.m.renderPermissionModal(width)
}

func (d screenDeps) SuspendAltScreen() bool     { return d.m.Screen.SuspendAltScreenForScrollbackDump }
func (d screenDeps) HistoryBrowseMouseOff() bool { return d.m.Viewport.HistoryBrowseMouseOff }

// Compile-time check
var _ screen.Deps = screenDeps{}
