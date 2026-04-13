package main

import (
	"testing"
	"time"
)

func TestEffectiveSpinnerTip(t *testing.T) {
	cases := []struct {
		elapsed time.Duration
		enabled bool
		want    string
	}{
		{0, false, ""},
		{0, true, spinnerTipPromptQueue},
		{29 * time.Second, true, spinnerTipPromptQueue},
		{30 * time.Second, true, spinnerTipBtw},
		{30*time.Minute - time.Nanosecond, true, spinnerTipBtw},
		{30 * time.Minute, true, spinnerTipClear},
	}
	for _, tc := range cases {
		got := effectiveSpinnerTip(tc.elapsed, tc.enabled)
		if got != tc.want {
			t.Fatalf("elapsed=%v enabled=%v: got %q want %q", tc.elapsed, tc.enabled, got, tc.want)
		}
	}
}
