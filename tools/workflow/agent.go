package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

// workflowLogDir is set by the engine to persist per-agent traces.
var workflowLogDir string
var workflowLogMu sync.Mutex

// SetWorkflowLogDir sets the directory for per-agent debug traces.
func SetWorkflowLogDir(dir string) { workflowLogDir = dir }

func wfLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if workflowLogDir != "" {
		workflowLogMu.Lock()
		defer workflowLogMu.Unlock()
		path := filepath.Join(workflowLogDir, "agent-traces.log")
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			f.WriteString(msg)
			f.Close()
		}
	}
}

// completion carries the result of an agent execution back to the event loop.
type completion struct {
	resolve   func(interface{}) error
	reject    func(interface{}) error
	rawResult interface{}
	err       error
	vm        *goja.Runtime
}

// AgentRunnerFunc is the function signature for spawning a subagent.
type AgentRunnerFunc func(raw []byte, cfg AgentRunConfig) (string, bool, error)

// AgentRunConfig carries the necessary configuration for spawning a subagent.
type AgentRunConfig struct {
	WorkDir             string
	ProjectRoot         string
	SessionID           string
	TasksDir            string
	AvailableMCPServers []string
	MessagesJSON        json.RawMessage
	SystemPrompt        []string
	MainLoopModel       string
	ProgressCallback    func(json.RawMessage)
	NotificationCB      func(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64)
	ToolPermissionJSON  json.RawMessage
	ParentToolUseID     string
}

var agentRunnerFn AgentRunnerFunc

func SetAgentRunner(fn AgentRunnerFunc) { agentRunnerFn = fn }

func bindAgent(vm *goja.Runtime, state *RunState, cfg EngineConfig, completionCh chan<- completion) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) {
			panic(vm.NewTypeError("agent() requires a prompt string as first argument"))
		}
		prompt := call.Arguments[0].String()

		var opts *AgentOpts
		if len(call.Arguments) >= 2 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
			opts = extractAgentOpts(call.Arguments[1])
		}

		if state.Budget.IsExhausted() {
			panic(vm.NewGoError(fmt.Errorf("budget exhausted")))
		}

		select {
		case <-state.abortCh:
			return goja.Null()
		default:
		}

		callHash := HashAgentCall(prompt, opts)
		if cached := state.Journal.Lookup(callHash); cached != nil {
			var result any
			if err := json.Unmarshal(cached.Result, &result); err == nil {
				return vm.ToValue(result)
			}
		}

		if !state.AcquireSlot() {
			return goja.Null()
		}

		promise, resolve, reject := vm.NewPromise()

		agentID := state.newAgentProgressID()
		label := prompt
		if opts != nil && opts.Label != "" {
			label = opts.Label
		} else if len(prompt) > 80 {
			label = prompt[:80] + "..."
		}
		if state.WorkflowProgressCallback != nil {
			state.WorkflowProgressCallback(agentID, "running", "agent: "+label)
		}

		state.AgentCount.Add(1)
		go func() {
			defer state.ReleaseSlot()

			result, err := executeAgent(prompt, opts, callHash, state, cfg)
			if err != nil {
				if state.WorkflowProgressCallback != nil {
						state.WorkflowProgressCallback(agentID, "completed", "agent error")
					}
					completionCh <- completion{reject: reject, err: err}
				return
			}

			var jsResult any
			if opts != nil && len(opts.Schema) > 0 {
				extracted := extractJSON(result)
				adapted := adaptToSchema(extracted, opts.Schema)
				wfLog( "[wf-agent] JSON_STAGES hash=%s rawType=%T extractedType=%T adaptedType=%T\n",
					callHash[:12], result, extracted, adapted)
				jsResult = adapted
			} else {
				jsResult = result
			}

			if state.WorkflowProgressCallback != nil {
					state.WorkflowProgressCallback(agentID, "completed", "agent: "+label)
				}
				completionCh <- completion{resolve: resolve, rawResult: jsResult, vm: vm}
			}()

		return vm.ToValue(promise)
	}
}

func executeAgent(prompt string, opts *AgentOpts, callHash string, state *RunState, cfg EngineConfig) (string, error) {
	if agentRunnerFn == nil {
		return "", fmt.Errorf("workflow: agent runner not initialized")
	}

	var contextBuilder strings.Builder
	if len(cfg.Messages) > 0 {
		contextBuilder.WriteString("<PARENT_CONTEXT>\n")
		contextBuilder.WriteString("The parent conversation already read these files:\n\n")
		for _, m := range cfg.Messages {
			var blocks []struct {
				Type    string          `json:"type"`
				Text    string          `json:"text,omitempty"`
				Name    string          `json:"name,omitempty"`
				Input   json.RawMessage `json:"input,omitempty"`
				Content json.RawMessage `json:"content,omitempty"`
			}
			if json.Unmarshal([]byte(m.Content), &blocks) != nil {
				continue
			}
			for _, b := range blocks {
				switch b.Type {
				case "text":
					if len(b.Text) < 500 {
						contextBuilder.WriteString(b.Text + "\n")
					}
				case "tool_use":
					contextBuilder.WriteString("[Tool: " + b.Name + "]")
					if len(b.Input) > 0 {
						contextBuilder.WriteString(" " + string(b.Input))
					}
					contextBuilder.WriteString("\n")
				case "tool_result":
					var contentStr string
					if json.Unmarshal(b.Content, &contentStr) != nil {
						contentStr = string(b.Content)
					}
					const maxLen = 8000
					if len(contentStr) > maxLen {
						contentStr = contentStr[:maxLen] + "\n... [truncated]"
					}
					if len(contentStr) > 0 {
						contextBuilder.WriteString(contentStr + "\n")
					}
				}
			}
		}
		contextBuilder.WriteString("</PARENT_CONTEXT>\n\n")
	}

	promptText := contextBuilder.String() + prompt
	if opts != nil && len(opts.Schema) > 0 {
		promptText += `

<STRICT_OUTPUT_FORMAT>
You MUST respond with a single valid JSON object matching this schema:
` + string(opts.Schema) + `

RULES:
1. Your ENTIRE response must be parseable as JSON. No markdown, no explanation, no code fences.
2. Start your response with { and end with }.
3. Do NOT wrap the JSON in ` + "```" + `json blocks.
4. All fields in the schema are required unless marked optional.
5. If you cannot find any issues for a field, use an empty array [] or appropriate null/empty value.
</STRICT_OUTPUT_FORMAT>`
	}

	input := map[string]any{
		"description":      prompt,
		"prompt":           promptText,
		"subagent_type":    "general-purpose",
		"run_in_background": false,
	}
	if opts != nil {
		if opts.AgentType != "" {
			input["subagent_type"] = opts.AgentType
		}
		if opts.Model != "" {
			input["model"] = opts.Model
		}
		if opts.Isolation != "" {
			input["isolation"] = opts.Isolation
		}
	}

	raw, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("marshal agent input: %w", err)
	}

	var msgsJSON json.RawMessage
	if len(cfg.Messages) > 0 {
		msgsJSON, _ = json.Marshal(cfg.Messages)
	}

	runnerCfg := AgentRunConfig{
		WorkDir:             cfg.WorkDir,
		ProjectRoot:         cfg.ProjectRoot,
		SessionID:           cfg.SessionID,
		TasksDir:            cfg.TasksDir,
		AvailableMCPServers: cfg.AvailableMCPServers,
		MainLoopModel:       cfg.MainLoopModel,
		ParentToolUseID:     cfg.ToolUseID,
		MessagesJSON:        msgsJSON,
	}

	result, isErr, err := agentRunnerFn(raw, runnerCfg)
	if err != nil {
		wfLog( "[wf-agent] RUNNER_ERROR hash=%s err=%v\n", callHash[:12], err)
		return "", err
	}

	state.Journal.Record(callHash, json.RawMessage(fmt.Sprintf("%q", result)))

	wfLog("[wf-agent] RAW hash=%s len=%d isErr=%v preview=%.300s\n", callHash[:12], len(result), isErr, result)

	if isErr {
		wfLog( "[wf-agent] AGENT_ERR hash=%s result=%.500s\n", callHash[:12], result)
		return "", fmt.Errorf("agent error: %s", result)
	}

	var resp struct {
		Data struct {
			Message string `json:"message"`
			Output  string `json:"output"`
		} `json:"data"`
	}
	if json.Unmarshal([]byte(result), &resp) == nil {
		// Prefer output (actual agent response) over message (status text like "Agent completed")
		content := resp.Data.Output
		source := "OUTPUT"
		if content == "" {
			content = resp.Data.Message
			source = "MSG"
		}
		if content != "" {
			wfLog("[wf-agent] EXTRACT_%s hash=%s len=%d\n", source, callHash[:12], len(content))
			return content, nil
		}
		wfLog("[wf-agent] EMPTY_DATA hash=%s raw=%.200s\n", callHash[:12], result)
	} else {
		wfLog( "[wf-agent] NOT_JSON hash=%s\n", callHash[:12])
	}

	wfLog( "[wf-agent] RAW_RETURN hash=%s result=%.300s\n", callHash[:12], result)
	return result, nil
}

func extractJSON(text string) any {
	trimmed := strings.TrimSpace(text)
	var obj any
	if json.Unmarshal([]byte(trimmed), &obj) == nil {
		return obj
	}
	for _, fence := range []string{"```json", "```"} {
		if idx := strings.Index(trimmed, fence); idx >= 0 {
			start := idx + len(fence)
			if rest := trimmed[start:]; len(rest) > 0 && rest[0] == '\n' {
				start++
			}
			if end := strings.Index(trimmed[start:], "```"); end >= 0 {
				inner := strings.TrimSpace(trimmed[start : start+end])
				if json.Unmarshal([]byte(inner), &obj) == nil {
					return obj
				}
			}
		}
	}
	if idx := strings.Index(trimmed, "{"); idx >= 0 {
		depth := 0
		inString := false
		escaped := false
		for i := idx; i < len(trimmed); i++ {
			ch := trimmed[i]
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' && inString {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = !inString
				continue
			}
			if inString {
				continue
			}
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					if json.Unmarshal([]byte(trimmed[idx:i+1]), &obj) == nil {
						return obj
					}
					break
				}
			}
		}
	}
	if idx := strings.Index(trimmed, "["); idx >= 0 {
		if end := strings.LastIndex(trimmed, "]"); end > idx {
			if json.Unmarshal([]byte(trimmed[idx:end+1]), &obj) == nil {
				return obj
			}
		}
	}
	return trimmed
}

func adaptToSchema(val any, schema json.RawMessage) any {
	if val == nil {
		return val
	}
	var s struct {
		Type       string `json:"type"`
		Properties map[string]struct {
			Type  string `json:"type"`
			Items any    `json:"items"`
		} `json:"properties"`
	}
	if json.Unmarshal(schema, &s) != nil || s.Type != "object" || len(s.Properties) == 0 {
		return val
	}
	var arrayPropName string
	for name, prop := range s.Properties {
		if prop.Type == "array" {
			arrayPropName = name
			break
		}
	}
	if arr, ok := val.([]any); ok && arrayPropName != "" {
		return map[string]any{arrayPropName: arr}
	}
	if m, ok := val.(map[string]any); ok {
		if _, ok := m[arrayPropName]; ok {
			return val
		}
		for k, v := range m {
			if arr, ok := v.([]any); ok && arrayPropName != "" {
				result := make(map[string]any, len(m))
				for mk, mv := range m {
					if mk == k {
						result[arrayPropName] = arr
					} else {
						result[mk] = mv
					}
				}
				return result
			}
		}
		if arrayPropName != "" {
			return map[string]any{arrayPropName: m}
		}
	}
	return val
}

func extractAgentOpts(val goja.Value) *AgentOpts {
	exported := val.Export()
	if exported == nil {
		return nil
	}
	data, err := json.Marshal(exported)
	if err != nil {
		return nil
	}
	var opts AgentOpts
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil
	}
	return &opts
}
