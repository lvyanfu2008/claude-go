package localtools

import (
	"fmt"
	"strings"
	"testing"
)

func TestGetPatchFromContents_TrimsContext(t *testing.T) {
	// Simulate a 309-line file where 2 lines are deleted near line 200
	var oldLines []string
	for i := range 309 {
		if i == 199 || i == 200 {
			oldLines = append(oldLines, "deleted line "+fmt.Sprintf("%d", i+1))
		} else {
			oldLines = append(oldLines, "line "+fmt.Sprintf("%d", i+1))
		}
	}
	newLines := make([]string, 0, 307)
	for i := range 309 {
		if i == 199 || i == 200 {
			continue
		}
		newLines = append(newLines, oldLines[i])
	}

	oldContent := strings.Join(oldLines, "\n")
	newContent := strings.Join(newLines, "\n")

	hunks := GetPatchFromContents("test.java", oldContent, newContent)

	if len(hunks) == 0 {
		t.Fatal("expected at least one hunk")
	}

	// Should be a single hunk with limited context
	h := hunks[0]

	// OldStart should be around 197 (line 199 - 3 context lines, 1-based)
	if h.OldStart < 195 || h.OldStart > 198 {
		t.Errorf("OldStart=%d, expected around 197", h.OldStart)
	}

	// Total lines should be small: 3 context + 2 deleted + 3 context = ~8
	if h.OldLines+h.NewLines > 30 {
		t.Errorf("hunk too large: OldLines=%d, NewLines=%d, total lines=%d (want <=30)",
			h.OldLines, h.NewLines, len(h.Lines))
	}

	// Should contain exactly 2 deleted lines
	delCount := 0
	for _, l := range h.Lines {
		if strings.HasPrefix(l, "-") {
			delCount++
		}
	}
	if delCount != 2 {
		t.Errorf("expected 2 deleted lines, got %d", delCount)
	}
}

func TestGetPatchFromContents_MultipleHunks(t *testing.T) {
	// Two changes far apart in a 100-line file
	lines := make([]string, 100)
	for i := range 100 {
		lines[i] = "line " + fmt.Sprintf("%d", i+1)
	}

	oldContent := strings.Join(lines, "\n")
	// Modify line 10 and line 90
	newLines := make([]string, 100)
	copy(newLines, lines)
	newLines[9] = "modified line 10"
	newLines[89] = "modified line 90"
	newContent := strings.Join(newLines, "\n")

	hunks := GetPatchFromContents("test.java", oldContent, newContent)

	if len(hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(hunks))
	}

	// First hunk should be around line 10
	if hunks[0].OldStart < 7 || hunks[0].OldStart > 11 {
		t.Errorf("hunk 0 OldStart=%d, expected around 8-10", hunks[0].OldStart)
	}
	// Second hunk should be around line 90
	if hunks[1].OldStart < 85 || hunks[1].OldStart > 91 {
		t.Errorf("hunk 1 OldStart=%d, expected around 87-90", hunks[1].OldStart)
	}
}

