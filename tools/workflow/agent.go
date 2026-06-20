package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

// completion carries the result of an agent execution back to the event loop.
// rawResult is the Go value (not yet converted to goja.Value) because
// the goroutine must not call vm.ToValue() — Goja is not goroutine-safe.
type completion struct {
	resolve   func(interface{}) error
	reject    func(interface{}) error
	rawResult interface{} // raw Go value, converted to goja.Value in event loop
	err       error
	vm        *goja.Runtime // VM reference for conversion in event loop
}

// AgentRunnerFunc is the function signature for spawning a subagent.
// raw is JSON-encoded AgentToolInput, returns (resultJSON, isError, error).
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

// agentRunnerFn is set by the tools package during init to avoid circular imports.
var agentRunnerFn AgentRunnerFunc

// SetAgentRunner wires the agent execution function into the workflow package.
func SetAgentRunner(fn AgentRunnerFunc) {
	agentRunnerFn = fn
}

// bindAgent returns a JS-callable function: agent(prompt, opts?).
// Each call spawns a subagent via RunAgentTool in a goroutine.
// The function returns a Promise that resolves when the agent completes.
func bindAgent(vm *goja.Runtime, state *RunState, cfg EngineConfig, completionCh chan<- completion) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		// Parse prompt (required, first argument)
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) {
			panic(vm.NewTypeError("agent() requires a prompt string as first argument"))
		}
		prompt := call.Arguments[0].String()

		// Parse opts (optional, second argument)
		var opts *AgentOpts
		if len(call.Arguments) >= 2 && !goja.IsUndefined(call.Arguments[1]) && !goja.IsNull(call.Arguments[1]) {
			opts = extractAgentOpts(call.Arguments[1])
		}

		// Check budget before spawning
		if state.Budget.IsExhausted() {
			panic(vm.NewGoError(fmt.Errorf("budget exhausted: spent %d of %d tokens", state.Budget.Spent(), state.Budget.Total())))
		}

		// Check abort
		select {
		case <-state.abortCh:
			return goja.Null()
		default:
		}

		// Check resume cache
		callHash := HashAgentCall(prompt, opts)
		if cached := state.Journal.Lookup(callHash); cached != nil {
			var result any
			if err := json.Unmarshal(cached.Result, &result); err == nil {
				return vm.ToValue(result)
			}
		}

		// Acquire concurrency slot (blocks until available or aborted)
		if !state.AcquireSlot() {
			return goja.Null()
		}

		// Create a Goja Promise
		promise, resolve, reject := vm.NewPromise()

		// Emit agent progress to UI (use unique per-agent ID)
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

		// Spawn agent in goroutine
		state.AgentCount.Add(1)
		go func() {
			defer state.ReleaseSlot()

			// Execute agent
			result, err := executeAgent(prompt, opts, callHash, state, cfg)
			if err != nil {
				completionCh <- completion{reject: reject, err: err}
				return
			}

			// Parse result for JS (no Goja calls — must happen in event loop goroutine)
			var jsResult any
			if opts != nil && len(opts.Schema) > 0 {
				jsResult = adaptToSchema(extractJSON(result), opts.Schema)
			} else {
				jsResult = result
			}

			completionCh <- completion{resolve: resolve, rawResult: jsResult, vm: vm}
		}()

		return vm.ToValue(promise)
	}
}

// executeAgent builds AgentRunConfig and calls the injected runner function.
func executeAgent(prompt string, opts *AgentOpts, callHash string, state *RunState, cfg EngineConfig) (string, error) {
	if agentRunnerFn == nil {
		return "", fmt.Errorf("workflow: agent runner not initialized")
	}

	// Build readable context from parent messages: extract text, tool uses, and tool results.
	var contextBuilder strings.Builder
	if len(cfg.Messages) > 0 {
		contextBuilder.WriteString("<PARENT_CONTEXT>\n")
		contextBuilder.WriteString("The parent conversation already read these files. Their contents:\n\n")
		for _, m := range cfg.Messages {
			// Parse content blocks (assistant messages have [{"type":"text",...},{"type":"tool_use",...},{"type":"tool_result",...}])
			var blocks []struct {
				Type    string          `json:"type"`
				Text    string          `json:"text,omitempty"`
				Name    string          `json:"name,omitempty"`
				Input   json.RawMessage `json:"input,omitempty"`
				Content json.RawMessage `json:"content,omitempty"`
			}
			if json.Unmarshal([]byte(m.Content), &blocks) != nil {
				continue // skip unparseable messages
			}
			for _, b := range blocks {
				switch b.Type {
				case "text":
					// Skip very long text blocks (likely LLM narration, not useful)
					if len(b.Text) < 500 {
						contextBuilder.WriteString(b.Text)
						contextBuilder.WriteString("\n")
					}
				case "tool_use":
					contextBuilder.WriteString("[Tool: " + b.Name + "]")
					if len(b.Input) > 0 {
						contextBuilder.WriteString(" " + string(b.Input))
					}
					contextBuilder.WriteString("\n")
				case "tool_result":
					// Include file content from Read/Grep/Glob results
					var contentStr string
					if json.Unmarshal(b.Content, &contentStr) == nil {
						// Success: b.Content was a JSON string
					} else {
						contentStr = string(b.Content)
					}
					const maxLen = 8000
					if len(contentStr) > maxLen {
						contentStr = contentStr[:maxLen] + "\n... [truncated]"
					}
					if len(contentStr) > 0 {
						contextBuilder.WriteString(contentStr)
						contextBuilder.WriteString("\n")
					}
				}
			}
		}
		contextBuilder.WriteString("</PARENT_CONTEXT>\n\n")
	}

	promptText := contextBuilder.String() + prompt
	if opts != nil && len(opts.Schema) > 0 {
		promptText = promptText + `

<STRICT_OUTPUT_FORMAT>
You MUST respond with a single valid JSON object matching this schema:
` + string(opts.Schema) + `

RULES:
1. Your ENTIRE response must be parseable as JSON. No markdown, no explanation, no code fences.
2. Start your response with { and end with }.
3. Do NOT wrap the JSON in ` + "```" + `json blocks.
4. All fields in the schema are required unless marked optional.
5. If you cannot find any issues for a field, use an empty array [] or appropriate null/empty value — do NOT omit the field.
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

	// Marshal parent messages so subagent inherits file read context
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
		return "", err
	}

	// Record in journal for resume
	state.Journal.Record(callHash, json.RawMessage(fmt.Sprintf("%q", result)))

	if isErr {
		return "", fmt.Errorf("agent error: %s", result)
	}

	// Try to extract result from agent tool response: {"data":{"message":"...","output":"..."}}
	var resp struct {
		Data struct {
			Message string `json:"message"`
			Output  string `json:"output"`
		} `json:"data"`
	}
	if json.Unmarshal([]byte(result), &resp) == nil {
		if resp.Data.Message != "" {
			return resp.Data.Message, nil
		}
		if resp.Data.Output != "" {
			return resp.Data.Output, nil
		}
	}

	return result, nil
}


// extractJSON attempts to extract a JSON object from agent response text.
// Uses multiple strategies to handle various response formats.
func extractJSON(text string) any {
	trimmed := strings.TrimSpace(text)

	// 1. Try direct parse
	var obj any
	if json.Unmarshal([]byte(trimmed), &obj) == nil {
		return obj
	}

	// 2. Try to find JSON inside markdown code blocks ```json ... ``` or ``` ... ```
	for _, fence := range []string{"```json", "```"} {
		if idx := strings.Index(trimmed, fence); idx >= 0 {
			start := idx + len(fence)
			// Skip optional language tag newline
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

	// 3. Find outermost { ... } pair by tracking brace depth
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

	// 4. Try to find first [ ... ] pair
	if idx := strings.Index(trimmed, "["); idx >= 0 {
		if end := strings.LastIndex(trimmed, "]"); end > idx {
			if json.Unmarshal([]byte(trimmed[idx:end+1]), &obj) == nil {
				return obj
			}
		}
	}

	return trimmed // fallback to raw text
}

// adaptToSchema wraps the extracted value to match the schema when possible.
// Common case: agent returns [...] but schema expects {findings: [...]}.
func adaptToSchema(val any, schema json.RawMessage) any {
	if val == nil {
		return val
	}
	// Parse schema to find expected object properties
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

	// If val is an array and schema expects {singleArrayProp: [...]}, wrap it
	if arr, ok := val.([]any); ok {
		for propName, prop := range s.Properties {
			if prop.Type == "array" {
				return map[string]any{propName: arr}
			}
		}
	}

	// If val is a map but missing the wrapper key, and there's only one array prop, wrap
	if m, ok := val.(map[string]any); ok {
		hasWrapperKey := false
		for key := range s.Properties {
			if _, ok := m[key]; ok {
				hasWrapperKey = true
				break
			}
		}
		if !hasWrapperKey {
			for propName, prop := range s.Properties {
				if prop.Type == "array" {
					if _, ok := m[propName]; !ok {
						return map[string]any{propName: val}
					}
				}
			}
		}
	}

	return val
}

// extractAgentOpts converts a Goja value to AgentOpts.
func extractAgentOpts(val goja.Value) *AgentOpts {
	exported := val.Export()
	if exported == nil {
		return nil
	}
	// Re-marshal through JSON to get clean struct
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
