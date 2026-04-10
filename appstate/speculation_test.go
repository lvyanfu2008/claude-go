package appstate

import (
	"encoding/json"
	"testing"
)

func TestCompletionBoundary_roundTrip(t *testing.T) {
	b := CompletionBoundary{Type: BoundaryEdit, CompletedAt: 42, ToolName: "Edit", FilePath: "a.go"}
	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	var got CompletionBoundary
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Type != BoundaryEdit || got.FilePath != "a.go" || got.CompletedAt != 42 {
		t.Fatalf("%+v", got)
	}
}

func TestNormalizeSpeculationCollections_active(t *testing.T) {
	s := SpeculationState{Status: SpeculationStatusActive, ID: "x"}
	normalizeSpeculationCollections(&s)
	if s.Messages == nil || s.WrittenPaths == nil {
		t.Fatal("expected non-nil slices")
	}
}

func TestNormalizeSpeculationCollections_idleNoOp(t *testing.T) {
	s := IdleSpeculationState()
	normalizeSpeculationCollections(&s)
	if s.Messages != nil || s.WrittenPaths != nil {
		t.Fatalf("%+v", s)
	}
}

func TestSpeculationState_idleJSON(t *testing.T) {
	s := IdleSpeculationState()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"status":"idle"}` {
		t.Fatalf("got %s", b)
	}
}

func TestStore_Update(t *testing.T) {
	st := NewStore(DefaultAppState())
	st.Update(func(prev AppState) AppState {
		prev.Verbose = true
		return prev
	})
	if !st.GetState().Verbose {
		t.Fatal("expected verbose")
	}
}
