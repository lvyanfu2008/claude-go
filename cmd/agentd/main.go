package main

import (
	"bufio"
	"context"
	"fmt"
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"goc/claudeinit"
	"goc/engine"
	"goc/gou/conversation"
	"goc/sessiontranscript"
	"goc/types"
)

func main() {
	// Load settings (API keys, etc.) from settings.go.json
	if err := claudeinit.Init(context.Background(), claudeinit.Options{NonInteractive: true}); err != nil {
		fmt.Fprintf(os.Stderr, "agentd: claudeinit: %v\n", err)
		os.Exit(1)
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
	submitFn := agentdSubmitFn(cwd, sessionID, perms)

	orc := engine.NewOrchestrator(store, tr, events, submitFn, perms)

	// 发送初始 state_snapshot
	events.OnStateSnapshot(store.Messages, engine.StateMetadata{
		SessionID: sessionID,
	})

	// 上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
						Allow: p.Decision == "allow",
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
