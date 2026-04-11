package toolexecution

import (
	"context"
	"encoding/json"
	"fmt"

	"goc/permissionrules"
	"goc/types"
)

// RuleBasedToolPermissionsChecker is optional on [Tool] for permissions.ts step 1c (tool-specific rule/safety checks).
// Return nil for passthrough. For [PermissionAsk], set [PermissionDecision.AskKind] to [PermissionAskKindRuleContent] or [PermissionAskKindSafetyCheck] so the ask propagates from [CheckRuleBasedPermissions] (1f–1g); other asks are ignored at this layer (TS returns null).
type RuleBasedToolPermissionsChecker interface {
	CheckPermissionsFromRules(ctx context.Context, input json.RawMessage, tcx *ToolUseContext) *PermissionDecision
}

// CheckRuleBasedPermissions mirrors checkRuleBasedPermissions (toolHooks.ts / permissions.ts) steps 1a–1b, 1c–1g.
// Uses [ToolUseContext.ToolPermission] when set for 1a–1b; nil permission data is treated as no alwaysDeny/alwaysAsk rules.
// Step 1b: whole-tool ask on Bash is skipped when [WholeToolAskSkippedForBash1b] (see [BashSandboxRule1b] on tcx).
//
// TS: src/utils/permissions/permissions.ts L1071+.
func CheckRuleBasedPermissions(
	ctx context.Context,
	tool Tool,
	input json.RawMessage,
	tcx *ToolUseContext,
) *PermissionDecision {
	_ = ctx
	tcxUse := tcx
	if tcxUse == nil {
		tcxUse = &ToolUseContext{}
	}
	if rd := wholeToolAlwaysDenyAsk(tool.Name(), input, tcxUse.ToolPermission, tcxUse.BashSandboxRule1b); rd != nil {
		return rd
	}
	if checker, ok := tool.(RuleBasedToolPermissionsChecker); ok {
		raw := checker.CheckPermissionsFromRules(ctx, input, tcxUse)
		return toolRuleLayerObjection(raw)
	}
	return nil
}

// toolRuleLayerObjection keeps only deny and the ask variants that TS surfaces from checkRuleBasedPermissions (1d, 1f, 1g).
func toolRuleLayerObjection(d *PermissionDecision) *PermissionDecision {
	if d == nil {
		return nil
	}
	switch d.Behavior {
	case PermissionDeny:
		return d
	case PermissionAsk:
		if d.AskKind == PermissionAskKindRuleContent || d.AskKind == PermissionAskKindSafetyCheck {
			return d
		}
		return nil
	default:
		return nil
	}
}

// RuleBasedDecisionForTool applies alwaysDeny / alwaysAsk rules (permissions.ts 1a–1b) for a tool name without tool input.
// Bash sandbox 1b bypass is not applied (no input); use [CheckRuleBasedPermissions] when input is available.
// When perm is nil, rule lists are empty (same as TS with no merged deny/ask rules).
func RuleBasedDecisionForTool(toolName string, perm *types.ToolPermissionContextData) *PermissionDecision {
	return wholeToolAlwaysDenyAsk(toolName, nil, perm, nil)
}

// wholeToolAlwaysDenyAsk implements permissions.ts 1a–1b including Bash whole-tool ask bypass (1b).
func wholeToolAlwaysDenyAsk(toolName string, input json.RawMessage, perm *types.ToolPermissionContextData, bash1b *BashSandboxRule1b) *PermissionDecision {
	var pc types.ToolPermissionContextData
	if perm != nil {
		pc = *perm
	}
	types.NormalizeToolPermissionContextData(&pc)
	spec := types.ToolSpec{Name: toolName}
	if dr := permissionrules.GetDenyRuleForTool(pc, spec); dr != nil {
		d := DenyDecision(fmt.Sprintf("Permission to use %s has been denied.", toolName))
		return &d
	}
	if ar := permissionrules.GetAskRuleForTool(pc, spec); ar != nil {
		_ = ar
		if WholeToolAskSkippedForBash1b(toolName, input, bash1b) {
			return nil
		}
		a := AskDecision(permissionRequestMessage(toolName))
		return &a
	}
	return nil
}

func permissionRequestMessage(toolName string) string {
	return fmt.Sprintf("Permission required to use tool %s", toolName)
}
