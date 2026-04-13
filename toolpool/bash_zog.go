package toolpool

import (
	"strings"

	"goc/ccb-engine/bashzog"
	"goc/internal/toolvalidator"
	"goc/types"
)

// ReplaceBashToolSpecIfZogMode appends a [bashzog.ZogToolName] row (Go Zog wire / embedded snapshot)
// when GO_TOOL_INPUT_VALIDATOR=zog. The original "Bash" row from tools_api.json is unchanged.
func ReplaceBashToolSpecIfZogMode(specs []types.ToolSpec) ([]types.ToolSpec, error) {
	if toolvalidator.InputValidatorMode() != "zog" {
		return specs, nil
	}
	for i := range specs {
		if strings.TrimSpace(specs[i].Name) == bashzog.ZogToolName {
			return specs, nil
		}
	}
	zogSpec, err := bashzog.BashZogToolSpec()
	if err != nil {
		return nil, err
	}
	merged := zogSpec
	for i := range specs {
		if strings.TrimSpace(specs[i].Name) == "Bash" {
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
	}
	out := make([]types.ToolSpec, len(specs)+1)
	copy(out, specs)
	out[len(specs)] = merged
	return out, nil
}
