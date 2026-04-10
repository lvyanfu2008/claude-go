// Package toolpolicy implements optional Go-side tool allowlisting before BridgeRunner emits execute_tool.
// See docs/plans/go-policy-ts-pure-execution.md (P0).
package toolpolicy

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// EnforcementEnabled is true when CCB_ENGINE_ENFORCE_ALLOWED_TOOLS=1.
func EnforcementEnabled() bool {
	return strings.TrimSpace(os.Getenv("CCB_ENGINE_ENFORCE_ALLOWED_TOOLS")) == "1"
}

// DenyReason returns a non-empty message if the tool must not run (before schema-valid execute_tool).
// When enforcement is off, always returns "".
func DenyReason(permissionContext json.RawMessage, toolName string) string {
	if !EnforcementEnabled() {
		return ""
	}
	if len(permissionContext) == 0 {
		return "permission_context required when CCB_ENGINE_ENFORCE_ALLOWED_TOOLS=1"
	}
	var pc struct {
		AllowedTools []string `json:"allowedTools"`
	}
	if err := json.Unmarshal(permissionContext, &pc); err != nil {
		return "invalid permission_context JSON: " + err.Error()
	}
	if len(pc.AllowedTools) == 0 {
		return "allowedTools must be non-empty when CCB_ENGINE_ENFORCE_ALLOWED_TOOLS=1"
	}
	for _, n := range pc.AllowedTools {
		if n == toolName {
			return ""
		}
	}
	return fmt.Sprintf("tool %q not in allowedTools", toolName)
}
