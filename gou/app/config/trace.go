package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"

	"goc/ccb-engine/debugpath"
	"goc/types"
)

// traceLogger is set by SetupTrace from GOU_DEMO_LOG_FILE or GOU_DEMO_LOG.
var traceLogger *log.Logger

func defaultGouDemoTracePath() string {
	p := debugpath.ResolveLogPath()
	if p != "" {
		return p
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("gou-demo-trace-%d.txt", os.Getpid()))
}

// SetupTrace initializes the trace logger from environment variables. Returns a cleanup func.
func SetupTrace() (cleanup func()) {
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
		traceLogger = log.New(f, "[gou-demo] ", flags)
		return func() { _ = f.Close() }
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GOU_DEMO_LOG")))
	if v != "1" && v != "true" && v != "yes" && v != "on" {
		return func() {}
	}
	// GOU_DEMO_LOG=1: writing to stderr while the TUI runs may corrupt line order and layout.
	if EnvTruthy("GOU_DEMO_LOG_STDERR") {
		traceLogger = log.New(os.Stderr, "[gou-demo] ", flags)
		return func() {}
	}
	if isatty.IsTerminal(os.Stderr.Fd()) {
		p := defaultGouDemoTracePath()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "[gou-demo] trace mkdir %q: %v; falling back to stderr\n", filepath.Dir(p), err)
			traceLogger = log.New(os.Stderr, "[gou-demo] ", flags)
			return func() {}
		}
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gou-demo] trace open %q: %v; falling back to stderr\n", p, err)
			traceLogger = log.New(os.Stderr, "[gou-demo] ", flags)
			return func() {}
		}
		debugpath.MaybeUpdateLatestSymlink(p)
		traceLogger = log.New(f, "[gou-demo] ", flags)
		lp := debugpath.LatestLinkPathFor(p)
		if lp != "" {
			traceLogger.Printf("trace -> %s points to %s (TTY: stderr+TUI garbles; or GOU_DEMO_LOG_FILE=...)", lp, p)
		} else {
			traceLogger.Printf("trace -> %s (TTY: stderr+TUI garbles output; use this file or GOU_DEMO_LOG_FILE=...)", p)
		}
		return func() { _ = f.Close() }
	}
	traceLogger = log.New(os.Stderr, "[gou-demo] ", flags)
	return func() {}
}

// Tracef writes a trace log line when tracing is enabled.
func Tracef(format string, args ...any) {
	if traceLogger != nil {
		traceLogger.Printf(format, args...)
	}
}

// LogToolUseContext dumps ProcessUserInputContext / ToolUseContext JSON when CLAUDE_CODE_LOG_TOOL_USE_CONTEXT
// or GOU_DEMO_LOG_TOOL_USE_CONTEXT is set (requires GOU_DEMO_LOG=1 or GOU_DEMO_LOG_FILE so [traceLogger] is configured — stderr+TUI is avoided by default).
// Values: 1|true|summary — summary snapshot; full — entire serializable context (large). JSON is one-line (no indent).
func LogToolUseContext(rc *types.ProcessUserInputContextData) {
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
	if traceLogger == nil {
		return
	}
	b, err := types.FormatProcessInputContextForLog(rc, full)
	if err != nil {
		Tracef("ToolUseContext log: marshal: %v", err)
		return
	}
	mode := "summary"
	if full {
		mode = "full"
	}
	traceLogger.Printf("ToolUseContext (%s JSON):\n%s\n", mode, string(b))
}

func PreviewForTrace(s string, max int) string {
	if max <= 0 {
		max = 120
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + fmt.Sprintf("…(%d runes)", len(r))
}
