package engine

import (
	"context"
	"encoding/json"
)

// PermissionDecision 是用户对权限询问的回复。
type PermissionDecision struct {
	Allow        bool
	Reason       string
	UpdatedInput json.RawMessage `json:"updated_input,omitempty"`
}

// PermissionBridge 处理工具权限询问。
// 实现者负责向用户展示询问内容并返回用户的决策。
//
// Bubble Tea 实现：弹窗阻塞等按键
// Agentd 实现：写 stdout 发 permission_ask → 读 stdin 等 permission_response
type PermissionBridge interface {
	// AskPermission 向用户询问是否允许执行指定工具。
	// ctx 超时应返回 error，调用方应视为拒绝。
	AskPermission(ctx context.Context, toolName string, input json.RawMessage) (PermissionDecision, error)
}
