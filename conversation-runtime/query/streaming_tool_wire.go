package query

import (
	"encoding/json"
	"strings"

	"goc/conversation-runtime/streamingtool"
)

type stubToolBehavior struct {
	name string
}

func (s stubToolBehavior) Name() string { return s.name }

func (s stubToolBehavior) InputOK(input []byte) (parsed any, ok bool) {
	return nil, true
}

func (s stubToolBehavior) IsConcurrencySafe(parsed any) bool { return false }

func (s stubToolBehavior) InterruptBehavior() string { return "block" }

// makeFindToolBehavior builds [streamingtool.ToolBehavior] lookup from tools[] JSON (name field only; TS parity expands later).
func makeFindToolBehavior(toolsJSON json.RawMessage) func(string) (streamingtool.ToolBehavior, bool) {
	names := map[string]struct{}{}
	if len(toolsJSON) > 0 {
		var arr []map[string]any
		if err := json.Unmarshal(toolsJSON, &arr); err == nil {
			for _, o := range arr {
				n, _ := o["name"].(string)
				n = strings.TrimSpace(n)
				if n != "" {
					names[n] = struct{}{}
				}
			}
		}
	}
	return func(name string) (streamingtool.ToolBehavior, bool) {
		if _, ok := names[name]; !ok {
			return nil, false
		}
		return stubToolBehavior{name: name}, true
	}
}

type queryToolUseContextPort struct {
	root *streamingtool.AbortController
}

func newQueryToolUseContextPort(root *streamingtool.AbortController) *queryToolUseContextPort {
	return &queryToolUseContextPort{root: root}
}

func (p *queryToolUseContextPort) QueryAbort() *streamingtool.AbortController { return p.root }

func (p *queryToolUseContextPort) SetInProgressToolUseIDs(updater func(prev map[string]struct{}) map[string]struct{}) {
	_ = updater
}

func (p *queryToolUseContextPort) SetHasInterruptibleToolInProgress(v bool) { _ = v }
