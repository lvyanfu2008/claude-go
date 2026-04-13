package toolpool

import (
	"strings"

	"goc/ccb-engine/bashzog"
	"goc/internal/toolvalidator"
	"goc/types"
)

// ReplaceBashToolSpecIfZogMode replaces the tools_api "Bash" row with [bashzog.ZogToolName] when
// GO_TOOL_INPUT_VALIDATOR=zog (same slot / merged limits). Drops any duplicate BashZog rows.
// If the list already has BashZog and no Bash, returns specs unchanged (idempotent).
func ReplaceBashToolSpecIfZogMode(specs []types.ToolSpec) ([]types.ToolSpec, error) {
	if toolvalidator.InputValidatorMode() != "zog" {
		return specs, nil
	}
	var hasBash, hasZog bool
	for _, t := range specs {
		switch strings.TrimSpace(t.Name) {
		case "Bash":
			hasBash = true
		case bashzog.ZogToolName:
			hasZog = true
		}
	}
	if hasZog && !hasBash {
		return specs, nil
	}
	zogSpec, err := bashzog.BashZogToolSpec()
	if err != nil {
		return nil, err
	}
	merged := zogSpec
	for i := range specs {
		if strings.TrimSpace(specs[i].Name) != "Bash" {
			continue
		}
		b := specs[i]
		merged.MaxResultSizeChars = b.MaxResultSizeChars
		merged.Strict = b.Strict
		merged.ShouldDefer = b.ShouldDefer
		merged.AlwaysLoad = b.AlwaysLoad
		merged.InterruptBehavior = b.InterruptBehavior
		merged.SearchHint = b.SearchHint
		merged.Aliases = append([]string(nil), b.Aliases...)
		break
	}
	out := make([]types.ToolSpec, 0, len(specs))
	emitted := false
	for _, t := range specs {
		n := strings.TrimSpace(t.Name)
		switch n {
		case "Bash":
			if !emitted {
				out = append(out, merged)
				emitted = true
			}
		case bashzog.ZogToolName:
			continue
		default:
			out = append(out, t)
		}
	}
	if !emitted {
		out = append(out, merged)
	}
	return out, nil
}
