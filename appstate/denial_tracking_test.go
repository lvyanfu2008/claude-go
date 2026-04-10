package appstate

import "testing"

func TestDenialTracking_recordAndFallback(t *testing.T) {
	s := CreateDenialTrackingState()
	s = RecordDenial(s)
	s = RecordDenial(s)
	s = RecordDenial(s)
	if !ShouldFallbackToPrompting(s) {
		t.Fatal("expected fallback after 3 consecutive")
	}
	s = RecordSuccess(CreateDenialTrackingState())
	s = RecordDenial(s)
	if ShouldFallbackToPrompting(s) {
		t.Fatal("unexpected fallback")
	}
}
