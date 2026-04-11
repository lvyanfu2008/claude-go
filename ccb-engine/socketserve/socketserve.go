// Package socketserve runs the ccb-engine Unix-socket protocol (SubmitUserTurn + bridge).
// Embedded in gou-demo; headless automation uses cmd/ccb-socket-host.
package socketserve

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"goc/ccb-engine/internal/anthropic"
	"goc/ccb-engine/internal/engine"
	"goc/ccb-engine/internal/llm"
	"goc/ccb-engine/internal/protocol"
	"goc/ccb-engine/internal/toolpolicy"
	"goc/ccb-engine/submitfill"
)

// Logf prints optional diagnostics (e.g. to stderr or a trace logger); nil is a no-op.
type Logf func(format string, args ...any)

type submitUserTurnRequest struct {
	V       string `json:"v,omitempty"`
	Method  string `json:"method"`
	ID      string `json:"id"`
	Payload struct {
		Text           string          `json:"text"`
		Messages       json.RawMessage `json:"messages,omitempty"`
		Tools          json.RawMessage `json:"tools,omitempty"`
		System         string          `json:"system,omitempty"`
		ClientStateRev *uint64         `json:"client_state_rev,omitempty"`

		FetchSystemPromptIfEmpty bool            `json:"fetch_system_prompt_if_empty,omitempty"`
		Cwd                      string          `json:"cwd,omitempty"`
		ExtraClaudeMdRoots       []string        `json:"extra_claude_md_roots,omitempty"`
		CustomSystemPrompt       string          `json:"custom_system_prompt,omitempty"`
		AppendSystemPrompt       string          `json:"append_system_prompt,omitempty"`
		ModelID                  string          `json:"model,omitempty"`
		PermissionContext        json.RawMessage `json:"permission_context,omitempty"`
	} `json:"payload"`
}

type toolResultInbound struct {
	CallID    string `json:"call_id"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
}

// Run listens on socketPath until ctx is cancelled. Each accepted connection is handled by [HandleConn].
// Removes an existing socket file before bind (same as standalone serve).
func Run(ctx context.Context, socketPath string, logf Logf) error {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if strings.TrimSpace(socketPath) == "" {
		return fmt.Errorf("socketserve: empty socket path")
	}
	_ = os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("socketserve listen %s: %w", socketPath, err)
	}
	defer ln.Close()
	if err := os.Chmod(socketPath, 0o600); err != nil {
		logf("socketserve: chmod socket: %v", err)
	}

	completer := llm.NewFromEnv()
	defaultTools := anthropic.DefaultStubTools()
	anthropic.LogToolsLoaded("socketserve_listen", "", "default_stub", defaultTools)

	logf("socketserve listening on %s", socketPath)

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, aerr := ln.Accept()
		if aerr != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logf("socketserve accept: %v", aerr)
			continue
		}
		go HandleConn(conn, completer, defaultTools)
	}
}

func readOneLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	return line, nil
}

// HandleConn serves one Unix (or TCP for tests) connection: line-delimited JSON protocol.
func HandleConn(conn net.Conn, completer llm.TurnCompleter, defaultTools []anthropic.ToolDefinition) {
	defer conn.Close()

	lines := make(chan string, 128)
	go func() {
		defer close(lines)
		r := bufio.NewReader(conn)
		for {
			line, err := readOneLine(r)
			if err != nil {
				return
			}
			if line == "" {
				continue
			}
			lines <- line
		}
	}()

	var writeMu sync.Mutex
	writeEv := func(ev protocol.StreamEvent) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writeLine(conn, ev)
	}
	writeEvBridge := func(ev protocol.StreamEvent) error {
		return writeLine(conn, ev)
	}

	for line := range lines {
		req, err := parseSubmitUserTurn(line)
		if err != nil {
			_ = writeEv(protocol.ErrEvent("invalid_request", err.Error()))
			_ = writeEv(protocol.ResponseEnd(""))
			continue
		}
		if req.V != "" && req.V != protocol.Version {
			_ = writeEv(protocol.ErrEvent("version_mismatch", fmt.Sprintf("client protocol %q not supported; want %q", req.V, protocol.Version)))
			_ = writeEv(protocol.ResponseEnd(req.ID))
			continue
		}
		if err := validateSubmitPayload(&req); err != nil {
			_ = writeEv(protocol.ErrEvent("invalid_request", err.Error()))
			_ = writeEv(protocol.ResponseEnd(req.ID))
			continue
		}

		tools := defaultTools
		toolsSource := "default_stub"
		if len(req.Payload.Tools) > 0 {
			var parsed []anthropic.ToolDefinition
			if err := json.Unmarshal(req.Payload.Tools, &parsed); err != nil {
				_ = writeEv(protocol.ErrEvent("invalid_request", "tools: "+err.Error()))
				_ = writeEv(protocol.ResponseEnd(req.ID))
				continue
			}
			if len(parsed) > 0 {
				tools = parsed
				toolsSource = "payload_json"
			}
		}
		anthropic.LogToolsLoaded("socketserve", req.ID, toolsSource, tools)

		turnCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		bridge := engine.NewBridgeRunner(turnCtx, &writeMu, writeEvBridge)
		sess := engine.NewSession(func(ev protocol.StreamEvent) {
			_ = writeEv(ev)
		})
		bridge.SetStateRevProvider(sess.StateRev)
		if toolpolicy.EnforcementEnabled() {
			bridge.SetToolExecutePolicy(map[string]any{
				"decision": "allow",
				"source":   "ccb-engine",
			})
		}

		msgsRaw := req.Payload.Messages
		systemStr := req.Payload.System
		fillOpts := submitfill.Options{
			FetchIfEmpty:       submitfill.FetchDesired(req.Payload.FetchSystemPromptIfEmpty),
			Cwd:                req.Payload.Cwd,
			ToolsJSON:          req.Payload.Tools,
			ExtraClaudeMdRoots: req.Payload.ExtraClaudeMdRoots,
			CustomSystemPrompt: req.Payload.CustomSystemPrompt,
			AppendSystemPrompt: req.Payload.AppendSystemPrompt,
			ModelID:            req.Payload.ModelID,
		}
		var errFill error
		systemStr, msgsRaw, errFill = submitfill.ApplyIfEmpty(systemStr, msgsRaw, fillOpts)
		if errFill != nil {
			cancel()
			_ = writeEv(protocol.ErrEvent("invalid_request", "system_context: "+errFill.Error()))
			_ = writeEv(protocol.ResponseEnd(req.ID))
			continue
		}

		if len(msgsRaw) > 0 {
			var msgs []anthropic.Message
			if err := json.Unmarshal(msgsRaw, &msgs); err != nil {
				cancel()
				_ = writeEv(protocol.ErrEvent("invalid_request", "messages: "+err.Error()))
				_ = writeEv(protocol.ResponseEnd(req.ID))
				continue
			}
			sess.HydrateFromMessages(msgs)
		}
		if req.Payload.Text != "" {
			sess.AppendUserText(req.Payload.Text)
		}

		runErr := make(chan error, 1)
		go func() {
			runErr <- sess.RunTurn(turnCtx, completer, tools, systemStr, bridge, false,
				engine.WithPermissionContext(req.Payload.PermissionContext),
				engine.WithModelID(req.Payload.ModelID))
		}()

	turnLoop:
		for {
			select {
			case err := <-runErr:
				cancel()
				_ = err
				break turnLoop
			case nextLine, ok := <-lines:
				if !ok {
					cancel()
					<-runErr
					return
				}
				if tryHandleTurnInbound(nextLine, bridge, cancel) {
					continue
				}
				continue
			}
		}

		_ = writeEv(protocol.ResponseEnd(req.ID))
	}
}

func parseSubmitUserTurn(line string) (submitUserTurnRequest, error) {
	var req submitUserTurnRequest
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return req, err
	}
	if req.Method != "SubmitUserTurn" {
		return req, fmt.Errorf("expected method SubmitUserTurn, got %q", req.Method)
	}
	return req, nil
}

func validateSubmitPayload(req *submitUserTurnRequest) error {
	hasMsgs := len(req.Payload.Messages) > 0
	hasText := strings.TrimSpace(req.Payload.Text) != ""
	if !hasMsgs && !hasText {
		return fmt.Errorf("need non-empty payload.text and/or payload.messages")
	}
	return nil
}

func tryHandleTurnInbound(line string, bridge *engine.BridgeRunner, cancel context.CancelFunc) bool {
	var methodProbe struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal([]byte(line), &methodProbe); err != nil {
		return false
	}
	if methodProbe.Method == "CancelTurn" {
		cancel()
		return true
	}

	var tr toolResultInbound
	if err := json.Unmarshal([]byte(line), &tr); err != nil {
		return false
	}
	if tr.CallID == "" || tr.ToolUseID == "" {
		return false
	}
	if methodProbe.Method == "SubmitUserTurn" {
		return false
	}

	bridge.DeliverToolResult(tr.CallID, tr.ToolUseID, tr.Content, tr.IsError)
	return true
}

func writeLine(conn net.Conn, ev protocol.StreamEvent) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(b, '\n'))
	return err
}
