// Command gou-demo is a minimal Bubble Tea full-screen UI: virtualscroll + markdown + tool blocks (Phase 4 messagerow).
// Model replies appear inside the TUI. Default uses the terminal alt-screen (full-screen); after quit, that buffer is gone — use -no-alt-screen (or GOU_DEMO_NO_ALT_SCREEN=1) so output stays in normal shell scrollback.
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
// Go local tool parity (streaming parity + [skilltools.ParityToolRunner]): Bash is allowed by default (same as TS); set GOU_DEMO_NO_LOCAL_BASH=1 to disable unless CCB_ENGINE_LOCAL_BASH=1. AskUserQuestion auto-picks the first option per question unless GOU_DEMO_NO_ASK_AUTO_FIRST=1. WebFetch is allowed by default; set CCB_ENGINE_DISABLE_WEB_FETCH=1 to block network fetches in the Go runner. See docs/plans/go-tools-parity.md.
//
// System # Language / # Output Style: merged from ~/.claude/settings.json and project .claude/settings.go.json / settings.local.json (see settingsfile; project settings.json is TS-only). CLAUDE_CODE_LANGUAGE and CLAUDE_CODE_OUTPUT_STYLE_* override when set (non-empty); built-in outputStyle keys Explanatory/Learning use prompts from src/constants/outputStyles.ts (embedded).
// Extra CLAUDE.md roots: optional runtimeContext.toolPermissionContext.additionalWorkingDirectories (JSON) and/or GOU_DEMO_EXTRA_CLAUDE_MD_ROOTS / CLAUDE_CODE_EXTRA_CLAUDE_MD_ROOTS (comma or PATH-style list). Paths from runtime/env are always scanned when passed (see [querycontext.ExtraClaudeMdRootsForFetch]); CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1 is only needed for env-only flows in claudemd that do not pass explicit roots.
// Debug log (optional): GOU_DEMO_LOG_FILE=/path/to.log, or GOU_DEMO_LOG=1 (default file path matches TS getDebugLogPath via goc/ccb-engine/debugpath when stderr is TTY). GOU_DEMO_LOG_STDERR=1 forces stderr (may corrupt TUI). Lines are prefixed [gou-demo].
// ToolUseContext dump: CLAUDE_CODE_LOG_TOOL_USE_CONTEXT or GOU_DEMO_LOG_TOOL_USE_CONTEXT = 1|summary|full (with logging enabled) prints JSON after each BuildDemoParams; full includes the entire commands[] snapshot.
// Store transcript dump: GOU_DEMO_LOG_STORE_MESSAGES=1 (with GOU_DEMO_LOG=1 or GOU_DEMO_LOG_FILE) writes [conversation.Store].Messages as indented JSON at after_apply_user_input, before_ccbhydrate, and after stream turn_complete / response_end. Each dump truncates after ~512KiB.
// Virtual-scroll stats line (messages N, visible [a,b), spacers…): set GOU_DEMO_SCROLL_STATS=1 (default off).
//
// Keys: ↑/↓/PgUp/PgDn scroll the message pane, End bottom, Enter send (Ctrl+J / Alt+Enter newline; Shift+↑↓ move line). F2 toggles slash picker. Ctrl+o toggles TS-style transcript screen (frozen message tail; Esc/q/ctrl+c closes; ctrl+e toggles show-all hint). In prompt mode, q or Esc quit. Columns < 80 use a shorter header/footer (TS REPL isNarrow). Terminal tab title: OSC 0 unless CLAUDE_CODE_DISABLE_TERMINAL_TITLE=1; loading shows a "…" prefix. CLAUDE_CODE_PERMISSION_MODE sets tool permission mode for submits (TS toolPermissionContext.mode).
// Theme: CLAUDE_CODE_THEME=light (after merged settings env) selects a higher-contrast palette; see [theme.InitFromThemeName]. GOU_DEMO_STATUS_LINE=1 shows theme/msg counts above the prompt.
// Slash: /name is resolved in-process — disk skills via [goc/slashresolve.ResolveDiskSkill], bundled prompts via [goc/slashresolve.ResolveBundledSkill] (embedded markdown under slashresolve/bundleddata). Other prompt commands need a disk skill (SkillRoot) or a bundled definition. Unknown names default to a normal prompt; GOU_DEMO_SLASH_STRICT_UNKNOWN=1 uses TS-style Unknown skill for names matching looksLikeCommand when /name is not an existing root path (non-Windows).
// MCP skills (scheme-2 R0/R1): -mcp-commands-json=path or GOU_DEMO_MCP_COMMANDS_JSON → JSON array of types.Command merged into Skill/commands (enable FEATURE_MCP_SKILLS=1 for listing).
// MCP tool defs (assembleToolPool): -mcp-tools-json=path or GOU_DEMO_MCP_TOOLS_JSON → JSON array merged into Options.Tools when GOU_DEMO_USE_EMBEDDED_TOOLS_API=1 (see mcpcommands.EnvToolsJSONPath).
//
// Session JSONL (optional): GOU_DEMO_RECORD_TRANSCRIPT=1 persists via [goc/sessiontranscript] (~/.claude/projects/.../<session>.jsonl). Streaming parity also wires [query.QueryDeps.OnQueryYield] so each assistant/tool_result yield is logged incrementally (deduped by message UUID); turn end still calls maybeRecordTranscript for a full-store sync. Set GOU_DEMO_SESSION_ID to a UUID or the store gets a random UUID when the default "demo" id is invalid. Use -no-seed for cleaner UUIDs in demo history.
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
	"slices"
	"strings"
	"time"

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
	// GOU_DEMO_LOG=1: writing to stderr while Bubble Tea uses the alt screen corrupts line order and layout.
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
		fmt.Fprintf(os.Stderr, "[gou-demo] trace -> %s (TTY: stderr+TUI garbles output; use this file or GOU_DEMO_LOG_FILE=...)\n", p)
		gouDemoTrace = log.New(f, "[gou-demo] ", flags)
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

type model struct {
	store  *conversation.Store
	pr     prompt.Model
	width  int
	height int
	cols   int // content width for wrap + virtual scroll

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
	lastEmittedTitlePlain string

	// Transcript screen (TS REPL.tsx Screen prompt|transcript + frozen lengths).
	uiScreen              gouDemoScreen
	transcriptFreezeN     int
	transcriptShowAll     bool
	promptSavedScrollTop  int
	promptSavedSticky     bool
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
	noAltScreenFlag := flag.Bool("no-alt-screen", false, "do not use alt-screen: conversation stays in normal terminal scrollback (visible after exit)")
	mcpCommandsJSON := flag.String("mcp-commands-json", "", "JSON array file of MCP prompt commands (types.Command); overrides "+mcpcommands.EnvCommandsJSONPath)
	mcpToolsJSON := flag.String("mcp-tools-json", "", "JSON array file of MCP tool definitions for assembleToolPool; overrides "+mcpcommands.EnvToolsJSONPath)
	// Backward compat: real LLM used to be opt-in via -ccb-inline; it is now the default. Flag is a no-op.
	ccbInlineCompat := flag.Bool("ccb-inline", false, "deprecated: no-op (real LLM is default). Use -fake-stream for simulation only")
	flag.Parse()
	_ = ccbInlineCompat

	noAltScreen := *noAltScreenFlag
	if !noAltScreen {
		v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_NO_ALT_SCREEN")))
		noAltScreen = v == "1" || v == "true" || v == "yes" || v == "on"
	}

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
	if !noAltScreen {
		opts = append(opts, tea.WithAltScreen())
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
		tr = &sessiontranscript.Store{
			SessionID:   st.ConversationID,
			OriginalCwd: cwd,
			Cwd:         cwd,
		}
	}

	return &model{
		store:               st,
		pr:                  pr,
		sticky:              true,
		heightCache:         make(map[string]int),
		skillListingSent:    make(map[string]struct{}),
		titleH:              1,
		streamH:             4,
		mcpCommandsJSONPath: mcpCommandsJSONPath,
		mcpToolsJSONPath:    mcpToolsJSONPath,
		tsBridge:            tsBridge,
		transcript:          tr,
		permissionMode:      gouDemoPermissionModeFromEnv(),
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
		text := fmt.Sprintf("## Seed %d\n\nScroll the **virtual** list. Inline `code` here.", i+1)
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

func (m *model) inputAreaHeight() int {
	n := m.pr.LineCount() + 2
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
	foot := joinFooterLines(transcriptFooterLines(narrow, m.transcriptShowAll), m.cols)
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
		oldCols := m.cols
		m.width = msg.Width
		m.height = msg.Height
		m.cols = max(12, msg.Width-4)
		_ = m.pr.Update(msg)
		if oldCols > 0 && oldCols != m.cols && len(m.heightCache) > 0 {
			virtualscroll.ScaleHeightCache(m.heightCache, oldCols, m.cols)
		} else {
			m.rebuildHeightCache()
		}
		return m, nil

	case gouPermissionAskMsg:
		m.permAsk = &permissionAskOverlay{
			toolName:  msg.toolName,
			toolUseID: msg.toolUseID,
			input:     msg.input,
			prompt:    msg.prompt,
			replyCh:   msg.replyCh,
		}
		return m, nil

	case tea.KeyMsg:
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
		if m.permAsk == nil && m.uiScreen == gouDemoScreenPrompt && msg.String() == "ctrl+o" {
			m.enterTranscriptScreen()
			m.slashPick = nil
			m.rebuildHeightCache()
			return m, nil
		}
		if m.handleTranscriptKey(msg) {
			return m, nil
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
						if !gouDemoEnvTruthy("GOU_DEMO_SKIP_SKILL_LISTING") {
							listingSent := m.skillListingSent
							if gouDemoEnvTruthy("GOU_DEMO_SKILL_LISTING_EVERY_TURN") {
								listingSent = make(map[string]struct{})
							}
							if s, ok := commands.AppendSkillListingForAPI(skillListing, hasSkillTool, listingSent, nil); ok {
								listing = s
							}
						}
						gouDemoLogStoreMessages("before_ccbhydrate", m.store)
						msgsJSON, errL := ccbhydrate.MessagesJSONWithLeadingMeta(m.store.Messages, userCtxReminder, listing, toolSpecs, normOpts)
						if errL != nil {
							gouDemoTracef("gou-demo: MessagesJSONWithLeadingMeta error: %v", errL)
							m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: skill listing hydrate: %v", errL)))
							m.rebuildHeightCache()
						} else {
							reqID := fmt.Sprintf("turn-%d", time.Now().UnixNano())
							m.store.ClearStreaming()
							// TS: skill_listing attachment is pushed to mutableMessages before callModel (QueryEngine attachment case).
							if strings.TrimSpace(listing) != "" {
								if att, ok := ccbhydrate.SkillListingStoreMessage(listing); ok {
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
								te := toolexecution.ExecutionDeps{InvokeTool: runner.Run}
								// Opt-in TS permissions.ts 1b: whole-tool alwaysAsk on Bash skipped when input looks sandboxed (see toolexecution.BashSandboxRule1b).
								if gouDemoEnvTruthy("GOU_TOOLEXEC_BASH_SANDBOX_1B") {
									te.SandboxingEnabled = true
									te.AutoAllowBashWholeToolAskWhenSandboxed = true
								}
								m.installAskResolver(&te)
								qdeps.ToolexecutionDeps = te
								if m.transcript != nil && gouDemoEnvTruthy("GOU_DEMO_RECORD_TRANSCRIPT") {
									tr := m.transcript
									store := m.store
									qdeps.OnQueryYield = func(ctx context.Context, y query.QueryYield) error {
										if y.Message == nil {
											return nil
										}
										all := slices.Clone(store.Messages)
										all = append(all, *y.Message)
										_, err := tr.RecordTranscript(ctx, []types.Message{*y.Message}, sessiontranscript.RecordOpts{AllMessages: all})
										return err
									}
								}
								msgsForQ := slices.Clone(m.store.Messages)
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
								m.queryBusy = true
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
					return m, cmd
				}
				gouDemoTracef("starting fake streamTick path")
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
				return m, tea.Batch(cmd, tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg { return streamTick{} }))
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
		m.store.AppendMessage(msg.Message)
		m.rebuildHeightCache()
		if m.uiScreen != gouDemoScreenTranscript {
			m.sticky = true
			m.scrollTop = 1 << 30
		}
		return m, nil

	case gouQueryDoneMsg:
		m.queryBusy = false
		if msg.Err != nil {
			m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: query streaming: %v", msg.Err)))
			m.rebuildHeightCache()
		} else if gouDemoEnvTruthy("GOU_DEMO_BELL") {
			fmt.Print("\a")
		}
		gouDemoLogStoreMessages("after_query_stream", m.store)
		if m.transcript != nil {
			m.maybeRecordTranscript()
		}
		m.rebuildHeightCache()
		if m.uiScreen != gouDemoScreenTranscript {
			m.sticky = true
			m.scrollTop = 1 << 30
		}
		return m, nil

	case streamTick:
		if len(m.streamChunks) == 0 || m.streamIdx >= len(m.streamChunks) {
			return m, nil
		}
		m.store.AppendStreamingChunk(m.streamChunks[m.streamIdx])
		m.streamIdx++
		if m.streamIdx < len(m.streamChunks) {
			return m, tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg { return streamTick{} })
		}
		raw, _ := json.Marshal([]map[string]string{{"type": "text", "text": strings.TrimSpace(m.store.StreamingText)}})
		m.store.AppendMessage(types.Message{
			UUID:    fmt.Sprintf("a-%d", time.Now().UnixNano()),
			Type:    types.MessageTypeAssistant,
			Content: raw,
		})
		m.store.ClearStreaming()
		m.streamChunks = nil
		m.streamIdx = 0
		m.queryBusy = false
		m.rebuildHeightCache()
		gouDemoTracef("fake streamTick finished storeMessages=%d", len(m.store.Messages))
		if m.transcript != nil {
			m.maybeRecordTranscript()
		}
		if m.uiScreen != gouDemoScreenTranscript {
			m.sticky = true
			m.scrollTop = 1 << 30
		}
		return m, nil

	case ccbstream.Msg:
		ev := ccbstream.StreamEvent(msg)
		if gouDemoTrace != nil {
			switch ev.Type {
			case "assistant_delta":
				gouDemoTracef("ui ccbstream.Msg assistant_delta textLen=%d", len(ev.Text))
			case "error":
				gouDemoTracef("ui ccbstream.Msg error code=%q message=%q", ev.Code, ev.Message)
			default:
				gouDemoTracef("ui ccbstream.Msg type=%s", ev.Type)
			}
		}
		ccbstream.Apply(m.store, ev)
		if ev.Type == "turn_complete" || ev.Type == "response_end" {
			gouDemoLogStoreMessages("after_stream_"+ev.Type, m.store)
		}
		m.rebuildHeightCache()
		if m.transcript != nil && (ev.Type == "turn_complete" || ev.Type == "response_end") {
			m.maybeRecordTranscript()
		}
		// Model events often arrive while the user has scrolled up; always jump to bottom so the reply is visible.
		if m.uiScreen != gouDemoScreenTranscript {
			switch ev.Type {
			case "assistant_delta", "tool_use", "tool_result", "turn_complete", "error":
				m.sticky = true
				m.scrollTop = 1 << 30
			}
		}
		return m, nil
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
	return lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf(
		"theme=%s msgs=%d items=%d cols=%d sticky=%v",
		theme.ActiveTheme(), n, vk, m.cols, m.sticky))
}

func (m *model) rebuildHeightCache() {
	keys := m.store.ItemKeys()
	virtualscroll.PruneHeightCache(m.heightCache, keys)
	cols := m.cols
	if cols < 1 {
		cols = 40
	}
	for i := range m.store.Messages {
		m.heightCache[keys[i]] = m.measureMessageRows(m.store.Messages[i], cols)
	}
}

// measureMessageRows matches final View styling (ANSI + wrap) for VirtualMessageList heightCache parity.
func (m *model) measureMessageRows(msg types.Message, cols int) int {
	header := lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(msg.Type)).Render(string(msg.Type))
	body := formatMessageSegments(messagerow.SegmentsFromMessage(msg), cols)
	block := header + "\n" + body
	return max(1, layout.WrappedRowCount(block, cols))
}

func (m *model) refineVisibleHeights(keys []string, start, end, n int) {
	cols := m.cols
	if cols < 1 {
		return
	}
	for i := start; i < end && i < n; i++ {
		m.heightCache[keys[i]] = m.measureMessageRows(m.store.Messages[i], cols)
	}
}

func (m *model) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	keys := m.scrollItemKeys()
	n := len(keys)
	vpH := listViewportH(m)

	// Phase 3: refine last frame's visible rows (TS measureRef), then recompute range with fresh heights.
	if m.prevRange != nil && n > 0 {
		m.refineVisibleHeights(keys, m.prevRange.Start, m.prevRange.End, n)
	}

	vr := virtualscroll.ComputeRange(virtualscroll.RangeInput{
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
	})

	m.mountedKeys = make(map[string]struct{}, max(0, vr.End-vr.Start))
	for i := vr.Start; i < vr.End && i < n; i++ {
		m.mountedKeys[keys[i]] = struct{}{}
	}

	if m.prevRange == nil {
		m.prevRange = &virtualscroll.Range{}
	}
	m.prevRange.Start, m.prevRange.End = vr.Start, vr.End

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

	var msgPane strings.Builder
	if gouDemoScrollStatsEnabled() {
		msgPane.WriteString(lipgloss.NewStyle().Faint(true).Render(
			fmt.Sprintf("messages %d  cols=%d  visible [%d,%d)  topSpacer=%d bottomSpacer=%d sticky=%v",
				n, m.cols, vr.Start, vr.End, vr.TopSpacer, vr.BottomSpacer, m.sticky)))
		msgPane.WriteByte('\n')
	}

	cols := m.cols
	for i := vr.Start; i < vr.End && i < n; i++ {
		msg := m.store.Messages[i]
		key := keys[i]
		h := m.heightCache[key]
		block := renderMessageRow(msg, cols, h)
		msgPane.WriteString(block)
		if i+1 < vr.End {
			msgPane.WriteByte('\n')
		}
	}
	// Same transcript as TS: show in-flight assistant text in the main pane, not only the small stream: strip.
	if m.uiScreen != gouDemoScreenTranscript && strings.TrimSpace(m.store.StreamingText) != "" {
		msgPane.WriteByte('\n')
		head := lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
		msgPane.WriteString(head)
		msgPane.WriteByte('\n')
		msgPane.WriteString(styleMarkdownTokens(markdown.CachedLexerStreaming(m.store.StreamingText), cols))
	}

	// pad message pane to fixed height for stable layout
	lines := strings.Split(msgPane.String(), "\n")
	for len(lines) < vpH {
		lines = append(lines, "")
	}
	if len(lines) > vpH {
		// Height cache can be slightly low vs lipgloss; sticky bottom must keep the *tail* or newest lines vanish.
		if m.sticky {
			lines = lines[len(lines)-vpH:]
		} else {
			lines = lines[:vpH]
		}
	}
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteByte('\n')

	if m.uiScreen != gouDemoScreenTranscript {
		streamLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("stream: ")
		var streamBody string
		if m.store.StreamingText != "" {
			toks := markdown.CachedLexerStreaming(m.store.StreamingText)
			streamBody = styleMarkdownTokens(toks, m.cols)
		} else {
			streamBody = lipgloss.NewStyle().Faint(true).Render("(idle)")
		}
		streamWrapped := layout.WrapForViewport(streamLabel+streamBody, m.width-2)
		streamRows := strings.Split(streamWrapped, "\n")
		for len(streamRows) < m.streamH {
			streamRows = append(streamRows, "")
		}
		if len(streamRows) > m.streamH {
			streamRows = streamRows[:m.streamH]
		}
		b.WriteString(strings.Join(streamRows, "\n"))
		b.WriteByte('\n')
	}
	if s := m.statusLineString(); s != "" {
		b.WriteString(s)
		b.WriteByte('\n')
	}

	if m.uiScreen == gouDemoScreenTranscript {
		foot := joinFooterLines(transcriptFooterLines(narrow, m.transcriptShowAll), m.cols)
		b.WriteString(lipgloss.NewStyle().Faint(true).Width(m.cols).Render(foot))
	} else {
		promptView := m.pr.View()
		hintText := replChromeFooterHint(narrow)
		if frag := replChromePermissionFragment(m.permissionMode, narrow); frag != "" {
			hintText = frag + " · " + hintText
		}
		hint := lipgloss.NewStyle().Faint(true).Width(m.cols).Render(hintText)
		b.WriteString(promptView)
		b.WriteByte('\n')
		b.WriteString(hint)
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

func renderMessageRow(m types.Message, cols, maxRows int) string {
	header := lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(m.Type)).Render(string(m.Type))
	body := formatMessageSegments(messagerow.SegmentsFromMessage(m), cols)
	block := header + "\n" + body
	wrapped := layout.WrapForViewport(block, cols)
	rows := strings.Split(wrapped, "\n")
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	return strings.Join(rows, "\n")
}

// formatMessageSegments mirrors Message.tsx per-block branches (text→markdown, tool_use/tool_result/thinking).
func formatMessageSegments(segs []messagerow.Segment, cols int) string {
	var parts []string
	for _, seg := range segs {
		switch seg.Kind {
		case messagerow.SegTextMarkdown:
			parts = append(parts, styleMarkdownTokens(markdown.CachedLexer(seg.Text), cols))
		case messagerow.SegToolUse:
			parts = append(parts, lipgloss.NewStyle().Foreground(theme.ToolUseAccent()).Bold(true).Render("⚙ "+seg.Text))
		case messagerow.SegToolResult:
			st := lipgloss.NewStyle().Foreground(theme.DimMuted())
			if seg.IsToolError {
				st = lipgloss.NewStyle().Foreground(theme.ToolError())
			}
			body := textutil.LinkifyOSC8(seg.Text)
			parts = append(parts, st.Render("↩ "+body))
		case messagerow.SegThinking:
			body := textutil.LinkifyOSC8(seg.Text)
			parts = append(parts, lipgloss.NewStyle().Bold(true).Render("● "+body))
		case messagerow.SegDisplayHint:
			parts = append(parts, lipgloss.NewStyle().Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(seg.Text)))
		case messagerow.SegServerToolUse:
			parts = append(parts, lipgloss.NewStyle().Foreground(theme.ServerAccent()).Bold(true).Render("⎈ "+seg.Text))
		case messagerow.SegAdvisorToolResult:
			st := lipgloss.NewStyle().Foreground(theme.AdvisorAccent())
			if seg.IsToolError {
				st = lipgloss.NewStyle().Foreground(theme.ToolError())
			}
			body := textutil.LinkifyOSC8(seg.Text)
			parts = append(parts, st.Render("✧ "+body))
		case messagerow.SegGroupedToolUse:
			parts = append(parts, lipgloss.NewStyle().Foreground(theme.GroupedAccent()).Bold(true).Render("▦ "+seg.Text))
		case messagerow.SegCollapsedReadSearch:
			parts = append(parts, lipgloss.NewStyle().Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(seg.Text)))
		default:
			parts = append(parts, lipgloss.NewStyle().Faint(true).Render(textutil.LinkifyOSC8(seg.Text)))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
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
