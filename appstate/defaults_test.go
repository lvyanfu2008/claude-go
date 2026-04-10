package appstate

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDefaultAppState_JSONRoundTrip(t *testing.T) {
	st := DefaultAppState()
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	var back AppState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.ExpandedView != ExpandedNone {
		t.Fatalf("expandedView: %q", back.ExpandedView)
	}
	if back.SelectedIPAgentIndex != -1 {
		t.Fatalf("selectedIPAgentIndex: %d", back.SelectedIPAgentIndex)
	}
	if back.Speculation.Status != SpeculationStatusIdle {
		t.Fatalf("speculation status: %q", back.Speculation.Status)
	}
}

func TestDefaultAppState_initialMessageNull(t *testing.T) {
	st := DefaultAppState()
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"initialMessage":null`) {
		t.Fatalf("want initialMessage null, got substring missing in %d bytes", len(b))
	}
}

func TestDefaultAppState_notificationsCurrentNull(t *testing.T) {
	st := DefaultAppState()
	b, err := json.Marshal(st.Notifications)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"current":null`) {
		t.Fatalf("want current null, got %s", b)
	}
}
