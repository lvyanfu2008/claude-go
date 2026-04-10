package virtualscroll

import "testing"

func TestBuildOffsets_missingUsesDefault(t *testing.T) {
	keys := []string{"a", "b", "c"}
	cache := map[string]int{"a": 10}
	off := BuildOffsets(keys, cache, DefaultEstimate)
	if off[0] != 0 || off[1] != 10 || off[2] != 10+DefaultEstimate || off[3] != off[2]+DefaultEstimate {
		t.Fatalf("offsets = %v", off)
	}
}

func TestComputeRange_coldStart(t *testing.T) {
	n := 50
	keys := make([]string, n)
	for i := range keys {
		keys[i] = string(rune('a' + (i % 26)))
	}
	out := ComputeRange(RangeInput{
		ItemKeys:    keys,
		HeightCache: nil,
		ScrollTop:   -1,
		ViewportH:   0,
		IsSticky:    false,
		ListOrigin:  0,
	})
	if out.End-out.Start != ColdStartCount && n >= ColdStartCount {
		t.Fatalf("expected %d items, got [%d,%d)", ColdStartCount, out.Start, out.End)
	}
	if out.Start != n-ColdStartCount {
		t.Fatalf("start=%d end=%d", out.Start, out.End)
	}
}

func TestComputeRange_stickyTail(t *testing.T) {
	keys := []string{"k0", "k1", "k2", "k3"}
	cache := map[string]int{"k0": 5, "k1": 5, "k2": 5, "k3": 5}
	out := ComputeRange(RangeInput{
		ItemKeys:    keys,
		HeightCache: cache,
		ScrollTop:   100,
		ViewportH:   10,
		IsSticky:    true,
		ListOrigin:  0,
	})
	if out.End != 4 {
		t.Fatalf("end=%d want 4", out.End)
	}
	if out.Start < 0 || out.Start > 3 {
		t.Fatalf("start=%d", out.Start)
	}
	// Suffix height from start must be >= viewport+overscan unless start==0
	suffix := out.TotalHeight - out.Offsets[out.Start]
	if suffix < 10+OverscanRows && out.Start != 0 {
		t.Fatalf("suffix=%d too small", suffix)
	}
}

func TestComputeRange_stickyTail_suffixCoversBudgetWithHugeLeadingMessage(t *testing.T) {
	keys := []string{"k0", "k1", "k2", "k3", "k4"}
	cache := map[string]int{"k0": 100, "k1": 1, "k2": 1, "k3": 1, "k4": 1}
	vp := 10
	budget := vp + OverscanRows
	out := ComputeRange(RangeInput{
		ItemKeys:    keys,
		HeightCache: cache,
		ScrollTop:   0,
		ViewportH:   vp,
		IsSticky:    true,
		ListOrigin:  0,
	})
	suffix := out.TotalHeight - out.Offsets[out.Start]
	if suffix < budget && out.Start != 0 {
		t.Fatalf("sticky suffix=%d want >= %d (start=%d total=%d offsets=%v)",
			suffix, budget, out.Start, out.TotalHeight, out.Offsets)
	}
}

func TestComputeRange_nonStickyBinarySearch(t *testing.T) {
	keys := []string{"a", "b", "c", "d"}
	cache := map[string]int{"a": 10, "b": 10, "c": 10, "d": 10}
	off := BuildOffsets(keys, cache, DefaultEstimate)
	// scroll so that lo lands in third block
	scrollTop := off[2] + 3
	out := ComputeRange(RangeInput{
		ItemKeys:     keys,
		HeightCache:  cache,
		ScrollTop:    scrollTop,
		PendingDelta: 0,
		ViewportH:    5,
		IsSticky:     false,
		ListOrigin:   0,
	})
	if out.Start > 2 {
		t.Fatalf("start=%d expected <=2 for scrollTop=%d offsets=%v", out.Start, scrollTop, off)
	}
	if out.End <= out.Start {
		t.Fatalf("empty range [%d,%d)", out.Start, out.End)
	}
	if out.TopSpacer != off[out.Start] {
		t.Fatalf("topSpacer=%d want %d", out.TopSpacer, off[out.Start])
	}
	wantBottom := off[len(keys)] - off[out.End]
	if out.BottomSpacer != wantBottom {
		t.Fatalf("bottomSpacer=%d want %d", out.BottomSpacer, wantBottom)
	}
}

func TestPruneHeightCache(t *testing.T) {
	c := map[string]int{"a": 1, "b": 2, "c": 3}
	PruneHeightCache(c, []string{"a", "c"})
	if len(c) != 2 || c["b"] != 0 {
		t.Fatalf("after prune: %v", c)
	}
}

func TestScaleHeightCache(t *testing.T) {
	c := map[string]int{"x": 10, "y": 20}
	ScaleHeightCache(c, 80, 40)
	if c["x"] != 20 || c["y"] != 40 {
		t.Fatalf("got %v", c)
	}
}

func TestScrollOffsetForIndex(t *testing.T) {
	off := []int{0, 5, 12}
	if ScrollOffsetForIndex(off, 1, 100) != 105 {
		t.Fatal()
	}
	if ScrollOffsetForIndex(off, 99, 0) != -1 {
		t.Fatal()
	}
}
