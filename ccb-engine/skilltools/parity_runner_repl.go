package skilltools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// replInnerToolNames matches TS REPL_ONLY_TOOLS (src/tools/REPLTool/constants.ts) —
// primitives routed inside REPL when CLAUDE_REPL_MODE hides them from the outer pool.
var replInnerToolNames = map[string]struct{}{
	"Read": {}, "Write": {}, "Edit": {}, "Glob": {}, "Grep": {},
	"Bash": {}, "NotebookEdit": {}, "Agent": {},
}

func replInnerAllowed(name string) bool {
	_, ok := replInnerToolNames[strings.TrimSpace(name)]
	return ok
}

type replToolInput struct {
	Tool  string          `json:"tool"`
	Input json.RawMessage `json:"input"`
	Batch []struct {
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"batch"`
}

type replStep struct {
	Name  string
	Input json.RawMessage
}

func (in *replToolInput) normalizedSteps() []replStep {
	if len(in.Batch) > 0 {
		out := make([]replStep, 0, len(in.Batch))
		for _, b := range in.Batch {
			raw := b.Input
			if raw == nil {
				raw = json.RawMessage(`{}`)
			}
			out = append(out, replStep{Name: b.Name, Input: raw})
		}
		return out
	}
	if strings.TrimSpace(in.Tool) != "" {
		raw := in.Input
		if raw == nil {
			raw = json.RawMessage(`{}`)
		}
		return []replStep{{Name: in.Tool, Input: raw}}
	}
	return nil
}

// runREPLTool executes one or more REPL_ONLY primitives (TS: inside REPL VM).
// Input shapes: {"tool":"Read","input":{...}} or {"batch":[{"name":"Read","input":{...}}, ...]}.
func (r ParityToolRunner) runREPLTool(ctx context.Context, toolUseID string, input json.RawMessage) (string, bool, error) {
	var in replToolInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", true, err
	}
	steps := in.normalizedSteps()
	if len(steps) == 0 {
		return "", true, fmt.Errorf(`REPL input: use {"tool":"Read","input":{...}} or {"batch":[{"name":"Read","input":{...}}]}`)
	}
	var blocks []string
	for i, st := range steps {
		nm := strings.TrimSpace(st.Name)
		if nm == "" || nm == "REPL" {
			return "", true, fmt.Errorf("REPL step %d: invalid tool name %q", i, st.Name)
		}
		if !replInnerAllowed(nm) {
			return "", true, fmt.Errorf("REPL step %d: tool %q is not allowed inside REPL", i, nm)
		}
		subID := fmt.Sprintf("%s#repl%d", toolUseID, i)
		out, isErr, err := r.dispatchTool(ctx, nm, subID, st.Input)
		if err != nil {
			return "", true, err
		}
		prefix := fmt.Sprintf("[%s] ", nm)
		if isErr {
			prefix = fmt.Sprintf("[%s ERROR] ", nm)
		}
		blocks = append(blocks, prefix+strings.TrimSpace(out))
	}
	return strings.Join(blocks, "\n\n"), false, nil
}
