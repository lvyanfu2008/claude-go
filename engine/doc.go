// Package engine 提供无 UI 依赖的核心对话循环。
//
// Orchestrator 接收用户输入，执行 ProcessUserInput → Query 循环，
// 通过 EventHandler 输出事件，通过 PermissionBridge 处理权限询问。
//
// Bubble Tea TUI 模式和 agentd 模式共享此包，各实现自己的
// EventHandler 和 PermissionBridge，通过接口注入。
//
// 约束：
//   - engine/ 不 import bubbletea, lipgloss, 或 gou/ 下任何包
//   - 所有输出通过 EventHandler，所有输入通过 SubmitInput/Abort
//   - 权限询问通过 PermissionBridge，不阻塞 stdin reader goroutine
package engine
