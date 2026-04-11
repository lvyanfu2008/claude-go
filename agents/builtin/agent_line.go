package builtin

import (
	"fmt"
	"strings"
)

// FormatAgentLine mirrors src/tools/AgentTool/prompt.ts formatAgentLine:
// `- agentType: whenToUse (Tools: ...)`.
func FormatAgentLine(a BuiltinAgent) string {
	return fmt.Sprintf("- %s: %s (Tools: %s)", a.AgentType, a.WhenToUse, formatAgentToolsDescription(a))
}

func formatAgentToolsDescription(a BuiltinAgent) string {
	hasAllow := len(a.Tools) > 0
	hasDeny := len(a.DisallowedTools) > 0
	if hasAllow && hasDeny {
		deny := make(map[string]struct{}, len(a.DisallowedTools))
		for _, t := range a.DisallowedTools {
			deny[t] = struct{}{}
		}
		var eff []string
		for _, t := range a.Tools {
			if _, skip := deny[t]; !skip {
				eff = append(eff, t)
			}
		}
		if len(eff) == 0 {
			return "None"
		}
		return strings.Join(eff, ", ")
	}
	if hasAllow {
		return strings.Join(a.Tools, ", ")
	}
	if hasDeny {
		return "All tools except " + strings.Join(a.DisallowedTools, ", ")
	}
	return "All tools"
}
