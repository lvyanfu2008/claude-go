package message

import (
	"strings"
	"testing"
)

func TestListMouseWheelStep(t *testing.T) {
	tests := []struct {
		name string
		vpH  int
		want int
	}{
		{"zero viewport", 0, 1},
		{"negative viewport", -1, 1},
		{"tiny viewport", 5, 1},
		{"small viewport", 12, 1},
		{"medium viewport", 24, 2},
		{"large viewport", 60, 5},
		{"huge viewport", 120, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ListMouseWheelStep(tt.vpH); got != tt.want {
				t.Errorf("ListMouseWheelStep(%d) = %d, want %d", tt.vpH, got, tt.want)
			}
		})
	}
}

func TestScrollbarThumb(t *testing.T) {
	tests := []struct {
		name      string
		vpH, tH, sT int
		wantStart, wantLen int
	}{
		{
			name:      "fits in viewport",
			vpH:       20, tH: 10, sT: 0,
			wantStart: 0, wantLen: 20,
		},
		{
			name:      "zero viewport",
			vpH:       0, tH: 100, sT: 0,
			wantStart: 0, wantLen: 0,
		},
		{
			name:      "at top",
			vpH:       20, tH: 100, sT: 0,
			wantStart: 0, wantLen: 4,
		},
		{
			name:      "at bottom",
			vpH:       20, tH: 100, sT: 80,
			wantStart: 16, wantLen: 4,
		},
		{
			name:      "in middle",
			vpH:       20, tH: 100, sT: 40,
			wantStart: 8, wantLen: 4,
		},
		{
			name:      "negative scroll clamps to zero",
			vpH:       20, tH: 100, sT: -5,
			wantStart: 0, wantLen: 4,
		},
		{
			name:      "overflow scroll clamps to max",
			vpH:       20, tH: 100, sT: 200,
			wantStart: 16, wantLen: 4,
		},
		{
			name:      "large content small thumb",
			vpH:       24, tH: 10000, sT: 5000,
			wantStart: 11, wantLen: 1,
		},
		{
			name:      "thumb respects viewport bounds",
			vpH:       10, tH: 50, sT: 0,
			wantStart: 0, wantLen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotLen := ScrollbarThumb(tt.vpH, tt.tH, tt.sT)
			if gotStart != tt.wantStart || gotLen != tt.wantLen {
				t.Errorf("ScrollbarThumb(%d,%d,%d) = (%d,%d), want (%d,%d)",
					tt.vpH, tt.tH, tt.sT, gotStart, gotLen, tt.wantStart, tt.wantLen)
			}
			// Invariant: thumb must fit in viewport
			if gotStart < 0 || gotStart+gotLen > tt.vpH {
				t.Errorf("thumb [%d,%d) out of viewport bounds [0,%d)", gotStart, gotStart+gotLen, tt.vpH)
			}
		})
	}
}

func TestScrollbarThumbProportional(t *testing.T) {
	// As scrollTop increases, thumb start should increase monotonically
	vpH, tH := 20, 100
	var lastStart int
	for sT := 0; sT <= 80; sT += 10 {
		start, _ := ScrollbarThumb(vpH, tH, sT)
		if start < lastStart {
			t.Errorf("ScrollbarThumb not monotonic: at scrollTop=%d, start=%d < previous=%d", sT, start, lastStart)
		}
		lastStart = start
	}
}

func TestJoinLinesWithScrollbar(t *testing.T) {
	t.Run("no scrollbar when barW=0", func(t *testing.T) {
		lines := []string{"line1", "line2"}
		got := JoinLinesWithScrollbar(lines, 40, 3, 10, 0, 0)
		if got != "line1\nline2" {
			t.Errorf("expected joined lines, got %q", got)
		}
	})

	t.Run("no scrollbar when vpH=0", func(t *testing.T) {
		lines := []string{"line1"}
		got := JoinLinesWithScrollbar(lines, 40, 0, 10, 0, 1)
		if got != "line1" {
			t.Errorf("expected single line, got %q", got)
		}
	})

	t.Run("adds scrollbar column", func(t *testing.T) {
		lines := []string{"hello", "world"}
		got := JoinLinesWithScrollbar(lines, 10, 3, 10, 0, 1)
		split := strings.Split(got, "\n")
		if len(split) != 3 {
			t.Fatalf("expected 3 lines (2 content + 1 empty), got %d", len(split))
		}
		// Each line should end with │ or ┃ (scrollbar char)
		for _, s := range split {
			if !strings.Contains(s, "│") && !strings.Contains(s, "┃") {
				t.Errorf("line %q missing scrollbar character", s)
			}
		}
	})

	t.Run("pads short lines", func(t *testing.T) {
		lines := []string{"hi"}
		got := JoinLinesWithScrollbar(lines, 10, 1, 5, 0, 1)
		// Should contain padding spaces before scrollbar
		if !strings.Contains(got, " ") {
			t.Error("expected padding spaces in output")
		}
	})

	t.Run("fills missing rows", func(t *testing.T) {
		lines := []string{"only one line"}
		got := JoinLinesWithScrollbar(lines, 20, 5, 50, 25, 1)
		split := strings.Split(got, "\n")
		if len(split) != 5 {
			t.Errorf("expected 5 rows, got %d", len(split))
		}
	})

	t.Run("thumb appears when content overflows", func(t *testing.T) {
		lines := []string{"a", "b", "c"}
		got := JoinLinesWithScrollbar(lines, 10, 3, 30, 15, 1)
		if !strings.Contains(got, "┃") {
			t.Error("expected thumb character ┃ for overflowing content")
		}
	})

	t.Run("full track when at top of overflowing content", func(t *testing.T) {
		lines := []string{"a"}
		got := JoinLinesWithScrollbar(lines, 10, 2, 30, 0, 1)
		if !strings.Contains(got, "┃") {
			t.Error("expected thumb even at top of overflowing content")
		}
	})
}
