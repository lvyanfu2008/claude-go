// Command process-user-input reads a JSON envelope on stdin and writes
// a union JSON on stdout:
// - { "kind": "result", "result": ProcessUserInputBaseResult }
// - { "kind": "execution_request", "action": ExecutionRequest, "actions"?: ExecutionRequest[] }
// - { "kind": "execution_result", "executionResult": ProcessUserInputBaseResult }
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"goc/diagnostics"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/types"
)

const protocolVersion = "goc-process-user-input-v1"

// stdinEnvelope is the JSON request shape for this CLI (protocolVersion + args + optional fields).
type stdinEnvelope struct {
	V                   string                                       `json:"v"`
	PermissionMode      types.PermissionMode                         `json:"permissionMode,omitempty"`
	Args                processuserinput.ProcessUserInputArgs        `json:"args"`
	ExecutionResult     *processuserinput.ProcessUserInputBaseResult `json:"executionResult,omitempty"`
	ExecutionResultKind string                                       `json:"executionResultKind,omitempty"`
	StatePatchAck       *processuserinput.StatePatchAck              `json:"statePatchAck,omitempty"`
	// GoCommandsLoad optional: extra options for Go slash/skill loading (cwd override, touchedFiles, auth). Commands always come from Go.
	GoCommandsLoad *goCommandsLoad `json:"goCommandsLoad,omitempty"`
}

type stdoutKind string

const (
	stdoutKindResult           stdoutKind = "result"
	stdoutKindExecutionRequest stdoutKind = "execution_request"
	stdoutKindExecutionResult  stdoutKind = "execution_result"
)

type stdoutEnvelope struct {
	Kind            stdoutKind                                   `json:"kind"`
	Result          *processuserinput.ProcessUserInputBaseResult `json:"result,omitempty"`
	Action          *processuserinput.ExecutionRequest           `json:"action,omitempty"`
	Actions         []processuserinput.ExecutionRequest          `json:"actions,omitempty"`
	ExecutionResult *processuserinput.ProcessUserInputBaseResult `json:"executionResult,omitempty"`
	StatePatchBatch *processuserinput.StatePatchBatch            `json:"statePatchBatch,omitempty"`
}

const envEmitExecutionResult = "CLAUDE_CODE_GO_PROCESS_USER_INPUT_EMIT_EXECUTION_RESULT"
const envConsumeExecutionResult = "CLAUDE_CODE_GO_PROCESS_USER_INPUT_CONSUME_EXECUTION_RESULT"

func envEnabledDefaultTrue(name string) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return true
	}
	switch strings.ToLower(raw) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func maybeConsumeExecutionResultFromStdin(
	env stdinEnvelope,
	consume bool,
) *stdoutEnvelope {
	if env.ExecutionResult == nil || !consume {
		return nil
	}
	executionResult := reduceConsumedExecutionResult(
		env.ExecutionResultKind,
		env.ExecutionResult,
	)
	return &stdoutEnvelope{
		Kind:            stdoutKindExecutionResult,
		ExecutionResult: executionResult,
	}
}

func reduceConsumedExecutionResult(
	kind string,
	in *processuserinput.ProcessUserInputBaseResult,
) *processuserinput.ProcessUserInputBaseResult {
	if in == nil {
		return in
	}
	if kind == "hooks_plan" && in.HooksReducerInput != nil {
		if in.HooksReducerInput.Blocked || in.HooksReducerInput.PreventContinuation {
			in.ShouldQuery = false
		}
	}
	return in
}

func buildStdoutEnvelope(
	out *processuserinput.ProcessUserInputBaseResult,
	emitExecutionResult bool,
) stdoutEnvelope {
	envOut := stdoutEnvelope{}
	if out != nil && len(out.ExecutionSequence) > 0 {
		envOut.Kind = stdoutKindExecutionRequest
		envOut.Actions = out.ExecutionSequence
		envOut.Action = &out.ExecutionSequence[0]
		return envOut
	}
	if out != nil && out.Execution != nil {
		envOut.Kind = stdoutKindExecutionRequest
		envOut.Action = out.Execution
		return envOut
	}
	if emitExecutionResult {
		envOut.Kind = stdoutKindExecutionResult
		envOut.ExecutionResult = out
		return envOut
	}
	envOut.Kind = stdoutKindResult
	envOut.Result = out
	return envOut
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "process-user-input: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Start context load tracking
	tracker := diagnostics.NewContextLoadTracker()
	tracker.StartPhase("total_processing")

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		tracker.Complete(0, false)
		return fmt.Errorf("read stdin: %w", err)
	}
	if len(data) == 0 {
		tracker.Complete(0, false)
		return fmt.Errorf("empty stdin")
	}

	var env stdinEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		tracker.Complete(0, false)
		return fmt.Errorf("json: %w", err)
	}
	if env.V != "" && env.V != protocolVersion {
		tracker.Complete(0, false)
		return fmt.Errorf("unsupported protocol %q (want %q)", env.V, protocolVersion)
	}

	args := env.Args
	pm := env.PermissionMode
	if pm == "" {
		pm = types.PermissionDefault
	}

	// Slash/skill list comes from applyGoCommandsLoad; drop any stale commands slice from the request body.
	rc := args.Context
	rc.Options.Commands = nil

	p := &processuserinput.ProcessUserInputParams{
		Input:                    args.Input,
		PreExpansionInput:        args.PreExpansionInput,
		Mode:                     args.Mode,
		PastedContents:           args.PastedContents,
		IdeSelection:             args.IdeSelection,
		Messages:                 args.Messages,
		UUID:                     args.UUID,
		IsAlreadyProcessing:      args.IsAlreadyProcessing,
		QuerySource:              args.QuerySource,
		SkipSlashCommands:        args.SkipSlashCommands,
		BridgeOrigin:             args.BridgeOrigin,
		IsMeta:                   args.IsMeta,
		SkipAttachments:          args.SkipAttachments,
		Commands:                 nil,
		PermissionMode:           pm,
		RuntimeContext:           &rc,
		BridgeAttachmentMessages: args.BridgeAttachmentMessages,
		StatePatchAck:            args.StatePatchAck,
		// Bash / slash / hooks: nil (MVP); GetAttachmentMessages nil unless stdin omits bridgeAttachmentMessages.
	}

	tracker.StartPhase("command_loading")
	if err := applyGoCommandsLoad(context.Background(), p, env.GoCommandsLoad); err != nil {
		tracker.EndPhase("command_loading")
		tracker.Complete(0, false)
		return fmt.Errorf("go commands: %w", err)
	}
	tracker.EndPhase("command_loading")

	logPath := strings.TrimSpace(os.Getenv(envPuiDebugLog))
	toStderr := isEnvTruthy(os.Getenv(envPuiDebugStderr))

	tracker.StartPhase("debug_logging")
	// Distinct marker so session logs show this turn used the Go CLI.
	logProcessUserInputDebug(logPath, toStderr, "via", map[string]any{
		"engine":   "go",
		"protocol": protocolVersion,
	})
	logToolUseContextForCLI(logPath, toStderr, p.RuntimeContext)
	logProcessUserInputDebug(logPath, toStderr, "IN", buildInPayload(&args))
	if env.StatePatchAck != nil {
		logProcessUserInputDebug(logPath, toStderr, "IN_STATE_PATCH_ACK", map[string]any{
			"patchId":    env.StatePatchAck.PatchID,
			"applied":    env.StatePatchAck.Applied,
			"reason":     env.StatePatchAck.Reason,
			"newVersion": env.StatePatchAck.NewVersion,
		})
	}
	if env.ExecutionResult != nil {
		logProcessUserInputDebug(logPath, toStderr, "IN_EXECUTION_RESULT", buildResultPayload("stdin", env.ExecutionResult))
		if consumed := maybeConsumeExecutionResultFromStdin(
			env,
			envEnabledDefaultTrue(envConsumeExecutionResult),
		); consumed != nil {
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			if err := enc.Encode(consumed); err != nil {
				tracker.EndPhase("debug_logging")
				tracker.Complete(0, false)
				return fmt.Errorf("encode consumed execution_result: %w", err)
			}
			tracker.EndPhase("debug_logging")
			tracker.Complete(1, true)
			return nil
		}
	}
	tracker.EndPhase("debug_logging")

	tracker.StartPhase("callback_wiring")
	wireProcessUserInputCallbacks(p, logPath, toStderr)
	tracker.EndPhase("callback_wiring")

	tracker.StartPhase("process_user_input")
	out, err := processuserinput.ProcessUserInput(context.Background(), p)
	tracker.EndPhase("process_user_input")

	if err != nil {
		tracker.StartPhase("error_logging")
		logProcessUserInputDebug(logPath, toStderr, "ERROR", map[string]any{"error": err.Error()})
		tracker.EndPhase("error_logging")
		tracker.Complete(0, false)
		return err
	}

	tracker.StartPhase("result_logging")
	logProcessUserInputDebug(logPath, toStderr, "AFTER_BASE", buildResultPayload("", out))
	logProcessUserInputDebug(logPath, toStderr, "OUT", buildResultPayload("go-cli", out))
	tracker.EndPhase("result_logging")

	tracker.StartPhase("output_encoding")
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	envOut := buildStdoutEnvelope(
		out,
		envEnabledDefaultTrue(envEmitExecutionResult),
	)
	if out != nil && out.StatePatchBatch != nil {
		envOut.StatePatchBatch = out.StatePatchBatch
	}
	if err := enc.Encode(envOut); err != nil {
		tracker.EndPhase("output_encoding")
		tracker.Complete(0, false)
		return fmt.Errorf("encode result: %w", err)
	}
	tracker.EndPhase("output_encoding")

	tracker.Complete(1, true)
	return nil
}
