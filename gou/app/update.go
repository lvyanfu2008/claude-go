// Command gou-demo is a minimal Bubble Tea full-screen UI: new message renderer + markdown + tool blocks (Phase 4 messagerow).
// Extracted [model.Update] branches: update_streaming.go (query yield / NDJSON), update_layout.go (window resize).
// Layout uses the full terminal size (main buffer by default). Optional GOU_DEMO_ALT_SCREEN=1 uses the alternate buffer (no shell scrollback mixing).
// With GOU_DEMO_LOG=1, trace uses the same path rules as TS debug log (see goc/ccb-engine/debugpath); on TTY without GOU_DEMO_LOG_FILE, trace goes to that file, not stderr.
//
// Run from repo: cd goc && go run ./cmd/gou-demo
//
// Flags: -transcript=file.json (UI or API messages), -replay-cc=events.ndjson, -stream-stdin (pipe NDJSON),
// Real model: [goc/conversation-runtime/query.Query] HTTP streaming parity when ANTHROPIC_API_KEY (or ANTHROPIC_AUTH_TOKEN) is set
// and GOU_QUERY_STREAMING_PARITY=1 or GOU_DEMO_STREAMING_TOOL_EXECUTION=1 (see [query.BuildQueryConfig]).
// When a tool gate returns ask, GOU_QUERY_ASK_STRATEGY=allow auto-allows for headless demo (maps to [toolexecution.ExecutionDeps.AskResolver]).
// GOU_TOOLEXEC_BASH_SANDBOX_1B=1 enables permissions.ts whole-tool ask bypass on Bash when the tool input carries a non-empty command without dangerously_disable_sandbox (see toolexecution.WholeToolAskSkippedForBash1b).
// Go-side init port (subset of TS init.ts): gou-demo runs [goc/claudeinit.Init] (includes [settingsfile.EnsureProjectClaudeEnvOnce]). See docs/plans/go-init-port.md.
// Go local tool parity (streaming parity + [skilltools.ParityToolRunner]): Bash is allowed by default (same as TS); set GOU_DEMO_NO_LOCAL_BASH=1 to disable unless CCB_ENGINE_LOCAL_BASH=1. PowerShell is off unless CCB_ENGINE_LOCAL_POWERSHELL=1 (uses pwsh or powershell.exe). AskUserQuestion auto-picks the first option per question unless GOU_DEMO_NO_ASK_AUTO_FIRST=1. WebFetch is allowed by default; set CCB_ENGINE_DISABLE_WEB_FETCH=1 to block network fetches in the Go runner. See docs/plans/go-tools-parity.md.
//
// System # Language / # Output Style: merged from ~/.harness/settings.go.json and project .harness/settings.go.json / settings.local.json (see settingsfile; project settings.go.json is TS-only). CLAUDE_CODE_LANGUAGE and CLAUDE_CODE_OUTPUT_STYLE_* override when set (non-empty); built-in outputStyle keys Explanatory/Learning use prompts from src/constants/outputStyles.ts (embedded).
// Extra HARNESS.md roots: optional runtimeContext.toolPermissionContext.additionalWorkingDirectories (JSON) and/or GOU_DEMO_EXTRA_CLAUDE_MD_ROOTS / CLAUDE_CODE_EXTRA_CLAUDE_MD_ROOTS (comma or PATH-style list). Paths from runtime/env are always scanned when passed (see [querycontext.ExtraClaudeMdRootsForFetch]); CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1 is only needed for env-only flows in claudemd that do not pass explicit roots.
// Debug log (optional): GOU_DEMO_LOG_FILE=/path/to.log, or GOU_DEMO_LOG=1 (default file path matches TS getDebugLogPath via goc/ccb-engine/debugpath when stderr is TTY). GOU_DEMO_LOG_STDERR=1 forces stderr (may corrupt TUI). Lines are prefixed [gou-demo].
// ToolUseContext dump: CLAUDE_CODE_LOG_TOOL_USE_CONTEXT or GOU_DEMO_LOG_TOOL_USE_CONTEXT = 1|summary|full (with logging enabled) prints JSON after each BuildDemoParams; full includes the entire commands[] snapshot.
// Read/Grep/Glob stream tail: default keeps each tool_use + tool_result as separate rows (avoids looking like history was cleared). Set GOU_DEMO_COLLAPSE_READ_SEARCH_TAIL=1 for TS-style merge into collapsed_read_search (gou/ccbstream/apply.go).
// Prompt: merged one-line Grep/Glob/Read summaries (GOU_DEMO_TOOL_USE_SUMMARY_LINE) wait GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS after each assistant message first appears (default 2000 ms) while full Search/Read rows are shown; set to 0 to collapse immediately.
//
// Keys: ↑/↓/PgUp/PgDn scroll the message pane, End bottom. Prompt: default Enter send; Alt+Enter or Option+Enter (macOS) newline when the terminal sends Meta; Ctrl+J / LF newline. GOU_DEMO_REPL_ENTER_SUBMITS=0 for chat mode (Enter newline, Alt+Enter send). Shift+↑↓ move line. F2 toggles the slash list; leading "/" (TS) or mid-input " … /tok" shows the list; ↑/↓ move selection; Tab inserts; Enter applies selection and runs submit; input stays in the main field. Ctrl+l forces a full-screen clear + redraw (TS Global app:redraw). Ctrl+o toggles TS-style transcript (frozen tail; / search with n/N when not in dump; search bar Esc clears; ctrl+e show-all expands collapsed/grouped + full tool_result bodies except in dump). In the main prompt, user messages that contain only tool_result / advisor_tool_result blocks are omitted from the list (no "user / ↩ tool_result …" stub row); mixed user rows still fold tool_result bodies to one line + (ctrl+o to expand). Transcript (compact): same omission + tool_result folded on user rows; assistant rows show ⏺+⎿ summaries. Ctrl+e show-all or [ dump shows full blocks. [ (no search bar) enables dump: show-all + plain transcript to scrollback (Printf). v opens frozen transcript in $VISUAL/$EDITOR via temp file (tea.ExecProcess). Transcript pager (search bar closed, not dump): arrows/pgup/pgdn/end, j/k, g, G/shift+g, ctrl+u/d, ctrl+b/f, b, space (full page), ctrl+n/p (line). Esc/q/ctrl+c exit transcript when search bar closed. In prompt mode, q or Esc quit. Columns < 80 use a shorter header/footer (TS REPL isNarrow). Terminal tab title: OSC 0 unless CLAUDE_CODE_DISABLE_TERMINAL_TITLE=1; loading shows a "…" prefix. CLAUDE_CODE_PERMISSION_MODE sets tool permission mode for submits (TS toolPermissionContext.mode).
// Theme: CLAUDE_CODE_THEME=light (after merged settings env) selects a higher-contrast palette; see [theme.InitFromThemeName]. GOU_DEMO_STATUS_LINE=1 shows theme/msg counts above the prompt.
// Message pane: new renderer ([message.VirtualList] in [gou/message]) drives both prompt and transcript screens. Prompt uses [bubbles/viewport] by default (full-document scroll + ctrl+y fold-all); disable with GOU_DEMO_BUBBLES_VIEWPORT=0|false|off|no to render the visible slice directly on top of m.scrollTop.
// Mouse: SGR mouse (cell motion) enables wheel + plain left-drag on the message list when not disabled by env. Set GOU_DEMO_DISABLE_MOUSE_SCROLL=1 to ignore wheel/drag in-app. Mirror TS fullscreen.ts: CLAUDE_CODE_DISABLE_MOUSE=1 or GOU_DEMO_DISABLE_MOUSE=1 omits SGR mouse (keyboard scroll still works), unless GOU_DEMO_DISALLOW_DISABLE_MOUSE=1. One-column TUI scrollbar is on by default when the pane is wide enough; GOU_DEMO_MESSAGE_SCROLLBAR=0|false|off|no or GOU_DEMO_NO_SCROLLBAR=1 turns it off. Alternate screen: opt-in GOU_DEMO_ALT_SCREEN=1 (default main buffer). Bubbles viewport: at-top wheel-up can release mouse for host scrollback; opt out with GOU_DEMO_MSG_HISTORY_MOUSE_RELEASE=0|false|off|no.
// Slash: /name is resolved in-process — disk skills via [goc/slashresolve.ResolveDiskSkill], bundled prompts via [goc/slashresolve.ResolveBundledSkill] (embedded markdown under slashresolve/skills/bundled). Other prompt commands need a disk skill (SkillRoot) or a bundled definition. Unknown names that look like command names and are not root filesystem paths (non-Windows) return TS-style Unknown skill without calling the model; otherwise the line is treated as a normal user prompt.
// MCP skills (scheme-2 R0/R1): -mcp-commands-json=path or GOU_DEMO_MCP_COMMANDS_JSON → JSON array of types.Command merged into Skill/commands (enable FEATURE_MCP_SKILLS=1 for listing).
// MCP tool defs (assembleToolPool): -mcp-tools-json=path or GOU_DEMO_MCP_TOOLS_JSON → JSON array merged into Options.Tools when GOU_DEMO_USE_EMBEDDED_TOOLS_API=1 (see mcpcommands.EnvToolsJSONPath).
//
// Session JSONL (default on): persists via [goc/sessiontranscript] (~/.harness/projects/.../<session>.jsonl). After each successful ProcessUserInput + ApplyBaseResult, maybeRecordTranscript runs so user rows land before streaming yields. Streaming parity wires [query.QueryDeps.OnQueryYield] to RecordTranscript with a growing turn prefix (same as TS recordTranscript(messages)) so parentUuid chains; each yield is deduped by message UUID; turn end still calls maybeRecordTranscript for a full-store sync. File-history-snapshot stubs: default at most one line per session (before the first non-meta user) unless CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING (TS fileHistory off); GOU_DEMO_FILE_HISTORY_SNAPSHOT_EACH_USER=1 restores one stub before every new non-meta user; GOU_DEMO_SKIP_FILE_HISTORY_SNAPSHOT=1 omits stubs. User message UUIDs follow TS (crypto.randomUUID via process-user-input when DemoConfig.uuid is unset). Set GOU_DEMO_SESSION_ID to a UUID or the store gets a random UUID when the default "demo" id is invalid.
// Skill listing follows TS delta (sentSkillNames): later submits omit skills already injected. Set GOU_DEMO_SKILL_LISTING_EVERY_TURN=1 to use a fresh sent map each submit so the full listing is attached every round (debug only; not TS production behavior).
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"goc/ccb-engine/diaglog"
	"runtime"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"goc/gou/ccbstream"
	"goc/gou/commandqueue"
	goumsg "goc/gou/message"
	"goc/gou/messagerow"
	"goc/gou/prompt"
	"goc/gou/pui"
	"goc/gou/theme"
	"goc/tools/toolexecution"
	"goc/types"
)

func (m *model) handleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.msgHistoryBrowseMouseOff {
		m.msgHistoryBrowseMouseOff = false
		m2, cmd := m.handleKeyMsgPreserving(msg)
		if cmd == nil {
			return m2, teaGlobalRedrawCmd()
		}
		return m2, tea.Sequence(teaGlobalRedrawCmd(), cmd)
	}
	return m.handleKeyMsgPreserving(msg)
}

func (m *model) handleKeyMsgPreserving(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.permModal.IsActive() && msg.String() == "ctrl+c" {
		if m.queryCancel != nil {
			m.queryCancel()
			m.queryCancel = nil
		}
		m.permModal.Dismiss()
		return m, nil
	}
	if m.permModal.IsActive() {
		m.permModal.Update(msg)
		return m, nil
	}
	if msg.String() == "ctrl+l" ||
		(msg.String() == "x" && m.hasCompletedAgents()) {
		m.evictCompletedAgents()
		return m, teaGlobalRedrawCmd()
	}
	if m.msgViewportWanted() && msg.String() == "ctrl+y" {
		m.msgFoldAll = !m.msgFoldAll
		m.msgFoldRev++
		return m, nil
	}
	if m.uiScreen == gouDemoScreenPrompt && msg.String() == "ctrl+o" {
		m.slashPicker.Dismiss()
		return m, m.enterTranscriptScreen()
	}
	if m.uiScreen == gouDemoScreenPrompt {
		if msg.String() == "f5" {
			gouDemoTracef("f5 pressed: entering manual render mode (buffering events)")
			m.manualRenderMode = true
			return m, nil
		}
		if msg.String() == "f6" {
			gouDemoTracef("f6 pressed: flushing %d buffered events", len(m.pendingEvents))
			m.manualRenderMode = false
			var cmds []tea.Cmd
			for _, e := range m.pendingEvents {
				_, cmd := m.Update(e)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			m.pendingEvents = nil
			if len(cmds) > 0 {
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}
	}

	if handled, cmd := m.handleTranscriptKey(msg); handled {
		return m, cmd
	}
	// @-mention autocomplete: Tab/Enter/↑/↓/Esc (must run before slash list nav).
	if m.uiScreen == gouDemoScreenPrompt && m.handleAtSuggestKeys(msg) == 1 {
		return m, nil
	}

	// Slash command list: ↑/↓/Tab must win over message-pane scroll (see isListViewportScrollKey).
	if m.uiScreen == gouDemoScreenPrompt && m.handleSlashListNavKey(msg) {
		return m, nil
	}
	if m.msgViewportWanted() && isListViewportScrollKey(msg.String()) {
		diaglog.Line("[key] handleKeyMsgPreserving: msgViewportWanted=true, key=%s, calling handleMsgViewportScrollKey", msg.String())
		return m, m.handleMsgViewportScrollKey(msg)
	} else if isListViewportScrollKey(msg.String()) {
		diaglog.Line("[key] handleKeyMsgPreserving: msgViewportWanted=false, key=%s, handling with traditional scroll", msg.String())
		return m, m.handleTraditionalScrollKey(msg)
	}
	switch msg.String() {
	case "ctrl+c":
		if m.queryBusy && m.queryCancel != nil {
			m.queryCancel()
			m.queryCancel = nil
			return m, nil
		}
		now := time.Now()
		if now.Sub(m.lastCtrlC) < 800*time.Millisecond && m.ctrlCPending {
			m.ctrlCPending = false
			if m.suggestionEngine != nil {
				m.suggestionEngine.FileIndex().Stop()
			}
			return m, tea.Quit
		}
		m.lastCtrlC = now
		m.ctrlCPending = true
		return m, nil
	case "esc":
		if m.slashListVisible() {
			m.slashPicker.Dismiss()
			// If the list is auto-shown (via leading "/"), clear the input
			// so shouldShowTSSlashList stops returning true.
			if m.pr.Value() != "" {
				m.pr.SetValue("")
			}
			m.syncSlashListAfterPrompt()
			return m, nil
		}
		if m.slashResultPanel != nil {
			m.clearSlashResultPanel()
			m.rebuildHeightCache()
			return m, nil
		}
		if m.uiScreen == gouDemoScreenTranscript {
			return m, m.exitTranscriptScreenWithPostCmd()
		}
		if m.suggestionEngine != nil {
			m.suggestionEngine.FileIndex().Stop()
		}
		return m, tea.Quit
	case "f2":
		m.loadSlashCommandsOnce()
		m.slashPicker.SetCommands(m.slashCommands)
		m.slashPicker.ToggleUserManual()
		return m, nil
	}
	if m.uiScreen == gouDemoScreenTranscript {
		switch msg.String() {
		case "q":
			return m, m.exitTranscriptScreenWithPostCmd()
		case "up":
			m.sticky = false
			m.scrollTop = max(0, m.scrollTop-1)
			return m, nil
		case "down":
			m.sticky = false
			m.scrollTop += 1
			return m, nil
		case "pgup":
			m.sticky = false
			m.scrollTop = max(0, m.scrollTop-listViewportH(m)/2)
			return m, nil
		case "pgdown":
			m.sticky = false
			m.scrollTop += listViewportH(m) / 2
			return m, nil
		case "end":
			m.sticky = true
			m.scrollTop = 1 << 30
			return m, nil
		}
		return m, nil
	}

	// Slash list: Enter applies the highlighted command and runs full submit.
	if m.uiScreen == gouDemoScreenPrompt && m.slashListVisible() && isPromptEnterKey(msg) {
		if len(m.visibleSlashList()) > 0 {
			m.applySlashTab()
			fullPrompt := strings.TrimRight(m.pr.Value(), "\r\n")
			m.pr.SetValue("")
			m.slashPicker.Dismiss()
			m.syncSlashListAfterPrompt()
			line := strings.TrimSpace(fullPrompt)
			if line == "" {
				return m, nil
			}
			return m.gouSubmitFromPromptText(fullPrompt, line)
		}
	}
	m.pr.Update(prompt.NormalizeTTYNewlineKey(msg))
	m.syncAtSuggestions()
	m.syncSlashListAfterPrompt()
	if m.pr.Submitted() {
		fullPrompt := strings.TrimRight(m.pr.Value(), "\r\n")
		m.pr.SetValue("")
		line := strings.TrimSpace(fullPrompt)
		if line == "" {
			return m, nil
		}
		return m.gouSubmitFromPromptText(fullPrompt, line)
	}
	// On Windows, ConPTY doesn't handle ANSI erase sequences correctly,
	// leaving ghost characters. Force a full redraw on every keypress
	// so the next frame writes explicit spaces instead of relying on CSI K/J.
	if runtime.GOOS == "windows" {
		return m, teaGlobalRedrawCmd()
	}
	return m, nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// When the interactive hooks config menu is active, delegate all updates to it.
	if m.hooksConfigMenu != nil {
		hm, _ := m.hooksConfigMenu.Update(msg)
		m.hooksConfigMenu = hm.(*hooksConfigMenu)
		if m.hooksConfigMenu.IsDone() {
			m.hooksConfigMenu = nil
		}
		return m, nil
	}

	// When the interactive question UI is active, delegate all updates to it.
	if m.questionUI != nil {
		qm, _ := m.questionUI.Update(msg)
		m.questionUI = qm.(*questionModel) // questionModel.Update always returns *questionModel, nil cmd
		if m.questionUI.IsDone() {
			reply := permissionAskReply{}
			if m.questionUI.IsCancelled() {
				reply.dec = toolexecution.DenyDecision("User declined to answer questions")
			} else {
				updatedInput := m.questionUI.BuildUpdatedInput(m.questionUI.originalInput)
				reply.dec = toolexecution.PermissionDecision{
					Behavior:     toolexecution.PermissionAllow,
					UpdatedInput: updatedInput,
				}
			}
			// Send reply through the questionUI's own channel (not permAsk).
			if m.questionUI.replyCh != nil {
				select {
				case m.questionUI.replyCh <- reply:
				default:
				}
			}
			m.questionUI = nil
		}
		return m, nil
	}

	if m.manualRenderMode {
		switch msg.(type) {
		case ccbstream.Msg, gouQueryDoneMsg, gouQueryYieldMsg, gouStreamEventMsg, gouSpinnerTickMsg, gouStreamingToolUsesMsg, gouToolSummaryDelayTickMsg, gouMemoryAppendMsg, compactPhaseMsg:
			m.pendingEvents = append(m.pendingEvents, msg)
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleUpdateWindowSize(msg)

	case pui.GouHooksMenuMsg:
		m.hooksConfigMenu = m.newHooksConfigMenuFromMsg(msg)
		return m, nil

	case gouPermissionAskMsg:
		if len(msg.questions) > 0 {
			// AskUserQuestion: switch to interactive question UI.
			m.questionUI = newQuestionModel(msg.questions, msg.replyCh, m.width, m.height)
			// Store the original input for building updatedInput on submit.
			m.questionUI.originalInput = msg.input
			return m, nil
		}
		m.permModal.Activate(msg.toolName, string(msg.input), msg.replyCh)
		return m, nil

	case gouTranscriptEditorPrepMsg:
		return m, m.handleTranscriptEditorChainMsg(msg)
	case gouTranscriptEditorExecDoneMsg:
		return m, m.handleTranscriptEditorChainMsg(msg)
	case gouTranscriptEditorClearStatusMsg:
		return m, m.handleTranscriptEditorChainMsg(msg)

	case tea.MouseClickMsg, tea.MouseMotionMsg, tea.MouseWheelMsg, tea.MouseReleaseMsg:
		if m.msgHistoryBrowseMouseOff && m.msgViewportWanted() {
			return m, nil
		}
		if handled, cmd := m.tryHandleMessageListMouse(msg); handled {
			return m, cmd
		}

	case tea.KeyPressMsg:
		return m.handleKeyMsg(msg)

	case gouQueryYieldMsg:
		return m.handleUpdateGouQueryYield(msg)

	case gouStreamEventMsg:
		return m.handleUpdateGouStreamEvent(msg)

	case gouStreamingToolUsesMsg:
		return m.handleUpdateGouStreamingToolUses(msg)

	case gouSpinnerTickMsg: //120ms
		return m.handleUpdateGouSpinnerTick(msg)

	case gouQueryDoneMsg:
		return m.handleUpdateGouQueryDone(msg)

	case gouMemoryAppendMsg:
		return m.handleUpdateGouMemoryAppend(msg)

	case compactPhaseMsg:
		return m.handleUpdateCompactPhase(msg)

	case gouToolSummaryDelayTickMsg:
		return m.handleUpdateToolSummaryDelayTick(msg)

	case ccbstream.Msg:
		return m.handleUpdateCCBStream(msg)

	case AgentRegisteredMsg:
		m.agentTasks.Register(msg.Task)
		return m, func() tea.Msg { return AgentTaskTickMsg{} }

	case AgentProgressMsg:
		m.agentTasks.UpdateProgress(msg.AgentID, msg.Progress)
		// Also update main session token count from sub-agent progress
		if msg.Progress != nil && msg.Progress.TokenCount > 0 {
			m.spinnerTokens += msg.Progress.TokenCount
			if m.agentTasks != nil {
				m.agentTasks.UpdateProgress("main-session", &AgentTaskProgress{
					TokenCount: m.spinnerTokens,
				})
			}
		}
		return m, nil

	case AgentCompletedMsg:
		m.agentTasks.Complete(msg.AgentID, msg.Status)
		if runtime.GOOS == "windows" {
			return m, teaGlobalRedrawCmd()
		}
		return m, nil

	case AgentTaskTickMsg:
		m.agentTasks.EvictExpired(time.Now())
		if m.agentTasks.Count() > 0 {
			return m, taskListTickCmdAgent()
		}
		return m, nil

	case commandQueueNotifyMsg:
		// TUI idle: auto-submit a new query with pending bg agent notifications.
		// TUI busy: notifications are injected at query start by msgsForQ prepend.
		if m.queryBusy || m.lastQueryParams == nil {
			return m, nil
		}
		if !commandqueue.HasPendingNotifications() {
			return m, nil
		}
		// Drain notifications into messages
		msgs := slices.Clone(m.store.Messages)
		for _, cmd := range commandqueue.DrainCommandQueue() {
			content, _ := json.Marshal([]map[string]any{{
				"type": "text",
				"text": "A background agent completed a task:\n" + cmd.Value,
			}})
			msgs = append(msgs, types.Message{
				Type:    "user",
				Content: content,
			})
		}
		m.store.AppendMessage(msgs[len(msgs)-1])
		m.rebuildHeightCache()
		// Reuse last query params with updated messages
		qp := *m.lastQueryParams
		qp.Messages = msgs
		m.beginQuerySpinner()
		m.queryBusy = true
		m.store.ClearStreamingToolUses()
		ctx, cancel := context.WithCancel(context.Background())
		m.queryCancel = cancel
		runQueryStreamingParityTurn(ctx, m.ccbSend, qp)
		return m, nil

	case taskListTickMsg:
		m.taskList.poll()
		return m, taskListTickCmd(m.taskList)
	}

	if syn, ok := prompt.SyntheticTTYKeyFromUnknownMsg(msg); ok {
		return m.handleKeyMsg(syn)
	}
	if m.uiScreen != gouDemoScreenTranscript {
		m.pr.Update(msg)
	}
	return m, nil
}

// taskListViewMaxDisplay matches the line budget for [model.View] (task list after stream rows); keep in sync with that block.
func (m *model) taskListViewMaxDisplay() int {
	if m.height <= 10 {
		return 0
	}
	return min(10, max(3, m.height-14))
}

// taskListViewReservedRows is the vertical space between the message pane and the status line
// that the task list can occupy. [listViewportH] must subtract this so the full frame
// (title + messages + stream strip + task block + status + input) does not exceed [model.height]
// and the input area stays visible.
func (m *model) taskListViewReservedRows() int {
	if m.uiScreen == gouDemoScreenTranscript {
		return 0
	}
	if m.taskList == nil || !m.taskList.isVisible() {
		return 0
	}
	// Upper bound: standalone header (1) + at most N task lines + at most one hidden-summary line.
	md := m.taskListViewMaxDisplay()
	if md == 0 {
		return 2 // header + " … +…" (task_list.view maxDisplay=0)
	}
	return 2 + md
}

func listViewportH(m *model) int {
	streamReserve := m.streamH
	if m.uiScreen == gouDemoScreenTranscript {
		streamReserve = 0
	}
	h := m.height - m.titleH - streamReserve - m.bottomChromeHeight() - 1
	if gouDemoStatusLineEnabled() && m.statusLineString() != "" {
		h--
	}
	if m.uiScreen != gouDemoScreenTranscript {
		h -= m.taskListViewReservedRows()
		// Reserve space for spinner
		if m.queryBusy {
			h-- // spinner row
		}
	}
	if h < 3 {
		h = 3
	}
	return h
}

func (m *model) statusLineString() string {
	if !gouDemoStatusLineEnabled() {
		return ""
	}
	n := len(m.store.Messages)
	vk := len(m.store.ItemKeys())
	s := fmt.Sprintf("theme=%s msgs=%d items=%d cols=%d sticky=%v",
		theme.ActiveTheme(), n, vk, m.cols, m.sticky)
	return lipgloss.NewStyle().Faint(true).Render(s)
}

func (m *model) rebuildHeightCache() {
	m.rebuildHeightCacheCalls++
	m.syncMsgFirstShownAt()

	m.groupedAgentLookups = messagerow.BuildGroupedAgentLookups(m.store.Messages)

	// Convert bool map to struct{} map for existing formatMessageSegments logic
	m.resolvedToolIDs = make(map[string]struct{})
	for k, v := range m.groupedAgentLookups.ResolvedToolUseIDs {
		if v {
			m.resolvedToolIDs[k] = struct{}{}
		}
	}
	if m.heightCache == nil {
		m.heightCache = make(map[string]int)
	}
	hl := m.transcriptSearchHighlightNeedle()
	baseCols := m.cols
	if baseCols < 1 {
		baseCols = 40
	}
	m.msgScrollbarW = 0
	m.msgBodyCols = baseCols
	m.fillMessageHeightCache(baseCols, hl)
	vp := listViewportH(m)
	if gouDemoMessageScrollbarStrip() && baseCols >= 18 && vp >= 3 {
		if m.messageScrollContentHeight() > vp {
			narrow := baseCols - 1
			if narrow >= 8 {
				m.fillMessageHeightCache(narrow, hl)
				if m.messageScrollContentHeight() > vp {
					m.msgScrollbarW = 1
					m.msgBodyCols = narrow
				} else {
					m.fillMessageHeightCache(baseCols, hl)
				}
			}
		}
	}
}

// measureMessageRows returns terminal row count for heightCache / scrollbar offsets.
// It uses the new message renderer ([goumsg.Dispatcher.Measure]) with the same [goumsg.RenderContext]
// fields as [MessageRendererIntegration.ComputeVisibleRange], plus legacy parity tweaks:
//   - [skipFoldedToolResultStubInPrompt] → 0 (omitted user stub rows)
//   - user messages: +1 line (legacy messagerow appended a trailing newline before [layout.WrappedRowCount])
//   - non-attachment: at least 1 row when non-empty measure path would otherwise undercount
//
// Transcript search highlight (searchHL) only affected the old messagerow path; Measure does not widen/wrap on hl.
func (m *model) messagerowOpts(msg types.Message) *messagerow.RenderOpts {
	if m.uiScreen == gouDemoScreenPrompt {
		active := m.queryBusy &&
			len(m.store.Messages) > 0 &&
			m.store.Messages[len(m.store.Messages)-1].UUID == msg.UUID &&
			msg.Type == types.MessageTypeCollapsedReadSearch &&
			!m.store.HasStreaming()
		return &messagerow.RenderOpts{
			FoldToolResultBody:         true,
			CollapsedReadSearchActive:  active,
			GroupedAgentLookups:        m.groupedAgentLookups,
			ResolvedToolUseIDs:         m.resolvedToolIDs,
			SuppressToolUseSummaryLine: m.suppressToolUseSummaryLine(msg),
		}
	}
	if m.uiScreen == gouDemoScreenTranscript {
		ro := &messagerow.RenderOpts{
			GroupedAgentLookups:        m.groupedAgentLookups,
			VerboseCollapsedReadSearch: true,
			ResolvedToolUseIDs:         m.resolvedToolIDs,
			TranscriptMode:             true,
		}
		if m.transcriptShowAll || m.transcriptDumpMode {
			ro.ShowAllInTranscript = true
		} else {
			// Compact transcript (TS): fold tool_result bodies on user rows; assistant row shows ⏺+⎿ via [formatMessageSegments].
			ro.FoldToolResultBody = true
		}
		return ro
	}
	return &messagerow.RenderOpts{
		GroupedAgentLookups: m.groupedAgentLookups,
		ResolvedToolUseIDs:  m.resolvedToolIDs,
	}
}

func (m *model) measureMessageRows(msg types.Message, cols int, searchHL string) int {
	_ = searchHL
	if m.skipFoldedToolResultStubInPrompt(msg) {
		return 0
	}
	m.integrateMessageRenderer()
	if cols < 1 {
		cols = 40
	}
	isTranscript := m.uiScreen == gouDemoScreenTranscript
	verbose := m.transcriptShowAll || (m.uiScreen == gouDemoScreenTranscript && m.transcriptSearchOpen)
	cw := cols
	ctx := &goumsg.RenderContext{
		Width:          cols,
		Theme:          m.msgRenderer.Palette(),
		IsTranscript:   isTranscript,
		IsStatic:       isTranscript,
		Verbose:        verbose,
		Highlighter:    markdownHighlighter,
		AddMargin:      true,
		ContainerWidth: &cw,
	}
	h, err := m.msgRenderer.MeasureMessage(&msg, ctx)
	if err != nil {
		h = 1
	}
	if msg.Type == types.MessageTypeUser {
		// Match legacy height: user block had an extra trailing newline before WrappedRowCount.
		h++
	}
	if msg.Type == types.MessageTypeAttachment {
		return h
	}
	return max(1, h)
}

func extractPartialJSONField(input string, field string) string {
	marker1 := `"` + field + `":"`
	marker2 := `"` + field + `": "`

	idx := strings.Index(input, marker2)
	if idx == -1 {
		idx = strings.Index(input, marker1)
		if idx == -1 {
			return ""
		}
		idx += len(marker1)
	} else {
		idx += len(marker2)
	}

	end := strings.IndexByte(input[idx:], '"')
	if end == -1 {
		return input[idx:]
	}
	return input[idx : idx+end]
}

func (m *model) measureTranscriptStreamingToolRow(group GroupedStreamingTool, cols int, searchHL string) int {
	if !group.IsGroup {
		tu := group.Single
		head := lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
		facing, paren, _ := messagerow.ToolChromeParts(tu.Name, json.RawMessage(tu.UnparsedInput))
		if facing == "" {
			facing = tu.Name
		}
		namePart := facing
		if strings.TrimSpace(searchHL) != "" {
			namePart = highlightSearchPlain(namePart, searchHL, transcriptSearchHLStyle())
		}
		// 所有工具都显示活动状态
		activityLine := messagerow.ActivityLineForToolUse(tu.Name, json.RawMessage(tu.UnparsedInput))
		if activityLine == "" {
			// 如果没有活动描述，使用工具名
			activityLine = namePart
			if p := strings.TrimSpace(paren); p != "" {
				activityLine += " " + p
			}
		}
		// 添加省略号表示正在执行
		activityLine += "…"
		// 添加交互提示
		toolLine := toolRowLeadPrefix(false) + lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render(activityLine) + lipgloss.NewStyle().Faint(true).Render(messagerow.CtrlOToExpandHint)
		block := head + "\n" + toolLine
		return messagePaneGutterRowCount(block, cols)
	}

	head := lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
	summary := messagerow.SearchReadSummaryText(true, group.SearchCount, group.ReadCount, group.ListCount, 0, 0, 0, 0, 0, nil, nil, nil)
	toolLine := toolRowLeadPrefix(false) + lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render(summary) + lipgloss.NewStyle().Faint(true).Render(messagerow.CtrlOToExpandHint)
	block := head + "\n" + toolLine
	for _, item := range group.Items {
		path := extractPartialJSONField(item.UnparsedInput, "file_path")
		if path == "" {
			path = extractPartialJSONField(item.UnparsedInput, "path")
		}
		if path == "" {
			path = extractPartialJSONField(item.UnparsedInput, "pattern")
		}
		if path == "" {
			path = "..."
		}
		treeLine := lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render("  ⎿  " + path)
		block += "\n" + treeLine
	}
	return messagePaneGutterRowCount(block, cols)
}

func (m *model) renderTranscriptStreamingToolRow(group GroupedStreamingTool, cols, h int, searchHL string) string {
	var block string
	if !group.IsGroup {
		tu := group.Single
		head := lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
		facing, paren, _ := messagerow.ToolChromeParts(tu.Name, json.RawMessage(tu.UnparsedInput))
		if facing == "" {
			facing = tu.Name
		}
		namePart := facing
		if strings.TrimSpace(searchHL) != "" {
			namePart = highlightSearchPlain(namePart, searchHL, transcriptSearchHLStyle())
		}
		// 所有工具都显示活动状态
		activityLine := messagerow.ActivityLineForToolUse(tu.Name, json.RawMessage(tu.UnparsedInput))
		if activityLine == "" {
			// 如果没有活动描述，使用工具名
			activityLine = namePart
			if p := strings.TrimSpace(paren); p != "" {
				activityLine += " " + p
			}
		}
		// 添加省略号表示正在执行
		activityLine += "…"
		// 添加交互提示
		toolLine := toolRowLeadPrefix(false) + lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render(activityLine) + lipgloss.NewStyle().Faint(true).Render(messagerow.CtrlOToExpandHint)
		block = head + "\n" + toolLine
	} else {
		head := lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
		summary := messagerow.SearchReadSummaryText(true, group.SearchCount, group.ReadCount, group.ListCount, 0, 0, 0, 0, 0, nil, nil, nil)
		toolLine := toolRowLeadPrefix(false) + lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render(summary) + lipgloss.NewStyle().Faint(true).Render(messagerow.CtrlOToExpandHint)
		block = head + "\n" + toolLine
		for _, item := range group.Items {
			path := extractPartialJSONField(item.UnparsedInput, "file_path")
			if path == "" {
				path = extractPartialJSONField(item.UnparsedInput, "path")
			}
			if path == "" {
				path = extractPartialJSONField(item.UnparsedInput, "pattern")
			}
			if path == "" {
				path = "..."
			}
			treeLine := lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Render("  ⎿  " + path)
			block += "\n" + treeLine
		}
	}

	block = applyMessagePaneGutter(block, cols)
	lines := strings.Split(block, "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h && h > 0 {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}
