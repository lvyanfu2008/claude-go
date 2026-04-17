//go:build integration

package engine

import (
	"context"
	"os"
	"testing"

	"goc/ccb-engine/llmturn"
)

func TestSessionRunTurn_LiveAPI(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" {
		t.Skip("set ANTHROPIC_API_KEY or ANTHROPIC_AUTH_TOKEN in the environment")
	}
	completer := llmturn.NewFromEnv()
	sess := NewSession(nil)
	sess.AppendUserText("Reply with exactly the word: pong")
	ctx := context.Background()
	err := sess.RunTurn(ctx, completer, nil, "", StubRunner{}, false)
	if err != nil {
		t.Fatal(err)
	}
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if len(sess.messages) < 2 {
		t.Fatalf("expected >=2 messages, got %d", len(sess.messages))
	}
}
