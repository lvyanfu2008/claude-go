package slashresolve

import (
	"encoding/json"
	"testing"

	"goc/types"
	"goc/utils"
)

func TestSlashResolveResult_JSON_unmarshal_effortString(t *testing.T) {
	raw := `{
		"userText": "hello",
		"allowedTools": ["Bash"],
		"model": "claude-sonnet-4-20250514",
		"effort": "high",
		"source": "ts_bridge",
		"bridgeMeta": { "bridgeVersion": "0.2.0", "latencyMs": 1, "requestId": "r1" }
	}`
	var out types.SlashResolveResult
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	if out.UserText != "hello" {
		t.Fatal(out.UserText)
	}
	if out.Effort == nil {
		t.Fatal("effort nil")
	}
	lvl, ok := out.Effort.Level()
	if !ok || lvl != utils.EffortHigh {
		t.Fatalf("%v", out.Effort.String())
	}
}
