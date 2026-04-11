package engine

import (
	"context"
	"encoding/json"

	"goc/ccb-engine/internal/toolsearch"
)

// ToolRunner produces tool_result content strings for the Messages API.
type ToolRunner interface {
	Run(ctx context.Context, name, toolUseID string, input json.RawMessage) (content string, isError bool, err error)
}

// StubRunner implements ToolRunner with deterministic JSON (no TS bridge).
type StubRunner struct{}

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
