package workflow

import (
	"encoding/json"
	"fmt"

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

		// Emit agent progress to UI
		label := prompt
		if opts != nil && opts.Label != "" {
			label = opts.Label
		} else if len(prompt) > 80 {
			label = prompt[:80] + "..."
		}
		state.emitProgress("running", "agent: "+label)

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
				var obj any
				if json.Unmarshal([]byte(result), &obj) == nil {
					jsResult = obj
				} else {
					jsResult = result
				}
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

	input := map[string]any{
		"description":      prompt,
		"prompt":           prompt,
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

	runnerCfg := AgentRunConfig{
		WorkDir:             cfg.WorkDir,
		ProjectRoot:         cfg.ProjectRoot,
		SessionID:           cfg.SessionID,
		TasksDir:            cfg.TasksDir,
		AvailableMCPServers: cfg.AvailableMCPServers,
		MainLoopModel:       cfg.MainLoopModel,
		ParentToolUseID:     cfg.ToolUseID,
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

	// Try to extract the message from the agent tool response format {"data":{"message":"..."}}
	var resp struct {
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}
	if json.Unmarshal([]byte(result), &resp) == nil && resp.Data.Message != "" {
		return resp.Data.Message, nil
	}

	return result, nil
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
