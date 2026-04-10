package types

import (
	"encoding/json"
	"strings"
)

const toolContextLogCommandNameSample = 30
const toolContextLogMessageSample = 5

// FormatProcessInputContextForLog serializes [ProcessUserInputContextData] (ToolUseContext + options) for debug logs.
// full: entire value as JSON (very large when options.commands is the full builtin table).
// !full: compact summary — command/tool counts, head of names, message count + sample types/uuids.
// Output is json.Marshal (single line, no indent) so logs stay grep-friendly; use jq for pretty-print.
func FormatProcessInputContextForLog(rc *ProcessUserInputContextData, full bool) ([]byte, error) {
	if rc == nil {
		return []byte("null"), nil
	}
	if full {
		return json.Marshal(rc)
	}
	return json.Marshal(summarizeProcessInputContext(rc))
}

func summarizeProcessInputContext(rc *ProcessUserInputContextData) map[string]any {
	o := rc.Options
	cmdNames := make([]string, 0, len(o.Commands))
	for _, c := range o.Commands {
		n := strings.TrimSpace(c.Name)
		if n != "" {
			cmdNames = append(cmdNames, n)
		}
	}
	head := cmdNames
	if len(head) > toolContextLogCommandNameSample {
		head = append([]string(nil), head[:toolContextLogCommandNameSample]...)
	}
	toolNames := toolDefNamesFromToolsJSON(o.Tools)
	toolsHead := toolNames
	if len(toolsHead) > toolContextLogCommandNameSample {
		toolsHead = append([]string(nil), toolsHead[:toolContextLogCommandNameSample]...)
	}
	msgSample := make([]map[string]any, 0, toolContextLogMessageSample)
	for i, m := range rc.Messages {
		if i >= toolContextLogMessageSample {
			break
		}
		msgSample = append(msgSample, map[string]any{
			"type": string(m.Type),
			"uuid": m.UUID,
		})
	}
	return map[string]any{
		"toolUseContext": map[string]any{
			"options": map[string]any{
				"commandsTotal":           len(o.Commands),
				"commandNamesHead":        head,
				"mainLoopModel":           o.MainLoopModel,
				"debug":                   o.Debug,
				"verbose":                 o.Verbose,
				"isNonInteractiveSession": o.IsNonInteractiveSession,
				"thinkingConfig":          o.ThinkingConfig,
				"toolsJsonBytes":          len(o.Tools),
				"toolDefsTotal":           len(toolNames),
				"toolDefNamesHead":        toolsHead,
				"querySource":             o.QuerySource,
			},
			"messagesTotal":  len(rc.Messages),
			"messagesSample": msgSample,
		},
	}
}

func toolDefNamesFromToolsJSON(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var defs []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &defs); err != nil {
		return nil
	}
	out := make([]string, 0, len(defs))
	for _, d := range defs {
		n := strings.TrimSpace(d.Name)
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}
