package paritytools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTestingPermission_dataString(t *testing.T) {
	t.Parallel()
	out, isErr, err := TestingPermissionFromJSON([]byte(`{}`))
	if err != nil || isErr {
		t.Fatal(err, isErr)
	}
	if !strings.Contains(out, "TestingPermission executed successfully") {
		t.Fatalf("out=%s", out)
	}
}

func TestListPeers_emptyArray(t *testing.T) {
	t.Parallel()
	out, isErr, err := ListPeersFromJSON(nil)
	if err != nil || isErr {
		t.Fatal(err)
	}
	var w map[string]any
	if err := json.Unmarshal([]byte(out), &w); err != nil {
		t.Fatal(err)
	}
	if _, ok := w["data"].([]any); !ok {
		t.Fatalf("got %#v", w["data"])
	}
}

func TestSleep_contextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := SleepFromJSON(ctx, []byte(`{"seconds":10}`))
	if err == nil {
		t.Fatal("expected cancel")
	}
}

func TestVerifyPlanExecution_unavailableShape(t *testing.T) {
	t.Parallel()
	out, isErr, err := VerifyPlanExecutionFromJSON(nil)
	if err != nil || isErr {
		t.Fatal(err, isErr)
	}
	if !strings.Contains(out, `"success":false`) {
		t.Fatalf("out=%s", out)
	}
}
