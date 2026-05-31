package layout

import "testing"

func TestVirtualScrollStateComputeRange(t *testing.T) {
	vs := &VirtualScrollState{
		RowHeights: []int{5, 5, 5, 5, 5, 5, 5, 5, 5, 5},
		ScrollTop:  0,
		ViewportH:  20,
		Overscan:   2,
	}
	from, to, offset, total := vs.ComputeRange()
	if from != 0 {
		t.Errorf("from: got %d, want 0", from)
	}
	if offset != 0 {
		t.Errorf("offset: got %d, want 0", offset)
	}
	if total != 50 {
		t.Errorf("total: got %d, want 50", total)
	}
	_ = to
}

func TestVirtualScrollStateStickyBottom(t *testing.T) {
	vs := &VirtualScrollState{
		RowHeights:   []int{5, 5, 5, 5, 5},
		ScrollTop:    5,
		ViewportH:    20,
		Overscan:     2,
		StickyBottom: true,
	}
	vs.UpdateForNewContent(8)
	if vs.ScrollTop <= 5 {
		t.Errorf("expected scrollTop > 5 after stickyBottom update, got %d", vs.ScrollTop)
	}
}

func TestVirtualScrollStateUpdateHeights(t *testing.T) {
	vs := &VirtualScrollState{
		RowHeights: []int{3, 3, 3},
		ViewportH:  10,
		Overscan:   1,
	}
	vs.UpdateHeight(1, 10)
	if vs.RowHeights[1] != 10 {
		t.Errorf("expected height 10, got %d", vs.RowHeights[1])
	}
}

func TestVirtualScrollStateSetRowCount(t *testing.T) {
	vs := &VirtualScrollState{
		RowHeights: []int{1, 1, 1},
		ViewportH:  10,
	}
	vs.SetRowCount(5)
	if len(vs.RowHeights) != 5 {
		t.Errorf("expected 5 rows, got %d", len(vs.RowHeights))
	}
	vs.SetRowCount(2)
	if len(vs.RowHeights) != 2 {
		t.Errorf("expected 2 rows, got %d", len(vs.RowHeights))
	}
}

func TestVirtualScrollStateNew(t *testing.T) {
	vs := NewVirtualScrollState(10, 30)
	if len(vs.RowHeights) != 10 {
		t.Errorf("expected 10 rows, got %d", len(vs.RowHeights))
	}
	if vs.ViewportH != 30 {
		t.Errorf("expected viewportH 30, got %d", vs.ViewportH)
	}
	if vs.Overscan != 5 {
		t.Errorf("expected default overscan 5, got %d", vs.Overscan)
	}
}

func TestVirtualScrollStateComputeRangeEmpty(t *testing.T) {
	vs := &VirtualScrollState{
		RowHeights: []int{},
		ViewportH:  20,
	}
	from, to, offset, total := vs.ComputeRange()
	if from != 0 || to != 0 || offset != 0 || total != 0 {
		t.Errorf("expected all zeros for empty list, got from=%d to=%d offset=%d total=%d",
			from, to, offset, total)
	}
}

func TestVirtualScrollStateComputeRangeScrolled(t *testing.T) {
	vs := &VirtualScrollState{
		RowHeights: []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
		ScrollTop:  25,
		ViewportH:  20,
		Overscan:   2,
	}
	from, to, offset, total := vs.ComputeRange()
	// scrollTop 25 should land in row 2 (offsets[2]=20 <= 25 < offsets[3]=30).
	if total != 100 {
		t.Errorf("total: got %d, want 100", total)
	}
	if from < 0 {
		t.Errorf("from should be >= 0, got %d", from)
	}
	if to <= from {
		t.Errorf("to (%d) should be > from (%d)", to, from)
	}
	if offset < 0 {
		t.Errorf("offset should be >= 0, got %d", offset)
	}
}

func TestVirtualScrollStateComputeRangeBeyondEnd(t *testing.T) {
	vs := &VirtualScrollState{
		RowHeights: []int{5, 5, 5},
		ScrollTop:  100,
		ViewportH:  20,
	}
	from, to, offset, total := vs.ComputeRange()
	if total != 15 {
		t.Errorf("total: got %d, want 15", total)
	}
	_ = from
	_ = to
	_ = offset
}
