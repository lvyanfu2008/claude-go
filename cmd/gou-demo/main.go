// Command gou-demo is a minimal Bubble Tea full-screen UI: virtualscroll + markdown + tool blocks (Phase 4 messagerow).
// Extracted [model.Update] branches: update_streaming.go (query yield / NDJSON / fake stream), update_layout.go (window resize).
// Model replies appear inside the TUI. By default the program redraws in the normal buffer so shell scrollback above the session stays available (no tea.WithAltScreen). Set GOU_DEMO_ALT_SCREEN=1 to use the alternate screen (reliable in-pane mouse wheel; previous terminal contents restored on exit).
// With GOU_DEMO_LOG=1, trace uses the same path rules as TS debug log (see goc/ccb-engine/debugpath); on TTY without GOU_DEMO_LOG_FILE, trace goes to that file, not stderr.
//
// Run from repo: cd goc && go run ./cmd/gou-demo
//
// Flags: -transcript=file.json (UI or API messages), -no-seed (or GOU_DEMO_NO_SEED=1) to skip the 45 demo seed rows,
// -replay-cc=events.ndjson, -stream-stdin (pipe NDJSON),
// Real model: [goc/conversation-runtime/query.Query] HTTP streaming parity when ANTHROPIC_API_KEY (or ANTHROPIC_AUTH_TOKEN) is set
// and GOU_QUERY_STREAMING_PARITY=1 or GOU_DEMO_STREAMING_TOOL_EXECUTION=1 (see [query.BuildQueryConfig]).
// Use -fake-stream (or GOU_DEMO_USE_FAKE_STREAM=1) for a UI-only simulated stream with no HTTP (no apilog bodies on send).
// When a tool gate returns ask, GOU_QUERY_ASK_STRATEGY=allow auto-allows for headless demo (maps to [toolexecution.ExecutionDeps.AskResolver]).
// GOU_TOOLEXEC_BASH_SANDBOX_1B=1 enables permissions.ts whole-tool ask bypass on Bash when the tool input carries a non-empty command without dangerously_disable_sandbox (see toolexecution.WholeToolAskSkippedForBash1b).
// Go-side init port (subset of TS init.ts): GOU_DEMO_GO_INIT=1 runs [goc/claudeinit.Init] instead of only [settingsfile.EnsureProjectClaudeEnvOnce] (Init includes Ensure). See docs/plans/go-init-port.md.
// Go local tool parity (streaming parity + [skilltools.ParityToolRunner]): Bash is allowed by default (same as TS); set GOU_DEMO_NO_LOCAL_BASH=1 to disable unless CCB_ENGINE_LOCAL_BASH=1. PowerShell is off unless CCB_ENGINE_LOCAL_POWERSHELL=1 (uses pwsh or powershell.exe). AskUserQuestion auto-picks the first option per question unless GOU_DEMO_NO_ASK_AUTO_FIRST=1. WebFetch is allowed by default; set CCB_ENGINE_DISABLE_WEB_FETCH=1 to block network fetches in the Go runner. See docs/plans/go-tools-parity.md.
//
// System # Language / # Output Style: merged from ~/.claude/settings.json and project .claude/settings.go.json / settings.local.json (see settingsfile; project settings.json is TS-only). CLAUDE_CODE_LANGUAGE and CLAUDE_CODE_OUTPUT_STYLE_* override when set (non-empty); built-in outputStyle keys Explanatory/Learning use prompts from src/constants/outputStyles.ts (embedded).
// Extra CLAUDE.md roots: optional runtimeContext.toolPermissionContext.additionalWorkingDirectories (JSON) and/or GOU_DEMO_EXTRA_CLAUDE_MD_ROOTS / CLAUDE_CODE_EXTRA_CLAUDE_MD_ROOTS (comma or PATH-style list). Paths from runtime/env are always scanned when passed (see [querycontext.ExtraClaudeMdRootsForFetch]); CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1 is only needed for env-only flows in claudemd that do not pass explicit roots.
// Debug log (optional): GOU_DEMO_LOG_FILE=/path/to.log, or GOU_DEMO_LOG=1 (default file path matches TS getDebugLogPath via goc/ccb-engine/debugpath when stderr is TTY). GOU_DEMO_LOG_STDERR=1 forces stderr (may corrupt TUI). Lines are prefixed [gou-demo].
// ToolUseContext dump: CLAUDE_CODE_LOG_TOOL_USE_CONTEXT or GOU_DEMO_LOG_TOOL_USE_CONTEXT = 1|summary|full (with logging enabled) prints JSON after each BuildDemoParams; full includes the entire commands[] snapshot.
// Store transcript dump: GOU_DEMO_LOG_STORE_MESSAGES=1 (with GOU_DEMO_LOG=1 or GOU_DEMO_LOG_FILE) writes [conversation.Store].Messages as indented JSON at after_apply_user_input, before_ccbhydrate, and after stream turn_complete / response_end. Each dump truncates after ~512KiB.
// Virtual-scroll stats line (messages N, visible [a,b), spacers…): set GOU_DEMO_SCROLL_STATS=1 (default off).
// Read/Grep/Glob stream tail: default keeps each tool_use + tool_result as separate rows (avoids looking like history was cleared). Set GOU_DEMO_COLLAPSE_READ_SEARCH_TAIL=1 for TS-style merge into collapsed_read_search (gou/ccbstream/apply.go).
//
// Keys: ↑/↓/PgUp/PgDn scroll the message pane, End bottom, Enter send (Shift+Enter / Ctrl+J / Alt+Enter newline; Shift+↑↓ move line). F2 toggles slash picker. Ctrl+l forces a full-screen clear + redraw (TS Global app:redraw). Ctrl+o toggles TS-style transcript (frozen tail; / search with n/N when not in dump; search bar Esc clears; ctrl+e expands collapsed/grouped except in dump). In the main prompt, user messages that contain only tool_result / advisor_tool_result blocks are omitted from the list (no "user / ↩ tool_result …" stub row); mixed user rows still fold tool_result bodies to one line + (ctrl+o to expand). Transcript view shows full blocks when opened. [ (no search bar) enables dump: show-all + plain transcript to scrollback (Printf). v opens frozen transcript in $VISUAL/$EDITOR via temp file (tea.ExecProcess). Transcript pager (search bar closed, not dump): arrows/pgup/pgdn/end, j/k, g, G/shift+g, ctrl+u/d, ctrl+b/f, b, space (full page), ctrl+n/p (line). Esc/q/ctrl+c exit transcript when search bar closed. In prompt mode, q or Esc quit. Columns < 80 use a shorter header/footer (TS REPL isNarrow). Terminal tab title: OSC 0 unless CLAUDE_CODE_DISABLE_TERMINAL_TITLE=1; loading shows a "…" prefix. CLAUDE_CODE_PERMISSION_MODE sets tool permission mode for submits (TS toolPermissionContext.mode).
// Theme: CLAUDE_CODE_THEME=light (after merged settings env) selects a higher-contrast palette; see [theme.InitFromThemeName]. GOU_DEMO_STATUS_LINE=1 shows theme/msg counts above the prompt.
// Virtual scroll: CLAUDE_CODE_DISABLE_VIRTUAL_SCROLL=1 widens the mounted-item cap (min(n,2000)) via [virtualscroll.RangeInput.MaxMountedItemsOverride]; see gouDemoVirtualScrollDisabled in repl_chrome.go (TS Messages.tsx gate; not a full non-virtual Ink path).
// Prompt message list uses [bubbles/viewport] by default (same scrolling style as go-tui: full-document scroll + ctrl+y fold-all). Disable with GOU_DEMO_BUBBLES_VIEWPORT=0|false|off|no, or use legacy virtualscroll only with GOU_DEMO_LEGACY_VIRTUAL_MESSAGE_SCROLL=1. Exceeding GOU_DEMO_VIEWPORT_MAX_LINES (default 20000 wrapped rows) falls back to classic virtualscroll. Transcript mode always uses the legacy pane.
// Mouse: tea.WithMouseCellMotion enables wheel + plain left-drag scroll on the message list. Set GOU_DEMO_DISABLE_MOUSE_SCROLL=1 to ignore wheel/drag in-app while mouse mode may still be on. Mirror TS fullscreen.ts: CLAUDE_CODE_DISABLE_MOUSE=1 or GOU_DEMO_DISABLE_MOUSE=1 omits SGR mouse so the terminal can use native selection/copy (keyboard scroll still works), unless GOU_DEMO_DISALLOW_DISABLE_MOUSE=1 (ignore those vars and do not run history-browse tea.DisableMouse). Optional one-column TUI scrollbar: GOU_DEMO_MESSAGE_SCROLLBAR=1; GOU_DEMO_NO_SCROLLBAR=1 forces it off. Alternate screen (tea.WithAltScreen): GOU_DEMO_ALT_SCREEN=1. Bubbles viewport (default): go-tui/main/test.go style at-top wheel-up runs tea.DisableMouse for host scrollback; any key runs EnableMouseCellMotion+ClearScreen; opt out with GOU_DEMO_MSG_HISTORY_MOUSE_RELEASE=0|false|off|no.
// Slash: /name is resolved in-process — disk skills via [goc/slashresolve.ResolveDiskSkill], bundled prompts via [goc/slashresolve.ResolveBundledSkill] (embedded markdown under slashresolve/bundleddata). Other prompt commands need a disk skill (SkillRoot) or a bundled definition. Unknown names default to a normal prompt; GOU_DEMO_SLASH_STRICT_UNKNOWN=1 uses TS-style Unknown skill for names matching looksLikeCommand when /name is not an existing root path (non-Windows).
// MCP skills (scheme-2 R0/R1): -mcp-commands-json=path or GOU_DEMO_MCP_COMMANDS_JSON → JSON array of types.Command merged into Skill/commands (enable FEATURE_MCP_SKILLS=1 for listing).
// MCP tool defs (assembleToolPool): -mcp-tools-json=path or GOU_DEMO_MCP_TOOLS_JSON → JSON array merged into Options.Tools when GOU_DEMO_USE_EMBEDDED_TOOLS_API=1 (see mcpcommands.EnvToolsJSONPath).
//
// Session JSONL (optional): GOU_DEMO_RECORD_TRANSCRIPT=1 persists via [goc/sessiontranscript] (~/.claude/projects/.../<session>.jsonl). After each successful ProcessUserInput + ApplyBaseResult, maybeRecordTranscript runs so user rows land before streaming yields. Streaming parity wires [query.QueryDeps.OnQueryYield] to RecordTranscript with a growing turn prefix (same as TS recordTranscript(messages)) so parentUuid chains; each yield is deduped by message UUID; turn end still calls maybeRecordTranscript for a full-store sync. File-history-snapshot stubs: default at most one line per session (before the first non-meta user) unless CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING (TS fileHistory off); GOU_DEMO_FILE_HISTORY_SNAPSHOT_EACH_USER=1 restores one stub before every new non-meta user; GOU_DEMO_SKIP_FILE_HISTORY_SNAPSHOT=1 omits stubs. User message UUIDs follow TS (crypto.randomUUID via process-user-input when DemoConfig.uuid is unset). Set GOU_DEMO_SESSION_ID to a UUID or the store gets a random UUID when the default "demo" id is invalid. Use -no-seed for cleaner UUIDs in demo history.
// Skill listing follows TS delta (sentSkillNames): later submits omit skills already injected. Set GOU_DEMO_SKILL_LISTING_EVERY_TURN=1 to use a fresh sent map each submit so the full listing is attached every round (debug only; not TS production behavior).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"

	"goc/ccb-engine/apilog"
	"goc/ccb-engine/debugpath"
	"goc/ccb-engine/settingsfile"
	"goc/ccb-engine/skilltools"
	"goc/claudeinit"
	"goc/commands"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	"goc/gou/ccbhydrate"
	"goc/gou/ccbstream"
	"goc/gou/conversation"
	"goc/gou/layout"
	"goc/gou/markdown"
	"goc/gou/messagerow"
	"goc/gou/prompt"
	"goc/gou/pui"
	"goc/gou/textutil"
	"goc/gou/theme"
	"goc/gou/transcript"
	"goc/gou/virtualscroll"
	"goc/mcpcommands"
	"goc/messagesapi"
	"goc/modelenv"
	"goc/querycontext"
	"goc/sessiontranscript"
	"goc/toolexecution"
	"goc/tscontext"
	"goc/types"
)

// gouDemoTrace is set by setupGouDemoTrace from GOU_DEMO_LOG_FILE or GOU_DEMO_LOG.
var gouDemoTrace *log.Logger

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

const gouDemoLogStoreMessagesMaxBytes = 512 * 1024

// gouDemoLogStoreMessages dumps store.Messages when GOU_DEMO_LOG_STORE_MESSAGES=1 and trace logging is on.
func gouDemoLogStoreMessages(tag string, s *conversation.Store) {
	if s == nil || !gouDemoEnvTruthy("GOU_DEMO_LOG_STORE_MESSAGES") {
		return
	}
	if gouDemoTrace == nil {
		return
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.Messages); err != nil {
		gouDemoTracef("store.Messages %s: json encode: %v", tag, err)
		return
	}
	out := bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
	if len(out) > gouDemoLogStoreMessagesMaxBytes {
		out = append(out[:gouDemoLogStoreMessagesMaxBytes], []byte("\n…(truncated)")...)
	}
	gouDemoTrace.Printf("store.Messages %s count=%d streamingTextLen=%d\n%s\n", tag, len(s.Messages), len(s.StreamingText), string(out))
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

func gouDemoScrollStatsEnabled() bool {
	return gouDemoEnvTruthy("GOU_DEMO_SCROLL_STATS")
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
			"[gou-demo] CLAUDE_CODE_LOG_API_* is set, but this run uses -fake-stream (or GOU_DEMO_USE_FAKE_STREAM / GOU_DEMO_CCB_INLINE=0).\n"+
				"           No HTTP → apilog will not append request/response lines. For real HTTP logs, omit -fake-stream and set ANTHROPIC_API_KEY plus GOU_QUERY_STREAMING_PARITY=1 or GOU_DEMO_STREAMING_TOOL_EXECUTION=1.\n")
		return
	}
	if !gouDemoHasLLMKeys() {
		fmt.Fprintf(os.Stderr,
			"[gou-demo] CLAUDE_CODE_LOG_API_* is set, but no ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, or OPENAI_API_KEY is set.\n"+
				"           Put keys in ~/.claude/settings.json or project .claude/settings.go.json env, or export them.\n")
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

// gouStreamingToolUsesMsg carries in-flight tool_use snapshots from [query.QueryDeps.OnStreamingToolUses].
// Uses==nil clears the store (Anthropic message_stop); non-nil replaces the live list (may be empty).
type gouStreamingToolUsesMsg struct {
	Uses []query.StreamingToolUseLive
}

// gouQueryDoneMsg marks completion of a query streaming parity turn (Err set on failure).
type gouQueryDoneMsg struct {
	Err error
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
func runQueryStreamingParityTurn(programSend func(tea.Msg), qp query.QueryParams) {
	go func() {
		ctx := context.Background()
		for y, err := range query.Query(ctx, qp) {
			if err != nil {
				if programSend != nil {
					programSend(gouQueryDoneMsg{Err: err})
				}
				return
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

type streamTick struct{}

// teardropAsterisk matches TS constants/figures.ts TEARDROP_ASTERISK (Spinner.tsx).
const teardropAsterisk = "\u273b"

func spinnerTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return gouSpinnerTickMsg{} })
}

func (m *model) beginQuerySpinner() {
	m.queryBusyStartedAt = time.Now()
	m.spinnerVerb = pickSpinnerVerb()
	m.spinnerFrame = 0
}

func (m *model) endQuerySpinner() {
	m.spinnerVerb = ""
	m.queryBusyStartedAt = time.Time{}
	m.spinnerFrame = 0
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
	h := m.streamH
	if h < 2 {
		h = 2
	}
	w := m.width - 2
	if w < 8 {
		w = m.cols
	}
	if m.queryBusy {
		verb := strings.TrimSpace(m.spinnerVerb)
		if verb == "" {
			verb = "Working"
		}
		frames := []string{"…", ".", "..", "..."}
		sfx := frames[m.spinnerFrame%len(frames)]
		row0 := lipgloss.NewStyle().Bold(true).Render(teardropAsterisk + " " + verb + sfx)
		row1 := ""
		if gouDemoSpinnerTipsEnabled() && !m.queryBusyStartedAt.IsZero() {
			if tip := effectiveSpinnerTip(time.Since(m.queryBusyStartedAt), true); tip != "" {
				row1 = lipgloss.NewStyle().Faint(true).Render("Tip: " + tip)
			}
		}
		restH := h - 2
		if restH < 0 {
			restH = 0
		}
		var streamTail string
		if strings.TrimSpace(m.store.StreamingText) != "" {
			toks := markdown.CachedLexerStreaming(m.store.StreamingText)
			streamTail = styleMarkdownTokens(toks, m.cols)
		} else {
			streamTail = lipgloss.NewStyle().Faint(true).Render("(streaming)")
		}
		wrapped := layout.WrapForViewport(streamTail, w)
		tailLines := strings.Split(wrapped, "\n")
		tailLines = padStreamRows(tailLines, restH)
		out := append([]string{row0, row1}, tailLines...)
		return padStreamRows(out, h)
	}
	if strings.TrimSpace(m.store.StreamingText) == "" {
		return padStreamRows(nil, h)
	}
	streamLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("stream: ")
	toks := markdown.CachedLexerStreaming(m.store.StreamingText)
	streamBody := styleMarkdownTokens(toks, m.cols)
	streamWrapped := layout.WrapForViewport(streamLabel+streamBody, w)
	streamRows := strings.Split(streamWrapped, "\n")
	return padStreamRows(streamRows, h)
}

type model struct {
	store  *conversation.Store
	pr     prompt.Model
	width  int
	height int
	cols   int // terminal content width (title/footer); message list may use msgBodyCols when a scrollbar strip is shown

	// msgBodyCols is wrap width for virtual message rows (m.cols or m.cols-1). msgScrollbarW is 0 or 1.
	msgBodyCols   int
	msgScrollbarW int

	permAsk           *permissionAskOverlay
	slashPick         *slashPickerOverlay
	slashCommands     []types.Command
	slashCommandsOnce bool

	scrollTop    int
	pendingDelta int
	sticky       bool
	heightCache  map[string]int
	prevRange    *virtualscroll.Range
	mountedKeys  map[string]struct{}

	// streamChunks appends whole fragments so ``` fences stay valid while streaming.
	streamChunks []string
	streamIdx    int

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

	// tsBridge when non-nil supplies in-process snapshot for commands/tools/prompt parts (tests; former TS bridge removed).
	tsBridge *tscontext.Snapshot

	// transcript when non-nil (GOU_DEMO_RECORD_TRANSCRIPT=1) appends messages after each completed turn.
	transcript *sessiontranscript.Store

	// REPL chrome (terminal title, permission pill): see repl_chrome.go.
	permissionMode        types.PermissionMode
	queryBusy             bool
	queryBusyStartedAt    time.Time
	spinnerVerb           string
	spinnerFrame          int
	lastEmittedTitlePlain string

	// Transcript screen (TS REPL.tsx Screen prompt|transcript + frozenTranscriptState).
	uiScreen             gouDemoScreen
	transcriptFrozen     *frozenTranscriptSnapshot // nil in prompt; set on enterTranscriptScreen
	transcriptShowAll    bool
	transcriptDumpMode   bool // [ : dump-to-scrollback + uncapped show-all (TS dumpMode)
	promptSavedScrollTop int
	promptSavedSticky    bool

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
	resolvedToolIDs   map[string]struct{}
	groupedAgentLookups *messagerow.GroupedAgentLookups
	lastMainLoopModel string

	// rebuildHeightCacheCalls increments in rebuildHeightCache (tests: streaming skip policy).
	rebuildHeightCacheCalls int
}

func main() {
	if gouDemoEnvTruthy("GOU_DEMO_GO_INIT") {
		if err := claudeinit.Init(context.Background(), claudeinit.Options{NonInteractive: true}); err != nil {
			log.Fatalf("gou-demo: claudeinit (GOU_DEMO_GO_INIT): %v", err)
		}
		defer claudeinit.RunCleanups()
	} else {
		if err := settingsfile.EnsureProjectClaudeEnvOnce(); err != nil {
			log.Fatalf("gou-demo: project settings: %v", err)
		}
	}
	theme.InitFromThemeName(os.Getenv("CLAUDE_CODE_THEME"))
	// Env merge matches [settingsfile.ApplyMergedClaudeSettingsEnv]: user ~/.claude/settings.json,
	// project .claude/settings.go.json, settings.local.json. Project .claude/settings.json is TS-only
	// (see settingsfile package doc); put GOU_DEMO_* / CCB_ENGINE_* in settings.go.json or export in shell.
	apilog.PrepareIfEnabled()
	apilog.MaybePrintDiag()
	traceCleanup := setupGouDemoTrace()
	defer traceCleanup()

	transcriptPath := flag.String("transcript", "", "load messages from JSON file (UI []Message or API [{role,content}]); skips built-in seed")
	noSeed := flag.Bool("no-seed", false, "start with an empty transcript (no 45 demo seed messages). Same as GOU_DEMO_NO_SEED=1")
	replayCC := flag.String("replay-cc", "", "apply ccb-engine NDJSON stream events from file (protocol-v1 StreamEvent lines), then open TUI")
	streamStdin := flag.Bool("stream-stdin", false, "read NDJSON stream events from stdin (pipe from ccb-engine); open /dev/tty for keys when available")
	fakeStreamFlag := flag.Bool("fake-stream", false, "do not call the model: simulated stream only (no HTTP; no apilog bodies)")
	mcpCommandsJSON := flag.String("mcp-commands-json", "", "JSON array file of MCP prompt commands (types.Command); overrides "+mcpcommands.EnvCommandsJSONPath)
	mcpToolsJSON := flag.String("mcp-tools-json", "", "JSON array file of MCP tool definitions for assembleToolPool; overrides "+mcpcommands.EnvToolsJSONPath)
	// Backward compat: real LLM used to be opt-in via -ccb-inline; it is now the default. Flag is a no-op.
	ccbInlineCompat := flag.Bool("ccb-inline", false, "deprecated: no-op (real LLM is default). Use -fake-stream for simulation only")
	flag.Parse()
	_ = ccbInlineCompat

	st := &conversation.Store{ConversationID: "demo"}
	if gouDemoEnvTruthy("GOU_DEMO_RECORD_TRANSCRIPT") {
		if sid := strings.TrimSpace(os.Getenv("GOU_DEMO_SESSION_ID")); sessiontranscript.IsValidUUID(sid) {
			st.ConversationID = sid
		} else if !sessiontranscript.IsValidUUID(st.ConversationID) {
			st.ConversationID = sessiontranscript.NewUUID()
		}
	}
	skipSeed := *noSeed || gouDemoEnvTruthy("GOU_DEMO_NO_SEED")
	if *transcriptPath != "" {
		msgs, err := transcript.LoadFile(*transcriptPath)
		if err != nil {
			log.Fatalf("transcript: %v", err)
		}
		st.Messages = msgs
	} else if !skipSeed {
		seedDemo(st)
	}
	if *replayCC != "" {
		if err := ccbstream.ReplayFile(*replayCC, st); err != nil {
			log.Fatalf("replay-cc: %v", err)
		}
	}

	if gouDemoEnvTruthy("GOU_DEMO_TS_CONTEXT_BRIDGE") {
		log.Fatalf("gou-demo: GOU_DEMO_TS_CONTEXT_BRIDGE is no longer supported (scripts/go-context-bridge.ts removed). Use Go prompt assembly and GOU_DEMO_USE_EMBEDDED_TOOLS_API / MCP JSON; unset GOU_DEMO_TS_CONTEXT_BRIDGE.")
	}

	m := newModel(st, strings.TrimSpace(*mcpCommandsJSON), strings.TrimSpace(*mcpToolsJSON), nil)

	opts := []tea.ProgramOption{}
	if gouDemoAltScreenEnabled() {
		opts = append(opts, tea.WithAltScreen())
	}
	if gouDemoMouseCellMotionEnabled() {
		opts = append(opts, tea.WithMouseCellMotion())
	}
	if *streamStdin {
		tty, err := os.Open("/dev/tty")
		if err == nil {
			opts = append(opts, tea.WithInput(tty))
			defer tty.Close()
		}
	}
	inlineCCB := true
	if *fakeStreamFlag {
		inlineCCB = false
	}
	if gouDemoEnvTruthy("GOU_DEMO_USE_FAKE_STREAM") {
		inlineCCB = false
	}
	// Legacy: explicit disable (used when real LLM was opt-in)
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_CCB_INLINE"))); v == "0" || v == "false" || v == "off" || v == "no" {
		inlineCCB = false
	}
	p := tea.NewProgram(m, opts...)
	m.BindCCB(p.Send, inlineCCB)
	gouDemoWarnApilogExpectations(inlineCCB)
	gouDemoTracef("startup messages=%d ccbInline=%v", len(st.Messages), inlineCCB)
	if *streamStdin {
		ccbstream.Feed(os.Stdin, p)
	}
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newModel(st *conversation.Store, mcpCommandsJSONPath, mcpToolsJSONPath string, tsBridge *tscontext.Snapshot) *model {
	pr := prompt.New()

	var tr *sessiontranscript.Store
	if gouDemoEnvTruthy("GOU_DEMO_RECORD_TRANSCRIPT") {
		cwd, _ := os.Getwd()
		fhSnap := !gouDemoEnvTruthy("GOU_DEMO_SKIP_FILE_HISTORY_SNAPSHOT")
		fhEachUser := gouDemoEnvTruthy("GOU_DEMO_FILE_HISTORY_SNAPSHOT_EACH_USER")
		tr = &sessiontranscript.Store{
			SessionID:                 st.ConversationID,
			OriginalCwd:               cwd,
			Cwd:                       cwd,
			FileHistorySnapshotOnUser: fhSnap,
			// Default: at most one stub snapshot per session (TS often shows one line with checkpointing off or single-turn).
			FileHistorySnapshotOnce: fhSnap && !fhEachUser,
		}
	}

	lm := strings.TrimSpace(modelenv.FirstNonEmpty())
	if lm == "" {
		lm = pui.DefaultMainLoopModelForDemo()
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
		permissionMode:      gouDemoPermissionModeFromEnv(),
		useMsgViewport:      gouDemoBubblesViewport(),
	}
}

func (m *model) maybeRecordTranscript() {
	if m.transcript == nil {
		return
	}
	msgs := slices.Clone(m.store.Messages)
	_, err := m.transcript.RecordTranscript(context.Background(), msgs, sessiontranscript.RecordOpts{AllMessages: msgs})
	if err != nil && gouDemoTrace != nil {
		gouDemoTracef("RecordTranscript: %v", err)
	}
}

// BindCCB wires Bubble Tea Send and whether real HTTP streaming parity is allowed (vs simulated stream only).
func (m *model) BindCCB(send func(tea.Msg), inline bool) {
	m.ccbSend = send
	m.ccbInline = inline
}

func seedDemo(s *conversation.Store) {
	for i := range 45 {
		text := fmt.Sprintf("## Seed %d\n\nScroll the message list. Inline `code` here.", i+1)
		if i%7 == 0 {
			text += "\n\n```text\nmulti\nline\nfence\n```\n\nMore body."
		}
		mt := types.MessageTypeUser
		if i%2 == 1 {
			mt = types.MessageTypeAssistant
		}
		raw, _ := json.Marshal([]map[string]string{{"type": "text", "text": text}})
		s.AppendMessage(types.Message{
			UUID:    fmt.Sprintf("seed-%04d", i),
			Type:    mt,
			Content: raw,
		})
	}
	// Tool-use / tool_result pair (Message.tsx assistant + user branches).
	toolRaw, _ := json.Marshal([]map[string]any{
		{"type": "text", "text": "Calling a tool."},
		{"type": "tool_use", "id": "demo-tool-1", "name": "Bash", "input": map[string]any{"command": "ls -la"}},
	})
	s.AppendMessage(types.Message{
		UUID:    "seed-tool-assistant",
		Type:    types.MessageTypeAssistant,
		Content: toolRaw,
	})
	resRaw, _ := json.Marshal([]map[string]any{
		{"type": "tool_result", "tool_use_id": "demo-tool-1", "content": "total 0\ndrwxr-xr-x  .\n", "is_error": false},
	})
	s.AppendMessage(types.Message{
		UUID:    "seed-tool-user",
		Type:    types.MessageTypeUser,
		Content: resRaw,
	})

	assistDisplay, _ := json.Marshal([]map[string]any{{"type": "text", "text": "Read **src/foo.go** (display line)."}})
	displayMsg := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "seed-grouped-display",
		Content: assistDisplay,
	}
	s.AppendMessage(types.Message{
		Type:           types.MessageTypeGroupedToolUse,
		UUID:           "seed-grouped-1",
		ToolName:       "Read",
		Messages:       []types.Message{{Type: types.MessageTypeAssistant, UUID: "g-a1", Content: assistDisplay}},
		Results:        []types.Message{{Type: types.MessageTypeUser, UUID: "g-u1", Content: resRaw}},
		DisplayMessage: &displayMsg,
	})

	s.AppendMessage(types.Message{
		Type:          types.MessageTypeCollapsedReadSearch,
		UUID:          "seed-collapsed-1",
		SearchCount:   2,
		ReadCount:     5,
		ListCount:     1,
		ReadFilePaths: []string{"src/a.ts", "src/b.ts"},
		SearchArgs:    []string{"TODO"},
		DisplayMessage: &types.Message{
			Type:    types.MessageTypeAssistant,
			UUID:    "seed-collapsed-display",
			Content: assistDisplay,
		},
	})

	serverBlock, _ := json.Marshal([]map[string]any{{
		"type": "server_tool_use", "id": "srv1", "name": "example_server_tool", "input": map[string]any{"q": "x"},
	}})
	s.AppendMessage(types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "seed-server-tool",
		Content: serverBlock,
	})
	advisorBlock, _ := json.Marshal([]map[string]any{{
		"type": "advisor_tool_result", "id": "adv1", "content": "suggestion: try Y",
	}})
	s.AppendMessage(types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "seed-advisor",
		Content: advisorBlock,
	})
}

func (m *model) Init() tea.Cmd {
	return nil
}

// teaGlobalRedrawCmd mirrors TS useGlobalKeybindings app:redraw (ctrl+l): clear the terminal
// so the next frame repaints fully (e.g. after the host cleared scrollback with Cmd+K).
func teaGlobalRedrawCmd() tea.Cmd {
	return func() tea.Msg { return tea.ClearScreen() }
}

func (m *model) inputAreaHeight() int {
	n := m.pr.LineCount() + 2
	if m.uiScreen != gouDemoScreenTranscript && !gouDemoBuiltinStatusLineDisabled() {
		n++
	}
	if n < 4 {
		n = 4
	}
	if n > 16 {
		n = 16
	}
	return n
}

// bottomChromeHeight is prompt input height or transcript footer height (TS transcript has no prompt).
func (m *model) bottomChromeHeight() int {
	if m.uiScreen != gouDemoScreenTranscript {
		return m.inputAreaHeight()
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

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleUpdateWindowSize(msg)

	case gouPermissionAskMsg:
		m.permAsk = &permissionAskOverlay{
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

	case tea.MouseMsg:
		if m.msgHistoryBrowseMouseOff && m.msgViewportWanted() {
			return m, nil
		}
		if handled, cmd := m.tryHandleMessageListMouse(msg); handled {
			return m, cmd
		}

	case tea.KeyMsg:
		if m.msgHistoryBrowseMouseOff && m.msgViewportWanted() {
			m.msgHistoryBrowseMouseOff = false
			return m, tea.Sequence(tea.EnableMouseCellMotion, tea.ClearScreen)
		}
		if m.permAsk != nil && msg.String() == "ctrl+c" {
			m.finishPermissionAsk(permissionAskReply{dec: toolexecution.DenyDecision("interrupted"), err: nil})
			return m, tea.Quit
		}
		if m.handlePermissionKey(msg) {
			return m, nil
		}
		if m.slashPick != nil {
			if m.handleSlashPickerKey(msg) {
				return m, nil
			}
		}
		if msg.String() == "ctrl+l" {
			return m, teaGlobalRedrawCmd()
		}
		if m.msgViewportWanted() && msg.String() == "ctrl+y" {
			m.msgFoldAll = !m.msgFoldAll
			m.msgFoldRev++
			return m, nil
		}
		if m.permAsk == nil && m.uiScreen == gouDemoScreenPrompt && msg.String() == "ctrl+o" {
			m.slashPick = nil
			m.rebuildHeightCache()
			return m, m.enterTranscriptScreen()
		}
		if handled, cmd := m.handleTranscriptKey(msg); handled {
			return m, cmd
		}
		if m.msgViewportWanted() && isListViewportScrollKey(msg.String()) {
			return m, m.handleMsgViewportScrollKey(msg)
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.slashPick != nil {
				m.slashPick = nil
				return m, nil
			}
			return m, tea.Quit
		case "f2":
			m.toggleSlashPicker()
			return m, nil
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
		if m.uiScreen == gouDemoScreenTranscript {
			return m, nil
		}
		m.pr.Update(msg)
		if m.pr.Submitted() {
			fullPrompt := strings.TrimRight(m.pr.Value(), "\r\n")
			m.pr.SetValue("")
			line := strings.TrimSpace(fullPrompt)
			if line == "" {
				return m, nil
			}
			var cmd tea.Cmd
			gouDemoTracef("enter input=%q", previewForTrace(line, 120))
			cwd, _ := os.Getwd()
			toolProjectRoot := resolveToolProjectRoot(cwd)
			mergedLang, mergedOutName, mergedOutPrompt := gouDemoMergedSystemLocale()
			preExp := fullPrompt
			demoCfg := pui.DemoConfig{
				SessionID:           m.store.ConversationID,
				Language:            mergedLang,
				MCPCommandsJSONPath: m.mcpCommandsJSONPath,
				MCPToolsJSONPath:    m.mcpToolsJSONPath,
				PreExpansionInput:   &preExp,
				PermissionMode:      &m.permissionMode,
			}
			if m.tsBridge != nil {
				demoCfg.TSContextBridge = m.tsBridge
			}
			params, err := pui.BuildDemoParams(line, m.store, demoCfg)
			if err != nil {
				gouDemoTracef("BuildDemoParams error: %v", err)
				m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: build params: %v", err)))
				m.rebuildHeightCache()
				m.sticky = true
				m.scrollTop = 1 << 30
				return m, cmd
			}
			if params.RuntimeContext != nil {
				gouDemoLogToolUseContext(params.RuntimeContext)
			}
			params.ProcessSlashCommand = pui.NewSlashResolveProcessSlashCommand(pui.SlashResolveHandlerOptions{
				SessionID: m.store.ConversationID,
			})
			gouDemoTracef("ProcessUserInput start priorMsgs=%d", len(m.store.Messages))
			r, err := processuserinput.ProcessUserInput(context.Background(), params)
			gouDemoTracef("ProcessUserInput end err=%v", err)
			if err != nil {
				m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("processUserInput: %v", err)))
				m.rebuildHeightCache()
				m.sticky = true
				m.scrollTop = 1 << 30
				return m, cmd
			}
			out := pui.ApplyBaseResult(m.store, r, &m.processUserInputBaseResultHandoff)
			gouDemoLogStoreMessages("after_apply_user_input", m.store)
			gouDemoTracef("after ApplyBaseResult shouldQuery=%v effectiveShouldQuery=%v hadExecutionRequest=%v messagesAppended=%d",
				r != nil && r.ShouldQuery, out.EffectiveShouldQuery, out.HadExecutionRequest, len(r.Messages))
			if out.NextInput != "" {
				m.pr.SetValue(out.NextInput)
			}
			m.rebuildHeightCache()
			m.sticky = true
			m.scrollTop = 1 << 30
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
					return ccbhydrate.MessagesJSONNormalized(m.store.Messages, toolSpecs, normOpts)
				}
				if m.ccbInline && m.ccbSend != nil {
					baseMsgs, err := tryMsgs()
					if err != nil {
						gouDemoTracef("gou-demo: ccbhydrate.MessagesJSON error: %v (fallback fake stream)", err)
						m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: ccb messages JSON: %v (fallback fake stream)", err)))
						m.rebuildHeightCache()
					} else if len(bytes.TrimSpace(baseMsgs)) < 3 || bytes.Equal(bytes.TrimSpace(baseMsgs), []byte("[]")) {
						gouDemoTracef("gou-demo: empty messages JSON bytes=%d (fake stream)", len(baseMsgs))
						m.store.AppendMessage(pui.SystemNotice("gou-demo: empty chat transcript (fallback fake stream)"))
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
						// Prefer ToolUseContext (from [pui.BuildDemoParams]); live model env chain (incl. settings merge) vs default.
						mainLoopModel := pui.DefaultMainLoopModelForDemo()
						if params.RuntimeContext != nil && strings.TrimSpace(params.RuntimeContext.ToolUseContext.Options.MainLoopModel) != "" {
							mainLoopModel = strings.TrimSpace(params.RuntimeContext.ToolUseContext.Options.MainLoopModel)
						} else if m := modelenv.FirstNonEmpty(); m != "" {
							mainLoopModel = m
						}
						m.lastMainLoopModel = mainLoopModel
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
							CustomSystemPrompt: customSys,
							Gou:                gouOpts,
							ExtraClaudeMdRoots: extraRoots,
						}
						if m.tsBridge != nil {
							fetchOpts.TSSnapshot = m.tsBridge
						}
						partsRes, errParts := querycontext.FetchSystemPromptParts(context.Background(), fetchOpts)
						var guidance string
						var userCtxReminder string
						if errParts != nil {
							gouDemoTracef("FetchSystemPromptParts: %v (fallback BuildGouDemoSystemPrompt)", errParts)
							m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: system context: %v (using base prompt only)", errParts)))
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
							fullParts := querycontext.AppendSystemContextParts(base, partsRes.SystemContext)
							guidance = strings.Join(fullParts, "\n\n")
						}

						listing := ""
						var listingMeta *ccbhydrate.SkillListingMeta
						if !gouDemoEnvTruthy("GOU_DEMO_SKIP_SKILL_LISTING") {
							listingSent := m.skillListingSent
							if gouDemoEnvTruthy("GOU_DEMO_SKILL_LISTING_EVERY_TURN") {
								listingSent = make(map[string]struct{})
							}
							if s, n, initial, ok := commands.AppendSkillListingForAPI(skillListing, hasSkillTool, listingSent, nil); ok {
								listing = s
								listingMeta = &ccbhydrate.SkillListingMeta{SkillCount: n, IsInitial: initial}
							}
						}
						gouDemoLogStoreMessages("before_ccbhydrate", m.store)
						msgsJSON, errL := ccbhydrate.MessagesJSONWithLeadingMeta(m.store.Messages, userCtxReminder, listing, listingMeta, toolSpecs, normOpts)
						if errL != nil {
							gouDemoTracef("gou-demo: MessagesJSONWithLeadingMeta error: %v", errL)
							m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: skill listing hydrate: %v", errL)))
							m.rebuildHeightCache()
						} else {
							reqID := fmt.Sprintf("turn-%d", time.Now().UnixNano())
							m.store.ClearStreaming()
							m.store.ClearStreamingToolUses()
							// TS: skill_listing attachment is pushed to mutableMessages before callModel (QueryEngine attachment case).
							if strings.TrimSpace(listing) != "" {
								if att, ok := ccbhydrate.SkillListingStoreMessage(listing, listingMeta); ok {
									m.store.AppendMessage(att)
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
									SessionID: m.store.ConversationID,
								},
								WorkDir:          cwdAbs,
								ProjectRoot:      toolProjectRoot,
								LocalBashDefault: true,
								AskAutoFirst:     !gouDemoEnvTruthy("GOU_DEMO_NO_ASK_AUTO_FIRST"),
								MainLoopModel:    mainLoopModel,
							}
							if gouDemoPreferQueryStreamingParity() {
								var userCtx map[string]string
								if errParts == nil {
									userCtx = gouDemoUserContextMapForQuery(partsRes.UserContext)
								}
								tcx := types.ToolUseContext{}
								if params.RuntimeContext != nil {
									tcx = params.RuntimeContext.ToolUseContext
								}
								qdeps := query.ProductionDeps()
								te := toolexecution.ExecutionDeps{
									InvokeTool:     runner.Run,
									MainLoopModel:  mainLoopModel,
									ReadToolRoots:  runner.ToolReadMappingRoots(),
									ReadToolMemCWD: runner.ToolReadMappingMemCWD(),
								}
								// Opt-in TS permissions.ts 1b: whole-tool alwaysAsk on Bash skipped when input looks sandboxed (see toolexecution.BashSandboxRule1b).
								if gouDemoEnvTruthy("GOU_TOOLEXEC_BASH_SANDBOX_1B") {
									te.SandboxingEnabled = true
									te.AutoAllowBashWholeToolAskWhenSandboxed = true
								}
								m.installAskResolver(&te)
								qdeps.ToolexecutionDeps = te
								// Snapshot matches qp.Messages (TS QueryEngine messages at callModel): includes skill_listing row if appended above.
								msgsForQ := slices.Clone(m.store.Messages)
								if send := m.ccbSend; send != nil {
									qdeps.OnStreamingToolUses = func(ctx context.Context, uses []query.StreamingToolUseLive) error {
										send(gouStreamingToolUsesMsg{Uses: uses})
										return nil
									}
								}
								if m.transcript != nil && gouDemoEnvTruthy("GOU_DEMO_RECORD_TRANSCRIPT") {
									tr := m.transcript
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
								qp := query.QueryParams{
									Messages:        msgsForQ,
									SystemPrompt:    query.AsSystemPrompt([]string{guidance}),
									UserContext:     userCtx,
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
									reqID, len(m.store.Messages), len(toolsJSON))
								m.beginQuerySpinner()
								m.queryBusy = true
								m.store.ClearStreamingToolUses()
								runQueryStreamingParityTurn(m.ccbSend, qp)
								usedCCB = true
							} else {
								m.store.AppendMessage(pui.SystemNotice(
									"gou-demo: ccb-engine/localturn was removed. For a real model reply, set ANTHROPIC_API_KEY (or ANTHROPIC_AUTH_TOKEN) and GOU_QUERY_STREAMING_PARITY=1 or GOU_DEMO_STREAMING_TOOL_EXECUTION=1. Or use -fake-stream for a simulated reply only.",
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
				gouDemoTracef("starting fake streamTick path")
				m.beginQuerySpinner()
				m.queryBusy = true
				m.streamChunks = []string{
					"## Streamed reply\n\n",
					"Chunks preserve ``` fences.\n\n```go\n",
					`fmt.Println("gou-demo")`,
					"\n```\n\n",
					"Done.",
				}
				m.streamIdx = 0
				m.store.ClearStreaming()
				m.store.ClearStreamingToolUses()
				return m, tea.Batch(cmd, tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg { return streamTick{} }), spinnerTickCmd())
			}
			gouDemoTracef("no query path (effectiveShouldQuery=%v hadExecutionRequest=%v)", out.EffectiveShouldQuery, out.HadExecutionRequest)
			if !out.EffectiveShouldQuery && !out.HadExecutionRequest {
				sq := false
				if r != nil {
					sq = r.ShouldQuery
				}
				m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf(
					"gou-demo: 未调用模型（shouldQuery=%v）。全屏 TUI 里看回复；本终端 shell 不会打印对话正文。调试: GOU_DEMO_LOG=1 stderr",
					sq)))
				m.rebuildHeightCache()
				m.sticky = true
				m.scrollTop = 1 << 30
			}
			return m, cmd
		}
		return m, nil

	case gouQueryYieldMsg:
		return m.handleUpdateGouQueryYield(msg)

	case gouStreamingToolUsesMsg:
		return m.handleUpdateGouStreamingToolUses(msg)

	case gouSpinnerTickMsg:
		return m.handleUpdateGouSpinnerTick(msg)

	case gouQueryDoneMsg:
		return m.handleUpdateGouQueryDone(msg)

	case streamTick:
		return m.handleUpdateStreamTick(msg)

	case ccbstream.Msg:
		return m.handleUpdateCCBStream(msg)
	}

	if m.uiScreen != gouDemoScreenTranscript {
		m.pr.Update(msg)
	}
	return m, nil
}

func listViewportH(m *model) int {
	streamReserve := m.streamH
	if m.uiScreen == gouDemoScreenTranscript {
		streamReserve = 0
	}
	h := m.height - m.titleH - streamReserve - m.bottomChromeHeight() - 2
	if gouDemoStatusLineEnabled() {
		h--
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

// measureMessageRows matches final View styling (ANSI + wrap) for VirtualMessageList heightCache parity.
func (m *model) messagerowOpts(msg types.Message) *messagerow.RenderOpts {
	if m.uiScreen == gouDemoScreenPrompt {
		active := m.queryBusy &&
			len(m.store.Messages) > 0 &&
			m.store.Messages[len(m.store.Messages)-1].UUID == msg.UUID &&
			msg.Type == types.MessageTypeCollapsedReadSearch &&
			strings.TrimSpace(m.store.StreamingText) == ""
		return &messagerow.RenderOpts{
			FoldToolResultBody:        true,
			CollapsedReadSearchActive: active,
			GroupedAgentLookups:       m.groupedAgentLookups,
		}
	}
	if m.uiScreen == gouDemoScreenTranscript {
		ro := &messagerow.RenderOpts{
			GroupedAgentLookups:        m.groupedAgentLookups,
			VerboseCollapsedReadSearch: true,
		}
		if m.transcriptShowAll || m.transcriptDumpMode {
			ro.ShowAllInTranscript = true
		}
		return ro
	}
	return &messagerow.RenderOpts{
		GroupedAgentLookups: m.groupedAgentLookups,
	}
}

func (m *model) measureMessageRows(msg types.Message, cols int, searchHL string) int {
	if m.skipFoldedToolResultStubInPrompt(msg) {
		return 0
	}
	segs := messagerow.SegmentsFromMessageOpts(msg, m.messagerowOpts(msg))
	if msg.Type == types.MessageTypeAttachment && len(segs) == 0 {
		return 0
	}
	var header string
	if msg.Type != types.MessageTypeAttachment {
		switch msg.Type {
		case types.MessageTypeUser, types.MessageTypeAssistant:
		default:
			header = lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(msg.Type)).Render(string(msg.Type))
		}
	}
	body := formatMessageSegments(segs, cols, m.showToolUseCtrlOExpandHint(), m.resolvedToolIDs, msg.Type == types.MessageTypeAssistant, searchHL)
	body = withUserPromptPointerIfNeeded(msg, body)
	block := body
	if header != "" {
		block = header + "\n" + body
	}
	r := layout.WrappedRowCount(block, cols)
	if msg.Type == types.MessageTypeAttachment {
		return r
	}
	return max(1, r)
}

func (m *model) measureTranscriptStreamingToolRow(tu conversation.StreamingToolUse, cols int, searchHL string) int {
	head := lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
	namePart := tu.Name
	if strings.TrimSpace(searchHL) != "" {
		namePart = highlightSearchPlain(tu.Name, searchHL, transcriptSearchHLStyle())
	}
	toolLine := lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Bold(true).Render("⚙ "+namePart) + lipgloss.NewStyle().Faint(true).Render(" · streaming")
	block := head + "\n" + toolLine
	if s := strings.TrimSpace(tu.UnparsedInput); s != "" {
		maxW := cols * 4
		if maxW < 80 {
			maxW = 80
		}
		prev := previewForTrace(s, maxW)
		if strings.TrimSpace(searchHL) != "" {
			prev = highlightSearchPlain(prev, searchHL, transcriptSearchHLStyle())
		}
		block += "\n" + lipgloss.NewStyle().Faint(true).Render(prev)
	}
	return max(1, layout.WrappedRowCount(block, cols))
}

func (m *model) renderTranscriptStreamingToolRow(tu conversation.StreamingToolUse, cols, h int, searchHL string) string {
	head := lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
	namePart := tu.Name
	if strings.TrimSpace(searchHL) != "" {
		namePart = highlightSearchPlain(tu.Name, searchHL, transcriptSearchHLStyle())
	}
	toolLine := lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Bold(true).Render("⚙ "+namePart) + lipgloss.NewStyle().Faint(true).Render(" · streaming")
	var b strings.Builder
	b.WriteString(head)
	b.WriteByte('\n')
	b.WriteString(toolLine)
	if s := strings.TrimSpace(tu.UnparsedInput); s != "" {
		b.WriteByte('\n')
		maxW := cols * 4
		if maxW < 80 {
			maxW = 80
		}
		prev := previewForTrace(s, maxW)
		if strings.TrimSpace(searchHL) != "" {
			prev = highlightSearchPlain(prev, searchHL, transcriptSearchHLStyle())
		}
		b.WriteString(lipgloss.NewStyle().Faint(true).Render(prev))
	}
	out := b.String()
	lines := strings.Split(out, "\n")
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h && h > 0 {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}

func (m *model) refineVisibleHeights(keys []string, start, end, n int) {
	cols := m.messageBodyColsForLayout()
	if cols < 1 {
		return
	}
	hl := m.transcriptSearchHighlightNeedle()
	msgView := m.messagesForScroll()
	msgN := len(msgView)
	st := m.transcriptStreamingToolsForView()
	for i := start; i < end && i < n; i++ {
		if i < msgN {
			h := m.measureMessageRows(msgView[i], cols, hl)
			if i > 0 && userAssistantPairBlankLine(msgView[i-1], msgView[i]) {
				h++
			}
			m.heightCache[keys[i]] = h
			continue
		}
		ti := i - msgN
		if ti >= 0 && ti < len(st) {
			h := m.measureTranscriptStreamingToolRow(st[ti], cols, hl)
			if ti == 0 && msgN > 0 && msgView[msgN-1].Type == types.MessageTypeUser {
				h++
			}
			m.heightCache[keys[i]] = h
		} else {
			m.heightCache[keys[i]] = 1
		}
	}
}

func (m *model) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	vpH := listViewportH(m)
	bodyCols := m.messageBodyColsForLayout()
	useVp := m.useMsgViewport && m.uiScreen == gouDemoScreenPrompt && !m.msgViewportFallback
	if useVp {
		m.msgViewportSyncGeometry()
		m.applyMsgViewportContentFromView()
		if m.msgViewportFallback {
			useVp = false
		}
	}

	var b strings.Builder
	narrow := m.cols > 0 && m.cols < 80
	plainTitle := replChromeComposeTerminalTitle(m.store.ConversationID, m.queryBusy, strings.TrimSpace(m.store.StreamingText) != "")
	if !gouDemoTerminalTitleDisabled() && plainTitle != m.lastEmittedTitlePlain {
		m.lastEmittedTitlePlain = plainTitle
		if osc := oscSetWindowTitle(plainTitle); osc != "" {
			b.WriteString(osc)
		}
	}
	topBar := replChromeTopBar(narrow)
	if m.uiScreen == gouDemoScreenTranscript {
		topBar = replChromeTranscriptTopBar(narrow)
	}
	title := lipgloss.NewStyle().Bold(true).Render(topBar)
	b.WriteString(title)
	b.WriteByte('\n')

	if useVp {
		b.WriteString(m.messagePaneViewportBlock(vpH, bodyCols))
		b.WriteByte('\n')
	} else {
		keys := m.scrollItemKeys()
		n := len(keys)
		if !m.sticky {
			m.clampScrollTopForVirtualList()
		}

		if m.prevRange != nil && n > 0 {
			m.refineVisibleHeights(keys, m.prevRange.Start, m.prevRange.End, n)
		}

		ri := virtualscroll.RangeInput{
			ItemKeys:     keys,
			HeightCache:  m.heightCache,
			ScrollTop:    m.scrollTop,
			PendingDelta: m.pendingDelta,
			ViewportH:    vpH,
			IsSticky:     m.sticky,
			ListOrigin:   0,
			PrevRange:    m.prevRange,
			MountedKeys:  m.mountedKeys,
			FastScroll:   false,
		}
		if gouDemoVirtualScrollDisabled() && n > 0 {
			ri.MaxMountedItemsOverride = min(n, 2000)
		}
		vr := virtualscroll.ComputeRange(ri)

		m.mountedKeys = make(map[string]struct{}, max(0, vr.End-vr.Start))
		for i := vr.Start; i < vr.End && i < n; i++ {
			m.mountedKeys[keys[i]] = struct{}{}
		}

		if m.prevRange == nil {
			m.prevRange = &virtualscroll.Range{}
		}
		m.prevRange.Start, m.prevRange.End = vr.Start, vr.End

		var msgPane strings.Builder
		if gouDemoScrollStatsEnabled() {
			msgPane.WriteString(lipgloss.NewStyle().Faint(true).Render(
				fmt.Sprintf("messages %d  cols=%d  visible [%d,%d)  topSpacer=%d bottomSpacer=%d sticky=%v",
					n, m.cols, vr.Start, vr.End, vr.TopSpacer, vr.BottomSpacer, m.sticky)))
			msgPane.WriteByte('\n')
		}

		hl := m.transcriptSearchHighlightNeedle()
		msgView := m.messagesForScroll()
		msgN := len(msgView)
		stRows := m.transcriptStreamingToolsForView()
		for i := vr.Start; i < vr.End && i < n; i++ {
			key := keys[i]
			h := m.heightCache[key]
			var block string
			if i < msgN {
				msg := msgView[i]
				block = m.renderMessageRow(msg, bodyCols, h, hl)
			} else {
				ti := i - msgN
				if ti < 0 || ti >= len(stRows) {
					continue
				}
				block = m.renderTranscriptStreamingToolRow(stRows[ti], bodyCols, h, hl)
			}
			if strings.TrimSpace(block) == "" && h <= 0 {
				continue
			}
			if msgPane.Len() > 0 {
				msgPane.WriteByte('\n')
				if i > vr.Start {
					needExtra := false
					if i < msgN {
						needExtra = userAssistantPairBlankLine(msgView[i-1], msgView[i])
					} else if i == msgN && msgN > 0 {
						needExtra = msgView[msgN-1].Type == types.MessageTypeUser
					}
					if needExtra {
						msgPane.WriteByte('\n')
					}
				}
			}
			msgPane.WriteString(block)
		}
		if m.uiScreen != gouDemoScreenTranscript && len(m.store.StreamingToolUses) > 0 {
			for _, tu := range m.store.StreamingToolUses {
				if msgPane.Len() > 0 {
					msgPane.WriteByte('\n')
				}
				head := lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
				msgPane.WriteString(head)
				msgPane.WriteByte('\n')
				toolTitle := lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Bold(true).Render("⚙ "+tu.Name) + lipgloss.NewStyle().Faint(true).Render(" · streaming")
				msgPane.WriteString(toolTitle)
				if s := strings.TrimSpace(tu.UnparsedInput); s != "" {
					msgPane.WriteByte('\n')
					maxW := bodyCols * 4
					if maxW < 80 {
						maxW = 80
					}
					msgPane.WriteString(lipgloss.NewStyle().Faint(true).Render(previewForTrace(s, maxW)))
				}
			}
		}
		if m.uiScreen != gouDemoScreenTranscript && strings.TrimSpace(m.store.StreamingText) != "" {
			if msgPane.Len() > 0 {
				msgPane.WriteByte('\n')
				if streamGapAfterUserMessage(msgView) {
					msgPane.WriteByte('\n')
				}
			} else {
				msgPane.WriteByte('\n')
			}
			head := lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
			msgPane.WriteString(head)
			msgPane.WriteByte('\n')
			msgPane.WriteString(styleMarkdownTokens(markdown.CachedLexerStreaming(m.store.StreamingText), bodyCols))
		}

		lines := strings.Split(msgPane.String(), "\n")
		for len(lines) < vpH {
			lines = append(lines, "")
		}
		if len(lines) > vpH {
			if m.sticky {
				lines = lines[len(lines)-vpH:]
			} else {
				lines = lines[:vpH]
			}
		}
		totalScroll := m.messageScrollContentHeight()
		b.WriteString(joinMessagePaneLinesWithScrollbar(lines, bodyCols, vpH, totalScroll, m.scrollTop, m.msgScrollbarW))
		b.WriteByte('\n')
	}

	if m.uiScreen != gouDemoScreenTranscript {
		streamRows := m.promptBottomStreamRows()
		b.WriteString(strings.Join(streamRows, "\n"))
		b.WriteByte('\n')
	}
	if s := m.statusLineString(); s != "" {
		b.WriteString(s)
		b.WriteByte('\n')
	}

	if m.uiScreen == gouDemoScreenTranscript {
		foot := joinFooterLines(transcriptChromeFootLines(m, narrow), m.cols)
		b.WriteString(lipgloss.NewStyle().Faint(true).Width(m.cols).Render(foot))
	} else {
		if s := m.builtinStatusLineView(); s != "" {
			b.WriteString(s)
			b.WriteByte('\n')
		}
		promptView := userInputViewWithPromptPrefix(m)
		hintText := strings.TrimSpace(replChromeFooterHint(narrow))
		frag := strings.TrimSpace(replChromePermissionFragment(m.permissionMode, narrow))
		var hintLine string
		switch {
		case frag != "" && hintText != "":
			hintLine = frag + " · " + hintText
		case frag != "":
			hintLine = frag
		default:
			hintLine = hintText
		}
		b.WriteString(promptView)
		if hintLine != "" {
			hint := lipgloss.NewStyle().Faint(true).Width(m.cols).Render(hintLine)
			b.WriteByte('\n')
			b.WriteString(hint)
		}
	}
	out := lipgloss.NewStyle().MaxWidth(m.width).Render(b.String())
	if m.permAsk != nil {
		mod := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(m.renderPermissionModal(m.width))
		out = lipgloss.JoinVertical(lipgloss.Left, out, mod)
	}
	if m.slashPick != nil {
		overlay := m.renderSlashPicker(m.width, min(14, m.height/3))
		out = lipgloss.JoinVertical(lipgloss.Left, out, overlay)
	}
	return out
}

func (m *model) showToolUseCtrlOExpandHint() bool {
	return m.uiScreen == gouDemoScreenPrompt && !m.transcriptDumpMode
}

// userAssistantPairBlankLine is true when the UI inserts one empty line between adjacent
// user and assistant scroll rows (either order).
func userAssistantPairBlankLine(a, b types.Message) bool {
	u, aType := types.MessageTypeUser, types.MessageTypeAssistant
	return (a.Type == u && b.Type == aType) || (a.Type == aType && b.Type == u)
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

// userMessageIsOmittableToolResultStub is true when the user row would render only folded tool_result
// stubs (no visible user text). Empty text blocks and whitespace-only text are ignored so
// [{"type":"text","text":""},{"type":"tool_result",...}] still omits. Call after [messagerow.NormalizeMessageJSON]
// so API-shaped rows with content only in Message.{role,content} match.
func userMessageIsOmittableToolResultStub(msg types.Message) bool {
	if msg.Type != types.MessageTypeUser || len(msg.Content) == 0 {
		return false
	}
	var blocks []types.MessageContentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil || len(blocks) == 0 {
		return false
	}
	hasTool := false
	for _, b := range blocks {
		switch b.Type {
		case "tool_result", "advisor_tool_result":
			hasTool = true
		case "text":
			if strings.TrimSpace(b.Text) != "" {
				return false
			}
		default:
			return false
		}
	}
	return hasTool
}

func (m *model) skipFoldedToolResultStubInPrompt(msg types.Message) bool {
	if m.uiScreen != gouDemoScreenPrompt {
		return false
	}
	if messagerow.VerboseToolOutputEnabled() {
		return false
	}
	msg = messagerow.NormalizeMessageJSON(msg)
	return userMessageIsOmittableToolResultStub(msg)
}

func userPromptPrefixStyled() string {
	return lipgloss.NewStyle().Faint(true).Foreground(theme.DimMuted()).Render(UserPromptPointerGlyph() + " ")
}

// userInputViewWithPromptPrefix prepends the same dim "> " as user rows on the first line of the bottom input.
func userInputViewWithPromptPrefix(m *model) string {
	v := m.pr.View()
	prefix := userPromptPrefixStyled()
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
	prefix := userPromptPrefixStyled()
	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		return prefix
	}
	lines[0] = prefix + lines[0]
	return strings.Join(lines, "\n")
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
		default:
			header = lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(msg.Type)).Render(string(msg.Type))
		}
	}
	body := formatMessageSegments(segs, cols, m.showToolUseCtrlOExpandHint(), m.resolvedToolIDs, msg.Type == types.MessageTypeAssistant, searchHL)
	body = withUserPromptPointerIfNeeded(msg, body)
	block := body
	if header != "" {
		block = header + "\n" + body
	}
	wrapped := layout.WrapForViewport(block, cols)
	rows := strings.Split(wrapped, "\n")
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	return strings.Join(rows, "\n")
}

func toolRowLeadPrefix() string {
	glyph := "\u25cf " // ● — TS figures.BLACK_CIRCLE non-darwin
	if runtime.GOOS == "darwin" {
		glyph = "\u23fa " // ⏺ — TS figures.BLACK_CIRCLE on darwin
	}
	return lipgloss.NewStyle().Foreground(theme.DimMuted()).Render(glyph)
}

// prefixToolGlyphFirstLine prepends the dim tool lead (⏺ / ●) to the first line of rendered assistant text.
func prefixToolGlyphFirstLine(body string) string {
	if body == "" {
		return toolRowLeadPrefix()
	}
	p := toolRowLeadPrefix()
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

// formatMessageSegments mirrors Message.tsx per-block branches (text→markdown, tool_use/tool_result/thinking).
// assistantLeadGlyph prefixes the first non-empty assistant text segment (TS-style ⏺ before the opening sentence).
// searchHL applies transcript search highlight to visible plain substrings (TS useSearchHighlight).
func formatMessageSegments(segs []messagerow.Segment, cols int, toolUseCtrlOHint bool, resolved map[string]struct{}, assistantLeadGlyph bool, searchHL string) string {
	hlSt := transcriptSearchHLStyle()
	withHL := func(s string) string {
		if strings.TrimSpace(searchHL) == "" {
			return s
		}
		return highlightSearchPlain(s, searchHL, hlSt)
	}
	var parts []string
	assistantTextLeadDone := false
	for i, seg := range segs {
		switch seg.Kind {
		case messagerow.SegTextMarkdown:
			textForMd := seg.Text
			if strings.TrimSpace(searchHL) != "" {
				textForMd = highlightSearchPlain(seg.Text, searchHL, hlSt)
			}
			md := styleMarkdownTokens(markdown.CachedLexer(textForMd), cols)
			if assistantLeadGlyph && !assistantTextLeadDone && strings.TrimSpace(seg.Text) != "" {
				assistantTextLeadDone = true
				md = prefixToolGlyphFirstLine(md)
			}
			parts = append(parts, md)
		case messagerow.SegToolUse:
			if seg.ToolFacing != "" {
				row1 := ""
				if !priorNonEmptyAssistantText(segs, i) {
					row1 = toolRowLeadPrefix()
				}
				row1 += lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Bold(true).Render(withHL(seg.ToolFacing))
				if p := strings.TrimSpace(seg.ToolParen); p != "" {
					row1 += " (" + withHL(p) + ")"
				}
				var toolLines []string
				toolLines = append(toolLines, row1)
				if !toolUseResolved(resolved, seg.ToolUseID) {
					if act := strings.TrimSpace(seg.Text); act != "" {
						actLine := lipgloss.NewStyle().Foreground(theme.DimMuted()).Render(withHL(act) + "…")
						if toolUseCtrlOHint {
							actLine += lipgloss.NewStyle().Faint(true).Render(" (ctrl+o to expand)")
						}
						toolLines = append(toolLines, actLine)
					}
					if h := strings.TrimSpace(seg.ToolHint); h != "" {
						toolLines = append(toolLines, lipgloss.NewStyle().Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(h))))
					}
				}
				parts = append(parts, strings.Join(toolLines, "\n"))
			} else {
				line := lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Bold(true).Render("⚙ " + withHL(seg.Text))
				if toolUseCtrlOHint {
					line += lipgloss.NewStyle().Faint(true).Render(" (ctrl+o to expand)")
				}
				parts = append(parts, line)
			}
		case messagerow.SegToolResult:
			st := lipgloss.NewStyle().Foreground(theme.DimMuted())
			if seg.IsToolError {
				st = lipgloss.NewStyle().Foreground(theme.ToolError())
			}
			body := textutil.LinkifyOSC8(seg.Text)
			line := st.Render("↩ " + withHL(body))
			if seg.ToolBodyOmitted && toolUseCtrlOHint {
				line += lipgloss.NewStyle().Faint(true).Render(" (ctrl+o to expand)")
			}
			parts = append(parts, line)
		case messagerow.SegThinking:
			body := textutil.LinkifyOSC8(seg.Text)
			parts = append(parts, lipgloss.NewStyle().Bold(true).Render("● "+withHL(body)))
		case messagerow.SegDisplayHint:
			parts = append(parts, lipgloss.NewStyle().Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(withHL(seg.Text))))
		case messagerow.SegServerToolUse:
			if seg.ToolFacing != "" {
				row1 := ""
				if !priorNonEmptyAssistantText(segs, i) {
					row1 = toolRowLeadPrefix()
				}
				row1 += lipgloss.NewStyle().Foreground(theme.ServerAccent()).Bold(true).Render(withHL(seg.ToolFacing))
				if p := strings.TrimSpace(seg.ToolParen); p != "" {
					row1 += " (" + withHL(p) + ")"
				}
				var toolLines []string
				toolLines = append(toolLines, row1)
				if !toolUseResolved(resolved, seg.ToolUseID) {
					if act := strings.TrimSpace(seg.Text); act != "" {
						actLine := lipgloss.NewStyle().Foreground(theme.DimMuted()).Render(withHL(act) + "…")
						if toolUseCtrlOHint {
							actLine += lipgloss.NewStyle().Faint(true).Render(" (ctrl+o to expand)")
						}
						toolLines = append(toolLines, actLine)
					}
					if h := strings.TrimSpace(seg.ToolHint); h != "" {
						toolLines = append(toolLines, lipgloss.NewStyle().Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(h))))
					}
				}
				parts = append(parts, strings.Join(toolLines, "\n"))
			} else {
				line := lipgloss.NewStyle().Foreground(theme.ServerAccent()).Bold(true).Render("⎈ " + withHL(seg.Text))
				if toolUseCtrlOHint {
					line += lipgloss.NewStyle().Faint(true).Render(" (ctrl+o to expand)")
				}
				parts = append(parts, line)
			}
		case messagerow.SegAdvisorToolResult:
			st := lipgloss.NewStyle().Foreground(theme.AdvisorAccent())
			if seg.IsToolError {
				st = lipgloss.NewStyle().Foreground(theme.ToolError())
			}
			body := textutil.LinkifyOSC8(seg.Text)
			line := st.Render("✧ " + withHL(body))
			if seg.ToolBodyOmitted && toolUseCtrlOHint {
				line += lipgloss.NewStyle().Faint(true).Render(" (ctrl+o to expand)")
			}
			parts = append(parts, line)
		case messagerow.SegGroupedToolUse:
			parts = append(parts, lipgloss.NewStyle().Foreground(theme.GroupedAccent()).Bold(true).Render("▦ "+withHL(seg.Text)))
		case messagerow.SegCollapsedReadSearch:
			parts = append(parts, lipgloss.NewStyle().Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(withHL(seg.Text))))
		case messagerow.SegSkillListingAvailable:
			n := seg.Num
			if n < 1 {
				n = 1
			}
			word := "skills"
			if n == 1 {
				word = "skill"
			}
			line := lipgloss.NewStyle().Bold(true).Render(strconv.Itoa(n)) + " " + word + " available"
			parts = append(parts, line)
		default:
			parts = append(parts, lipgloss.NewStyle().Faint(true).Render(textutil.LinkifyOSC8(withHL(seg.Text))))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// styleMarkdownTokens applies lipgloss to block tokens (mirrors Markdown.tsx roles, terminal-only).
func styleMarkdownTokens(toks []markdown.Token, cols int) string {
	if len(toks) == 0 {
		return ""
	}
	var parts []string
	for _, t := range toks {
		switch t.Type {
		case "heading":
			lv := min(max(t.Level, 1), 6)
			line := strings.Repeat("#", lv) + " " + t.Text
			parts = append(parts, lipgloss.NewStyle().Bold(true).Foreground(theme.MarkdownHeading()).Render(line))
		case "code":
			cb := "```" + t.Lang + "\n" + t.Text
			if t.Text != "" && !strings.HasSuffix(t.Text, "\n") {
				cb += "\n"
			}
			cb += "```"
			parts = append(parts, lipgloss.NewStyle().Faint(true).Render(cb))
		case "list_item":
			parts = append(parts, lipgloss.NewStyle().Render("- "+t.Text))
		case "blockquote":
			parts = append(parts, lipgloss.NewStyle().Italic(true).Render("> "+strings.ReplaceAll(t.Text, "\n", "\n> ")))
		case "hr":
			parts = append(parts, lipgloss.NewStyle().Faint(true).Render("---"))
		default:
			parts = append(parts, t.Text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}
