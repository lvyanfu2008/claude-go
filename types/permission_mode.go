// Mirrors src/types/permissions.ts PermissionMode / ExternalPermissionMode.
package types

// PermissionMode is the tool permission mode (user-addressable + internal).
type PermissionMode string

const (
	PermissionAcceptEdits      PermissionMode = "acceptEdits"
	PermissionBypassPermissions PermissionMode = "bypassPermissions"
	PermissionDefault          PermissionMode = "default"
	PermissionDontAsk          PermissionMode = "dontAsk"
	PermissionPlan             PermissionMode = "plan"
	PermissionAuto             PermissionMode = "auto"
	PermissionBubble           PermissionMode = "bubble"
)
