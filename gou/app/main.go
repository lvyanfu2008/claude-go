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
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-isatty"

	"goc/ccb-engine/apilog"
	"goc/ccb-engine/debugpath"
	"goc/ccb-engine/settingsfile"
	"goc/claudeinit"
	"goc/commands"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	"goc/gou/ccbstream"
	"goc/gou/conversation"
	"goc/gou/ink/integration"
	"goc/gou/layout"
	"goc/gou/markdown"
	"goc/gou/prompt"
	"goc/gou/textutil"
	"goc/gou/theme"
	"goc/gou/transcript"
	"goc/modelenv"
	"goc/sessiontranscript"
	"goc/types"
)

// gouDemoTrace is set by setupGouDemoTrace from GOU_DEMO_LOG_FILE or GOU_DEMO_LOG.
var gouDemoTrace *log.Logger

// markdownHighlighter is the global code highlighter instance
var markdownHighlighter *markdown.Highlighter

// messagePaneGutterCols is the uniform left indent for message pane body lines (alignment with wrap width).
const messagePaneGutterCols = 0

func messageWrapCols(cols int) int {
	if cols <= messagePaneGutterCols+8 {
		return max(8, cols)
	}
	return cols - messagePaneGutterCols
}

// applyMessagePaneGutter wraps block to (cols − gutter) and prefixes each line with two spaces.
func applyMessagePaneGutter(block string, cols int) string {
	if block == "" {
		return ""
	}
	if messagePaneGutterCols == 0 {
		return layout.WrapForViewport(block, cols)
	}
	wrapCols := messageWrapCols(cols)
	wrapped := layout.WrapForViewport(block, wrapCols)
	prefix := strings.Repeat(" ", messagePaneGutterCols)
	lines := strings.Split(wrapped, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

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
			lines[i] = textutil.AssistantBullet() + line
		} else {
			lines[i] = "  " + line
		}
	}
	return strings.Join(lines, "\n")
}

// messagePaneGutterRowCount matches [applyMessagePaneGutter] line count for height cache parity.
func messagePaneGutterRowCount(block string, cols int) int {
	g := applyMessagePaneGutter(block, cols)
	if g == "" {
		return 1
	}
	return max(1, strings.Count(g, "\n")+1)
}

// wrapHeadingForMessagePane wraps heading content to (messageWrapCols − levelPad) so after [applyMessagePaneGutter]
// each physical line still includes the ATX level indent on continuations (not only the global two spaces).
func wrapHeadingForMessagePane(content string, levelPad string, cols int) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	innerW := messageWrapCols(cols) - len(levelPad)
	if innerW < 8 {
		innerW = max(8, messageWrapCols(cols)-2)
	}
	wrapped := layout.WrapForViewport(content, innerW)
	if levelPad == "" {
		return wrapped
	}
	lines := strings.Split(wrapped, "\n")
	for i := range lines {
		lines[i] = levelPad + lines[i]
	}
	return strings.Join(lines, "\n")
}

// gouDemoMergedSystemLocale mirrors apiparity.GouDemo: user + project settings.go.json / settings.local.json language/outputStyle with env override.
// resolveToolProjectRoot returns CCB_ENGINE_PROJECT_ROOT if set, else the nearest Go project marker from cwd, else abs(cwd).
func resolveToolProjectRoot(cwd string) string {
	if r := strings.TrimSpace(os.Getenv("CCB_ENGINE_PROJECT_ROOT")); r != "" {
		if a, err := filepath.Abs(r); err == nil {
			return a
		}
	}
	if pr, err := settingsfile.FindClaudeProjectRoot(cwd); err == nil {
		return pr
	}
	if a, err := filepath.Abs(cwd); err == nil {
		return a
	}
	return cwd
}

func gouDemoMergedSystemLocale() (lang, outputStyleName, outputStylePrompt string) {
	projRoot := settingsfile.ProjectRootLastResolved()
	locLang, locStyleKey, err := settingsfile.MergeGouDemoLocalePrefs(projRoot, true)
	if err != nil {
		gouDemoTracef("MergeGouDemoLocalePrefs: %v", err)
		locLang, locStyleKey = "", ""
	}
	lang = strings.TrimSpace(os.Getenv("CLAUDE_CODE_LANGUAGE"))
	if lang == "" {
		lang = locLang
	}
	on, op := commands.ResolveGouDemoOutputStyle(
		os.Getenv("CLAUDE_CODE_OUTPUT_STYLE_NAME"),
		os.Getenv("CLAUDE_CODE_OUTPUT_STYLE_PROMPT"),
		locStyleKey,
	)
	return lang, on, op
}

func defaultGouDemoTracePath() string {
	p := debugpath.ResolveLogPath()
	if p != "" {
		return p
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("gou-demo-trace-%d.txt", os.Getpid()))
}

func setupGouDemoTrace() (cleanup func()) {
	path := strings.TrimSpace(os.Getenv("GOU_DEMO_LOG_FILE"))
	flags := log.LstdFlags | log.Lmicroseconds
	if path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			log.Printf("gou-demo: mkdir %q: %v", filepath.Dir(path), err)
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Printf("gou-demo: GOU_DEMO_LOG_FILE %q: %v", path, err)
			return func() {}
		}
		debugpath.MaybeUpdateLatestSymlink(path)
		gouDemoTrace = log.New(f, "[gou-demo] ", flags)
		return func() { _ = f.Close() }
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_LOG")))
	if v != "1" && v != "true" && v != "yes" && v != "on" {
		return func() {}
	}
	// GOU_DEMO_LOG=1: writing to stderr while the TUI runs may corrupt line order and layout.
	if gouDemoEnvTruthy("GOU_DEMO_LOG_STDERR") {
		gouDemoTrace = log.New(os.Stderr, "[gou-demo] ", flags)
		return func() {}
	}
	if isatty.IsTerminal(os.Stderr.Fd()) {
		p := defaultGouDemoTracePath()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "[gou-demo] trace mkdir %q: %v; falling back to stderr\n", filepath.Dir(p), err)
			gouDemoTrace = log.New(os.Stderr, "[gou-demo] ", flags)
			return func() {}
		}
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gou-demo] trace open %q: %v; falling back to stderr\n", p, err)
			gouDemoTrace = log.New(os.Stderr, "[gou-demo] ", flags)
			return func() {}
		}
		debugpath.MaybeUpdateLatestSymlink(p)
		gouDemoTrace = log.New(f, "[gou-demo] ", flags)
		lp := debugpath.LatestLinkPathFor(p)
		if lp != "" {
			gouDemoTrace.Printf("trace -> %s points to %s (TTY: stderr+TUI garbles; or GOU_DEMO_LOG_FILE=...)", lp, p)
		} else {
			gouDemoTrace.Printf("trace -> %s (TTY: stderr+TUI garbles output; use this file or GOU_DEMO_LOG_FILE=...)", p)
		}
		return func() { _ = f.Close() }
	}
	gouDemoTrace = log.New(os.Stderr, "[gou-demo] ", flags)
	return func() {}
}

func gouDemoTracef(format string, args ...any) {
	if gouDemoTrace != nil {
		gouDemoTrace.Printf(format, args...)
	}
}

// gouDemoLogToolUseContext dumps ProcessUserInputContext / ToolUseContext JSON when CLAUDE_CODE_LOG_TOOL_USE_CONTEXT
// or GOU_DEMO_LOG_TOOL_USE_CONTEXT is set (requires GOU_DEMO_LOG=1 or GOU_DEMO_LOG_FILE so [gouDemoTrace] is configured — stderr+TUI is avoided by default).
// Values: 1|true|summary — summary snapshot; full — entire serializable context (large). JSON is one-line (no indent).
func gouDemoLogToolUseContext(rc *types.ProcessUserInputContextData) {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("CLAUDE_CODE_LOG_TOOL_USE_CONTEXT")))
	if v == "" {
		v = strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_LOG_TOOL_USE_CONTEXT")))
	}
	if v == "" {
		return
	}
	full := v == "full"
	if !full && v != "1" && v != "true" && v != "yes" && v != "on" && v != "summary" {
		return
	}
	if gouDemoTrace == nil {
		return
	}
	b, err := types.FormatProcessInputContextForLog(rc, full)
	if err != nil {
		gouDemoTracef("ToolUseContext log: marshal: %v", err)
		return
	}
	mode := "summary"
	if full {
		mode = "full"
	}
	gouDemoTrace.Printf("ToolUseContext (%s JSON):\n%s\n", mode, string(b))
}

func gouDemoEnvTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func gouDemoEnvFalsy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "0" || v == "false" || v == "no" || v == "off"
}

func gouDemoStatusLineEnabled() bool {
	return gouDemoEnvTruthy("GOU_DEMO_STATUS_LINE")
}

func gouDemoEnvWantsApiBodyLog() bool {
	return gouDemoEnvTruthy("CLAUDE_CODE_LOG_API_REQUEST_BODY") || gouDemoEnvTruthy("CLAUDE_CODE_LOG_API_RESPONSE_BODY")
}

func gouDemoHasLLMKeys() bool {
	for _, k := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "OPENAI_API_KEY"} {
		if strings.TrimSpace(os.Getenv(k)) != "" {
			return true
		}
	}
	return false
}

// gouDemoWarnApilogExpectations prints stderr hints when CLAUDE_CODE_LOG_API_* cannot produce HTTP body logs.
func gouDemoWarnApilogExpectations(ccbInline bool) {
	if !gouDemoEnvWantsApiBodyLog() {
		return
	}
	if !ccbInline {
		fmt.Fprintf(os.Stderr,
			"[gou-demo] CLAUDE_CODE_LOG_API_* is set, but this run has real HTTP / streaming parity disabled (GOU_DEMO_CCB_INLINE=0).\n"+
				"           No HTTP → apilog will not append request/response lines. Unset GOU_DEMO_CCB_INLINE and set ANTHROPIC_API_KEY plus GOU_QUERY_STREAMING_PARITY=1 or GOU_DEMO_STREAMING_TOOL_EXECUTION=1 for real API logs.\n")
		return
	}
	if !gouDemoHasLLMKeys() {
		fmt.Fprintf(os.Stderr,
			"[gou-demo] CLAUDE_CODE_LOG_API_* is set, but no ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, or OPENAI_API_KEY is set.\n"+
				"           Put keys in ~/.harness/settings.go.json or project .harness/settings.go.json env, or export them.\n")
	}
}

func previewForTrace(s string, max int) string {
	if max <= 0 {
		max = 120
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + fmt.Sprintf("…(%d runes)", len(r))
}

// gouQueryYieldMsg carries one assistant or user row from [query.Query] streaming parity (non-ccbstream protocol).
type gouQueryYieldMsg struct {
	Message types.Message
}

// gouStreamEventMsg carries a raw SSE stream event (content_block_delta) for incremental streaming text display.
// Mirrors TS stream_event yields in handleMessageFromStream → onStreamingText.
type gouStreamEventMsg struct {
	Raw json.RawMessage
}

// gouStreamingToolUsesMsg carries in-flight tool_use snapshots from [query.QueryDeps.OnStreamingToolUses].
// Uses==nil clears the store (Anthropic message_stop); non-nil replaces the live list (may be empty).
type gouStreamingToolUsesMsg struct {
	Uses []query.StreamingToolUseLive
}

// gouQueryDoneMsg marks completion of a query streaming parity turn (Err set on failure).
type gouQueryDoneMsg struct {
	Err error
}

// gouMemoryAppendMsg appends a system message on the main thread (e.g. subtype memory_saved from extract-memories).
type gouMemoryAppendMsg struct {
	Msg types.Message
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

// commandQueueNotifyMsg is sent when a background agent notification is enqueued.
// Triggers auto-submit when TUI is idle.
type commandQueueNotifyMsg struct{}

// compactPhaseMsg carries auto-compact phase updates for spinner verb changes.
// Phase values: "started", "summarizing", "done".
type compactPhaseMsg struct {
	Phase string
}

func gouDemoAnthropicAPIKey() string {
	k := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
	if k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN"))
}

// gouDemoPreferQueryStreamingParity is true when env gates parity and an Anthropic key is present (HTTP path usable).
func gouDemoPreferQueryStreamingParity() bool {
	if gouDemoAnthropicAPIKey() == "" {
		return false
	}
	cfg := query.BuildQueryConfig()
	return query.StreamingParityPathEnabled(cfg)
}

// gouDemoQueryMainLoopModel is the model id for HTTP streaming parity + ParityToolRunner.
// /model sets CLAUDE_CODE_MODEL in-process; that must override ToolUseContext.Options from
// [pui.BuildDemoParams] when they disagree (otherwise the API keeps an older id).
func gouDemoQueryMainLoopModel(params *processuserinput.ProcessUserInputParams) string {
	if cm := strings.TrimSpace(os.Getenv("CLAUDE_CODE_MODEL")); cm != "" {
		return cm
	}
	if params != nil && params.RuntimeContext != nil {
		if m := strings.TrimSpace(params.RuntimeContext.ToolUseContext.Options.MainLoopModel); m != "" {
			return m
		}
	}
	return modelenv.EffectiveMainLoopModel()
}

// gouDemoUserContextMapForQuery copies live user context for [query.PrependUserContext].
// Values must be raw (no <system-reminder> wrapper): TS prependUserContext wraps once per #key/value.
// Do not pass [querycontext.FormatUserContextReminder] here — that string is already wrapped for ccbhydrate lead-in only.
func gouDemoUserContextMapForQuery(uc map[string]string) map[string]string {
	if len(uc) == 0 {
		return nil
	}
	out := make(map[string]string)
	for k, v := range uc {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// runQueryStreamingParityTurn runs [query.Query] in a goroutine and forwards whole messages to the Bubble Tea program.
func runQueryStreamingParityTurn(ctx context.Context, programSend func(tea.Msg), qp query.QueryParams) {
	go func() {
		for y, err := range query.Query(ctx, qp) {
			if err != nil {
				if programSend != nil {
					programSend(gouQueryDoneMsg{Err: err})
				}
				return
			}
			if y.StreamEvent != nil && programSend != nil {
				programSend(gouStreamEventMsg{Raw: y.StreamEvent})
			}
			if y.Message != nil && programSend != nil {
				programSend(gouQueryYieldMsg{Message: *y.Message})
			}
			if y.Terminal != nil {
				// Query encodes model/stream failures on Terminal.Error (second iter return is always nil err).
				var doneErr error
				if y.Terminal.Error != nil {
					doneErr = y.Terminal.Error
				}
				if programSend != nil {
					programSend(gouQueryDoneMsg{Err: doneErr})
				}
				return
			}
		}
	}()
}

// teardropAsterisk matches TS constants/figures.ts TEARDROP_ASTERISK (Spinner.tsx).
const teardropAsterisk = "\u273b"

func spinnerTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return gouSpinnerTickMsg{} })
}

func (m *Model) beginQuerySpinner() {
	m.queryBusyStartedAt = time.Now()
	m.spinnerVerb = pickSpinnerVerb()
	m.spinnerFrame = 0
	m.spinnerTokens = 0
	if m.agentTasks != nil {
		m.agentTasks.RegisterMainSession()
	}
}

func (m *Model) endQuerySpinner() {
	m.spinnerVerb = ""
	m.queryBusyStartedAt = time.Time{}
	m.spinnerFrame = 0
	m.spinnerTokens = 0
	if m.agentTasks != nil {
		m.agentTasks.CompleteMainSession()
	}
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

func (m *Model) promptBottomStreamRows() []string {
	h := m.streamH
	if h < 1 {
		h = 1
	}
	return padStreamRows(nil, h)
}

// Config is the runtime configuration for the TUI app.
type Config struct {
	// SessionID is the conversation session ID. Auto-generated when empty.
	SessionID string
	// PermissionMode sets the tool permission mode for the session.
	PermissionMode types.PermissionMode
	// CWD is the working directory. Defaults to os.Getwd() when empty.
	CWD string
	// TranscriptPath is an optional JSON file to load initial messages from.
	TranscriptPath string
	// ReplayCCPath is an optional NDJSON stream file to replay before starting the TUI.
	ReplayCCPath string
	// StreamStdin feeds NDJSON stream events from stdin before opening the TUI.
	StreamStdin bool
	// MCPCommandsJSONPath overrides the path for MCP command definitions.
	MCPCommandsJSONPath string
	// MCPToolsJSONPath overrides the path for MCP tool definitions.
	MCPToolsJSONPath string
}

// Run initializes and runs the TUI application. It blocks until the user exits.
// Mirrors the combined behavior of TS cli.tsx + main.tsx REPL launch.
func Run(config_ Config) error {
	if integration.RunNewEngine() {
		return nil
	}
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
		st.Messages = sessiontranscript.ApplySnipRemovals(msgs)
	}
	if config_.ReplayCCPath != "" {
		if err := ccbstream.ReplayFile(config_.ReplayCCPath, st); err != nil {
			return fmt.Errorf("replay-cc: %w", err)
		}
	}

	mcpCmdPath := strings.TrimSpace(config_.MCPCommandsJSONPath)
	mcpToolPath := strings.TrimSpace(config_.MCPToolsJSONPath)
	m := NewModel(st, mcpCmdPath, mcpToolPath, nil)
	m.taskList.setAgentTasks(m.agentTasks)

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
	if m, ok := res.(*Model); ok && gouDemoAltScreenEnabled() && (gouDemoEnvTruthy("GOU_DEMO_DUMP_ON_EXIT") || m.transcriptDumpMode) {
		fmt.Print(transcriptExportPlain(m, exportTranscriptWidth(m)) + "\n")
	}
	return runErr
}

func (m *Model) maybeRecordTranscript() {
	if m.transcript == nil {
		return
	}
	msgs := slices.Clone(m.store.Messages)
	_, err := m.transcript.RecordTranscript(context.Background(), msgs, sessiontranscript.RecordOpts{AllMessages: msgs})
	if err != nil && gouDemoTrace != nil {
		gouDemoTracef("RecordTranscript: %v", err)
	}
}

// BindCCB wires Bubble Tea Send and whether real HTTP streaming parity is allowed.
func (m *Model) BindCCB(send func(tea.Msg), inline bool) {
	m.ccbSend = send
	m.ccbInline = inline
}

// teaGlobalRedrawCmd mirrors TS useGlobalKeybindings app:redraw (ctrl+l): clear the terminal
// so the next frame repaints fully (e.g. after the host cleared scrollback with Cmd+K).
func teaGlobalRedrawCmd() tea.Cmd {
	return func() tea.Msg { return tea.ClearScreen() }
}

func (m *Model) inputAreaHeight() int {
	h := m.pr.LineCount()
	if m.suggVisible && len(m.suggestions) > 0 {
		visibleRows := min(6, len(m.suggestions))
		h += 1 + visibleRows // title line + suggestion rows
	}
	if m.uiScreen != gouDemoScreenTranscript {
		h++ // horizontal rule above input
	}
	if m.uiScreen != gouDemoScreenTranscript && !gouDemoBuiltinStatusLineDisabled() {
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
func (m *Model) bottomChromeHeight() int {
	if m.uiScreen != gouDemoScreenTranscript {
		h := m.inputAreaHeight()
		h += m.slashResultPanelChromeExtra()
		h += m.slashListChromeExtra()
		// Separator line below input (always shown)
		h++
		// Agent footer height (only when sub-agents exist)
		if m.agentTasks != nil {
			agentTasks := m.agentTasks.VisibleTasks()
			if len(agentTasks) > 0 {
				footerLines := 1 + len(agentTasks)
				if footerLines > maxAgentFooterLines {
					footerLines = maxAgentFooterLines
				}
				h += footerLines
			}
		}
		return h
	}
	narrow := m.cols > 0 && m.cols < 80
	foot := joinFooterLines(transcriptChromeFootLines(m, narrow), m.cols)
	c := m.cols
	if c < 1 {
		c = 40
	}
	n := len(strings.Split(layout.WrapForViewport(foot, c), "\n"))
	return max(4, n+1)
}

func (m *Model) hasCompletedAgents() bool {
	return m.agentTasks != nil && m.agentTasks.HasCompletedTasks()
}

func (m *Model) evictCompletedAgents() {
	if m.agentTasks != nil {
		m.agentTasks.EvictCompleted()
	}
}

// handleKeyMsg is the tea.KeyPressMsg branch; also used when SyntheticTTYKeyFromUnknownMsg maps Kitty CSI to KeyPressMsg.
