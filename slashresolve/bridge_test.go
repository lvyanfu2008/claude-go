package slashresolve

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"goc/types"
)

func TestResolveViaBridge_contract(t *testing.T) {
	if _, err := exec.LookPath("bun"); err != nil {
		t.Skip("bun not in PATH")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	testDir := filepath.Dir(file)
	repoRoot := FindSlashResolveBridgeRoot(testDir)
	if repoRoot == "" {
		t.Skip("scripts/slash-resolve-bridge.ts not found; set SLASH_RESOLVE_BRIDGE_ROOT to the TS repo root (e.g. claude-code)")
	}

	req := BridgeRequest{
		CommandName: "demo",
		Cwd:         "/tmp",
		Args:        "",
		CommandJSON: json.RawMessage(`{"type":"prompt","name":"demo"}`),
	}
	res, err := ResolveViaBridge(repoRoot, req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != types.SlashResolveTSBridge {
		t.Fatalf("source %q", res.Source)
	}
	if res.BridgeMeta == nil || res.BridgeMeta.BridgeVersion == "" {
		t.Fatalf("bridge meta: %+v", res.BridgeMeta)
	}
	if res.UserText == "" {
		t.Fatal("empty userText")
	}
}
