package toolexecution

import (
	"context"
	"encoding/json"
	"fmt"

	"goc/permissionrules"
	"goc/types"
)

// CheckRuleBasedPermissions mirrors checkRuleBasedPermissions (toolHooks.ts / permissions.ts) steps 1a–1b.
// Uses [ToolUseContext.ToolPermission] when set (merged settings rules).
//
// TS: src/utils/permissions/permissions.ts L1071+.
func CheckRuleBasedPermissions(
	ctx context.Context,
	tool Tool,
	input json.RawMessage,
	tcx *ToolUseContext,
) *PermissionDecision {
	_ = ctx
	_ = input
	if tcx == nil || tcx.ToolPermission == nil {
		return nil
	}
	return RuleBasedDecisionForTool(tool.Name(), tcx.ToolPermission)
}

// RuleBasedDecisionForTool applies alwaysDeny / alwaysAsk rules (permissions.ts 1a–1b) for a tool name.
func RuleBasedDecisionForTool(toolName string, perm *types.ToolPermissionContextData) *PermissionDecision {
	if perm == nil {
		return nil
	}
	pc := *perm
	types.NormalizeToolPermissionContextData(&pc)
	spec := types.ToolSpec{Name: toolName}
	if dr := permissionrules.GetDenyRuleForTool(pc, spec); dr != nil {
		d := DenyDecision(fmt.Sprintf("Permission to use %s has been denied.", toolName))
		return &d
	}
	if ar := permissionrules.GetAskRuleForTool(pc, spec); ar != nil {
		a := AskDecision(permissionRequestMessage(toolName))
		_ = ar
		return &a
	}
	return nil
}

func permissionRequestMessage(toolName string) string {
	return fmt.Sprintf("Permission required to use tool %s", toolName)
}
