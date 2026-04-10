package appstate

// DenialTrackingState mirrors src/utils/permissions/denialTracking.ts DenialTrackingState.
type DenialTrackingState struct {
	ConsecutiveDenials int `json:"consecutiveDenials"`
	TotalDenials       int `json:"totalDenials"`
}

// DenialLimits mirrors TS DENIAL_LIMITS.
var DenialLimits = struct {
	MaxConsecutive int
	MaxTotal       int
}{
	MaxConsecutive: 3,
	MaxTotal:       20,
}

// CreateDenialTrackingState mirrors createDenialTrackingState().
func CreateDenialTrackingState() DenialTrackingState {
	return DenialTrackingState{}
}

// RecordDenial mirrors recordDenial().
func RecordDenial(state DenialTrackingState) DenialTrackingState {
	return DenialTrackingState{
		ConsecutiveDenials: state.ConsecutiveDenials + 1,
		TotalDenials:       state.TotalDenials + 1,
	}
}

// RecordSuccess mirrors recordSuccess().
func RecordSuccess(state DenialTrackingState) DenialTrackingState {
	if state.ConsecutiveDenials == 0 {
		return state
	}
	return DenialTrackingState{
		ConsecutiveDenials: 0,
		TotalDenials:       state.TotalDenials,
	}
}

// ShouldFallbackToPrompting mirrors shouldFallbackToPrompting().
func ShouldFallbackToPrompting(state DenialTrackingState) bool {
	return state.ConsecutiveDenials >= DenialLimits.MaxConsecutive ||
		state.TotalDenials >= DenialLimits.MaxTotal
}
