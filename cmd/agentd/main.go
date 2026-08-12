package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"

	"goc/claudeinit"
	"goc/commands"
	"goc/engine"
	"goc/gou/conversation"
	"goc/hookexec"
	_ "goc/plugins"
	"goc/sessiontranscript"
	"goc/types"
	"time"
)

func main() {
	// Load settings (API keys, etc.) from settings.go.json
	if err := claudeinit.Init(context.Background(), claudeinit.Options{NonInteractive: true}); err != nil {
		fmt.Fprintf(os.Stderr, "agentd: claudeinit: %v\n", err)
		os.Exit(1)
	}

	// WS 服务模式: agentd --serve --ws :8765
	if hasFlag(os.Args, "--serve") {
		addr := wsListenAddr(os.Args)
		serveWS(addr)
		return
	}

	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout
	enc := json.NewEncoder(writer)

	// 解析启动参数
	resumeSessionID := ""
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--resume" && i+1 < len(args) {
			resumeSessionID = args[i+1]
			i++
		}
	}

	// 初始化
	cwd, _ := os.Getwd()
	sessionID := resumeSessionID
	if sessionID == "" {
		sessionID = sessiontranscript.NewUUID()
	}

	store := &conversation.Store{ConversationID: sessionID}
	var tr *sessiontranscript.Store
	if resumeSessionID != "" {
		tr = &sessiontranscript.Store{
			SessionID:   sessionID,
			OriginalCwd: cwd,
			Cwd:         cwd,
		}
	}

	// 创建 stdio 事件处理器
	events := newStdioEventHandler(enc)
	permRespCh := make(chan engine.PermissionDecision, 1)
	perms := &stdioPermissionBridge{responseCh: permRespCh}

	// 创建 Orchestrator 的 SubmitFunc
	sess := newAgentdSession()
	submitFn := agentdSubmitFn(sess, func() string { return cwd }, sessionID, perms)

	orc := engine.NewOrchestrator(store, tr, events, submitFn, perms)

	// 发送初始 state_snapshot
	events.OnStateSnapshot(store.Messages, engine.StateMetadata{
		SessionID: sessionID,
	})

	// 启动时加载并发送命令列表，供 UI slash 补全使用
	if cmds, err := commands.GetCommandsWithDefaults(context.Background(), cwd); err == nil {
		events.OnCommandsList(cmds)
	}

	// 上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// SessionEnd hooks — mirrors cmd/claude
	transcriptPath := sessiontranscript.TranscriptPath(sessionID, cwd, "", sessiontranscript.ConfigHomeDir())
	mergedHooks, _ := hookexec.MergedHooksForCwd(cwd)
	sessionEndReason := "error"
	defer func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), time.Duration(hookexec.SessionEndHookTimeoutMs)*time.Millisecond)
		defer bgCancel()
		hookexec.RunSessionEndHooks(bgCtx, mergedHooks, cwd, hookexec.BaseHookInput{
			SessionID:      sessionID,
			TranscriptPath: transcriptPath,
			Cwd:            cwd,
		}, sessionEndReason, sessionID)
	}()
	_ = sessionEndReason
	_ = transcriptPath

	// 信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// inboxCh 接收 stdin reader 发来的消息
	type inboxMessage struct {
		text string
		mode types.PromptInputMode
	}
	inboxCh := make(chan inboxMessage, 16)
	abortCh := make(chan struct{}, 1)

	// Goroutine 1: Stdin Reader
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var msg engine.GatewayMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}

			switch msg.Type {
			case engine.MsgTypeUserMessage:
				var p engine.UserMessagePayload
				if err := json.Unmarshal(msg.Payload, &p); err == nil {
					inboxCh <- inboxMessage{text: p.Text, mode: types.PromptInputModePrompt}
				}

			case engine.MsgTypePermissionResponse:
				var p engine.PermissionResponsePayload
				if err := json.Unmarshal(msg.Payload, &p); err == nil {
					permRespCh <- engine.PermissionDecision{
						Allow:        p.Decision == "allow",
						UpdatedInput: p.UpdatedInput,
					}
				}

			case engine.MsgTypeAbort:
				abortCh <- struct{}{}
			}
		}
	}()

	// Goroutine 2: Main loop
	go func() {
		for {
			select {
			case msg := <-inboxCh:
				orc.SubmitInput(ctx, msg.text)

			case <-abortCh:
				orc.Abort(ctx)

			case <-ctx.Done():
				return
			}
		}
	}()

	// 等待退出信号
	<-sigCh
	cancel()

	// 发送退出事件
	enc.Encode(engine.AgentEvent{
		Type: engine.EventTypeError,
		Payload: mustMarshal(map[string]string{
			"message": "agentd shutting down",
		}),
	})
}

// mustMarshal 是 JSON 序列化的辅助函数，忽略错误。
func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// hasFlag 检查 os.Args 是否包含指定 flag。
func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

// wsListenAddr 解析 --ws 参数,默认 :8765。纯数字自动加 ":"。
func wsListenAddr(args []string) string {
	addr := ":8765"
	for i := 0; i < len(args); i++ {
		if args[i] == "--ws" && i+1 < len(args) {
			addr = args[i+1]
			i++
			break
		}
	}
	if addr[0] >= '0' && addr[0] <= '9' {
		addr = ":" + addr
	}
	return addr
}

// serveWS 启动 WebSocket 服务,每连接一个会话。
func serveWS(addr string) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ws upgrade: %v\n", err)
			return
		}
		go handleWSConnection(conn)
	})
	fmt.Fprintf(os.Stderr, "agentd WS server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "ws server: %v\n", err)
		os.Exit(1)
	}
}

// handleWSConnection 处理单个 WebSocket 连接(一个完整会话)。
func handleWSConnection(conn *websocket.Conn) {
	// cwd 是会话工作目录,初始为 agentd 进程启动目录;客户端可通过 set_cwd 覆盖。
	// 用闭包让 submitFn 每次调用都读到最新值。
	var cwdLock sync.Mutex
	cwd, _ := os.Getwd()

	sess := newAgentdSession()
	events := newWsEventHandler(conn)
	perms := &wsPermissionBridge{responseCh: make(chan engine.PermissionDecision, 1)}

	sessionID := sessiontranscript.NewUUID()
	store := &conversation.Store{ConversationID: sessionID}
	submitFn := agentdSubmitFn(sess, func() string {
		cwdLock.Lock()
		defer cwdLock.Unlock()
		return cwd
	}, sessionID, perms)
	orc := engine.NewOrchestrator(store, nil, events, submitFn, perms)

	ctx, cancel := context.WithCancel(context.Background())
	defer conn.Close()
	defer cancel()

	// 初始 state_snapshot + commands_list
	events.OnStateSnapshot(store.Messages, engine.StateMetadata{SessionID: sessionID})
	if cmds, err := commands.GetCommandsWithDefaults(ctx, func() string {
		cwdLock.Lock()
		defer cwdLock.Unlock()
		return cwd
	}()); err == nil {
		events.OnCommandsList(cmds)
	}

	// SessionEnd hooks — mirrors cmd/claude, using the session's current cwd.
	curCwd := func() string {
		cwdLock.Lock()
		defer cwdLock.Unlock()
		return cwd
	}
	transcriptPath := sessiontranscript.TranscriptPath(sessionID, curCwd(), "", sessiontranscript.ConfigHomeDir())
	mergedHooks, _ := hookexec.MergedHooksForCwd(curCwd())
	sessionEndReason := "error"
	defer func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), time.Duration(hookexec.SessionEndHookTimeoutMs)*time.Millisecond)
		defer bgCancel()
		hookexec.RunSessionEndHooks(bgCtx, mergedHooks, curCwd(), hookexec.BaseHookInput{
			SessionID:      sessionID,
			TranscriptPath: transcriptPath,
			Cwd:            curCwd(),
		}, sessionEndReason, sessionID)
	}()
	_ = sessionEndReason
	_ = transcriptPath

	type inboxMessage struct {
		text string
		mode types.PromptInputMode
	}
	inboxCh := make(chan inboxMessage, 16)
	abortCh := make(chan struct{}, 1)

	// readLoop: 读 WS 帧 → 分发
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msg engine.GatewayMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}
			switch msg.Type {
			case engine.MsgTypeUserMessage:
				var p engine.UserMessagePayload
				if err := json.Unmarshal(msg.Payload, &p); err == nil {
					inboxCh <- inboxMessage{text: p.Text, mode: types.PromptInputModePrompt}
				}
			case engine.MsgTypeSetCwd:
				var p engine.SetCwdPayload
				if err := json.Unmarshal(msg.Payload, &p); err == nil && p.Cwd != "" {
					cwdLock.Lock()
					cwd = p.Cwd
					newCwd := cwd
					cwdLock.Unlock()
					// 命令列表随工作目录更新(项目级 slash 命令跟随上报的 cwd)
					if cmds, err := commands.GetCommandsWithDefaults(ctx, newCwd); err == nil {
						events.OnCommandsList(cmds)
					}
				}
			case engine.MsgTypePermissionResponse:
				var p engine.PermissionResponsePayload
				if err := json.Unmarshal(msg.Payload, &p); err == nil {
					perms.responseCh <- engine.PermissionDecision{
						Allow:        p.Decision == "allow",
						UpdatedInput: p.UpdatedInput,
					}
				}
			case engine.MsgTypeAbort:
				abortCh <- struct{}{}
			}
		}
	}()

	// 主循环(镜像现有 stdio main loop)
	for {
		select {
		case msg := <-inboxCh:
			orc.SubmitInput(ctx, msg.text)
		case <-abortCh:
			orc.Abort(ctx)
		case <-ctx.Done():
			return
		case <-readDone:
			return
		}
	}
}
