package layout

// VirtualScrollState manages a virtualised list of rows: it tracks
// row heights, the current scroll position, viewport size, and overscan,
// and provides the visible range via ComputeRange.
//
// Mirror of the VirtualScrollState helper used in src/hooks/useVirtualScroll.ts
// (simplified: no HeightCache / ItemKeys — just a flat []int).
type VirtualScrollState struct {
	RowHeights   []int
	Offsets      []int
	ScrollTop    int
	ViewportH    int
	Overscan     int
	StickyBottom bool
}

// NewVirtualScrollState creates a state with rowCount initial rows
// each sized 1, and a default overscan of 5.
func NewVirtualScrollState(rowCount, viewportH int) *VirtualScrollState {
	heights := make([]int, rowCount)
	for i := range heights {
		heights[i] = 1
	}
	return &VirtualScrollState{
		RowHeights: heights,
		Offsets:    make([]int, rowCount+1),
		ViewportH:  viewportH,
		Overscan:   5,
	}
}

// ComputeRange returns the visible index interval, the scroll offset at
// 'from', and the total content height.  It caches prefix sums in
// vs.Offsets so that offsetsUpTo can re-use them.
func (vs *VirtualScrollState) ComputeRange() (from, to, offsetTop int, totalH int) {
	n := len(vs.RowHeights)
	if n == 0 {
		return 0, 0, 0, 0
	}

	// Build prefix offsets.
	vs.Offsets = make([]int, n+1)
	for i, h := range vs.RowHeights {
		vs.Offsets[i+1] = vs.Offsets[i] + h
	}
	totalH = vs.Offsets[n]

	scrollTop := vs.ScrollTop
	if scrollTop < 0 {
		scrollTop = 0
	}
	if scrollTop > totalH {
		scrollTop = totalH
	}

	// Binary-search for the first row whose end > (scrollTop - overscan).
	lo := scrollTop - vs.Overscan
	if lo < 0 {
		lo = 0
	}
	from = 0
	if lo > 0 {
		l, r := 0, n
		for l < r {
			m := (l + r) >> 1
			if vs.Offsets[m+1] <= lo {
				l = m + 1
			} else {
				r = m
			}
		}
		from = l
	}

	// Extend the visible range until we have enough content for the
	// viewport + 2x overscan.
	needed := vs.ViewportH + 2*vs.Overscan
	if needed <= 0 {
		needed = vs.ViewportH
	}
	coverage := 0
	to = from
	for to < n && coverage < needed {
		coverage += vs.RowHeights[to]
		to++
	}

	offsetTop = vs.Offsets[from]
	return
}

// UpdateForNewContent ensures at least rowCount rows exist and, when
// StickyBottom is true, pins the scroll position to the bottom.
func (vs *VirtualScrollState) UpdateForNewContent(rowCount int) {
	for len(vs.RowHeights) < rowCount {
		vs.RowHeights = append(vs.RowHeights, 1)
	}
	if vs.StickyBottom {
		total := 0
		for _, h := range vs.RowHeights {
			total += h
		}
		vs.ScrollTop = total - vs.ViewportH
		if vs.ScrollTop < 0 {
			vs.ScrollTop = 0
		}
	}
}

// UpdateHeight sets the cached height for a single row.
func (vs *VirtualScrollState) UpdateHeight(index, h int) {
	if index >= 0 && index < len(vs.RowHeights) {
		vs.RowHeights[index] = h
	}
}

// SetRowCount resizes the row slice.  Growing appends rows of height 1;
// shrinking truncates.
func (vs *VirtualScrollState) SetRowCount(n int) {
	if n == len(vs.RowHeights) {
		return
	}
	if n > len(vs.RowHeights) {
		for len(vs.RowHeights) < n {
			vs.RowHeights = append(vs.RowHeights, 1)
		}
	} else {
		vs.RowHeights = vs.RowHeights[:n]
	}
}

// offsetsUpTo returns the cumulative height of rows [0, n).
// Called by layoutVirtualList after ComputeRange has populated vs.Offsets.
func (vs *VirtualScrollState) offsetsUpTo(n int) int {
	if n < 0 {
		return 0
	}
	if n >= len(vs.Offsets) {
		return vs.Offsets[len(vs.Offsets)-1]
	}
	return vs.Offsets[n]
}
