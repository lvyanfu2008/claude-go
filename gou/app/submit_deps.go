package app

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/gou/app/submit"
	"goc/gou/conversation"
	"goc/gou/pui"
	"goc/services/sessionmemory"
	"goc/sessiontranscript"
	"goc/tools/localtools"
	"goc/tools/toolexecution"
	"goc/tools/toolresultpersist"
	"goc/tscontext"
	"goc/types"
)

// submitDeps adapts *model to submit.Deps.
type submitDeps struct {
	m *model
}

func (d submitDeps) Model() tea.Model                               { return d.m }
func (d submitDeps) ConversationStore() *conversation.Store         { return d.m.Conversation.Store }
func (d submitDeps) ConversationTSBridge() *tscontext.Snapshot      { return d.m.Conversation.TSBridge }
func (d submitDeps) ConversationReadFileState() *localtools.ReadFileState {
	return d.m.Conversation.ReadFileState
}
func (d submitDeps) GroupedAgentLookups() interface{} { return d.m.Conversation.GroupedAgentLookups }
func (d submitDeps) ResolvedToolIDs() map[string]struct{} { return d.m.Conversation.ResolvedToolIDs }
func (d submitDeps) MCPCommandsJSONPath() string              { return d.m.Tool.MCPCommandsJSONPath }
func (d submitDeps) MCPToolsJSONPath() string                 { return d.m.Tool.MCPToolsJSONPath }
func (d submitDeps) ToolResultState() *toolresultpersist.ContentReplacementState {
	return d.m.Tool.ResultState
}
func (d submitDeps) SessionMem() *sessionmemory.State { return d.m.Memory.SessionMem }
func (d submitDeps) CCBSend() func(interface{})       { return d.m.Query.CCBSend }
func (d submitDeps) CCBInline() bool                  { return d.m.Query.CCBInline }
func (d submitDeps) ModalAskAutoFirst() bool          { return d.m.Modal.AskAutoFirst }

func (d submitDeps) SkillListingSent() map[string]struct{}         { return d.m.Input.SkillListingSent }
func (d submitDeps) SetSkillListingSent(v map[string]struct{})     { d.m.Input.SkillListingSent = v }
func (d submitDeps) PRSetValue(v string)                           { d.m.Input.PR.SetValue(v) }
func (d submitDeps) ScrollSetSticky(v bool)                        { d.m.Scroll.Sticky = v }
func (d submitDeps) ScrollSetTop(v int)                            { d.m.Scroll.Top = v }
func (d submitDeps) QuerySetCancel(v context.CancelFunc)           { d.m.Query.Cancel = v }
func (d submitDeps) QuerySetBusy(v bool)                           { d.m.Query.Busy = v }
func (d submitDeps) QuerySetBusyStartedAt(v time.Time)             { d.m.Query.BusyStartedAt = v }
func (d submitDeps) QuerySetSpinnerVerb(v string)                  { d.m.Query.SpinnerVerb = v }
func (d submitDeps) QuerySetSpinnerFrame(v int)                    { d.m.Query.SpinnerFrame = v }

func (d submitDeps) RebuildHeightCache()  { d.m.rebuildHeightCache() }
func (d submitDeps) MaybeRecordTranscript() { d.m.maybeRecordTranscript() }
func (d submitDeps) SyncSlashListAfterPrompt() { d.m.syncSlashListAfterPrompt() }
func (d submitDeps) ApplySlashResultPanelFromSubmit(line string, r *processuserinput.ProcessUserInputBaseResult, out pui.ApplyProcessUserInputBaseResultOutcome) {
	d.m.applySlashResultPanelFromSubmit(line, r, out)
}
func (d submitDeps) InstallAskResolver(te *toolexecution.ExecutionDeps, askAutoFirst bool) {
	d.m.installAskResolver(te, askAutoFirst)
}

func (d submitDeps) MessageBodyColsForLayout() int  { return d.m.messageBodyColsForLayout() }
func (d submitDeps) MessageScrollContentHeight() int { return d.m.messageScrollContentHeight() }
func (d submitDeps) IntegrateMessageRenderer()       { d.m.integrateMessageRenderer() }
func (d submitDeps) FillMessageHeightCache(cols int, hl string) {
	d.m.fillMessageHeightCache(cols, hl)
}
func (d submitDeps) BeginQuerySpinner() { d.m.beginQuerySpinner() }
func (d submitDeps) EndQuerySpinner()   { d.m.endQuerySpinner() }
func (d submitDeps) LoadSlashCommandsOnce() { d.m.loadSlashCommandsOnce() }

func (d submitDeps) LastGuidance() string               { return d.m.Memory.LastGuidance }
func (d submitDeps) SetLastGuidance(v string)           { d.m.Memory.LastGuidance = v }
func (d submitDeps) LastUserCtx() map[string]string     { return d.m.Memory.LastUserCtx }
func (d submitDeps) SetLastUserCtx(v map[string]string) { d.m.Memory.LastUserCtx = v }
func (d submitDeps) LastSystemCtx() map[string]string   { return d.m.Memory.LastSystemCtx }
func (d submitDeps) SetLastSystemCtx(v map[string]string) { d.m.Memory.LastSystemCtx = v }

func (d submitDeps) PermissionMode() types.PermissionMode { return d.m.Chrome.PermissionMode }
func (d submitDeps) ScreenMode() int                      { return int(d.m.Screen.Mode) }
func (d submitDeps) LayoutCols() int                      { return d.m.Layout.Cols }
func (d submitDeps) LayoutHeight() int                    { return d.m.Layout.Height }
func (d submitDeps) ConversationTranscript() *sessiontranscript.Store { return d.m.Conversation.Transcript }

// Compile-time check
var _ submit.Deps = submitDeps{}
