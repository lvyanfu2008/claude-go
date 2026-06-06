package app

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"goc/gou/app/state"
	"goc/gou/app/update"
	"goc/tools/toolexecution"
)

// Compile-time check: *model implements update.Deps.
var _ update.Deps = (*model)(nil)

// ── Model ────────────────────────────────────────────────────────────────────

func (m *model) Model() tea.Model { return m }

// ── State accessors ──────────────────────────────────────────────────────────

func (m *model) GetLayout() *state.Layout              { return m.Layout }
func (m *model) GetScreen() *state.Screen               { return m.Screen }
func (m *model) GetScroll() *state.Scroll               { return m.Scroll }
func (m *model) GetQuery() *state.Query                 { return m.Query }
func (m *model) GetModal() *state.Modal                 { return m.Modal }
func (m *model) GetInput() *state.Input                 { return m.Input }
func (m *model) GetViewport() *state.Viewport           { return m.Viewport }
func (m *model) GetAgent() *state.Agent                 { return m.Agent }
func (m *model) GetMessageTracking() *state.MessageTracking { return m.MessageTracking }
func (m *model) GetConversation() *state.Conversation          { return m.Conversation }
func (m *model) GetManualRender() *state.ManualRender          { return m.ManualRender }

// ── Modal / intercept ────────────────────────────────────────────────────────

func (m *model) ModalQuestionActive() bool {
	return m.Modal.Question != nil
}

func (m *model) HandleModalQuestion(msg tea.Msg) (tea.Model, tea.Cmd) {
	qm, _ := m.Modal.Question.(*questionModel).Update(msg)
	m.Modal.Question = qm.(*questionModel)
	if m.Modal.Question.(*questionModel).IsDone() {
		reply := permissionAskReply{}
		if m.Modal.Question.(*questionModel).IsCancelled() {
			reply.dec = toolexecution.DenyDecision("User declined to answer questions")
		} else {
			updatedInput := m.Modal.Question.(*questionModel).BuildUpdatedInput(m.Modal.Question.(*questionModel).originalInput)
			reply.dec = toolexecution.PermissionDecision{
				Behavior:     toolexecution.PermissionAllow,
				UpdatedInput: updatedInput,
			}
		}
		if m.Modal.Question.(*questionModel).replyCh != nil {
			select {
			case m.Modal.Question.(*questionModel).replyCh <- reply:
			default:
			}
		}
		m.Modal.Question = nil
	}
	return m, nil
}

// ── Manual render ────────────────────────────────────────────────────────────

func (m *model) ManualRenderActive() bool {
	return m.ManualRender.Active
}

func (m *model) BufferManualRenderEvent(msg tea.Msg) {
	m.ManualRender.Events = append(m.ManualRender.Events, msg)
}

func (m *model) ShouldBufferInManualRender(msg tea.Msg) bool {
	// Already checked in Dispatcher for external types; no additional check needed.
	return false
}

// ── Action methods ───────────────────────────────────────────────────────────

func (m *model) RebuildHeightCache()                 { m.rebuildHeightCache() }
func (m *model) ClearTranscriptSearchState()         { m.clearTranscriptSearchState() }
func (m *model) EndQuerySpinner()                    { m.endQuerySpinner() }
func (m *model) MaybeRecordTranscript()              { m.maybeRecordTranscript() }
func (m *model) AnyToolSummaryDelayPending() bool    { return m.anyToolSummaryDelayPending() }
func (m *model) HandleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	return m.handleKeyMsg(msg)
}

// ── App-internal message dispatch ────────────────────────────────────────────

func (m *model) TryHandleAppMessage(msg tea.Msg) (bool, tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case gouPermissionAskMsg:
		permModel, permCmd := m.handlePermissionAsk(msg)
		return true, permModel, permCmd

	case gouTranscriptEditorPrepMsg:
		return true, m, m.handleTranscriptEditorChainMsg(msg)
	case gouTranscriptEditorExecDoneMsg:
		return true, m, m.handleTranscriptEditorChainMsg(msg)
	case gouTranscriptEditorClearStatusMsg:
		return true, m, m.handleTranscriptEditorChainMsg(msg)

	case tea.MouseClickMsg, tea.MouseMotionMsg, tea.MouseWheelMsg, tea.MouseReleaseMsg:
		if m.Viewport.HistoryBrowseMouseOff && m.msgViewportWanted() {
			return true, m, nil
		}
		if handled, cmd := m.tryHandleMessageListMouse(msg); handled {
			return true, m, cmd
		}
		return true, m, nil

	case AgentRegisteredMsg:
		m.Agent.Tasks.(*agentTaskStore).Register(msg.Task)
		return true, m, func() tea.Msg { return AgentTaskTickMsg{} }
	case AgentProgressMsg:
		m.Agent.Tasks.(*agentTaskStore).UpdateProgress(msg.AgentID, msg.Progress)
		return true, m, nil
	case AgentCompletedMsg:
		m.Agent.Tasks.(*agentTaskStore).Complete(msg.AgentID, msg.Status)
		return true, m, nil
	case AgentTaskTickMsg:
		m.Agent.Tasks.(*agentTaskStore).EvictExpired(time.Now())
		if m.Agent.Tasks.(*agentTaskStore).Count() > 0 {
			return true, m, taskListTickCmdAgent()
		}
		return true, m, nil

	case taskListTickMsg:
		m.Agent.TaskList.(*taskListModel).poll()
		return true, m, taskListTickCmd(m.Agent.TaskList.(*taskListModel))
	}

	return false, nil, nil
}

// handlePermissionAsk handles the gouPermissionAskMsg inline from the original Update().
func (m *model) handlePermissionAsk(msg gouPermissionAskMsg) (tea.Model, tea.Cmd) {
	if len(msg.questions) > 0 {
		// AskUserQuestion: switch to interactive question UI.
		m.Modal.Question = newQuestionModel(msg.questions, msg.replyCh, m.Layout.Width, m.Layout.Height)
		m.Modal.Question.(*questionModel).originalInput = msg.input
		return m, nil
	}
	m.Modal.Permission = &permissionAskOverlay{
		toolName:  msg.toolName,
		toolUseID: msg.toolUseID,
		input:     msg.input,
		prompt:    msg.prompt,
		replyCh:   msg.replyCh,
	}
	return m, nil
}
