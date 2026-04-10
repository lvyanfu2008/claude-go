// Package hooks runs Claude-Code-style command hooks: JSON on stdin, stdout lines,
// optional PromptRequest / PromptResponse on stdin (aligned with src/utils/hooks.ts).
package hooks

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"goc/ccb-engine/internal/protocol"
)

// PromptHandler answers PromptRequest lines from hook stdout (TS/Ink in full product).
type PromptHandler func(protocol.PromptRequest) (protocol.PromptResponse, error)

// ExecOptions configures hook subprocess execution.
type ExecOptions struct {
	Command  string // executed with sh -c
	Dir      string
	Env      []string
	JSONIn   string
	Timeout  time.Duration
	OnPrompt PromptHandler // if set, stdin stays open for prompt responses until hook exits
}

// Result is collected hook output.
type Result struct {
	Stdout, Stderr string
	ExitCode       int
	PromptsHandled int
}

// Exec runs a hook command similarly to execCommandHook in hooks.ts.
func Exec(ctx context.Context, opt ExecOptions) (Result, error) {
	if opt.Command == "" {
		return Result{}, errors.New("hooks: empty command")
	}
	if opt.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opt.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", opt.Command)
	if opt.Dir != "" {
		cmd.Dir = opt.Dir
	}
	cmd.Env = opt.Env

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return Result{}, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return Result{}, err
	}

	if err := cmd.Start(); err != nil {
		return Result{}, err
	}

	var stdoutBuf, stderrBuf strings.Builder
	var prompts int
	var outMu sync.Mutex

	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		b, _ := io.ReadAll(stderr)
		outMu.Lock()
		stderrBuf.Write(b)
		outMu.Unlock()
	}()

	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		if opt.OnPrompt != nil {
			defer func() { _ = stdin.Close() }()
		}

		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			outMu.Lock()
			stdoutBuf.WriteString(line)
			stdoutBuf.WriteByte('\n')
			outMu.Unlock()

			if opt.OnPrompt == nil {
				continue
			}
			trim := strings.TrimSpace(line)
			if trim == "" {
				continue
			}
			var pr protocol.PromptRequest
			if json.Unmarshal([]byte(trim), &pr) != nil || pr.Prompt == "" {
				continue
			}
			resp, herr := opt.OnPrompt(pr)
			if herr != nil {
				return
			}
			b, _ := json.Marshal(resp)
			if _, werr := stdin.Write(append(b, '\n')); werr != nil {
				return
			}
			outMu.Lock()
			prompts++
			outMu.Unlock()
		}
	}()

	_, werr := io.WriteString(stdin, opt.JSONIn)
	if werr == nil && !strings.HasSuffix(opt.JSONIn, "\n") {
		_, werr = stdin.Write([]byte{'\n'})
	}
	if opt.OnPrompt == nil {
		_ = stdin.Close()
	}
	if werr != nil {
		_ = cmd.Process.Kill()
		<-scanDone
		<-stderrDone
		return Result{}, werr
	}

	waitErr := cmd.Wait()
	<-scanDone
	<-stderrDone

	outMu.Lock()
	r := Result{
		Stdout:         stdoutBuf.String(),
		Stderr:         stderrBuf.String(),
		PromptsHandled: prompts,
	}
	if cmd.ProcessState != nil {
		r.ExitCode = cmd.ProcessState.ExitCode()
	}
	outMu.Unlock()

	if ctx.Err() != nil {
		return r, ctx.Err()
	}
	if waitErr != nil {
		if exit, ok := waitErr.(*exec.ExitError); ok {
			r.ExitCode = exit.ExitCode()
		}
		return r, fmt.Errorf("hook: %w", waitErr)
	}
	return r, nil
}

// DefaultTimeout is a reasonable upper bound for command hooks.
const DefaultTimeout = 120 * time.Second
