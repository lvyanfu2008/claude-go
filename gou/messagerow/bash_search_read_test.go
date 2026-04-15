package messagerow

import "testing"

func TestIsSearchOrReadBashCommand(t *testing.T) {
	t.Parallel()
	cases := []struct {
		cmd            string
		wantS, wantR, wantL bool
	}{
		{"ls", false, false, true},
		{"grep foo", true, false, false},
		{"cat x", false, true, false},
		{"cat x | grep y", true, true, false},
		{"ls && grep foo", true, false, true},
		{"echo hi", false, false, false},
		{"git status", false, false, false},
		{"npm install", false, false, false},
		// Redirects: first simple command word still classifies (mirrors TS splitCommandWithOperators + loop).
		{"grep foo > /tmp/out", true, false, false},
		{"cat x > /tmp/out", false, true, false},
	}
	for _, tc := range cases {
		gotS, gotR, gotL := IsSearchOrReadBashCommand(tc.cmd)
		if gotS != tc.wantS || gotR != tc.wantR || gotL != tc.wantL {
			t.Errorf("%q: got search=%v read=%v list=%v want search=%v read=%v list=%v",
				tc.cmd, gotS, gotR, gotL, tc.wantS, tc.wantR, tc.wantL)
		}
	}
}
