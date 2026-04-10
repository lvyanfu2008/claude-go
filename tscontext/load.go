package tscontext

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// DefaultBridgeExecTimeout is the wall-clock limit for one `bun run go-context-bridge` subprocess
// when GOU_DEMO_TS_BRIDGE_TIMEOUT_SEC is unset or invalid. Cold TS init() often needs several minutes;
// a 2-minute cap caused frequent timeouts and looked like a hang when stderr was buffered.
const DefaultBridgeExecTimeout = 5 * time.Minute

// BridgeExecTimeout is the default timeout (alias for docs / backward compat).
const BridgeExecTimeout = DefaultBridgeExecTimeout

// EffectiveBridgeExecTimeout returns the actual timeout (override with GOU_DEMO_TS_BRIDGE_TIMEOUT_SEC,
// integer seconds, minimum 30).
func EffectiveBridgeExecTimeout() time.Duration {
	s := strings.TrimSpace(os.Getenv("GOU_DEMO_TS_BRIDGE_TIMEOUT_SEC"))
	if s == "" {
		return DefaultBridgeExecTimeout
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 30 {
		return DefaultBridgeExecTimeout
	}
	return time.Duration(n) * time.Second
}

// BridgeRequest is the first-line JSON sent to go-context-bridge stdin.
type BridgeRequest struct {
	Cwd                          string   `json:"cwd,omitempty"`
	AdditionalWorkingDirectories []string `json:"additionalWorkingDirectories,omitempty"`
	CustomSystemPrompt           string   `json:"customSystemPrompt,omitempty"`
}

// LoadSnapshotOnce runs `bun run go-context-bridge` in repoRoot with inherited env.
// workingDir should be the user's cwd (gou-demo process); repoRoot is the claude-code checkout (contains package.json).
// Bun stderr is copied to os.Stderr in real time so startup progress is visible (stdout stays buffered for the JSON line).
func LoadSnapshotOnce(ctx context.Context, repoRoot string, workingDir string, extraClaudeMdRoots []string) (*Snapshot, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return nil, errors.New("tscontext: empty repoRoot")
	}
	wd := strings.TrimSpace(workingDir)
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			wd = "."
		}
	}

	req := BridgeRequest{
		Cwd:                          wd,
		AdditionalWorkingDirectories: append([]string(nil), extraClaudeMdRoots...),
	}
	reqLine, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	reqLine = append(reqLine, '\n')

	deadline := EffectiveBridgeExecTimeout()
	cctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	cmd := exec.CommandContext(cctx, "bun", "run", "go-context-bridge")
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	cmd.Stdin = bytes.NewReader(reqLine)

	var stdout, stderrBuf bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tscontext: go-context-bridge: %w\nstderr: %s", err, strings.TrimSpace(stderrBuf.String()))
	}

	first, err := readFirstLine(&stdout)
	if err != nil {
		return nil, err
	}
	first = strings.TrimSpace(first)
	if first == "" {
		return nil, errors.New("tscontext: empty stdout from go-context-bridge")
	}

	var errBody struct {
		Error string `json:"error"`
	}
	if json.Unmarshal([]byte(first), &errBody) == nil && strings.TrimSpace(errBody.Error) != "" {
		return nil, fmt.Errorf("tscontext: bridge error: %s", errBody.Error)
	}

	var snap Snapshot
	if err := json.Unmarshal([]byte(first), &snap); err != nil {
		return nil, fmt.Errorf("tscontext: decode snapshot: %w", err)
	}
	return &snap, nil
}

func readFirstLine(r *bytes.Buffer) (string, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 256*1024), 16*1024*1024)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", err
		}
		return "", errors.New("tscontext: no stdout line")
	}
	return sc.Text(), nil
}
