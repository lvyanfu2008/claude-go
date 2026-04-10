// Command gou-demo is a minimal Bubble Tea full-screen UI: virtualscroll + markdown + tool blocks (Phase 4 messagerow).
// Model replies appear inside the TUI. Default uses the terminal alt-screen (full-screen); after quit, that buffer is gone — use -no-alt-screen (or GOU_DEMO_NO_ALT_SCREEN=1) so output stays in normal shell scrollback.
// With GOU_DEMO_LOG=1, trace uses the same path rules as TS debug log (see goc/ccb-engine/debugpath); on TTY without GOU_DEMO_LOG_FILE, trace goes to that file, not stderr.
//
// Run from repo: cd goc && go run ./cmd/gou-demo
//
// Flags: -transcript=file.json (UI or API messages), -no-seed (or GOU_DEMO_NO_SEED=1) to skip the 45 demo rows so localturn does not send fake history,
// -replay-cc=events.ndjson, -stream-stdin (pipe NDJSON),
// By default gou-demo calls the real model in-process (goc/ccb-engine/localturn). Use -fake-stream (or GOU_DEMO_USE_FAKE_STREAM=1)
// for a UI-only simulated stream with no HTTP (no apilog bodies on send).
// Go-side init port (subset of TS init.ts): GOU_DEMO_GO_INIT=1 runs [goc/claudeinit.Init] instead of only [settingsfile.EnsureProjectClaudeEnvOnce] (Init includes Ensure). See docs/plans/go-init-port.md. Compatible with GOU_DEMO_TS_CONTEXT_BRIDGE (Bun snapshot runs after this block).
// Full TS system prompt + commands + tools (startup only): GOU_DEMO_TS_CONTEXT_BRIDGE=1 runs `bun run go-context-bridge` once at startup (default 5m timeout, override GOU_DEMO_TS_BRIDGE_TIMEOUT_SEC); Bun stderr streams to the terminal. Snapshot is cached in-process for all turns (no per-turn Bun). Requires repo root (ancestor with scripts/slash-resolve-bridge.ts). MCP/settings changes mid-session are not reflected until restart.
// Go local tool parity (embedded localturn, no socket): Bash is allowed by default (same as TS); set GOU_DEMO_NO_LOCAL_BASH=1 to disable unless CCB_ENGINE_LOCAL_BASH=1. AskUserQuestion auto-picks the first option per question unless GOU_DEMO_NO_ASK_AUTO_FIRST=1 (then use TS socket worker for real prompts). WebFetch needs CCB_ENGINE_WEB_FETCH=1 or GOU_DEMO_WEB_FETCH=1. See docs/plans/go-tools-parity.md.
// Real TS tools: GOU_DEMO_CCB_SOCKET=1 and CCB_ENGINE_SOCKET=/path/to.sock. gou-demo embeds the socket protocol (goc/ccb-engine/socketserve) when nothing is already listening on that path; if the socket is already accepting connections (e.g. another gou-demo or ccb-socket-host), this process does not remove it. Repo needs package.json (bun run ccb-engine-tool-worker). Override root with CLAUDE_CODE_REPO_ROOT.
// Optional GOU_DEMO_CCB_PERSIST_WORKER=1: one long-lived `bun ccb-engine-tool-worker` with CCB_WORKER_STDIN_LOOP (faster than spawning each turn; serializes turns on one stdin).
// Persist turns must not block [tea.Model.Update]: waiting on response_end while the same goroutine should drain programSend would deadlock the TUI; submit runs in a background goroutine with [ccbPersistTurnGate].
// Bun worker stderr does not go to the TUI terminal by default (would garble Bubble Tea). Default log: next to gou-demo trace — dirname(defaultGouDemoTracePath)/ccb-engine-tool-worker.stderr.log. Override with GOU_DEMO_CCB_WORKER_LOG_FILE=...; set GOU_DEMO_CCB_WORKER_STDERR=1 to inherit os.Stderr (debug only).
//
// System # Language / # Output Style: merged from ~/.claude/settings.json and project .claude/settings.go.json / settings.local.json (see settingsfile; project settings.json is TS-only). CLAUDE_CODE_LANGUAGE and CLAUDE_CODE_OUTPUT_STYLE_* override when set (non-empty); built-in outputStyle keys Explanatory/Learning use prompts from src/constants/outputStyles.ts (embedded).
// Extra CLAUDE.md roots: optional runtimeContext.toolPermissionContext.additionalWorkingDirectories (JSON) and/or GOU_DEMO_EXTRA_CLAUDE_MD_ROOTS / CLAUDE_CODE_EXTRA_CLAUDE_MD_ROOTS (comma or PATH-style list); with CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1 they are scanned like TS add-dir (see [querycontext.ExtraClaudeMdRootsForFetch]).
// Debug log (optional): GOU_DEMO_LOG_FILE=/path/to.log, or GOU_DEMO_LOG=1 (default file path matches TS getDebugLogPath via goc/ccb-engine/debugpath when stderr is TTY). GOU_DEMO_LOG_STDERR=1 forces stderr (may corrupt TUI). Lines are prefixed [gou-demo].
// ToolUseContext dump: CLAUDE_CODE_LOG_TOOL_USE_CONTEXT or GOU_DEMO_LOG_TOOL_USE_CONTEXT = 1|summary|full (with logging enabled) prints JSON after each BuildDemoParams; full includes the entire commands[] snapshot.
// Virtual-scroll stats line (messages N, visible [a,b), spacers…): set GOU_DEMO_SCROLL_STATS=1 (default off).
//
// Keys: ↑/↓/PgUp/PgDn scroll the message pane, End sticky-to-bottom, q quit, Enter send prompt.
// Slash: /name is resolved in-process — disk skills via [goc/slashresolve.ResolveDiskSkill], bundled skills via [goc/slashresolve.ResolveBundledSkill] (embedded TS-expanded prompts under slashresolve/bundleddata). Optional Bun scripts/slash-resolve-bridge.ts remains for commands not covered by the Go embed.
// MCP skills (scheme-2 R0/R1): -mcp-commands-json=path or GOU_DEMO_MCP_COMMANDS_JSON → JSON array of types.Command merged into Skill/commands (enable FEATURE_MCP_SKILLS=1 for listing).
// MCP tool defs (assembleToolPool): -mcp-tools-json=path or GOU_DEMO_MCP_TOOLS_JSON → JSON array merged into Options.Tools when GOU_DEMO_USE_EMBEDDED_TOOLS_API=1 (see mcpcommands.EnvToolsJSONPath).
//
// Session JSONL (optional): GOU_DEMO_RECORD_TRANSCRIPT=1 persists turns via [goc/sessiontranscript] (~/.claude/projects/.../<session>.jsonl). Set GOU_DEMO_SESSION_ID to a UUID or the store gets a random UUID when the default "demo" id is invalid. Use -no-seed for cleaner UUIDs in demo history.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"

	"goc/ccb-engine/apilog"
	"goc/ccb-engine/debugpath"
	"goc/ccb-engine/localturn"
	"goc/ccb-engine/socketserve"
	"goc/ccb-engine/settingsfile"
	"goc/claudeinit"
	"goc/ccb-engine/skilltools"
	"goc/commands"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/gou/ccbhydrate"
	"goc/gou/ccbstream"
	"goc/messagesapi"
	"goc/gou/conversation"
	"goc/gou/layout"
	"goc/gou/markdown"
	"goc/gou/messagerow"
	"goc/gou/pui"
	"goc/gou/transcript"
	"goc/gou/virtualscroll"
	"goc/mcpcommands"
	"goc/querycontext"
	"goc/sessiontranscript"
	"goc/tscontext"
	"goc/types"
)

// gouDemoTrace is set by setupGouDemoTrace from GOU_DEMO_LOG_FILE or GOU_DEMO_LOG.
var gouDemoTrace *log.Logger

// gouDemoMergedSystemLocale mirrors apiparity.GouDemo: user + project settings.go.json / settings.local.json language/outputStyle with env override.
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

// Bun ccb-engine-tool-worker stderr must not default to os.Stderr: full-screen TUI + child stderr interleave and look like "the shell ran bun".
var (
	ccbWorkerStderrOnce sync.Once
	ccbWorkerStderrSink io.Writer = io.Discard
)

func resolveCcbWorkerStderrLogPath() string {
	if p := strings.TrimSpace(os.Getenv("GOU_DEMO_CCB_WORKER_LOG_FILE")); p != "" {
		return p
	}
	return filepath.Join(filepath.Dir(defaultGouDemoTracePath()), "ccb-engine-tool-worker.stderr.log")
}

func initCcbWorkerStderrSink() {
	p := resolveCcbWorkerStderrLogPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	ccbWorkerStderrSink = f
	gouDemoTracef("ccb-engine-tool-worker stderr -> %s", p)
}

func gouDemoCcbWorkerStderr() io.Writer {
	if gouDemoEnvTruthy("GOU_DEMO_CCB_WORKER_STDERR") {
		return os.Stderr
	}
	ccbWorkerStderrOnce.Do(initCcbWorkerStderrSink)
	return ccbWorkerStderrSink
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
				"           No HTTP → apilog will not append request/response lines. Omit -fake-stream for real localturn + logs.\n")
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

// runLocalTurnProgram runs goc/ccb-engine/localturn in a goroutine and forwards events to the Bubble Tea program.
func runLocalTurnProgram(programSend func(tea.Msg), p localturn.Params) {
	go func() {
		ctx := context.Background()
		_ = localturn.RunSubmitUserTurn(ctx, p, func(ev localturn.StreamEvent) {
			b, err := json.Marshal(ev)
			if err != nil {
				return
			}
			var c ccbstream.StreamEvent
			if json.Unmarshal(b, &c) != nil {
				return
			}
			if programSend != nil {
				programSend(ccbstream.Msg(c))
			}
		})
	}()
}

// gouDemoUseCcbSocketWorker is true when GOU_DEMO_CCB_SOCKET=1 and CCB_ENGINE_SOCKET is set (engine socket for the Bun worker).
func gouDemoUseCcbSocketWorker() bool {
	return gouDemoEnvTruthy("GOU_DEMO_CCB_SOCKET") && strings.TrimSpace(os.Getenv("CCB_ENGINE_SOCKET")) != ""
}

// unixSocketAlreadyListening returns true if a process accepts connections on the Unix socket path.
func unixSocketAlreadyListening(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	c, err := net.DialTimeout("unix", path, 150*time.Millisecond)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

func waitUnixSocketListening(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if unixSocketAlreadyListening(path) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout after %v", timeout)
}

func gouDemoEmbeddedSocketserveLogf(format string, args ...any) {
	gouDemoTracef(format, args...)
	if strings.HasPrefix(format, "socketserve listening") || strings.HasPrefix(format, "socketserve accept:") {
		fmt.Fprintf(os.Stderr, "[gou-demo] "+format+"\n", args...)
	}
}

// gouDemoWarnMissingCcbSocket prints stderr if socket mode is on but the Bun worker cannot reach the engine (dial fails).
func gouDemoWarnMissingCcbSocket() {
	if !gouDemoUseCcbSocketWorker() {
		return
	}
	p := strings.TrimSpace(os.Getenv("CCB_ENGINE_SOCKET"))
	if p == "" {
		return
	}
	if unixSocketAlreadyListening(p) {
		return
	}
	fmt.Fprintf(os.Stderr,
		"gou-demo: warning: cannot connect to CCB_ENGINE_SOCKET %q (embedded listener failed or another process holds the socket). Fix the path or run `cd goc && go run ./cmd/ccb-socket-host -socket %q` for a headless host.\n",
		p, p)
}

// gouDemoCcbPersistWorker uses one bun worker process with stdin loop instead of spawning per turn.
func gouDemoCcbPersistWorker() bool {
	return gouDemoEnvTruthy("GOU_DEMO_CCB_PERSIST_WORKER")
}

// ccbSocketPersist holds a single long-lived ccb-engine-tool-worker (CCB_WORKER_STDIN_LOOP=1).
var ccbSocketPersist struct {
	mu          sync.Mutex
	started     bool
	stdin       io.WriteCloser
	cmd         *exec.Cmd
	repoRoot    string
	socketPath  string
	responseEnd chan string
}

// ccbPersistTurnGate serializes persist-worker turns when submit runs off the UI thread (see file header).
var ccbPersistTurnGate sync.Mutex

func drainCcbPersistResponseEnds() {
	for {
		select {
		case <-ccbSocketPersist.responseEnd:
		default:
			return
		}
	}
}

func stopCcbSocketPersistWorkerLocked() {
	if ccbSocketPersist.stdin != nil {
		_ = ccbSocketPersist.stdin.Close()
	}
	if ccbSocketPersist.cmd != nil && ccbSocketPersist.cmd.Process != nil {
		_ = ccbSocketPersist.cmd.Process.Kill()
	}
	ccbSocketPersist.stdin = nil
	ccbSocketPersist.cmd = nil
	ccbSocketPersist.started = false
	drainCcbPersistResponseEnds()
}

func markCcbSocketPersistWorkerDead() {
	ccbSocketPersist.mu.Lock()
	defer ccbSocketPersist.mu.Unlock()
	stopCcbSocketPersistWorkerLocked()
}

// runCcbPersistWorkerStdoutScanner forwards NDJSON lines to the TUI; signals response_end ids on ccbSocketPersist.responseEnd.
func runCcbPersistWorkerStdoutScanner(stdout io.ReadCloser, programSend func(tea.Msg)) {
	defer stdout.Close()
	defer markCcbSocketPersistWorkerDead()

	s := bufio.NewScanner(stdout)
	const max = 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	s.Buffer(buf, max)
	for s.Scan() {
		var ev ccbstream.StreamEvent
		if json.Unmarshal(s.Bytes(), &ev) != nil || ev.Type == "" {
			continue
		}
		if programSend != nil {
			programSend(ccbstream.Msg(ev))
		}
		if ev.Type == "response_end" {
			select {
			case ccbSocketPersist.responseEnd <- ev.ID:
			default:
			}
		}
	}
}

func ensureCcbSocketPersistWorker(programSend func(tea.Msg), repoRoot, socketPath string) error {
	ccbSocketPersist.mu.Lock()
	defer ccbSocketPersist.mu.Unlock()

	if ccbSocketPersist.started && ccbSocketPersist.repoRoot == repoRoot && ccbSocketPersist.socketPath == socketPath {
		return nil
	}
	stopCcbSocketPersistWorkerLocked()

	cmd := exec.Command("bun", "run", "ccb-engine-tool-worker", socketPath)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"CCB_WORKER_CWD="+repoRoot,
		"CCB_WORKER_STDIN_LOOP=1",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return err
	}
	cmd.Stderr = gouDemoCcbWorkerStderr()
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return err
	}
	ccbSocketPersist.stdin = stdin
	ccbSocketPersist.cmd = cmd
	ccbSocketPersist.repoRoot = repoRoot
	ccbSocketPersist.socketPath = socketPath
	ccbSocketPersist.responseEnd = make(chan string, 32)
	ccbSocketPersist.started = true
	gouDemoTracef("ccb persist worker: started pid=%d repo=%s socket=%s", cmd.Process.Pid, repoRoot, socketPath)
	go runCcbPersistWorkerStdoutScanner(stdout, programSend)
	return nil
}

// submitCcbSocketPersistTurn writes one SubmitUserTurn line and waits for response_end (matching id or empty id on worker error).
// Must not hold ccbSocketPersist.mu while waiting so stdout scanner can mark the worker dead on EOF.
func submitCcbSocketPersistTurn(programSend func(tea.Msg), repoRoot, socketPath string, line []byte, reqID string) error {
	if err := ensureCcbSocketPersistWorker(programSend, repoRoot, socketPath); err != nil {
		return err
	}

	var endCh <-chan string
	var werr error
	func() {
		ccbSocketPersist.mu.Lock()
		defer ccbSocketPersist.mu.Unlock()
		if !ccbSocketPersist.started || ccbSocketPersist.stdin == nil {
			werr = fmt.Errorf("ccb persist worker: not running")
			return
		}
		drainCcbPersistResponseEnds()
		if _, err := ccbSocketPersist.stdin.Write(line); err != nil {
			werr = err
			return
		}
		endCh = ccbSocketPersist.responseEnd
	}()
	if werr != nil {
		return werr
	}
	if endCh == nil {
		return fmt.Errorf("ccb persist worker: not running")
	}

	deadline := time.After(35 * time.Minute)
	for {
		select {
		case id := <-endCh:
			if id == reqID || id == "" {
				return nil
			}
		case <-deadline:
			return fmt.Errorf("ccb persist worker: timeout waiting response_end for %s", reqID)
		}
	}
}

func findClaudeCodeRepoRoot(start string) (string, error) {
	if r := strings.TrimSpace(os.Getenv("CLAUDE_CODE_REPO_ROOT")); r != "" {
		return r, nil
	}
	dir := start
	for range 32 {
		pkg := filepath.Join(dir, "package.json")
		st, err := os.Stat(pkg)
		if err == nil && !st.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("find package.json above %s", start)
}

// runCcbSocketWorkerProgram spawns `bun run ccb-engine-tool-worker` (repo root) to drive the Go socket bridge with real TS tool execution.
func runCcbSocketWorkerProgram(programSend func(tea.Msg), reqID string, msgsJSON, toolsJSON json.RawMessage, guidance string) {
	socketPath := strings.TrimSpace(os.Getenv("CCB_ENGINE_SOCKET"))
	if socketPath == "" || programSend == nil {
		return
	}
	cwd, errWd := os.Getwd()
	if errWd != nil {
		cwd = "."
	}
	repoRoot, err := findClaudeCodeRepoRoot(cwd)
	if err != nil {
		gouDemoTracef("ccb socket worker: repo root: %v", err)
		return
	}
	type payload struct {
		Messages json.RawMessage `json:"messages,omitempty"`
		Tools    json.RawMessage `json:"tools,omitempty"`
		System   string          `json:"system,omitempty"`
	}
	type envelope struct {
		Method  string  `json:"method"`
		ID      string  `json:"id"`
		Payload payload `json:"payload"`
	}
	env := envelope{
		Method: "SubmitUserTurn",
		ID:     reqID,
		Payload: payload{
			Messages: msgsJSON,
			Tools:    toolsJSON,
			System:   guidance,
		},
	}
	line, err := json.Marshal(env)
	if err != nil {
		gouDemoTracef("ccb socket worker: marshal: %v", err)
		return
	}
	line = append(line, '\n')

	if gouDemoCcbPersistWorker() {
		lineCopy := bytes.Clone(line)
		repo := repoRoot
		sock := socketPath
		rid := reqID
		ps := programSend
		go func() {
			ccbPersistTurnGate.Lock()
			defer ccbPersistTurnGate.Unlock()
			if err := submitCcbSocketPersistTurn(ps, repo, sock, lineCopy, rid); err != nil {
				gouDemoTracef("ccb persist worker: %v", err)
			}
		}()
		return
	}

	cmd := exec.Command("bun", "run", "ccb-engine-tool-worker", socketPath)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "CCB_WORKER_CWD="+repoRoot)
	cmd.Stdin = bytes.NewReader(line)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		gouDemoTracef("ccb socket worker: stdout pipe: %v", err)
		return
	}
	cmd.Stderr = gouDemoCcbWorkerStderr()
	if err := cmd.Start(); err != nil {
		gouDemoTracef("ccb socket worker: start: %v", err)
		return
	}
	go func() {
		defer func() { _ = cmd.Wait() }()
		s := bufio.NewScanner(stdout)
		const max = 1024 * 1024
		buf := make([]byte, 0, 64*1024)
		s.Buffer(buf, max)
		for s.Scan() {
			var ev ccbstream.StreamEvent
			if json.Unmarshal(s.Bytes(), &ev) != nil || ev.Type == "" {
				continue
			}
			programSend(ccbstream.Msg(ev))
		}
	}()
}

type streamTick struct{}

type model struct {
	store  *conversation.Store
	ti     textinput.Model
	width  int
	height int
	cols   int // content width for wrap + virtual scroll

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
	inputH  int

	// ccbSend / ccbInline set by BindCCB after tea.NewProgram (localturn in-process when ccbInline).
	ccbSend   func(tea.Msg)
	ccbInline bool

	// skillListingSent tracks skill names already injected into the API transcript (TS sentSkillNames).
	skillListingSent map[string]struct{}

	// mcpCommandsJSONPath is -mcp-commands-json (overrides GOU_DEMO_MCP_COMMANDS_JSON when set).
	mcpCommandsJSONPath string
	// mcpToolsJSONPath is -mcp-tools-json (overrides GOU_DEMO_MCP_TOOLS_JSON when set).
	mcpToolsJSONPath string

	// tsBridge when non-nil (GOU_DEMO_TS_CONTEXT_BRIDGE=1) supplies TS fetchSystemPromptParts + commands + tools from startup Bun bridge.
	tsBridge *tscontext.Snapshot

	// transcript when non-nil (GOU_DEMO_RECORD_TRANSCRIPT=1) appends messages after each completed turn.
	transcript *sessiontranscript.Store
}

func main() {
	if gouDemoEnvTruthy("GOU_DEMO_GO_INIT") {
		if err := claudeinit.Init(context.Background(), claudeinit.Options{NonInteractive: true}); err != nil {
			log.Fatalf("gou-demo: claudeinit (GOU_DEMO_GO_INIT): %v", err)
		}
	} else {
		if err := settingsfile.EnsureProjectClaudeEnvOnce(); err != nil {
			log.Fatalf("gou-demo: project settings: %v", err)
		}
	}
	// Env merge matches [settingsfile.ApplyMergedClaudeSettingsEnv]: user ~/.claude/settings.json,
	// project .claude/settings.go.json, settings.local.json. Project .claude/settings.json is TS-only
	// (see settingsfile package doc); put GOU_DEMO_* / CCB_ENGINE_* in settings.go.json or export in shell.
	apilog.PrepareIfEnabled()
	apilog.MaybePrintDiag()
	traceCleanup := setupGouDemoTrace()
	defer traceCleanup()

	serveCtx, stopEmbeddedCcbServe := context.WithCancel(context.Background())
	defer stopEmbeddedCcbServe()
	if gouDemoUseCcbSocketWorker() {
		sp := strings.TrimSpace(os.Getenv("CCB_ENGINE_SOCKET"))
		fmt.Fprintf(os.Stderr, "[gou-demo] CCB socket mode: GOU_DEMO_CCB_SOCKET=1, CCB_ENGINE_SOCKET=%q\n", sp)
		if unixSocketAlreadyListening(sp) {
			gouDemoTracef("CCB_ENGINE_SOCKET already accepting at %s (embedded socketserve skipped)", sp)
			fmt.Fprintf(os.Stderr, "[gou-demo] using existing listener on %s (embedded socketserve skipped)\n", sp)
		} else {
			fmt.Fprintf(os.Stderr, "[gou-demo] starting embedded socketserve on %s …\n", sp)
			go func() {
				_ = socketserve.Run(serveCtx, sp, gouDemoEmbeddedSocketserveLogf)
			}()
			if err := waitUnixSocketListening(sp, 30*time.Second); err != nil {
				log.Fatalf("gou-demo: wait for embedded ccb socket %q: %v", sp, err)
			}
			gouDemoTracef("embedded socketserve ready on %s", sp)
			fmt.Fprintf(os.Stderr, "[gou-demo] embedded listener ready on %s (valid while this process runs; exit gou-demo closes it)\n", sp)
		}
	} else if gouDemoEnvTruthy("GOU_DEMO_CCB_SOCKET") && strings.TrimSpace(os.Getenv("CCB_ENGINE_SOCKET")) == "" {
		fmt.Fprintf(os.Stderr, "[gou-demo] GOU_DEMO_CCB_SOCKET is set but CCB_ENGINE_SOCKET is empty — socket bridge disabled. Set CCB_ENGINE_SOCKET in shell or .claude/settings.go.json env.\n")
	}

	transcriptPath := flag.String("transcript", "", "load messages from JSON file (UI []Message or API [{role,content}]); skips built-in seed")
	noSeed := flag.Bool("no-seed", false, "start with an empty transcript (no 45 demo seed messages); avoids sending fake history to localturn. Same as GOU_DEMO_NO_SEED=1")
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

	var tsBridge *tscontext.Snapshot
	if gouDemoEnvTruthy("GOU_DEMO_TS_CONTEXT_BRIDGE") {
		cwd, _ := os.Getwd()
		repoRoot := pui.FindRepoRootForBridge(cwd)
		if strings.TrimSpace(repoRoot) == "" {
			log.Fatalf("gou-demo: GOU_DEMO_TS_CONTEXT_BRIDGE=1 but repo root not found (walk upward from cwd=%q for scripts/slash-resolve-bridge.ts)", cwd)
		}
		fmt.Fprintf(os.Stderr,
			"[gou-demo] TS context bridge: running `bun run go-context-bridge` in %q (cwd=%q) — Bun stderr streams below until TS init finishes (cold start often 1–5 min; timeout %v, override GOU_DEMO_TS_BRIDGE_TIMEOUT_SEC=seconds). To skip: unset GOU_DEMO_TS_CONTEXT_BRIDGE.\n",
			repoRoot, cwd, tscontext.EffectiveBridgeExecTimeout())
		var err error
		extraRoots := querycontext.ExtraClaudeMdRootsForFetch(nil)
		tsBridge, err = tscontext.LoadSnapshotOnce(context.Background(), repoRoot, cwd, extraRoots)
		if err != nil {
			log.Fatalf("gou-demo: TS context bridge: %v", err)
		}
		gouDemoTracef("TS context bridge: loaded snapshot commandsBytes=%d toolsBytes=%d model=%q",
			len(tsBridge.Commands), len(tsBridge.Tools), tsBridge.MainLoopModel)
		fmt.Fprintf(os.Stderr, "[gou-demo] TS context bridge: cached system prompt + commands + tools from Bun (restart to refresh)\n")
	}

	m := newModel(st, strings.TrimSpace(*mcpCommandsJSON), strings.TrimSpace(*mcpToolsJSON), tsBridge)

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
	gouDemoWarnMissingCcbSocket()
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
	ti := textinput.New()
	ti.Placeholder = "message — Enter to send, q quit"
	ti.Focus()
	ti.CharLimit = 4000
	ti.Width = 60

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
		ti:                  ti,
		sticky:              true,
		heightCache:         make(map[string]int),
		skillListingSent:    make(map[string]struct{}),
		titleH:              1,
		streamH:             4,
		inputH:              3,
		mcpCommandsJSONPath: mcpCommandsJSONPath,
		mcpToolsJSONPath:    mcpToolsJSONPath,
		tsBridge:            tsBridge,
		transcript:          tr,
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

// BindCCB wires Bubble Tea Send and whether to use localturn for real model turns.
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
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		oldCols := m.cols
		m.width = msg.Width
		m.height = msg.Height
		m.cols = max(12, msg.Width-4)
		m.ti.Width = m.cols
		if oldCols > 0 && oldCols != m.cols && len(m.heightCache) > 0 {
			virtualscroll.ScaleHeightCache(m.heightCache, oldCols, m.cols)
		} else {
			m.rebuildHeightCache()
		}
		return m, textinput.Blink

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			m.sticky = false
			m.scrollTop = max(0, m.scrollTop-1)
			return m, nil
		case "down", "j":
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
		if msg.Type == tea.KeyEnter {
			line := strings.TrimSpace(m.ti.Value())
			var cmd tea.Cmd
			m.ti, cmd = m.ti.Update(msg)
			if line == "" {
				return m, cmd
			}
			gouDemoTracef("enter input=%q", previewForTrace(line, 120))
			cwd, _ := os.Getwd()
			repoRoot := pui.FindRepoRootForBridge(cwd)
			mergedLang, mergedOutName, mergedOutPrompt := gouDemoMergedSystemLocale()
			demoCfg := pui.DemoConfig{
				RepoRoot:            repoRoot,
				SessionID:           m.store.ConversationID,
				Language:            mergedLang,
				MCPCommandsJSONPath: m.mcpCommandsJSONPath,
				MCPToolsJSONPath:    m.mcpToolsJSONPath,
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
				RepoRoot:  repoRoot,
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
			gouDemoTracef("after ApplyBaseResult shouldQuery=%v effectiveShouldQuery=%v hadExecutionRequest=%v messagesAppended=%d",
				r != nil && r.ShouldQuery, out.EffectiveShouldQuery, out.HadExecutionRequest, len(r.Messages))
			if out.NextInput != "" {
				m.ti.SetValue(out.NextInput)
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
				normOpts := messagesapi.DefaultOptions()
				if gouDemoEnvTruthy("GOU_DEMO_NON_INTERACTIVE") {
					normOpts.NonInteractive = true
				}
				tryMsgs := func() (json.RawMessage, error) {
					return ccbhydrate.MessagesJSONNormalized(m.store.Messages, toolSpecs, normOpts)
				}
				if m.ccbInline && m.ccbSend != nil {
					baseMsgs, err := tryMsgs()
					if err != nil {
						gouDemoTracef("localturn: ccbhydrate.MessagesJSON error: %v (fake stream)", err)
						m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: ccb messages JSON: %v (fallback fake stream)", err)))
						m.rebuildHeightCache()
					} else if len(bytes.TrimSpace(baseMsgs)) < 3 || bytes.Equal(bytes.TrimSpace(baseMsgs), []byte("[]")) {
						gouDemoTracef("localturn: empty messages JSON bytes=%d (fake stream)", len(baseMsgs))
						m.store.AppendMessage(pui.SystemNotice("gou-demo: empty chat transcript for localturn (fallback fake stream)"))
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
						mainLoopModel := pui.DefaultMainLoopModelForDemo()
						if params.RuntimeContext != nil && strings.TrimSpace(params.RuntimeContext.ToolUseContext.Options.MainLoopModel) != "" {
							mainLoopModel = strings.TrimSpace(params.RuntimeContext.ToolUseContext.Options.MainLoopModel)
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
							if s, ok := commands.AppendSkillListingForAPI(skillListing, hasSkillTool, m.skillListingSent, nil); ok {
								listing = s
							}
						}
						msgsJSON, errL := ccbhydrate.MessagesJSONWithSkillListing(m.store.Messages, listing, toolSpecs, normOpts)
						if errL == nil && strings.TrimSpace(userCtxReminder) != "" {
							msgsJSON, errL = ccbhydrate.PrependUserMessageJSON(msgsJSON, userCtxReminder)
						}
						if errL != nil {
							gouDemoTracef("localturn: MessagesJSONWithSkillListing error: %v", errL)
							m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: skill listing hydrate: %v", errL)))
							m.rebuildHeightCache()
						} else {
							reqID := fmt.Sprintf("turn-%d", time.Now().UnixNano())
							m.store.ClearStreaming()
							gouDemoTracef("localturn.RunSubmitUserTurn start requestID=%s msgsJSONBytes=%d toolsBytes=%d systemBytes=%d",
								reqID, len(msgsJSON), len(toolsJSON), len(guidance))
							if gouDemoUseCcbSocketWorker() {
								gouDemoTracef("ccb socket worker: bun ccb-engine-tool-worker repoRoot=%s socket=%s", repoRoot, strings.TrimSpace(os.Getenv("CCB_ENGINE_SOCKET")))
								runCcbSocketWorkerProgram(m.ccbSend, reqID, msgsJSON, toolsJSON, guidance)
								usedCCB = true
							} else {
								cwdAbs, errAbs := filepath.Abs(cwd)
								if errAbs != nil {
									cwdAbs = cwd
								}
								var extraRoots []string
								if rr := strings.TrimSpace(repoRoot); rr != "" {
									if ra, e := filepath.Abs(rr); e == nil {
										extraRoots = append(extraRoots, ra)
									}
								}
								runner := skilltools.ParityToolRunner{
									DemoToolRunner: skilltools.DemoToolRunner{
										Commands:  params.Commands,
										RepoRoot:  repoRoot,
										SessionID: m.store.ConversationID,
									},
									WorkDir:          cwdAbs,
									ExtraRoots:       extraRoots,
									ProjectRoot:      repoRoot,
									LocalBashDefault: true,
									AskAutoFirst:     !gouDemoEnvTruthy("GOU_DEMO_NO_ASK_AUTO_FIRST"),
								}
								skillExpand := !gouDemoEnvTruthy("GOU_DEMO_NO_SKILL_EXPAND_USER_MSG")
								runLocalTurnProgram(m.ccbSend, localturn.Params{
									RequestID:               reqID,
									Messages:                msgsJSON,
									Tools:                   toolsJSON,
									System:                  guidance,
									SkillExpandUserFollowUp: skillExpand,
									Runner:                  runner,
								})
								usedCCB = true
							}
						}
					}
				}
				if usedCCB {
					return m, cmd
				}
				gouDemoTracef("starting fake streamTick path")
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
		var cmd tea.Cmd
		m.ti, cmd = m.ti.Update(msg)
		return m, cmd

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
		m.rebuildHeightCache()
		gouDemoTracef("fake streamTick finished storeMessages=%d", len(m.store.Messages))
		if m.transcript != nil {
			m.maybeRecordTranscript()
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
		m.rebuildHeightCache()
		if m.transcript != nil && (ev.Type == "turn_complete" || ev.Type == "response_end") {
			m.maybeRecordTranscript()
		}
		// Model events often arrive while the user has scrolled up; always jump to bottom so the reply is visible.
		switch ev.Type {
		case "assistant_delta", "tool_use", "tool_result", "turn_complete", "error":
			m.sticky = true
			m.scrollTop = 1 << 30
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func listViewportH(m *model) int {
	h := m.height - m.titleH - m.streamH - m.inputH - 2
	if h < 3 {
		h = 3
	}
	return h
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
	header := lipgloss.NewStyle().Bold(true).Foreground(roleColor(msg.Type)).Render(string(msg.Type))
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

	keys := m.store.ItemKeys()
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
	title := lipgloss.NewStyle().Bold(true).Render("gou-demo — grouped + collapsed + server blocks  ↑↓ PgUp/Dn  End  q")
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
	if strings.TrimSpace(m.store.StreamingText) != "" {
		msgPane.WriteByte('\n')
		head := lipgloss.NewStyle().Bold(true).Foreground(roleColor(types.MessageTypeAssistant)).Render(string(types.MessageTypeAssistant))
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

	b.WriteString(m.ti.View())
	return lipgloss.NewStyle().MaxWidth(m.width).Render(b.String())
}

func renderMessageRow(m types.Message, cols, maxRows int) string {
	header := lipgloss.NewStyle().Bold(true).Foreground(roleColor(m.Type)).Render(string(m.Type))
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
			parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true).Render("⚙ "+seg.Text))
		case messagerow.SegToolResult:
			parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("↩ "+seg.Text))
		case messagerow.SegThinking:
			parts = append(parts, lipgloss.NewStyle().Faint(true).Italic(true).Render(seg.Text))
		case messagerow.SegServerToolUse:
			parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true).Render("⎈ "+seg.Text))
		case messagerow.SegAdvisorToolResult:
			parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("183")).Render("✧ "+seg.Text))
		case messagerow.SegGroupedToolUse:
			parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).Render("▦ "+seg.Text))
		case messagerow.SegCollapsedReadSearch:
			parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true).Render("▤ "+seg.Text))
		default:
			parts = append(parts, lipgloss.NewStyle().Faint(true).Render(seg.Text))
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
			parts = append(parts, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(line))
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

func roleColor(t types.MessageType) lipgloss.Color {
	switch t {
	case types.MessageTypeUser:
		return lipgloss.Color("39")
	case types.MessageTypeAssistant:
		return lipgloss.Color("141")
	default:
		return lipgloss.Color("245")
	}
}
