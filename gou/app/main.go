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
// System # Language / # Output Style: merged from ~/.claude/settings.go.json and project .claude/settings.go.json / settings.local.json (see settingsfile; project settings.go.json is TS-only). CLAUDE_CODE_LANGUAGE and CLAUDE_CODE_OUTPUT_STYLE_* override when set (non-empty); built-in outputStyle keys Explanatory/Learning use prompts from src/constants/outputStyles.ts (embedded).
// Extra CLAUDE.md roots: optional runtimeContext.toolPermissionContext.additionalWorkingDirectories (JSON) and/or GOU_DEMO_EXTRA_CLAUDE_MD_ROOTS / CLAUDE_CODE_EXTRA_CLAUDE_MD_ROOTS (comma or PATH-style list). Paths from runtime/env are always scanned when passed (see [querycontext.ExtraClaudeMdRootsForFetch]); CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1 is only needed for env-only flows in claudemd that do not pass explicit roots.
// Debug log (optional): GOU_DEMO_LOG_FILE=/path/to.log, or GOU_DEMO_LOG=1 (default file path matches TS getDebugLogPath via goc/ccb-engine/debugpath when stderr is TTY). GOU_DEMO_LOG_STDERR=1 forces stderr (may corrupt TUI). Lines are prefixed [gou-demo].
// ToolUseContext dump: CLAUDE_CODE_LOG_TOOL_USE_CONTEXT or GOU_DEMO_LOG_TOOL_USE_CONTEXT = 1|summary|full (with logging enabled) prints JSON after each BuildDemoParams; full includes the entire commands[] snapshot.
// Read/Grep/Glob stream tail: default keeps each tool_use + tool_result as separate rows (avoids looking like history was cleared). Set GOU_DEMO_COLLAPSE_READ_SEARCH_TAIL=1 for TS-style merge into collapsed_read_search (gou/ccbstream/apply.go).
// Prompt: merged one-line Grep/Glob/Read summaries (GOU_DEMO_TOOL_USE_SUMMARY_LINE) wait GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS after each assistant message first appears (default 2000 ms) while full Search/Read rows are shown; set to 0 to collapse immediately.
//
// Keys: ↑/↓/PgUp/PgDn scroll the message pane, End bottom. Prompt: default Enter send; Alt+Enter or Option+Enter (macOS) newline when the terminal sends Meta; Ctrl+J / LF newline. GOU_DEMO_REPL_ENTER_SUBMITS=0 for chat mode (Enter newline, Alt+Enter send). Shift+↑↓ move line. F2 toggles the slash list; leading "/" (TS) or mid-input " … /tok" shows the list; ↑/↓ move selection; Tab inserts; Enter applies selection and runs submit; input stays in the main field. Ctrl+l forces a full-screen clear + redraw (TS Global app:redraw). Ctrl+o toggles TS-style transcript (frozen tail; / search with n/N when not in dump; search bar Esc clears; ctrl+e show-all expands collapsed/grouped + full tool_result bodies except in dump). In the main prompt, user messages that contain only tool_result / advisor_tool_result blocks are omitted from the list (no "user / ↩ tool_result …" stub row); mixed user rows still fold tool_result bodies to one line + (ctrl+o to expand). Transcript (compact): same omission + tool_result folded on user rows; assistant rows show ⏺+⎿ summaries. Ctrl+e show-all or [ dump shows full blocks. [ (no search bar) enables dump: show-all + plain transcript to scrollback (Printf). v opens frozen transcript in $VISUAL/$EDITOR via temp file (tea.ExecProcess). Transcript pager (search bar closed, not dump): arrows/pgup/pgdn/end, j/k, g, G/shift+g, ctrl+u/d, ctrl+b/f, b, space (full page), ctrl+n/p (line). Esc/q/ctrl+c exit transcript when search bar closed. In prompt mode, q or Esc quit. Columns < 80 use a shorter header/footer (TS REPL isNarrow). Terminal tab title: OSC 0 unless CLAUDE_CODE_DISABLE_TERMINAL_TITLE=1; loading shows a "…" prefix. CLAUDE_CODE_PERMISSION_MODE sets tool permission mode for submits (TS toolPermissionContext.mode).
// Theme: CLAUDE_CODE_THEME=light (after merged settings env) selects a higher-contrast palette; see [theme.InitFromThemeName]. GOU_DEMO_STATUS_LINE=1 shows theme/msg counts above the prompt.
// Message pane: new renderer ([message.VirtualList] in [gou/message]) drives both prompt and transcript screens. Prompt uses [bubbles/viewport] by default (full-document scroll + ctrl+y fold-all); disable with GOU_DEMO_BUBBLES_VIEWPORT=0|false|off|no to render the visible slice directly on top of m.Scroll.Top.
// Mouse: SGR mouse (cell motion) enables wheel + plain left-drag on the message list when not disabled by env. Set GOU_DEMO_DISABLE_MOUSE_SCROLL=1 to ignore wheel/drag in-app. Mirror TS fullscreen.ts: CLAUDE_CODE_DISABLE_MOUSE=1 or GOU_DEMO_DISABLE_MOUSE=1 omits SGR mouse (keyboard scroll still works), unless GOU_DEMO_DISALLOW_DISABLE_MOUSE=1. One-column TUI scrollbar is on by default when the pane is wide enough; GOU_DEMO_MESSAGE_SCROLLBAR=0|false|off|no or GOU_DEMO_NO_SCROLLBAR=1 turns it off. Alternate screen: opt-in GOU_DEMO_ALT_SCREEN=1 (default main buffer). Bubbles viewport: at-top wheel-up can release mouse for host scrollback; opt out with GOU_DEMO_MSG_HISTORY_MOUSE_RELEASE=0|false|off|no.
// Slash: /name is resolved in-process — disk skills via [goc/slashresolve.ResolveDiskSkill], bundled prompts via [goc/slashresolve.ResolveBundledSkill] (embedded markdown under slashresolve/skills/bundled). Other prompt commands need a disk skill (SkillRoot) or a bundled definition. Unknown names that look like command names and are not root filesystem paths (non-Windows) return TS-style Unknown skill without calling the model; otherwise the line is treated as a normal user prompt.
// MCP skills (scheme-2 R0/R1): -mcp-commands-json=path or GOU_DEMO_MCP_COMMANDS_JSON → JSON array of types.Command merged into Skill/commands (enable FEATURE_MCP_SKILLS=1 for listing).
// MCP tool defs (assembleToolPool): -mcp-tools-json=path or GOU_DEMO_MCP_TOOLS_JSON → JSON array merged into Options.Tools when GOU_DEMO_USE_EMBEDDED_TOOLS_API=1 (see mcpcommands.EnvToolsJSONPath).
//
// Session JSONL (default on): persists via [goc/sessiontranscript] (~/.claude/projects/.../<session>.jsonl). After each successful ProcessUserInput + ApplyBaseResult, maybeRecordTranscript runs so user rows land before streaming yields. Streaming parity wires [query.QueryDeps.OnQueryYield] to RecordTranscript with a growing turn prefix (same as TS recordTranscript(messages)) so parentUuid chains; each yield is deduped by message UUID; turn end still calls maybeRecordTranscript for a full-store sync. File-history-snapshot stubs: default at most one line per session (before the first non-meta user) unless CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING (TS fileHistory off); GOU_DEMO_FILE_HISTORY_SNAPSHOT_EACH_USER=1 restores one stub before every new non-meta user; GOU_DEMO_SKIP_FILE_HISTORY_SNAPSHOT=1 omits stubs. User message UUIDs follow TS (crypto.randomUUID via process-user-input when DemoConfig.uuid is unset). Set GOU_DEMO_SESSION_ID to a UUID or the store gets a random UUID when the default "demo" id is invalid.
// Skill listing follows TS delta (sentSkillNames): later submits omit skills already injected. Set GOU_DEMO_SKILL_LISTING_EVERY_TURN=1 to use a fresh sent map each submit so the full listing is attached every round (debug only; not TS production behavior).
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"goc/ccb-engine/diaglog"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"goc/ccb-engine/apilog"
	"goc/claudeinit"
	"goc/commands"
	"goc/compactservice"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	config "goc/gou/app/config"
	"goc/gou/ccbhydrate"
	"goc/gou/ccbstream"
	"goc/gou/conversation"
	"goc/gou/layout"
	"goc/gou/markdown"
	goumsg "goc/gou/message"
	"goc/gou/messagerow"
	"goc/gou/prompt"
	"goc/gou/pui"
	"goc/gou/segdiff"
	state "goc/gou/app/state"
	"goc/gou/suggestions"
	"goc/gou/textutil"
	"goc/gou/theme"
	"goc/gou/transcript"
	"goc/growthbook"
	"goc/hookexec"
	"goc/messagesapi"
	"goc/modelenv"
	"goc/querycontext"
	"goc/services/autodream"
	"goc/services/extractmemories"
	"goc/services/sessionmemory"
	"goc/sessiontranscript"
	"goc/tools"
	"goc/tools/localtools"
	"goc/tools/skilltools"
	"goc/tools/toolsearchwire"
	"goc/tools/toolexecution"
	"goc/tools/toolresultpersist"
	"goc/tscontext"
	"goc/types"
)

// markdownHighlighter is the global code highlighter instance
var markdownHighlighter *markdown.Highlighter

// applyAssistantStreamingGutter wraps streaming assistant text to (cols − 4) with a "⏺ "
// lead on the first line and 4-space indent on continuation lines, matching the completed
// assistant text layout in [goumsg.AssistantMessageRenderer.renderTextBlock].
func applyAssistantStreamingGutter(block string, cols int) string {
	if block == "" {
		return ""
	}
	wrapCols := cols - 4
	if wrapCols < 20 {
		wrapCols = 20
	}
	wrapped := layout.WrapForViewport(block, wrapCols)
	lines := strings.Split(wrapped, "\n")
	for i, line := range lines {
		if i == 0 {
			lines[i] = "  ⏺ " + line
		} else {
			lines[i] = "    " + line
		}
	}
	return strings.Join(lines, "\n")
}

// AgentRegisteredMsg is sent when a new agent task is registered.
type AgentRegisteredMsg struct {
	Task *AgentTaskState
}

// AgentProgressMsg carries a progress update for a running agent.
type AgentProgressMsg struct {
	AgentID  string
	Progress *AgentTaskProgress
}

// AgentCompletedMsg signals an agent completed, was killed, or failed.
type AgentCompletedMsg struct {
	AgentID string
	Status  string // "completed", "killed", "failed"
	Result  string
}

// AgentTaskTickMsg is sent by the 1s tick timer for coordinator panel refresh.
type AgentTaskTickMsg struct{}

func setupGouDemoTrace() (cleanup func()) {
	return config.SetupTrace()
}


func (m *model) beginQuerySpinner() {
	m.Query.BusyStartedAt = time.Now()
	m.Query.SpinnerVerb = pickSpinnerVerb()
	m.Query.SpinnerFrame = 0
}

func (m *model) endQuerySpinner() {
	m.Query.SpinnerVerb = ""
	m.Query.BusyStartedAt = time.Time{}
	m.Query.SpinnerFrame = 0
}

func padStreamRows(rows []string, h int) []string {
	for len(rows) < h {
		rows = append(rows, "")
	}
	if len(rows) > h {
		return rows[:h]
	}
	return rows
}

func (m *model) promptBottomStreamRows() []string {
	h := m.Layout.StreamH
	if h < 1 {
		h = 1
	}
	return padStreamRows(nil, h)
}

type model struct {
	*state.Conversation
	*state.Input
	*state.Scroll
	*state.Layout
	*state.Viewport
	*state.Screen
	*state.Query
	*state.Modal
	*state.Chrome
	*state.Agent
	*state.Memory
	*state.Tool
	*state.MessageTracking
	*state.ManualRender
	*state.Mouse

	slashListUser bool
	slashListSel  int

	msgRenderer *MessageRendererIntegration
}

// Run initializes and runs the TUI application. It blocks until the user exits.
// Mirrors the combined behavior of TS cli.tsx + main.tsx REPL launch.
func Run(config_ config.Config) error {
	if err := claudeinit.Init(context.Background(), claudeinit.Options{NonInteractive: true}); err != nil {
		return fmt.Errorf("claudeinit: %w", err)
	}
	defer claudeinit.RunCleanups()
	theme.InitFromThemeName(os.Getenv("CLAUDE_CODE_THEME"))

	hlConfig := markdown.DefaultHighlightConfig()
	var hlErr error
	markdownHighlighter, hlErr = markdown.NewHighlighter(hlConfig)
	if hlErr != nil {
		log.Printf("app: failed to create markdown highlighter: %v", hlErr)
	} else {
		log.Printf("app: markdown highlighter initialized with style=%s, formatter=%s", hlConfig.StyleName, hlConfig.FormatterName)
	}

	// Ensure Grep and Glob tools are available.
	if os.Getenv("EMBEDDED_SEARCH_TOOLS") != "" {
		gouDemoTracef("app: overriding EMBEDDED_SEARCH_TOOLS=%q", os.Getenv("EMBEDDED_SEARCH_TOOLS"))
	}
	os.Setenv("EMBEDDED_SEARCH_TOOLS", "0")

	apilog.PrepareIfEnabled()
	apilog.MaybePrintDiag()

	if os.Getenv("GOC_EXTRACT_MEMORIES") == "" {
		_ = os.Setenv("GOC_EXTRACT_MEMORIES", "1")
	}

	traceCleanup := setupGouDemoTrace()
	defer traceCleanup()

	if gouDemoEnvTruthy("GOU_DEMO_TS_CONTEXT_BRIDGE") {
		return fmt.Errorf("GOU_DEMO_TS_CONTEXT_BRIDGE is no longer supported")
	}

	sessionID := config_.SessionID
	if sessionID == "" || !sessiontranscript.IsValidUUID(sessionID) {
		sessionID = sessiontranscript.NewUUID()
	}
	st := &conversation.Store{ConversationID: sessionID}
	if config_.TranscriptPath != "" {
		msgs, err := transcript.LoadFile(config_.TranscriptPath)
		if err != nil {
			return fmt.Errorf("transcript: %w", err)
		}
		st.Messages = msgs
	}
	if config_.ReplayCCPath != "" {
		if err := ccbstream.ReplayFile(config_.ReplayCCPath, st); err != nil {
			return fmt.Errorf("replay-cc: %w", err)
		}
	}

	mcpCmdPath := strings.TrimSpace(config_.MCPCommandsJSONPath)
	mcpToolPath := strings.TrimSpace(config_.MCPToolsJSONPath)
	m := newModel(st, mcpCmdPath, mcpToolPath, nil)
	m.Agent.TaskList.(*taskListModel).setAgentTasks(m.Agent.Tasks.(*agentTaskStore))

	opts := []tea.ProgramOption{}
	if config_.StreamStdin {
		tty, err := os.Open("/dev/tty")
		if err == nil {
			opts = append(opts, tea.WithInput(tty))
			defer tty.Close()
		}
	}
	inlineCCB := true
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_CCB_INLINE"))); v == "0" || v == "false" || v == "off" || v == "no" {
		inlineCCB = false
	}
	p := tea.NewProgram(m, opts...)
	m.BindCCB(p.Send, inlineCCB)
	gouDemoWarnApilogExpectations(inlineCCB)
	gouDemoTracef("startup messages=%d ccbInline=%v", len(st.Messages), inlineCCB)
	if gouDemoKittyKeyboardEnabled() {
		defer func() { _ = prompt.WriteKittyKeyboardProtocolDisable() }()
	}
	if config_.StreamStdin {
		ccbstream.Feed(os.Stdin, p)
	}
	res, runErr := p.Run()
	if runErr != nil {
		fmt.Fprintln(os.Stderr, runErr)
	}
	if m, ok := res.(*model); ok && gouDemoAltScreenEnabled() && (gouDemoEnvTruthy("GOU_DEMO_DUMP_ON_EXIT") || m.Screen.DumpMode) {
		fmt.Print(transcriptExportPlain(m, exportTranscriptWidth(m)) + "\n")
	}
	return runErr
}

func newModel(st *conversation.Store, mcpCommandsJSONPath, mcpToolsJSONPath string, tsBridge *tscontext.Snapshot) *model {
	pr := prompt.New()
	pr.SetEnterSubmits(gouDemoPromptEnterSubmits())

	cwd, _ := os.Getwd()

	// Initialize @-mention autocomplete file index and engine.
	suggFI := suggestions.NewFileIndex(cwd)
	suggEngine := suggestions.NewSuggestionEngine(suggFI)

	fhSnap := !gouDemoEnvTruthy("GOU_DEMO_SKIP_FILE_HISTORY_SNAPSHOT")
	fhEachUser := gouDemoEnvTruthy("GOU_DEMO_FILE_HISTORY_SNAPSHOT_EACH_USER")
	tr := &sessiontranscript.Store{
		SessionID:                 st.ConversationID,
		OriginalCwd:               cwd,
		Cwd:                       cwd,
		FileHistorySnapshotOnUser: fhSnap,
		// Default: at most one stub snapshot per session (TS often shows one line with checkpointing off or single-turn).
		FileHistorySnapshotOnce: fhSnap && !fhEachUser,
	}

	lm := modelenv.EffectiveMainLoopModel()

	sessionMemState := sessionmemory.NewState()
	var toolResultState *toolresultpersist.ContentReplacementState
	if !gouDemoEnvFalsy("GOU_DEMO_TOOL_RESULT_PERSIST") {
		toolResultState = toolresultpersist.NewContentReplacementState()
	}
	return &model{
		Conversation: &state.Conversation{
			Store:           st,
			Transcript:      tr,
			ResolvedToolIDs: make(map[string]struct{}),
			ReadFileState:   localtools.NewReadFileState(),
			TSBridge:        tsBridge,
		},
		Input: &state.Input{
			PR:               pr,
			SkillListingSent: make(map[string]struct{}),
			SuggestionEngine: suggEngine,
		},
		Scroll:   state.NewScroll(),
		Layout:   state.NewLayout(),
		Viewport: &state.Viewport{Enabled: gouDemoBubblesViewport()},
		Screen:   &state.Screen{},
		Query:    &state.Query{},
		Modal:    &state.Modal{},
		Chrome: &state.Chrome{
			PermissionMode:    gouDemoPermissionModeFromEnv(),
			LastMainLoopModel: lm,
		},
		Agent: &state.Agent{
			TaskList: newTaskListModel(st.ConversationID),
			Tasks:    newAgentTaskStore(),
		},
		Memory: &state.Memory{
			AutoDream:   autodream.NewState(),
			ExtractMem:  extractmemories.NewState(),
			SessionMem:  sessionMemState,
			SessionHook: sessionmemory.Hook(sessionMemState, st.ConversationID, cwd),
		},
		Tool: &state.Tool{
			ResultState:         toolResultState,
			MCPCommandsJSONPath: mcpCommandsJSONPath,
			MCPToolsJSONPath:    mcpToolsJSONPath,
		},
		MessageTracking: &state.MessageTracking{
			FirstShownAt:            make(map[string]time.Time),
			LastAssistantContentLen: make(map[string]int),
		},
	}}

func (m *model) maybeRecordTranscript() {
	if m.Conversation.Transcript == nil {
		return
	}
	msgs := slices.Clone(m.Conversation.Store.Messages)
	_, err := m.Conversation.Transcript.RecordTranscript(context.Background(), msgs, sessiontranscript.RecordOpts{AllMessages: msgs})
	if err != nil {
		config.Tracef("RecordTranscript: %v", err)
	}
}

// BindCCB wires Bubble Tea Send and whether real HTTP streaming parity is allowed.
func (m *model) BindCCB(send func(tea.Msg), inline bool) {
	m.Query.CCBSend = func(msg interface{}) { send(msg.(tea.Msg)) }
	m.Query.CCBInline = inline
}

func (m *model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if gouDemoKittyKeyboardEnabled() {
		cmds = append(cmds, func() tea.Msg {
			_ = prompt.WriteKittyKeyboardProtocolEnable()
			return nil
		})
	}
	if gouDemoToolUseSummaryDelay() > 0 {
		cmds = append(cmds, tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return gouToolSummaryDelayTickMsg{} }))
	}
	cmds = append(cmds, taskListTickCmd(m.Agent.TaskList.(*taskListModel)))

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// teaGlobalRedrawCmd mirrors TS useGlobalKeybindings app:redraw (ctrl+l): clear the terminal
// so the next frame repaints fully (e.g. after the host cleared scrollback with Cmd+K).
func teaGlobalRedrawCmd() tea.Cmd {
	return func() tea.Msg { return tea.ClearScreen() }
}

func (m *model) inputAreaHeight() int {
	h := m.Input.PR.LineCount()
	if m.Input.SuggVisible && len(m.Input.Suggestions) > 0 {
		visibleRows := min(6, len(m.Input.Suggestions))
		h += 1 + visibleRows // title line + suggestion rows
	}
	if m.Screen.Mode != state.ScreenTranscript {
		h++ // horizontal rule above input
	}
	if m.Screen.Mode != state.ScreenTranscript && !gouDemoBuiltinStatusLineDisabled() {
		s := m.builtinStatusLineView()
		if s != "" {
			h += strings.Count(s, "\n") + 1
		}
	}
	if h < 2 {
		h = 2
	}
	if h > 16 {
		h = 16
	}
	return h
}

// promptAboveInputRuleLine is a faint full-width line between the context row and the multiline prompt.
func promptAboveInputRuleLine(cols int) string {
	if cols < 1 {
		cols = 40
	}
	rule := strings.Repeat("─", cols)
	return lipgloss.NewStyle().Faint(true).Foreground(theme.DimMuted()).Width(cols).Render(rule)
}

// bottomChromeHeight is prompt input height or transcript footer height (TS transcript has no prompt).
func (m *model) bottomChromeHeight() int {
	if m.Screen.Mode != state.ScreenTranscript {
		h := m.inputAreaHeight()
		h += m.slashResultPanelChromeExtra()
		h += m.slashListChromeExtra()
		return h
	}
	narrow := m.Layout.Cols > 0 && m.Layout.Cols < 80
	foot := joinFooterLines(transcriptChromeFootLines(m, narrow), m.Layout.Cols)
	c := m.Layout.Cols
	if c < 1 {
		c = 40
	}
	n := len(strings.Split(layout.WrapForViewport(foot, c), "\n"))
	return max(4, n+1)
}

// handleKeyMsg is the tea.KeyPressMsg branch; also used when SyntheticTTYKeyFromUnknownMsg maps Kitty CSI to KeyPressMsg.
func (m *model) handleKeyMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Viewport.HistoryBrowseMouseOff {
		m.Viewport.HistoryBrowseMouseOff = false
		m2, cmd := m.handleKeyMsgPreserving(msg)
		if cmd == nil {
			return m2, teaGlobalRedrawCmd()
		}
		return m2, tea.Sequence(teaGlobalRedrawCmd(), cmd)
	}
	return m.handleKeyMsgPreserving(msg)
}

func (m *model) handleKeyMsgPreserving(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.Modal.Permission != nil && msg.String() == "ctrl+c" {
		m.finishPermissionAsk(permissionAskReply{dec: toolexecution.DenyDecision("interrupted"), err: nil})
		if m.Query.Cancel != nil {
			m.Query.Cancel()
			m.Query.Cancel = nil
		}
		return m, nil
	}
	if m.handlePermissionKey(msg) {
		return m, nil
	}
	if msg.String() == "ctrl+l" {
		return m, teaGlobalRedrawCmd()
	}
	if m.msgViewportWanted() && msg.String() == "ctrl+y" {
		m.Viewport.FoldAll = !m.Viewport.FoldAll
		m.Viewport.FoldRev++
		return m, nil
	}
	if m.Modal.Permission == nil && m.Screen.Mode == state.ScreenPrompt && msg.String() == "ctrl+o" {
		m.slashListUser = false
		return m, m.enterTranscriptScreen()
	}
	if m.Modal.Permission == nil && m.Screen.Mode == state.ScreenPrompt {
		if msg.String() == "f5" {
			gouDemoTracef("f5 pressed: entering manual render mode (buffering events)")
			m.ManualRender.Active = true
			return m, nil
		}
		if msg.String() == "f6" {
			gouDemoTracef("f6 pressed: flushing %d buffered events", len(m.ManualRender.Events))
			m.ManualRender.Active = false
			var cmds []tea.Cmd
			for _, e := range m.ManualRender.Events {
				_, cmd := m.Update(e)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			m.ManualRender.Events = nil
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
	if m.Screen.Mode == state.ScreenPrompt && m.handleAtSuggestKeys(msg) == 1 {
		return m, nil
	}

	// Slash command list: ↑/↓/Tab must win over message-pane scroll (see isListViewportScrollKey).
	if m.Screen.Mode == state.ScreenPrompt && m.handleSlashListNavKey(msg) {
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
		if m.Query.Busy && m.Query.Cancel != nil {
			m.Query.Cancel()
			m.Query.Cancel = nil
			return m, nil
		}
		now := time.Now()
		if now.Sub(m.Query.LastCtrlC) < 800*time.Millisecond && m.Query.CtrlCPending {
			m.Query.CtrlCPending = false
			if m.Input.SuggestionEngine != nil {
				m.Input.SuggestionEngine.FileIndex().Stop()
			}
			return m, tea.Quit
		}
		m.Query.LastCtrlC = now
		m.Query.CtrlCPending = true
		return m, nil
	case "esc":
		if m.slashListUser {
			m.slashListUser = false
			return m, nil
		}
		if m.Input.SlashResultPanel != nil {
			m.clearSlashResultPanel()
			m.rebuildHeightCache()
			return m, nil
		}
		if m.Screen.Mode == state.ScreenTranscript {
			return m, m.exitTranscriptScreenWithPostCmd()
		}
		if m.Input.SuggestionEngine != nil {
			m.Input.SuggestionEngine.FileIndex().Stop()
		}
		return m, tea.Quit
	case "f2":
		m.toggleSlashListUser()
		return m, nil
	}
	if m.Screen.Mode == state.ScreenTranscript {
		switch msg.String() {
		case "q":
			return m, m.exitTranscriptScreenWithPostCmd()
		case "up":
			m.Scroll.Sticky = false
			m.Scroll.Top = max(0, m.Scroll.Top-1)
			return m, nil
		case "down":
			m.Scroll.Sticky = false
			m.Scroll.Top += 1
			return m, nil
		case "pgup":
			m.Scroll.Sticky = false
			m.Scroll.Top = max(0, m.Scroll.Top-listViewportH(m)/2)
			return m, nil
		case "pgdown":
			m.Scroll.Sticky = false
			m.Scroll.Top += listViewportH(m) / 2
			return m, nil
		case "end":
			m.Scroll.Sticky = true
			m.Scroll.Top = 1 << 30
			return m, nil
		}
		return m, nil
	}

	// Slash list: Enter applies the highlighted command and runs full submit.
	if m.Screen.Mode == state.ScreenPrompt && m.slashListVisible() && isPromptEnterKey(msg) {
		if len(m.visibleSlashList()) > 0 {
			m.applySlashTab()
			fullPrompt := strings.TrimRight(m.Input.PR.Value(), "\r\n")
			m.Input.PR.SetValue("")
			m.slashListUser = false
			m.syncSlashListAfterPrompt()
			line := strings.TrimSpace(fullPrompt)
			if line == "" {
				return m, nil
			}
			return m.gouSubmitFromPromptText(fullPrompt, line)
		}
	}
	m.Input.PR.Update(prompt.NormalizeTTYNewlineKey(msg))
	m.syncAtSuggestions()
	m.syncSlashListAfterPrompt()
	if m.Input.PR.Submitted() {
		fullPrompt := strings.TrimRight(m.Input.PR.Value(), "\r\n")
		m.Input.PR.SetValue("")
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
	// When the interactive question UI is active, delegate all updates to it.
	if m.Modal.Question != nil {
		qm, _ := m.Modal.Question.(*questionModel).Update(msg)
		m.Modal.Question = qm.(*questionModel) // questionModel.Update always returns *questionModel, nil cmd
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
			// Send reply through the questionUI's own channel (not permAsk).
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

	if m.ManualRender.Active {
		switch msg.(type) {
		case ccbstream.Msg, gouQueryDoneMsg, gouQueryYieldMsg, gouStreamEventMsg, gouSpinnerTickMsg, gouStreamingToolUsesMsg, gouToolSummaryDelayTickMsg, gouMemoryAppendMsg, compactPhaseMsg:
			m.ManualRender.Events = append(m.ManualRender.Events, msg)
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleUpdateWindowSize(msg)

	case gouPermissionAskMsg:
		if len(msg.questions) > 0 {
			// AskUserQuestion: switch to interactive question UI.
			m.Modal.Question = newQuestionModel(msg.questions, msg.replyCh, m.Layout.Width, m.Layout.Height)
			// Store the original input for building updatedInput on submit.
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

	case gouTranscriptEditorPrepMsg:
		return m, m.handleTranscriptEditorChainMsg(msg)
	case gouTranscriptEditorExecDoneMsg:
		return m, m.handleTranscriptEditorChainMsg(msg)
	case gouTranscriptEditorClearStatusMsg:
		return m, m.handleTranscriptEditorChainMsg(msg)

	case tea.MouseClickMsg, tea.MouseMotionMsg, tea.MouseWheelMsg, tea.MouseReleaseMsg:
		if m.Viewport.HistoryBrowseMouseOff && m.msgViewportWanted() {
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
		m.Agent.Tasks.(*agentTaskStore).Register(msg.Task)
		return m, func() tea.Msg { return AgentTaskTickMsg{} }

	case AgentProgressMsg:
		m.Agent.Tasks.(*agentTaskStore).UpdateProgress(msg.AgentID, msg.Progress)
		return m, nil

	case AgentCompletedMsg:
		m.Agent.Tasks.(*agentTaskStore).Complete(msg.AgentID, msg.Status)
		return m, nil

	case AgentTaskTickMsg:
		m.Agent.Tasks.(*agentTaskStore).EvictExpired(time.Now())
		if m.Agent.Tasks.(*agentTaskStore).Count() > 0 {
			return m, taskListTickCmdAgent()
		}
		return m, nil

	case taskListTickMsg:
		m.Agent.TaskList.(*taskListModel).poll()
		return m, taskListTickCmd(m.Agent.TaskList.(*taskListModel))
	}

	if syn, ok := prompt.SyntheticTTYKeyFromUnknownMsg(msg); ok {
		return m.handleKeyMsg(syn)
	}
	if m.Screen.Mode != state.ScreenTranscript {
		m.Input.PR.Update(msg)
	}
	return m, nil
}

// taskListViewMaxDisplay matches the line budget for [model.View] (task list after stream rows); keep in sync with that block.
func (m *model) taskListViewMaxDisplay() int {
	if m.Layout.Height <= 10 {
		return 0
	}
	return min(10, max(3, m.Layout.Height-14))
}

// taskListViewReservedRows is the vertical space between the message pane and the status line
// that the task list can occupy. [listViewportH] must subtract this so the full frame
// (title + messages + stream strip + task block + status + input) does not exceed [model.height]
// and the input area stays visible.
func (m *model) taskListViewReservedRows() int {
	if m.Screen.Mode == state.ScreenTranscript {
		return 0
	}
	if m.Agent.TaskList == nil || !m.Agent.TaskList.(*taskListModel).isVisible() {
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
	streamReserve := m.Layout.StreamH
	if m.Screen.Mode == state.ScreenTranscript {
		streamReserve = 0
	}
	h := m.Layout.Height - m.Layout.TitleH - streamReserve - m.bottomChromeHeight() - 1
	if gouDemoStatusLineEnabled() && m.statusLineString() != "" {
		h--
	}
	if m.Screen.Mode != state.ScreenTranscript {
		h -= m.taskListViewReservedRows()
		// Reserve lines for coordinator panel (main row + up to N agent rows)
		if cv := m.agentCoordinatorView(); cv != "" {
			lines := strings.Count(cv, "\n")
			if lines > 0 {
				h -= lines
			}
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
	n := len(m.Conversation.Store.Messages)
	vk := len(m.Conversation.Store.ItemKeys())
	s := fmt.Sprintf("theme=%s msgs=%d items=%d cols=%d sticky=%v",
		theme.ActiveTheme(), n, vk, m.Layout.Cols, m.Scroll.Sticky)
	return lipgloss.NewStyle().Faint(true).Render(s)
}

func (m *model) rebuildHeightCache() {
	m.MessageTracking.RebuildHeightCacheCalls++
	m.syncMsgFirstShownAt()

	m.Conversation.GroupedAgentLookups = messagerow.BuildGroupedAgentLookups(m.Conversation.Store.Messages)

	// Convert bool map to struct{} map for existing formatMessageSegments logic
	m.Conversation.ResolvedToolIDs = make(map[string]struct{})
	for k, v := range m.Conversation.GroupedAgentLookups.ResolvedToolUseIDs {
		if v {
			m.Conversation.ResolvedToolIDs[k] = struct{}{}
		}
	}
	if m.Scroll.HeightCache == nil {
		m.Scroll.HeightCache = make(map[string]int)
	}
	hl := m.transcriptSearchHighlightNeedle()
	baseCols := m.Layout.Cols
	if baseCols < 1 {
		baseCols = 40
	}
	m.Layout.MsgScrollbarW = 0
	m.Layout.MsgBodyCols = baseCols
	m.fillMessageHeightCache(baseCols, hl)
	vp := listViewportH(m)
	if gouDemoMessageScrollbarStrip() && baseCols >= 18 && vp >= 3 {
		if m.messageScrollContentHeight() > vp {
			narrow := baseCols - 1
			if narrow >= 8 {
				m.fillMessageHeightCache(narrow, hl)
				if m.messageScrollContentHeight() > vp {
					m.Layout.MsgScrollbarW = 1
					m.Layout.MsgBodyCols = narrow
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
	if m.Screen.Mode == state.ScreenPrompt {
		active := m.Query.Busy &&
			len(m.Conversation.Store.Messages) > 0 &&
			m.Conversation.Store.Messages[len(m.Conversation.Store.Messages)-1].UUID == msg.UUID &&
			msg.Type == types.MessageTypeCollapsedReadSearch &&
			!m.Conversation.Store.HasStreaming()
		return &messagerow.RenderOpts{
			FoldToolResultBody:         true,
			CollapsedReadSearchActive:  active,
			GroupedAgentLookups:        m.Conversation.GroupedAgentLookups,
			ResolvedToolUseIDs:         m.Conversation.ResolvedToolIDs,
			SuppressToolUseSummaryLine: m.suppressToolUseSummaryLine(msg),
		}
	}
	if m.Screen.Mode == state.ScreenTranscript {
		ro := &messagerow.RenderOpts{
			GroupedAgentLookups:        m.Conversation.GroupedAgentLookups,
			VerboseCollapsedReadSearch: true,
			ResolvedToolUseIDs:         m.Conversation.ResolvedToolIDs,
			TranscriptMode:             true,
		}
		if m.Screen.ShowAll || m.Screen.DumpMode {
			ro.ShowAllInTranscript = true
		} else {
			// Compact transcript (TS): fold tool_result bodies on user rows; assistant row shows ⏺+⎿ via [formatMessageSegments].
			ro.FoldToolResultBody = true
		}
		return ro
	}
	return &messagerow.RenderOpts{
		GroupedAgentLookups: m.Conversation.GroupedAgentLookups,
		ResolvedToolUseIDs:  m.Conversation.ResolvedToolIDs,
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
	isTranscript := m.Screen.Mode == state.ScreenTranscript
	verbose := m.Screen.ShowAll || (m.Screen.Mode == state.ScreenTranscript && m.Screen.SearchOpen)
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

func (m *model) View() tea.View {
	// When the interactive question UI is active, render it instead of the normal view.
	if m.Modal.Question != nil {
		return m.wrapRootView(m.Modal.Question.(*questionModel).View().Content)
	}

	if m.Layout.Width == 0 {
		return m.wrapRootView("Loading…")
	}

	vpH := listViewportH(m)
	bodyCols := m.messageBodyColsForLayout()
	useVp := m.msgViewportWanted()
	if useVp {
		m.msgViewportSyncGeometry()
		m.applyMsgViewportContentFromView()
		if m.Viewport.Fallback {
			useVp = false
		}
	}

	var b strings.Builder
	narrow := m.Layout.Cols > 0 && m.Layout.Cols < 80
	plainTitle := replChromeComposeTerminalTitle(m.Conversation.Store.ConversationID, m.Query.Busy, m.Conversation.Store.HasStreaming())
	if !gouDemoTerminalTitleDisabled() && plainTitle != m.Chrome.LastEmittedTitlePlain {
		m.Chrome.LastEmittedTitlePlain = plainTitle
		if osc := oscSetWindowTitle(plainTitle); osc != "" {
			b.WriteString(osc)
		}
	}
	topBar := replChromeTopBar(narrow)
	if m.Screen.Mode == state.ScreenTranscript {
		topBar = replChromeTranscriptTopBar(narrow)
	}
	title := lipgloss.NewStyle().Bold(true).Render(topBar)
	b.WriteString(title)
	b.WriteByte('\n')

	if useVp {
		// Bubbles viewport + new renderer (applyMsgViewportContentFromView populates the full document).
		b.WriteString(m.messagePaneViewportBlock(vpH, bodyCols))
		b.WriteByte('\n')
	} else {
		// New renderer without bubbles viewport: virtual slice uses m.Scroll.Top; scrollbar reflects renderer totals.
		msgPaneContent := m.renderMessagePaneWithNewRenderer()
		lines := strings.Split(msgPaneContent, "\n")
		if len(lines) > vpH {
			lines = lines[:vpH]
		}
		for len(lines) < vpH {
			lines = append(lines, "")
		}
		m.integrateMessageRenderer()
		messagesPtr := m.messagePtrSliceForNewRenderer()
		isTranscript := m.Screen.Mode == state.ScreenTranscript
		verbose := m.Screen.ShowAll || (m.Screen.Mode == state.ScreenTranscript && m.Screen.SearchOpen)
		_, _, totalHeight := m.msgRenderer.ComputeVisibleRange(
			messagesPtr,
			0,
			1,
			isTranscript,
			verbose,
			bodyCols,
		)
		b.WriteString(joinMessagePaneLinesWithScrollbar(lines, bodyCols, vpH, totalHeight, m.Scroll.Top, m.Layout.MsgScrollbarW))
		b.WriteByte('\n')
	}

	if m.Screen.Mode != state.ScreenTranscript {
		streamRows := m.promptBottomStreamRows()
		if len(streamRows) > 0 {
			b.WriteString(strings.Join(streamRows, "\n"))
			b.WriteByte('\n')
		}
		// Task list (inline, after stream rows)
		if m.Agent.TaskList != nil && m.Agent.TaskList.(*taskListModel).isVisible() {
			maxDisplay := m.taskListViewMaxDisplay()
			if tl := m.Agent.TaskList.(*taskListModel).view(maxDisplay, m.Layout.Cols); tl != "" {
				indented := applyMessagePaneGutter(tl, m.Layout.Width)
				b.WriteString(indented)
				b.WriteByte('\n')
			}
		}
		// Agent coordinator panel (below task list, above status line)
		if cv := m.agentCoordinatorView(); cv != "" {
			indented := applyMessagePaneGutter(cv, m.Layout.Width)
			b.WriteString(indented)
			b.WriteByte('\n')
		}
	}
	if s := m.statusLineString(); s != "" {
		b.WriteString(s)
		b.WriteByte('\n')
	}

	if m.Screen.Mode == state.ScreenTranscript {
		foot := joinFooterLines(transcriptChromeFootLines(m, narrow), m.Layout.Cols)
		b.WriteString(lipgloss.NewStyle().Faint(true).Width(m.Layout.Cols).Render(foot))
	} else {
		// @-mention autocomplete suggestions (above input footer area)
		if s := m.renderAtSuggestions(); s != "" {
			b.WriteString(s)
			b.WriteByte('\n')
		}
		if s := m.builtinStatusLineView(); s != "" {
			b.WriteString(s)
			b.WriteByte('\n')
		}
		b.WriteString(promptAboveInputRuleLine(m.Layout.Cols))
		b.WriteByte('\n')
		promptView := userInputViewWithPromptPrefix(m)
		b.WriteString(promptView)
		if blk := m.slashResultPanelViewBlock(); blk != "" {
			b.WriteByte('\n')
			b.WriteString(blk)
		}
		if m.slashListVisible() {
			if sp := m.renderSlashPicker(m.Layout.Cols, m.Layout.Height); sp != "" {
				b.WriteByte('\n')
				b.WriteString(sp)
			}
		}
	}
	out := lipgloss.NewStyle().MaxWidth(m.Layout.Width).Render(b.String())
	if m.Modal.Permission != nil {
		mod := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(m.renderPermissionModal(m.Layout.Width))
		out = lipgloss.JoinVertical(lipgloss.Left, out, mod)
	}
	return m.wrapRootView(out)
}

func (m *model) wrapRootView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = gouDemoAltScreenEnabled() && !m.Screen.SuspendAltScreenForScrollbackDump
	if gouDemoMouseCellMotionEnabled() {
		if m.Viewport.HistoryBrowseMouseOff {
			v.MouseMode = tea.MouseModeNone
		} else {
			v.MouseMode = tea.MouseModeCellMotion
		}
	}
	return v
}

func (m *model) showToolUseCtrlOExpandHint() bool {
	return m.Screen.Mode == state.ScreenPrompt && !m.Screen.DumpMode
}

// userAssistantPairBlankLine is true when the UI inserts one empty line between adjacent
// user and assistant scroll rows (either order).
func userAssistantPairBlankLine(a, b types.Message) bool {
	u, aType := types.MessageTypeUser, types.MessageTypeAssistant
	c := types.MessageTypeCollapsedReadSearch
	return a.Type == u && b.Type == aType || a.Type == c && b.Type == aType
}

// streamGapAfterUserMessage is true when the StreamingText tail should be separated from the
// message list by the same blank line as user↔assistant rows (last scroll message is user).
func streamGapAfterUserMessage(msgView []types.Message) bool {
	return len(msgView) > 0 && msgView[len(msgView)-1].Type == types.MessageTypeUser
}

func userMessageHasPromptText(msg types.Message) bool {
	if msg.Type != types.MessageTypeUser {
		return false
	}
	msg = messagerow.NormalizeMessageJSON(msg)
	if len(msg.Content) == 0 {
		return false
	}
	var blocks []types.MessageContentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return false
	}
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			return true
		}
	}
	return false
}

// userMessageRendersOnlyFoldedToolStubs is true when this user row would render only folded
// tool_result/advisor_tool_result stubs (ToolBodyOmitted), with no other visible segments.
// Matches actual [SegmentsFromMessageOpts] output so unknown content block types or API quirks
// do not leave a lone "↩ tool_result tool_use_id=…" line under assistant ⎿ summaries.
func (m *model) userMessageRendersOnlyFoldedToolStubs(msg types.Message) bool {
	if msg.Type != types.MessageTypeUser {
		return false
	}
	msg = messagerow.NormalizeMessageJSON(msg)
	if len(msg.Content) == 0 {
		return false
	}
	segs := messagerow.SegmentsFromMessageOpts(msg, m.messagerowOpts(msg))
	if len(segs) == 0 {
		return false
	}
	hasFoldedTool := false
	for _, s := range segs {
		switch s.Kind {
		case messagerow.SegTextMarkdown:
			if strings.TrimSpace(s.Text) != "" {
				return false
			}
		case messagerow.SegToolResult, messagerow.SegAdvisorToolResult:
			if !s.ToolBodyOmitted {
				return false
			}
			hasFoldedTool = true
		case messagerow.SegThinking:
			if strings.TrimSpace(s.Text) != "" {
				return false
			}
		default:
			return false
		}
	}
	return hasFoldedTool
}

// skipOmittableToolResultUserRow hides user messages that only render folded tool_result stubs
// (prompt always; transcript unless ctrl+e show-all or dump).
// The assistant tool_use row already shows the summary; omitting avoids duplicate ↩ tool_result lines.
func (m *model) skipFoldedToolResultStubInPrompt(msg types.Message) bool {
	if messagerow.VerboseToolOutputEnabled() {
		return false
	}
	if !m.userMessageRendersOnlyFoldedToolStubs(msg) {
		return false
	}
	if m.Screen.Mode == state.ScreenPrompt {
		return true
	}
	if m.Screen.Mode == state.ScreenTranscript && !m.Screen.ShowAll && !m.Screen.DumpMode {
		return true
	}
	return false
}

// userPromptPrefixStyled renders bright "> " for user rows (matches user message body emphasis).
func userPromptPrefixStyled(userMsgRowBg bool) string {
	st := lipgloss.NewStyle().Foreground(theme.UserMessageText()).Bold(true)
	if userMsgRowBg {
		st = st.Background(theme.UserMessageBackground())
	}
	return st.Render(UserPromptPointerGlyph() + " ")
}

// userInputViewWithPromptPrefix prepends the same dim "> " as user rows on the first line of the bottom input.
func userInputViewWithPromptPrefix(m *model) string {
	v := m.Input.PR.View()
	prefix := userPromptPrefixStyled(false)
	lines := strings.Split(v, "\n")
	if len(lines) == 0 {
		return prefix
	}
	lines[0] = prefix + lines[0]
	return strings.Join(lines, "\n")
}

// withUserPromptPointerIfNeeded prepends dim "> " before the first body line of user messages (same line as text).
func withUserPromptPointerIfNeeded(msg types.Message, body string) string {
	if msg.Type != types.MessageTypeUser || !userMessageHasPromptText(msg) || body == "" {
		return body
	}
	prefix := userPromptPrefixStyled(true)
	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		return prefix
	}
	lines[0] = prefix + lines[0]
	return strings.Join(lines, "\n")
}

// styleUserMessageLines applies a full-width gray background per row (ANSI-safe; lipgloss pads to cols).
func styleUserMessageLines(rows []string, cols int) string {
	st := lipgloss.NewStyle().Background(theme.UserMessageBackground()).Width(cols)
	var b strings.Builder
	for i, ln := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		// Check if the line already has ANSI escape sequences (already styled)
		// If it does, we need to be careful about applying additional styles
		if strings.Contains(ln, "\x1b[") {
			// Line already has ANSI codes, apply background without resetting
			// We'll wrap the existing styled text with background
			b.WriteString(st.Render(ln))
		} else {
			// Plain text, apply background
			b.WriteString(st.Render(ln))
		}
	}
	return b.String()
}

func withCollapsedSpaceIfNeeded(msg types.Message, body string) string {
	// Message pane uses [applyMessagePaneGutter] for uniform 2-space indent; no extra prefix here.
	if msg.Type != types.MessageTypeCollapsedReadSearch || body == "" {
		return body
	}

	return "  " + body
}

func (m *model) renderMessageRow(msg types.Message, cols, maxRows int, searchHL string) string {
	if m.skipFoldedToolResultStubInPrompt(msg) {
		return ""
	}
	segs := messagerow.SegmentsFromMessageOpts(msg, m.messagerowOpts(msg))
	var header string
	if msg.Type != types.MessageTypeAttachment {
		switch msg.Type {
		case types.MessageTypeUser:
			// No "user" title row: "> " on the first body line only (withUserPromptPointerIfNeeded).
		case types.MessageTypeAssistant:
			// No "assistant" title row — body starts directly (⏺/● lead still from formatMessageSegments).
		case types.MessageTypeCollapsedReadSearch, types.MessageTypeGroupedToolUse:
			// Same as assistant — no raw type label (TS compact collapsed row).
		default:
			header = lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(msg.Type)).Render(string(msg.Type))
		}
	}
	diaglog.Line("formatMessageSegments type %s, message %s", msg.Type, msg.Message)
	body := formatMessageSegments(segs, cols, m.showToolUseCtrlOExpandHint(), m.Conversation.ResolvedToolIDs, msg.Type == types.MessageTypeAssistant, searchHL, messagerow.CollectToolResultContentByToolUseID(m.Conversation.Store.Messages), true, msg.Type == types.MessageTypeUser)
	body = withUserPromptPointerIfNeeded(msg, body)
	body = withCollapsedSpaceIfNeeded(msg, body)
	block := body
	if header != "" {
		block = header + "\n" + body
	}
	wrapped := applyMessagePaneGutter(block, cols)
	rows := strings.Split(wrapped, "\n")
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	if msg.Type == types.MessageTypeUser {
		return "\n" + styleUserMessageLines(rows, cols) + "\n"
	}

	return strings.Join(rows, "\n")
}

func toolRowLeadPrefix(userRow bool) string {
	glyph := "\u25cf " // ● — TS figures.BLACK_CIRCLE non-darwin
	if runtime.GOOS == "darwin" {
		glyph = "\u23fa " // ⏺ — TS figures.BLACK_CIRCLE on darwin
	}
	return baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(glyph)
}

// prefixToolGlyphFirstLine prepends the dim tool lead (⏺ / ●) to the first line of rendered assistant text.
func prefixToolGlyphFirstLine(body string) string {
	if body == "" {
		return toolRowLeadPrefix(false)
	}
	p := toolRowLeadPrefix(false)
	i := strings.IndexByte(body, '\n')
	if i < 0 {
		return p + body
	}
	return p + body[:i] + body[i:]
}

func toolUseResolved(resolved map[string]struct{}, toolUseID string) bool {
	if resolved == nil || toolUseID == "" {
		return false
	}
	_, ok := resolved[toolUseID]
	return ok
}

// toolUseResolvedForDisplay treats a tool as resolved if it is in the resolved map, or (when detail is on)
// if tool_result payload exists for that id — avoids stale resolved maps skipping ⏺+⎿ stats.
func toolUseResolvedForDisplay(resolved map[string]struct{}, toolResultByID map[string]json.RawMessage, toolUseID string, allowResultPayloadAsResolved bool) bool {
	if toolUseID == "" {
		return false
	}
	if resolved != nil {
		if _, ok := resolved[toolUseID]; ok {
			return true
		}
	}
	if allowResultPayloadAsResolved && toolResultByID != nil {
		raw, ok := toolResultByID[toolUseID]
		if ok && len(raw) > 0 {
			return true
		}
	}
	return false
}

// toolUseSummaryLineResolvedForDisplay is true when every merged tool_use id in a SegToolUseSummaryLine has a result (or resolved map entry).
func toolUseSummaryLineResolvedForDisplay(resolved map[string]struct{}, toolResultByID map[string]json.RawMessage, seg messagerow.Segment, allowResultPayloadAsResolved bool) bool {
	ids := seg.ToolUseIDs
	if len(ids) == 0 {
		return toolUseResolvedForDisplay(resolved, toolResultByID, seg.ToolUseID, allowResultPayloadAsResolved)
	}
	for _, id := range ids {
		if !toolUseResolvedForDisplay(resolved, toolResultByID, id, allowResultPayloadAsResolved) {
			return false
		}
	}
	return true
}

// segmentJoinSeparator inserts an extra blank line after assistant prose before a merged Grep/Glob/Read summary line.
func segmentJoinSeparator(prev, cur messagerow.Segment) string {
	if prev.Kind == messagerow.SegTextMarkdown && strings.TrimSpace(prev.Text) != "" && cur.Kind == messagerow.SegToolUseSummaryLine {
		return "\n\n"
	}
	return "\n"
}

// transcriptAssistantPairBlankLine is true when the UI inserts one empty line between consecutive
// assistant rows in transcript (breathing room before the next ⏺ block).
func transcriptAssistantPairBlankLine(m *model, a, b types.Message) bool {
	if m == nil || m.Screen.Mode != state.ScreenTranscript {
		return false
	}
	return a.Type == types.MessageTypeAssistant && b.Type == types.MessageTypeAssistant
}

// priorNonEmptyAssistantText reports whether any earlier segment is non-empty assistant markdown.
// One ⏺/● marks the start of the assistant "paragraph"; tool title lines after that omit the lead glyph.
func priorNonEmptyAssistantText(segs []messagerow.Segment, idx int) bool {
	for j := 0; j < idx && j < len(segs); j++ {
		if segs[j].Kind == messagerow.SegTextMarkdown && strings.TrimSpace(segs[j].Text) != "" {
			return true
		}
	}
	return false
}

// baseMsgStyle adds the user-message row background so nested lipgloss.Render calls do not reset ANSI and punch holes in the gray bar.
func baseMsgStyle(userRow bool) lipgloss.Style {
	s := lipgloss.NewStyle()
	if userRow {
		s = s.Background(theme.UserMessageBackground())
	}
	return s
}

func logg(kind string, r string) {
	//return
	diaglog.Line("[goc/formatMessageSegments] seg kind=%s out=%s", kind, r)
}

// formatMessageSegments mirrors Message.tsx per-block branches (text→markdown, tool_use/tool_result/thinking).
// assistantLeadGlyph prefixes the first non-empty assistant text segment (TS-style ⏺ before the opening sentence).
// searchHL applies transcript search highlight to visible plain substrings (TS useSearchHighlight).
// showResolvedToolStats enables ⎿ TranscriptResolvedHintExtra for resolved Search/Read when tool_result JSON is available (prompt + transcript).
// userRow: when true, all lipgloss spans use the same row background as styleUserMessageLines (user-authored rows).
func formatMessageSegments(segs []messagerow.Segment, cols int, toolUseCtrlOHint bool, resolved map[string]struct{}, assistantLeadGlyph bool, searchHL string, toolResultByID map[string]json.RawMessage, showResolvedToolStats bool, userRow bool) string {
	hlSt := transcriptSearchHLStyle()
	withHL := func(s string) string {
		if strings.TrimSpace(searchHL) == "" {
			return s
		}
		return highlightSearchPlain(s, searchHL, hlSt)
	}
	var b strings.Builder
	var lastSegIdx int = -1
	assistantTextLeadDone := false
	for i, seg := range segs {
		var piece string
		switch seg.Kind {
		case messagerow.SegTextMarkdown:
			textForMd := seg.Text
			if strings.TrimSpace(searchHL) != "" {
				textForMd = highlightSearchPlain(seg.Text, searchHL, hlSt)
			}
			md := styleMarkdownTokens(markdown.CachedLexer(textForMd), cols, userRow)
			if assistantLeadGlyph && !assistantTextLeadDone && strings.TrimSpace(seg.Text) != "" {
				assistantTextLeadDone = true
				md = prefixToolGlyphFirstLine(md)
			}
			piece = md
			logg("SegTextMarkdown", piece)
		case messagerow.SegToolUse:
			if seg.ToolFacing != "" {
				row1 := ""
				if !priorNonEmptyAssistantText(segs, i) {
					row1 = toolRowLeadPrefix(userRow)
				}
				row1 += baseMsgStyle(userRow).Foreground(theme.ToolUseAccent()).Bold(true).Render(withHL(seg.ToolFacing))
				if p := strings.TrimSpace(seg.ToolParen); p != "" {
					row1 += " (" + withHL(p) + ")"
				}
				var toolLines []string
				toolLines = append(toolLines, row1)
				res := toolUseResolvedForDisplay(resolved, toolResultByID, seg.ToolUseID, showResolvedToolStats)
				if showResolvedToolStats && res {
					var raw json.RawMessage
					if toolResultByID != nil {
						raw = toolResultByID[seg.ToolUseID]
					}
					hint, extra := messagerow.TranscriptResolvedHintExtra(seg.ToolFacing, raw)
					if hint != "" {
						toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(hint))))
						if extra != "" {
							toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("     "+textutil.LinkifyOSC8(withHL(extra))))
						}
					}
				} else if !res {
					if act := strings.TrimSpace(seg.Text); act != "" {
						actLine := baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(withHL(act) + "…")
						if toolUseCtrlOHint {
							actLine += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
						}
						toolLines = append(toolLines, actLine)
					}
					if h := strings.TrimSpace(seg.ToolHint); h != "" {
						toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(h))))
					}
				}
				piece = strings.Join(toolLines, "\n")
			} else {
				line := baseMsgStyle(userRow).Foreground(theme.ToolUseAccent()).Bold(true).Render("⚙ " + withHL(seg.Text))
				if toolUseCtrlOHint {
					line += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
				}
				piece = line
			}
			logg("SegToolUse", piece)
		case messagerow.SegToolResult:
			piece = segdiff.FormatToolResultSegmentForTranscript(seg, userRow, toolUseCtrlOHint, cols, withHL, baseMsgStyle)
			logg("SegToolResult", piece)
		case messagerow.SegThinking:
			body := textutil.LinkifyOSC8(seg.Text)
			piece = baseMsgStyle(userRow).Bold(true).Render("● " + withHL(body))
			logg("SegThinking", piece)
		case messagerow.SegDisplayHint:
			piece = baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
			logg("SegDisplayHint", piece)
		case messagerow.SegServerToolUse:
			if seg.ToolFacing != "" {
				row1 := ""
				if !priorNonEmptyAssistantText(segs, i) {
					row1 = toolRowLeadPrefix(userRow)
				}
				row1 += baseMsgStyle(userRow).Foreground(theme.ServerAccent()).Bold(true).Render(withHL(seg.ToolFacing))
				if p := strings.TrimSpace(seg.ToolParen); p != "" {
					row1 += " (" + withHL(p) + ")"
				}
				var toolLines []string
				toolLines = append(toolLines, row1)
				res := toolUseResolvedForDisplay(resolved, toolResultByID, seg.ToolUseID, showResolvedToolStats)
				if showResolvedToolStats && res {
					var raw json.RawMessage
					if toolResultByID != nil {
						raw = toolResultByID[seg.ToolUseID]
					}
					hint, extra := messagerow.TranscriptResolvedHintExtra(seg.ToolFacing, raw)
					if hint != "" {
						toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(hint))))
						if extra != "" {
							toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("     "+textutil.LinkifyOSC8(withHL(extra))))
						}
					}
				} else if !res {
					if act := strings.TrimSpace(seg.Text); act != "" {
						actLine := baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(withHL(act) + "…")
						if toolUseCtrlOHint {
							actLine += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
						}
						toolLines = append(toolLines, actLine)
					}
					if h := strings.TrimSpace(seg.ToolHint); h != "" {
						toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(h))))
					}
				}
				piece = strings.Join(toolLines, "\n")
			} else {
				line := baseMsgStyle(userRow).Foreground(theme.ServerAccent()).Bold(true).Render("⎈ " + withHL(seg.Text))
				if toolUseCtrlOHint {
					line += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
				}
				piece = line
			}
			logg("SegServerToolUse", piece)
		case messagerow.SegAdvisorToolResult:
			st := baseMsgStyle(userRow).Foreground(theme.AdvisorAccent())
			if seg.IsToolError {
				st = baseMsgStyle(userRow).Foreground(theme.ToolError())
			}
			body := textutil.LinkifyOSC8(seg.Text)
			line := st.Render("✧ " + withHL(body))
			if seg.ToolBodyOmitted && toolUseCtrlOHint {
				line += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
			}
			piece = line
			logg("SegAdvisorToolResult", piece)
		case messagerow.SegGroupedToolUse:
			piece = baseMsgStyle(userRow).Foreground(theme.GroupedAccent()).Bold(true).Render("▦ " + withHL(seg.Text))
			logg("SegGroupedToolUse", piece)
		case messagerow.SegCollapsedReadSearch:
			piece = baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
			logg("SegCollapsedReadSearch", piece)
		case messagerow.SegToolUseSummaryLine:
			line := baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
			if !toolUseSummaryLineResolvedForDisplay(resolved, toolResultByID, seg, showResolvedToolStats) && toolUseCtrlOHint {
				line += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
			}
			piece = "  " + line
			logg("SegToolUseSummaryLine", piece)
		case messagerow.SegSkillListingAvailable:
			n := seg.Num
			if n < 1 {
				n = 1
			}
			word := "skills"
			if n == 1 {
				word = "skill"
			}
			piece = baseMsgStyle(userRow).Bold(true).Render(strconv.Itoa(n)) + baseMsgStyle(userRow).Render(" "+word+" available")
			logg("SegSkillListingAvailable", piece)
		default:
			piece = baseMsgStyle(userRow).Faint(true).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
			logg("default", piece)
		}
		if piece == "" {
			continue
		}
		if b.Len() > 0 && lastSegIdx >= 0 {
			b.WriteString(segmentJoinSeparator(segs[lastSegIdx], segs[i]))
		}
		b.WriteString(piece)
		lastSegIdx = i
	}
	return strings.TrimSpace(b.String())
}

// styleMarkdownInlineSegments renders paragraph/list_item runs with TS-style inline `code` color
// and strong/emphasis (terminal lipgloss).
func styleMarkdownInlineSegments(segs []markdown.InlineSegment, linePrefix string, userRow bool) string {
	if len(segs) == 0 {
		return ""
	}
	stCode := baseMsgStyle(userRow).Foreground(theme.MarkdownInlineCode())
	var stPlain, stBold, stItalic, stBoldItalic lipgloss.Style
	if userRow {
		ut := theme.UserMessageText()
		stPlain = baseMsgStyle(userRow).Foreground(ut).Bold(true)
		stBold = baseMsgStyle(userRow).Foreground(ut).Bold(true)
		stItalic = baseMsgStyle(userRow).Foreground(ut).Italic(true)
		stBoldItalic = baseMsgStyle(userRow).Foreground(ut).Bold(true).Italic(true)
	} else {
		stPlain = baseMsgStyle(userRow)
		stBold = baseMsgStyle(userRow).Bold(true)
		stItalic = baseMsgStyle(userRow).Italic(true)
		stBoldItalic = baseMsgStyle(userRow).Bold(true).Italic(true)
	}
	var b strings.Builder
	for i, seg := range segs {
		txt := seg.Text
		if i == 0 && linePrefix != "" {
			txt = linePrefix + txt
		}
		if seg.Code {
			b.WriteString(stCode.Render(txt))
			continue
		}
		var st lipgloss.Style
		switch {
		case seg.Bold && seg.Italic:
			st = stBoldItalic
		case seg.Bold:
			st = stBold
		case seg.Italic:
			st = stItalic
		default:
			st = stPlain
		}
		b.WriteString(st.Render(txt))
	}
	return b.String()
}

// headingMarkdownStyle is bold + heading color; level spacing is leading spaces only (no # in output).
func headingMarkdownStyle(userRow bool) lipgloss.Style {
	return baseMsgStyle(userRow).Bold(true).Foreground(theme.MarkdownHeading())
}

// styleMarkdownTokens applies lipgloss to block tokens (mirrors Markdown.tsx roles, terminal-only).
func styleMarkdownTokens(toks []markdown.Token, cols int, userRow bool) string {
	if len(toks) == 0 {
		return ""
	}
	var parts []string
	for _, t := range toks {
		switch t.Type {
		case "heading":
			lv := min(max(t.Level, 1), 6)
			levelPad := strings.Repeat(" ", (lv-1)*2)
			hst := headingMarkdownStyle(userRow)
			if len(t.Segments) > 0 {
				inner := styleMarkdownInlineSegments(t.Segments, "", userRow)
				rendered := hst.Render(inner)
				parts = append(parts, wrapHeadingForMessagePane(rendered, levelPad, cols))
			} else {
				plain := strings.TrimSpace(t.Text)
				wrapped := wrapHeadingForMessagePane(plain, levelPad, cols)
				lines := strings.Split(wrapped, "\n")
				var hb strings.Builder
				for i, ln := range lines {
					if i > 0 {
						hb.WriteByte('\n')
					}
					hb.WriteString(hst.Render(ln))
				}
				parts = append(parts, hb.String())
			}
		case "code":
			// Apply syntax highlighting if highlighter is available
			var highlightedCode string
			if markdownHighlighter != nil {
				highlighted, err := markdownHighlighter.HighlightCode(t.Text, t.Lang)
				if err == nil && highlighted != "" {
					highlightedCode = highlighted
				}
			}

			// If highlighting failed or highlighter is disabled, use plain code
			if highlightedCode == "" {
				cb := "```" + t.Lang + "\n" + t.Text
				if t.Text != "" && !strings.HasSuffix(t.Text, "\n") {
					cb += "\n"
				}
				cb += "```"
				parts = append(parts, baseMsgStyle(userRow).Faint(true).Render(cb))
			} else {
				// For highlighted code, just show the highlighted content without backticks
				parts = append(parts, baseMsgStyle(userRow).Render(highlightedCode))
			}
		case "list_item":
			indent := strings.Repeat(" ", t.ListIndent)
			var prefix string
			if t.ListContinuation {
				prefix = indent + "   "
			} else if t.ListOrdered && t.ListIndex > 0 {
				prefix = indent + fmt.Sprintf("%d. ", t.ListIndex)
			} else {
				prefix = indent + "- "
			}
			if len(t.Segments) > 0 {
				parts = append(parts, styleMarkdownInlineSegments(t.Segments, prefix, userRow))
			} else if userRow {
				parts = append(parts, baseMsgStyle(userRow).Foreground(theme.UserMessageText()).Bold(true).Render(prefix+t.Text))
			} else {
				parts = append(parts, baseMsgStyle(userRow).Render(prefix+t.Text))
			}
		case "blockquote":
			if len(t.Segments) > 0 {
				inner := styleMarkdownInlineSegments(t.Segments, "", userRow)
				pref := "> " + strings.ReplaceAll(inner, "\n", "\n> ")
				parts = append(parts, pref)
			} else if userRow {
				parts = append(parts, baseMsgStyle(userRow).Foreground(theme.UserMessageText()).Italic(true).Bold(true).Render("> "+strings.ReplaceAll(t.Text, "\n", "\n> ")))
			} else {
				parts = append(parts, baseMsgStyle(userRow).Italic(true).Render("> "+strings.ReplaceAll(t.Text, "\n", "\n> ")))
			}
		case "hr":
			parts = append(parts, baseMsgStyle(userRow).Faint(true).Render("---"))
		case "paragraph":
			if len(t.Segments) > 0 {
				parts = append(parts, styleMarkdownInlineSegments(t.Segments, "", userRow))
			} else {
				if userRow {
					parts = append(parts, baseMsgStyle(userRow).Foreground(theme.UserMessageText()).Bold(true).Render(t.Text))
				} else {
					parts = append(parts, t.Text)
				}
			}
		default:
			if userRow {
				parts = append(parts, baseMsgStyle(userRow).Foreground(theme.UserMessageText()).Bold(true).Render(t.Text))
			} else {
				parts = append(parts, t.Text)
			}
		}
	}
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			if toks[i-1].Type == "list_item" {
				b.WriteByte('\n')
			} else {
				b.WriteString("\n\n")
			}
		}
		b.WriteString(part)
	}
	return strings.TrimSpace(b.String())
}

// handleTraditionalScrollKey handles scroll keys when not using viewport (traditional virtual scrolling).
func (m *model) handleTraditionalScrollKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "down":
		m.Scroll.Sticky = false
		m.Scroll.Top += 1
		return nil
	case "up":
		m.Scroll.Sticky = false
		m.Scroll.Top = max(0, m.Scroll.Top-1)
		return nil
	case "pgdown", "space":
		m.Scroll.Sticky = false
		m.Scroll.Top += listViewportH(m) / 2
		return nil
	case "pgup", "b":
		m.Scroll.Sticky = false
		m.Scroll.Top = max(0, m.Scroll.Top-listViewportH(m)/2)
		return nil
	case "end", "G", "shift+g", "ctrl+end":
		m.Scroll.Sticky = true
		m.Scroll.Top = 1 << 30
		return nil
	case "home", "ctrl+home":
		m.Scroll.Sticky = false
		m.Scroll.Top = 0
		return nil
	case "ctrl+u":
		m.Scroll.Sticky = false
		m.Scroll.Top = max(0, m.Scroll.Top-listViewportH(m)/2)
		return nil
	case "ctrl+d":
		m.Scroll.Sticky = false
		m.Scroll.Top += listViewportH(m) / 2
		return nil
	case "ctrl+b":
		m.Scroll.Sticky = false
		m.Scroll.Top = max(0, m.Scroll.Top-listViewportH(m))
		return nil
	case "ctrl+f":
		m.Scroll.Sticky = false
		m.Scroll.Top += listViewportH(m)
		return nil
	case "ctrl+n":
		m.Scroll.Sticky = false
		m.Scroll.Top += 1
		return nil
	case "ctrl+p":
		m.Scroll.Sticky = false
		m.Scroll.Top = max(0, m.Scroll.Top-1)
		return nil
	}
	return nil
}
func (m *model) gouSubmitFromPromptText(fullPrompt, line string) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	gouDemoTracef("enter input=%q", previewForTrace(line, 120))
	cwd, _ := os.Getwd()
	toolProjectRoot := resolveToolProjectRoot(cwd)
	mergedLang, mergedOutName, mergedOutPrompt := gouDemoMergedSystemLocale()
	preExp := fullPrompt
	demoCfg := pui.DemoConfig{
		SessionID:           m.Conversation.Store.ConversationID,
		Language:            mergedLang,
		MCPCommandsJSONPath: m.Tool.MCPCommandsJSONPath,
		MCPToolsJSONPath:    m.Tool.MCPToolsJSONPath,
		PreExpansionInput:   &preExp,
		PermissionMode:      &m.Chrome.PermissionMode,
	}
	if m.Conversation.TSBridge != nil {
		demoCfg.TSContextBridge = m.Conversation.TSBridge
	}
	params, err := pui.BuildDemoParams(line, m.Conversation.Store, demoCfg)
	if err != nil {
		gouDemoTracef("BuildDemoParams error: %v", err)
		m.Conversation.Store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: build params: %v", err)))
		m.rebuildHeightCache()
		m.Scroll.Sticky = true
		m.Scroll.Top = 1 << 30
		return m, cmd
	}
	if params.RuntimeContext != nil {
		gouDemoLogToolUseContext(params.RuntimeContext)
	}
	params.ProcessSlashCommand = pui.NewSlashResolveProcessSlashCommand(pui.SlashResolveHandlerOptions{
		SessionID:        m.Conversation.Store.ConversationID,
		Store:            m.Conversation.Store,
		ReadFileState:    m.Conversation.ReadFileState,
		Cwd:              cwd,
		SessionMemState:  m.Memory.SessionMem,
		GuidancePtr:      &m.Memory.LastGuidance,
		UserContextPtr:   &m.Memory.LastUserCtx,
		SystemContextPtr: &m.Memory.LastSystemCtx,
	})
	gouDemoTracef("ProcessUserInput start priorMsgs=%d", len(m.Conversation.Store.Messages))
	r, err := processuserinput.ProcessUserInput(context.Background(), params)
	gouDemoTracef("ProcessUserInput end err=%v", err)
	if err != nil {
		m.Conversation.Store.AppendMessage(pui.SystemNotice(fmt.Sprintf("processUserInput: %v", err)))
		m.rebuildHeightCache()
		m.Scroll.Sticky = true
		m.Scroll.Top = 1 << 30
		return m, cmd
	}
	rStore := r
	if r != nil && strings.HasPrefix(line, "/") && r.Execution == nil && !r.ShouldQuery &&
		extractSlashLocalPanelText(r) != "" {
		rStore = slashResultForStoreOmittingPanelDupes(r)
	}
	out := pui.ApplyBaseResult(m.Conversation.Store, rStore, &m.Query.Handoff)
	gouDemoTracef("after ApplyBaseResult shouldQuery=%v effectiveShouldQuery=%v hadExecutionRequest=%v messagesAppended=%d",
		r != nil && r.ShouldQuery, out.EffectiveShouldQuery, out.HadExecutionRequest, len(rStore.Messages))
	if out.NextInput != "" {
		m.Input.PR.SetValue(out.NextInput)
		m.syncSlashListAfterPrompt()
	}
	m.applySlashResultPanelFromSubmit(line, r, out)
	m.rebuildHeightCache()
	m.Scroll.Sticky = true
	m.Scroll.Top = 1 << 30
	// Flush user (and any other new rows) before OnQueryYield appends streaming assistant/tool lines so JSONL follows conversation time order.
	m.maybeRecordTranscript()
	if out.EffectiveShouldQuery && !out.HadExecutionRequest {
		usedCCB := false
		var normToolsJSON json.RawMessage
		if params.RuntimeContext != nil {
			normToolsJSON = params.RuntimeContext.ToolUseContext.Options.Tools
		}
		var normToolDefs []struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(normToolsJSON, &normToolDefs)
		toolSpecs := make([]messagesapi.ToolSpec, 0, len(normToolDefs))
		for _, t := range normToolDefs {
			toolSpecs = append(toolSpecs, messagesapi.ToolSpec{Name: t.Name})
		}
		normOpts := messagesapi.OptionsFromEnv()
		if gouDemoEnvTruthy("GOU_DEMO_NON_INTERACTIVE") {
			normOpts.NonInteractive = true
		}
		tryMsgs := func() (json.RawMessage, error) {
			return ccbhydrate.MessagesJSONNormalized(m.Conversation.Store.Messages, toolSpecs, normOpts)
		}
		if m.Query.CCBInline && m.Query.CCBSend != nil {
			baseMsgs, err := tryMsgs()
			if err != nil {
				gouDemoTracef("gou-demo: ccbhydrate.MessagesJSON error: %v", err)
				m.Conversation.Store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: ccb messages JSON: %v", err)))
				m.rebuildHeightCache()
			} else if len(bytes.TrimSpace(baseMsgs)) < 3 || bytes.Equal(bytes.TrimSpace(baseMsgs), []byte("[]")) {
				gouDemoTracef("gou-demo: empty messages JSON bytes=%d", len(baseMsgs))
				m.Conversation.Store.AppendMessage(pui.SystemNotice("gou-demo: empty chat transcript (cannot call model)"))
				m.rebuildHeightCache()
			} else {
				var toolsJSON json.RawMessage
				if params.RuntimeContext != nil {
					toolsJSON = params.RuntimeContext.ToolUseContext.Options.Tools
				}
				var toolDefs []struct {
					Name string `json:"name"`
				}
				_ = json.Unmarshal(toolsJSON, &toolDefs)
				names := make([]string, 0, len(toolDefs))
				for _, t := range toolDefs {
					names = append(names, t.Name)
				}
				hasSkillTool := false
				skillNm := skilltools.SkillToolName()
				for _, t := range toolDefs {
					if t.Name == skillNm {
						hasSkillTool = true
						break
					}
				}
				skillListing := params.SkillListingCommands
				if len(skillListing) == 0 {
					skillListing = commands.SkillToolCommands(params.Commands)
				}
				discoverNm := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME"))
				mainLoopModel := gouDemoQueryMainLoopModel(params)
				m.Chrome.LastMainLoopModel = mainLoopModel
				gouOpts := commands.GouDemoSystemOpts{
					EnabledToolNames:       commands.EnabledToolNames(names),
					SkillToolCommands:      skillListing,
					ModelID:                mainLoopModel,
					Cwd:                    cwd,
					Language:               mergedLang,
					DiscoverSkillsToolName: discoverNm,
					NonInteractiveSession:  gouDemoEnvTruthy("GOU_DEMO_NON_INTERACTIVE"),
					OutputStyleName:        mergedOutName,
					OutputStylePrompt:      mergedOutPrompt,
				}
				commands.ApplyGouDemoRuntimeEnv(&gouOpts)
				var customSys, appendSys string
				if params.RuntimeContext != nil {
					if p := params.RuntimeContext.ToolUseContext.Options.CustomSystemPrompt; p != nil {
						customSys = strings.TrimSpace(*p)
					}
					if p := params.RuntimeContext.ToolUseContext.Options.AppendSystemPrompt; p != nil {
						appendSys = strings.TrimSpace(*p)
					}
				}
				extraRoots := querycontext.ExtraClaudeMdRootsForFetch(params.RuntimeContext)
				fetchOpts := querycontext.FetchOpts{
					CustomSystemPrompt:  customSys,
					Gou:                 gouOpts,
					ExtraClaudeMdRoots:  extraRoots,
					SessionStartSource:  "startup",
					HooksSessionID:      m.Conversation.Store.ConversationID,
					HooksTranscriptPath: "",
				}
				if m.Conversation.TSBridge != nil {
					fetchOpts.TSSnapshot = m.Conversation.TSBridge
				}
				partsRes, errParts := querycontext.FetchSystemPromptParts(context.Background(), fetchOpts)
				var guidance string
				var userCtxReminder string
				if errParts != nil {
					gouDemoTracef("FetchSystemPromptParts: %v (fallback BuildGouDemoSystemPrompt)", errParts)
					m.Conversation.Store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: system context: %v (using base prompt only)", errParts)))
					m.rebuildHeightCache()
					guidance = commands.BuildGouDemoSystemPrompt(gouOpts)
				} else {
					userCtxReminder = querycontext.FormatUserContextReminder(partsRes.UserContext)
					var base []string
					if customSys != "" {
						base = []string{customSys}
					} else {
						base = slices.Clone(partsRes.DefaultSystemPrompt)
					}
					if appendSys != "" {
						base = append(base, appendSys)
					}
					guidance = strings.Join(base, "\n\n")
				}
				m.Memory.LastGuidance = guidance

				listing := ""
				var listingMeta *ccbhydrate.SkillListingMeta
				if !gouDemoEnvTruthy("GOU_DEMO_SKIP_SKILL_LISTING") {
					listingSent := m.Input.SkillListingSent
					if gouDemoEnvTruthy("GOU_DEMO_SKILL_LISTING_EVERY_TURN") {
						listingSent = make(map[string]struct{})
					}
					if s, n, initial, ok := commands.AppendSkillListingForAPI(skillListing, hasSkillTool, listingSent, nil); ok {
						listing = s
						listingMeta = &ccbhydrate.SkillListingMeta{SkillCount: n, IsInitial: initial}
					}
				}
				msgsJSON, errL := ccbhydrate.MessagesJSONWithLeadingMeta(m.Conversation.Store.Messages, userCtxReminder, listing, listingMeta, toolSpecs, normOpts)
				if errL != nil {
					gouDemoTracef("gou-demo: MessagesJSONWithLeadingMeta error: %v", errL)
					m.Conversation.Store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: skill listing hydrate: %v", errL)))
					m.rebuildHeightCache()
				} else {
					// When dynamic tool loading is active, prepend <available-deferred-tools>
					// so the model knows which tools to discover via ToolSearch.
					msgsBefore := len(msgsJSON)
					if prep := toolsearchwire.PrepareMessagesForWire(msgsJSON, toolsJSON, mainLoopModel, false, false, m.Conversation.Store.Messages); len(prep) > 0 {
						msgsJSON = prep
					}
					// Persist announcement in store for delta tracking across turns.
					if len(msgsJSON) > msgsBefore {
						toolsearchwire.PersistDeferredAnnouncement(m.Conversation.Store, toolsJSON)
					}
					reqID := fmt.Sprintf("turn-%d", time.Now().UnixNano())
					m.Conversation.Store.ClearStreaming()
					m.Conversation.Store.ClearStreamingToolUses()
					// TS: skill_listing attachment is pushed to mutableMessages before callModel (QueryEngine attachment case).
					if strings.TrimSpace(listing) != "" {
						if att, ok := ccbhydrate.SkillListingStoreMessage(listing, listingMeta); ok {
							m.Conversation.Store.AppendMessage(att)
							m.rebuildHeightCache()
						}
					}
					gouDemoTracef("gou-demo model turn start requestID=%s msgsJSONBytes=%d toolsBytes=%d systemBytes=%d",
						reqID, len(msgsJSON), len(toolsJSON), len(guidance))
					cwdAbs, errAbs := filepath.Abs(cwd)
					if errAbs != nil {
						cwdAbs = cwd
					}
					runner := skilltools.ParityToolRunner{
						DemoToolRunner: skilltools.DemoToolRunner{
							Commands:  params.Commands,
							SessionID: m.Conversation.Store.ConversationID,
						},
						WorkDir:          cwdAbs,
						ProjectRoot:      toolProjectRoot,
						ReadFileState:    m.Conversation.ReadFileState,
						LocalBashDefault: true,
						AskAutoFirst:     !gouDemoEnvTruthy("GOU_DEMO_NO_ASK_AUTO_FIRST"),
						MainLoopModel:    mainLoopModel,
						Messages:         m.Conversation.Store.Messages,
						MessagesFunc:     func() []types.Message { return m.Conversation.Store.Messages },
						SystemPrompt:     []string{guidance},
						ProgressCallback: func(msg *types.Message) {
							if msg == nil {
								return
							}
							// Parse agent lifecycle events from progress messages
							if msg.Type == types.MessageTypeProgress && len(msg.Data) > 0 {
								var data struct {
									Type             string `json:"type"`
									AgentID          string `json:"agentId"`
									AgentType        string `json:"agentType"`
									Description      string `json:"description"`
									Name             string `json:"name"`
									IsBackground     bool   `json:"isBackground"`
									ParentToolUseID  string `json:"parentToolUseID"`
									TokenCount       int    `json:"tokenCount"`
									ToolUseCount     int    `json:"toolUseCount"`
									LastActivityDesc string `json:"lastActivityDesc"`
									Summary          string `json:"summary"`
									Status           string `json:"status"`
								}
								if json.Unmarshal(msg.Data, &data) == nil {
									switch data.Type {
									case "agent_registered":
										evictAfter := time.Now().Add(24 * time.Hour)
										task := &AgentTaskState{
											ID:              data.AgentID,
											AgentType:       data.AgentType,
											Description:     data.Description,
											Name:            data.Name,
											Status:          "running",
											StartTime:       time.Now(),
											IsBackground:    data.IsBackground,
											ParentToolUseID: data.ParentToolUseID,
											EvictAfter:      &evictAfter,
										}
										if send := m.Query.CCBSend; send != nil {
											send(AgentRegisteredMsg{Task: task})
										}
										// Still forward for message pane rendering
										if send := m.Query.CCBSend; send != nil {
											send(gouQueryYieldMsg{Message: *msg})
										}
										return
									case "agent_summary":
										if send := m.Query.CCBSend; send != nil {
											send(AgentProgressMsg{
												AgentID: data.AgentID,
												Progress: &AgentTaskProgress{
													Summary:          data.Summary,
													TokenCount:       data.TokenCount,
													ToolUseCount:     data.ToolUseCount,
													LastActivityDesc: data.LastActivityDesc,
												},
											})
										}
										return
									case "agent_completed":
										if send := m.Query.CCBSend; send != nil {
											send(AgentCompletedMsg{AgentID: data.AgentID, Status: "completed"})
										}
										return
									}
								}
							}
							// Default: forward as yield message for the message pane
							if send := m.Query.CCBSend; send != nil {
								send(gouQueryYieldMsg{Message: *msg})
							}
						},
						EditDeps: &localtools.EditDeps{
							OnNotebookEdit: func(absPath, oldString, newString string, replaceAll bool, roots []string, state *localtools.ReadFileState, userModified bool) (string, bool, error) {
								return tools.NotebookEditFromEdit(absPath, oldString, newString, replaceAll, roots)
							},
						},
					}
					if params.RuntimeContext != nil && params.RuntimeContext.ToolPermissionContext != nil {
						runner.ToolPermission = params.RuntimeContext.ToolPermissionContext
					}
					if gouDemoPreferQueryStreamingParity() {
						var userCtx map[string]string
						var systemCtx map[string]string
						if errParts == nil {
							userCtx = gouDemoUserContextMapForQuery(partsRes.UserContext)
							systemCtx = partsRes.SystemContext
						}
						m.Memory.LastUserCtx = userCtx
						m.Memory.LastSystemCtx = systemCtx
						tcx := types.ToolUseContext{}
						if params.RuntimeContext != nil {
							tcx = params.RuntimeContext.ToolUseContext
						}
						tcx.Options.MainLoopModel = mainLoopModel
						if m.Tool.ResultState != nil {
							tcx.ContentReplacementState = m.Tool.ResultState.ToJSON()
						}
						var trySMCompact compactservice.TrySessionMemoryCompactFn
						if m.Memory.SessionMem != nil {
							sessionID := m.Conversation.Store.ConversationID
							trySMCompact = func(ctx context.Context, messages []types.Message, agentID string, autoCompactThreshold *int) (*compactservice.CompactionResult, error) {
								return sessionmemory.TrySessionMemoryCompaction(
									ctx,
									m.Memory.SessionMem,
									sessionID,
									cwd,
									messages,
									"", // transcriptPath
									autoCompactThreshold,
									func(ctx context.Context, trigger string, model string) ([]types.Message, error) {
										runner := hookexec.SessionStartHookRunner(toolProjectRoot, cwd, sessionID, "")
										res, err := runner(ctx, compactservice.SessionStartHookTrigger(trigger),
											compactservice.SessionStartHookInput{Model: model})
										if err != nil {
											return nil, err
										}
										msgs := make([]types.Message, len(res))
										for i, r := range res {
											msgs[i] = r
										}
										return msgs, nil
									},
									agentID,
									mainLoopModel,
									nil, // planAttachmentProvider
								)
							}
						}
						qdeps := query.ProductionDeps(trySMCompact, func(phase string) {
					if send := m.Query.CCBSend; send != nil {
						send(compactPhaseMsg{Phase: phase})
					}
				})
						te := toolexecution.ExecutionDeps{
							InvokeTool:              runner.Run,
							MainLoopModel:           mainLoopModel,
							ReadToolRoots:           runner.ToolReadMappingRoots(),
							ReadToolMemCWD:          runner.ToolReadMappingMemCWD(),
							MultiMessageToolHandler: skilltools.NewSkillMultiMessageHandler(params.Commands, m.Conversation.Store.ConversationID, nil),
							QueryCanUseTool: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (toolexecution.PermissionDecision, error) {
								if toolName == "AskUserQuestion" {
									return toolexecution.AskDecision("Answer questions?"), nil
								}
								return toolexecution.AllowDecision(), nil
							},
						}
						// Opt-in TS permissions.ts 1b: whole-tool alwaysAsk on Bash skipped when input looks sandboxed (see toolexecution.BashSandboxRule1b).
						if gouDemoEnvTruthy("GOU_TOOLEXEC_BASH_SANDBOX_1B") {
							te.SandboxingEnabled = true
							te.AutoAllowBashWholeToolAskWhenSandboxed = true
						}
						// Tool result persistence: enabled by default (mirrors TS), set GOU_DEMO_TOOL_RESULT_PERSIST=0 to disable.
						// Large tool results are saved to disk and replaced with a preview in the tool_result block.
						if !gouDemoEnvFalsy("GOU_DEMO_TOOL_RESULT_PERSIST") {
							te.ToolResultPersistConfig = &toolexecution.ToolResultPersistConfig{
								SessionInfo: toolresultpersist.SessionInfo{
									SessionID: m.Conversation.Store.ConversationID,
									Cwd:       cwd,
								},
								ProcessOptions:          toolresultpersist.DefaultProcessOptions(),
								ContentReplacementState: m.Tool.ResultState,
							}
						}
						m.Modal.AskAutoFirst = !gouDemoEnvTruthy("GOU_DEMO_NO_ASK_AUTO_FIRST")
						m.installAskResolver(&te, m.Modal.AskAutoFirst)
						qdeps.ToolexecutionDeps = te
						// Wire tool result budget enforcement when persistence is enabled.
						// The closure captures the live Go *ContentReplacementState so mutations
						// survive across turns (mirrors TS shared ContentReplacementState instance).
						if m.Tool.ResultState != nil {
							statePtr := m.Tool.ResultState
							sessionInfo := te.ToolResultPersistConfig.SessionInfo
							qdeps.ApplyToolResultBudget = func(ctx context.Context, in *query.ToolResultBudgetInput) ([]types.Message, error) {
								return toolresultpersist.ApplyToolResultBudget(
									in.Messages,
									statePtr,
									sessionInfo,
									0,   // use default MaxToolResultsPerMessageChars
									nil, // skipToolNames
								), nil
							}
						}
						// Snapshot matches qp.Messages (TS QueryEngine messages at callModel): includes skill_listing row if appended above.
						msgsForQ := slices.Clone(m.Conversation.Store.Messages)
						// Prepend SessionStart hook_additional_context attachments (e.g. superpowers using-superpowers skill).
						// These are ephemeral — not persisted to store, only injected into the current query messages.
						if partsRes.SessionStartHookMessages != nil {
							msgsForQ = append(slices.Clone(partsRes.SessionStartHookMessages), msgsForQ...)
						}
						if send := m.Query.CCBSend; send != nil {
							qdeps.OnStreamingToolUses = func(ctx context.Context, uses []query.StreamingToolUseLive) error {
								send(gouStreamingToolUsesMsg{Uses: uses})
								return nil
							}
						}
						if m.Conversation.Transcript != nil {
							tr := m.Conversation.Transcript
							// Mirror TS recordTranscript(messages): each yield appends to the same turn prefix so
							// sessiontranscript dedup sees already-recorded user (and prior yields) before new rows.
							turnPrefix := slices.Clone(msgsForQ)
							qdeps.OnQueryYield = func(ctx context.Context, y query.QueryYield) error {
								if y.Message == nil {
									return nil
								}
								turnPrefix = append(turnPrefix, *y.Message)
								_, err := tr.RecordTranscript(ctx, turnPrefix, sessiontranscript.RecordOpts{AllMessages: turnPrefix})
								return err
							}
						}
						qdeps.OnQueryComplete = func(ctx context.Context, qcp query.QueryCompleteParams) {
							extractmemories.Execute(ctx, m.Memory.ExtractMem, extractmemories.ExtractionParams{
								Messages:       qcp.Messages,
								ToolUseContext: qcp.ToolUseContext,
								SystemPrompt:   qcp.SystemPrompt,
								UserContext:    qcp.UserContext,
								SystemContext:  qcp.SystemContext,
								Cwd:            qcp.Cwd,
								QuerySource:    qcp.QuerySource,
								NewUUID:        query.RandomUUID,
								SkipIndex:      growthbook.IsTenguMothCopse(),
								AppendSystemMessage: func(msg types.Message) {
									if send := m.Query.CCBSend; send != nil {
										send(gouMemoryAppendMsg{Msg: msg})
									}
								},
							})
							_, dreamErr := autodream.Execute(ctx, m.Memory.AutoDream,
								qcp.ToolUseContext, qcp.SystemPrompt,
								qcp.UserContext, qcp.SystemContext,
								qcp.QuerySource, query.RandomUUID,
								commands.ClaudeConfigHome(), qcp.Cwd,
								"", /* memoryDir — let Execute resolve */
								m.Conversation.Store.ConversationID,
							)
							if dreamErr != nil {
								gouDemoTracef("autodream: %v", dreamErr)
							}
							// Session memory extraction (TS sessionMemory post-turn hook).
							m.Memory.SessionHook(ctx, qcp)
						}
						qp := query.QueryParams{
							Messages:        msgsForQ,
							SystemPrompt:    query.AsSystemPrompt([]string{guidance}),
							UserContext:     userCtx,
							SystemContext:   systemCtx,
							ToolUseContext:  tcx,
							QuerySource:     params.QuerySource,
							StreamingParity: true,
							Deps:            &qdeps,
						}
						if params.RuntimeContext != nil && params.RuntimeContext.ToolPermissionContext != nil {
							pc := *params.RuntimeContext.ToolPermissionContext
							types.NormalizeToolPermissionContextData(&pc)
							qp.ToolPermissionContext = &pc
						}
						processuserinput.ApplyQueryHostEnvGates(&qp)
						processuserinput.WireToolexecutionFromProcessUserInput(&qp, params)
						gouDemoTracef("query streaming parity turn requestID=%s storeMsgs=%d toolsBytes=%d",
							reqID, len(m.Conversation.Store.Messages), len(toolsJSON))
						m.beginQuerySpinner()
						m.Query.Busy = true
						m.Conversation.Store.ClearStreamingToolUses()
						ctx, cancel := context.WithCancel(context.Background())
						m.Query.Cancel = cancel
						config.RunQueryStreamingParityTurn(ctx, func(msg tea.Msg) { m.Query.CCBSend(msg) }, qp)
						usedCCB = true
					} else {
						m.Conversation.Store.AppendMessage(pui.SystemNotice(
							"gou-demo: ccb-engine/localturn was removed. For a real model reply, set ANTHROPIC_API_KEY (or ANTHROPIC_AUTH_TOKEN) and GOU_QUERY_STREAMING_PARITY=1 or GOU_DEMO_STREAMING_TOOL_EXECUTION=1.",
						))
						m.rebuildHeightCache()
					}
				}
			}
		}
		if usedCCB {
			if cmd != nil {
				return m, tea.Batch(cmd, spinnerTickCmd())
			}
			return m, spinnerTickCmd()
		}
		if !m.Query.CCBInline {
			m.Conversation.Store.AppendMessage(pui.SystemNotice(
				"gou-demo: real HTTP / streaming parity is disabled (GOU_DEMO_CCB_INLINE=0). Unset it and set ANTHROPIC_API_KEY (or ANTHROPIC_AUTH_TOKEN) with GOU_QUERY_STREAMING_PARITY=1 or GOU_DEMO_STREAMING_TOOL_EXECUTION=1 for a model reply.",
			))
			m.rebuildHeightCache()
			m.Scroll.Sticky = true
			m.Scroll.Top = 1 << 30
		}
		if cmd != nil {
			return m, cmd
		}
		return m, nil
	}
	gouDemoTracef("no query path (effectiveShouldQuery=%v hadExecutionRequest=%v)", out.EffectiveShouldQuery, out.HadExecutionRequest)
	return m, cmd
}

// ---------------------------------------------------------------------------
// Backward-compat aliases; migrate callers to config. prefix over time.
// ---------------------------------------------------------------------------

// Type aliases for messages moved to config.
type gouQueryYieldMsg = config.QueryYieldMsg
type gouStreamEventMsg = config.StreamEventMsg
type gouStreamingToolUsesMsg = config.StreamingToolUsesMsg
type gouQueryDoneMsg = config.QueryDoneMsg
type gouSpinnerTickMsg = config.SpinnerTickMsg
type gouToolSummaryDelayTickMsg = config.ToolSummaryDelayTickMsg
type gouMemoryAppendMsg = config.MemoryAppendMsg
type compactPhaseMsg = config.CompactPhaseMsg

// Const alias (var, since Go does not support const aliases).
var teardropAsterisk = config.TeardropAsterisk

func gouDemoEnvTruthy(k string) bool                                         { return config.EnvTruthy(k) }
func gouDemoEnvFalsy(k string) bool                                          { return config.EnvFalsy(k) }
func gouDemoStatusLineEnabled() bool                                         { return config.StatusLineEnabled() }
func gouDemoTracef(f string, a ...any)                                       { config.Tracef(f, a...) }
func gouDemoLogToolUseContext(rc *types.ProcessUserInputContextData)          { config.LogToolUseContext(rc) }
func gouDemoWarnApilogExpectations(ccbInline bool)                           { config.WarnAPILogExpectations(ccbInline) }
func gouDemoPreferQueryStreamingParity() bool                                { return config.PreferQueryStreamingParity() }
func gouDemoQueryMainLoopModel(params *processuserinput.ProcessUserInputParams) string { return config.QueryMainLoopModel(params) }
func gouDemoUserContextMapForQuery(uc map[string]string) map[string]string   { return config.UserContextMapForQuery(uc) }
func resolveToolProjectRoot(cwd string) string                               { return config.ResolveToolProjectRoot(cwd) }
func gouDemoMergedSystemLocale() (lang, outputStyleName, outputStylePrompt string) { return config.MergedSystemLocale() }
func previewForTrace(s string, max int) string                               { return config.PreviewForTrace(s, max) }
func applyMessagePaneGutter(block string, cols int) string                   { return config.ApplyMessagePaneGutter(block, cols) }
func messagePaneGutterRowCount(block string, cols int) int                   { return config.MessagePaneGutterRowCount(block, cols) }
func wrapHeadingForMessagePane(content string, levelPad string, cols int) string { return config.WrapHeadingForMessagePane(content, levelPad, cols) }
func spinnerTickCmd() tea.Cmd                                                { return config.SpinnerTickCmd() }
