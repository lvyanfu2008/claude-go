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
	"os"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"goc/conversation-runtime/query"
	"goc/gou/conversation"
	"goc/gou/messagerow"
	"goc/gou/prompt"
	"goc/gou/pui"
	"goc/gou/suggestions"
	"goc/modelenv"
	"goc/services/autodream"
	"goc/services/extractmemories"
	"goc/services/sessionmemory"
	"goc/sessiontranscript"
	"goc/tools/localtools"
	"goc/tools/toolresultpersist"
	"goc/tscontext"
	"goc/types"
)

type model struct {
	store  *conversation.Store
	pr     prompt.Model
	width  int
	height int
	cols   int // terminal content width (title/footer); message list may use msgBodyCols when a scrollbar strip is shown

	// msgBodyCols is wrap width for virtual message rows (m.cols or m.cols-1). msgScrollbarW is 0 or 1.
	msgBodyCols   int
	msgScrollbarW int

	permModal         *permissionModalModel
	questionUI        *questionModel   // non-nil when interactive AskUserQuestion UI is active
	hooksConfigMenu   *hooksConfigMenu // non-nil when interactive hooks config menu is active
	askAutoFirst      bool             // cached from runner config, used by installAskResolver
	slashCommands     []types.Command
	slashCommandsOnce bool
	slashPicker       *slashPickerModel
	// slashResultPanel is local slash text output shown below the input until Esc (prompt screen only).
	slashResultPanel *string

	// @-mention autocomplete suggestions (see at_suggest.go)
	suggestionEngine *suggestions.SuggestionEngine
	suggestions      []suggestions.ScoredItem
	selectedSuggIdx  int
	suggVisible      bool

	scrollTop    int
	pendingDelta int
	sticky       bool
	heightCache  map[string]int

	// processUserInputBaseResultHandoff mirrors TS ProcessUserInputBaseResult non-messages fields after last Apply (shouldQuery, model, allowedTools, effort, resultText, nextInput, submitNextInput).
	processUserInputBaseResultHandoff pui.ProcessUserInputBaseResultHandoff

	// layout
	titleH  int
	streamH int // reserved lines for streaming strip inside message pane

	// ccbSend / ccbInline set by BindCCB after tea.NewProgram (real model path when ccbInline and streaming parity gates + key).
	ccbSend   func(tea.Msg)
	ccbInline bool

	// skillListingSent tracks skill names already injected into the API transcript (TS sentSkillNames).
	skillListingSent map[string]struct{}

	// mcpCommandsJSONPath is -mcp-commands-json (overrides GOU_DEMO_MCP_COMMANDS_JSON when set).
	mcpCommandsJSONPath string
	// mcpToolsJSONPath is -mcp-tools-json (overrides GOU_DEMO_MCP_TOOLS_JSON when set).
	mcpToolsJSONPath string

	// readFileState is the session-scoped read file state, used by /files and tool execution.
	readFileState *localtools.ReadFileState

	// tsBridge when non-nil supplies in-process snapshot for commands/tools/prompt parts (tests; former TS bridge removed).
	tsBridge *tscontext.Snapshot

	// transcript appends messages after each completed turn (session JSONL).
	transcript *sessiontranscript.Store

	// REPL chrome (terminal title, permission pill): see repl_chrome.go.
	permissionMode        types.PermissionMode
	queryBusy             bool
	queryBusyStartedAt    time.Time
	lastActivity          time.Time
	spinnerVerb           string
	spinnerFrame          int
	spinnerTokens         int
	preCompactVerb        string // saved spinner verb before compact, restored on done
	lastEmittedTitlePlain string

	// Ctrl+C interrupt support (TS app:interrupt → CancelRequestHandler + useExitOnCtrlCD).
	queryCancel  context.CancelFunc
	lastCtrlC    time.Time
	ctrlCPending bool

	// Transcript screen (TS REPL.tsx Screen prompt|transcript + frozenTranscriptState).
	uiScreen           gouDemoScreen
	transcriptFrozen   *frozenTranscriptSnapshot // nil in prompt; set on enterTranscriptScreen
	transcriptShowAll  bool
	transcriptDumpMode bool // [ : dump-to-scrollback + uncapped show-all (TS dumpMode)
	// suspendAltScreenForScrollbackDump exits the alternate buffer so bracket-dump (tea.Printf) hits host scrollback (Bubble Tea v2: no tea.ExitAltScreen).
	suspendAltScreenForScrollbackDump bool
	promptSavedScrollTop              int
	promptSavedSticky                 bool

	transcriptEditorBusy   bool
	transcriptEditorStatus string
	transcriptEditorGen    int

	transcriptSearchOpen   bool
	transcriptSearchQuery  string
	transcriptSearchHits   []int
	transcriptSearchCursor int

	// Message-list mouse scroll (see mouse_message_list.go; tea.WithMouseCellMotion).
	msgListMouseDragging bool
	msgListMouseLastY    int

	// Bubbles/viewport message pane (default on, prompt only); see message_viewport_pane.go.
	useMsgViewport      bool
	msgViewport         viewport.Model
	lastVpGeom          string
	lastVpContentSig    string
	vpNeedResizeContent bool
	msgFoldAll          bool
	msgFoldRev          int
	msgViewportFallback bool
	// msgHistoryBrowseMouseOff mirrors go-tui/main/test.go: at viewport top, wheel-up disables SGR mouse so the
	// terminal scrollback wheel works; any key runs EnableMouseCellMotion + ClearScreen (see Update).
	msgHistoryBrowseMouseOff bool

	// TS lookups.resolvedToolUseIDs + StatusLine mainLoopModel
	resolvedToolIDs     map[string]struct{}
	groupedAgentLookups *messagerow.GroupedAgentLookups
	lastMainLoopModel   string

	// rebuildHeightCacheCalls increments in rebuildHeightCache (tests: streaming skip policy).
	rebuildHeightCacheCalls int

	// msgFirstShownAt records when each message UUID first appeared (for GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS).
	msgFirstShownAt map[string]time.Time
	// msgLastAssistantContentLen tracks len(Content) per assistant UUID so streaming bumps reset the summary delay window.
	msgLastAssistantContentLen map[string]int

	// manual rendering mode (buffer events until flushed)
	manualRenderMode bool
	pendingEvents    []tea.Msg

	// New message rendering system integration
	msgRenderer *MessageRendererIntegration

	// autoDreamState tracks auto-dream scan throttle across turns.
	autoDreamState *autodream.State

	// extractMemState tracks post-turn extract-memories throttling and cursor (TS extractMemories).
	extractMemState *extractmemories.State

	// sessionMemState tracks post-turn session memory extraction (TS sessionMemory).
	sessionMemState *sessionmemory.State
	// sessionMemHook is the per-turn hook callback (mirrors TS initSessionMemory hook).
	sessionMemHook func(ctx context.Context, params query.QueryCompleteParams)
	// lastGuidance is the most recent system prompt guidance text, set after query building.
	lastGuidance string
	// lastUserCtx is the most recent user context map, set after query building.
	lastUserCtx map[string]string
	// lastSystemCtx is the most recent system context map, set after query building.
	lastSystemCtx map[string]string

	// Task list (mirrors TS TaskListV2)
	taskList *taskListModel

	// agentTasks tracks sub-agent tasks for the coordinator panel (TS LocalAgentTaskState registry).
	agentTasks *agentTaskStore

	// lastQueryParams stores the last query params for auto-submit of bg agent notifications.
	lastQueryParams *query.QueryParams

	// toolResultState tracks tool-result persistence decisions across turns.
	// Shared between the write path (per-tool persist) and read path (per-message budget enforcement).
	toolResultState *toolresultpersist.ContentReplacementState
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
		store:               st,
		pr:                  pr,
		sticky:              true,
		heightCache:         make(map[string]int),
		skillListingSent:    make(map[string]struct{}),
		resolvedToolIDs:     make(map[string]struct{}),
		lastMainLoopModel:   lm,
		titleH:              1,
		streamH:             4,
		mcpCommandsJSONPath: mcpCommandsJSONPath,
		mcpToolsJSONPath:    mcpToolsJSONPath,
		tsBridge:            tsBridge,
		transcript:          tr,
		readFileState:       localtools.NewReadFileState(),
		permissionMode:      gouDemoPermissionModeFromEnv(),
		useMsgViewport:      gouDemoBubblesViewport(),
		autoDreamState:      autodream.NewState(),
		extractMemState:     extractmemories.NewState(),
		sessionMemState:     sessionMemState,
		sessionMemHook:      sessionmemory.Hook(sessionMemState, st.ConversationID, cwd),
		suggestionEngine:    suggEngine,
		taskList:            newTaskListModel(st.ConversationID),
		agentTasks:          newAgentTaskStore(),
		slashPicker:         newSlashPickerModel(),
		permModal:           newPermissionModalModel(),
		toolResultState:     toolResultState,
	}
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
	cmds = append(cmds, taskListTickCmd(m.taskList))

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}
