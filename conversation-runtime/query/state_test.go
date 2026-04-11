package query

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestNewStateFromParams_autoCompactTrackingSeed(t *testing.T) {
	seed := json.RawMessage(`{"compacted":true}`)
	p := QueryParams{
		Messages:            []types.Message{{UUID: "m"}},
		ToolUseContext:      types.ToolUseContext{},
		AutoCompactTracking: seed,
	}
	st := NewStateFromParams(p)
	if string(st.AutoCompactTracking) != `{"compacted":true}` {
		t.Fatalf("%s", st.AutoCompactTracking)
	}
}
