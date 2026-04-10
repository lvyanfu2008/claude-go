package slashresolve

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"goc/types"
)

const bridgeScript = "scripts/slash-resolve-bridge.ts"

// FindSlashResolveBridgeRoot returns the directory containing scripts/slash-resolve-bridge.ts.
// If SLASH_RESOLVE_BRIDGE_ROOT is set to a valid path, that wins; otherwise walks upward from startDir.
// Empty string means not found (standalone goc without a TS monorepo checkout).
func FindSlashResolveBridgeRoot(startDir string) string {
	if p := strings.TrimSpace(os.Getenv("SLASH_RESOLVE_BRIDGE_ROOT")); p != "" {
		p = filepath.Clean(p)
		if fi, err := os.Stat(filepath.Join(p, bridgeScript)); err == nil && !fi.IsDir() {
			return p
		}
	}
	dir := filepath.Clean(startDir)
	for range 32 {
		if fi, err := os.Stat(filepath.Join(dir, bridgeScript)); err == nil && !fi.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// BridgeRequest is stdin JSON for the TS bridge worker.
type BridgeRequest struct {
	CommandName string          `json:"commandName"`
	Cwd         string          `json:"cwd"`
	Args        string          `json:"args"`
	CommandJSON json.RawMessage `json:"commandJson"`
}

// ResolveViaBridge runs the Bun bridge script (repo root) and returns SlashResolveResult.
// bridgeRoot is the repository root containing scripts/slash-resolve-bridge.ts.
func ResolveViaBridge(bridgeRoot string, req BridgeRequest) (types.SlashResolveResult, error) {
	script := filepath.Join(bridgeRoot, bridgeScript)
	in, err := json.Marshal(req)
	if err != nil {
		return types.SlashResolveResult{}, err
	}
	start := time.Now()
	cmd := exec.Command("bun", "run", script)
	cmd.Dir = bridgeRoot
	cmd.Stdin = bytes.NewReader(in)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("slashresolve bridge: %w: %s", err, stderr.String())
	}
	var out types.SlashResolveResult
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return types.SlashResolveResult{}, fmt.Errorf("slashresolve bridge json: %w", err)
	}
	if out.Source != types.SlashResolveTSBridge {
		return types.SlashResolveResult{}, errors.New("slashresolve bridge: response source must be ts_bridge")
	}
	if out.BridgeMeta == nil {
		return types.SlashResolveResult{}, errors.New("slashresolve bridge: BridgeMeta required")
	}
	if out.BridgeMeta.BridgeVersion == "" && out.BridgeMeta.LatencyMs == 0 && out.BridgeMeta.RequestID == "" {
		return types.SlashResolveResult{}, errors.New("slashresolve bridge: BridgeMeta must set at least one field")
	}
	if out.BridgeMeta.LatencyMs == 0 {
		ms := time.Since(start).Milliseconds()
		out.BridgeMeta.LatencyMs = ms
	}
	return out, nil
}
