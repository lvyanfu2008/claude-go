package main

import (
	"goc/appstate"
	"goc/gou/commandqueue"
	"goc/services/autodream"
	"goc/services/extractmemories"
	"goc/services/sessionmemory"
)

// agentdSession 聚合每个会话的独立状态。
// stdio 模式用进程级默认实例;WS 模式每连接创建一个。
type agentdSession struct {
	sessionStarted  bool
	extractMemState *extractmemories.State
	autoDreamState  *autodream.State
	sessionMemState *sessionmemory.State
	appStateStore   *appstate.Store
	cmdQueue        *commandqueue.Queue
}

// newAgentdSession 创建独立的会话状态(WS 每连接调用)。
func newAgentdSession() *agentdSession {
	return &agentdSession{
		extractMemState: extractmemories.NewState(),
		autoDreamState:  autodream.NewState(),
		sessionMemState: sessionmemory.NewState(),
		appStateStore:   appstate.NewStore(appstate.DefaultAppState()),
		cmdQueue:        commandqueue.NewQueue(),
	}
}
