package toolpool

import (
	"encoding/json"
	"strings"

	"goc/ccb-engine/bashzog"
	"goc/internal/toolvalidator"
	"goc/types"
)

// ReplaceBashToolSpecIfZogMode swaps the Bash entry to the Go-embedded snapshot when
// GO_TOOL_INPUT_VALIDATOR=zog (no runtime read of tools_api.json for Bash shape).
func ReplaceBashToolSpecIfZogMode(specs []types.ToolSpec) ([]types.ToolSpec, error) {
	if toolvalidator.InputValidatorMode() != "zog" {
		return specs, nil
	}
	patch, err := bashzog.BashToolSpec()
	if err != nil {
		return nil, err
	}
	out := make([]types.ToolSpec, len(specs))
	copy(out, specs)
	for i := range out {
		if strings.TrimSpace(out[i].Name) == "Bash" {
			merged := out[i]
			merged.Description = patch.Description
			merged.InputJSONSchema = append(json.RawMessage(nil), patch.InputJSONSchema...)
			out[i] = merged
			break
		}
	}
	return out, nil
}
