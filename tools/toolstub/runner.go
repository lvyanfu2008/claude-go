// Package toolstub holds shared tool execution helpers for local parity (e.g. [skilltools.ParityToolRunner]).
// It is not used by gou-demo's default HTTP streaming transcript path.
package toolstub

import (
	"context"
	"encoding/json"

	"goc/internal/toolsearch"
)

// ToolRunner produces tool_result content strings for parity / stub execution.
type ToolRunner interface {
	Run(ctx context.Context, name, toolUseID string, input json.RawMessage) (content string, isError bool, err error)
}

// StubRunner implements ToolRunner with deterministic JSON (no TS bridge).
type StubRunner struct{}

// Run implements ToolRunner.
func (StubRunner) Run(ctx context.Context, name, toolUseID string, input json.RawMessage) (string, bool, error) {
	if name == toolsearch.ToolSearchToolName {
		pending, names := MCPPendingsFromContext(ctx)
		return toolsearch.ExecToolSearchForRunner(input, ToolRegistryFromContext(ctx), pending, names)
	}
	payload := map[string]any{
		"stub":        true,
		"tool":        name,
		"tool_use_id": toolUseID,
		"input":       json.RawMessage(input),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", true, err
	}
	return string(b), false, nil
}
