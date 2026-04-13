package toolexecution

import (
	"encoding/json"
	"testing"

	"goc/ccb-engine/bashzog"
)

func TestBashInputUsesSandboxForRule1b(t *testing.T) {
	if !BashInputUsesSandboxForRule1b(json.RawMessage(`{"command":"ls"}`)) {
		t.Fatal("expected true")
	}
	if BashInputUsesSandboxForRule1b(json.RawMessage(`{}`)) {
		t.Fatal("empty command")
	}
	if BashInputUsesSandboxForRule1b(json.RawMessage(`{"command":"x","dangerously_disable_sandbox":true}`)) {
		t.Fatal("disabled sandbox")
	}
}

func TestWholeToolAskSkippedForBash1b(t *testing.T) {
	b := &BashSandboxRule1b{SandboxingEnabled: true, AutoAllowWholeToolAskWhenSandboxed: true}
	if !WholeToolAskSkippedForBash1b(BashToolName, json.RawMessage(`{"command":"ls"}`), b) {
		t.Fatal("bash+command+sandbox flags")
	}
	if !WholeToolAskSkippedForBash1b(bashzog.ZogToolName, json.RawMessage(`{"command":"ls"}`), b) {
		t.Fatal("bashzog tool name should match Bash 1b rule")
	}
	if WholeToolAskSkippedForBash1b("Read", json.RawMessage(`{"command":"ls"}`), b) {
		t.Fatal("non-bash")
	}
	if WholeToolAskSkippedForBash1b(BashToolName, json.RawMessage(`{"command":"ls"}`), nil) {
		t.Fatal("nil opts")
	}
}
