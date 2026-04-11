package localturn

import (
	"context"
	"encoding/json"
	"testing"
)

func TestQueryBridgeRun_invalidMessages(t *testing.T) {
	ctx := context.Background()
	err := QueryBridgeRun(ctx, QueryBridgeParams{
		RequestID: "r1",
		Messages:  json.RawMessage(`not-json`),
		System:    "s",
		Tools:     json.RawMessage(`[]`),
	}, func(StreamEvent) {})
	if err == nil {
		t.Fatal("expected error")
	}
}
