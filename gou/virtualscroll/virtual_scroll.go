// Package virtualscroll mirrors src/hooks/useVirtualScroll.ts: height cache,
// prefix offsets, and visible [start, end) range for VirtualMessageList.
package virtualscroll

import "math"

// Constants align with useVirtualScroll.ts (DEFAULT_ESTIMATE, OVERSCAN_ROWS, …).
const (
	DefaultEstimate   = 3
	OverscanRows      = 80
	ColdStartCount    = 30
	ScrollQuantum     = OverscanRows >> 1
	PessimisticHeight = 1
	MaxMountedItems   = 300
	SlideStep         = 25
)

// Range is a half-open index interval [Start, End) into the message list.
type Range struct {
	Start, End int
}

// RangeInput is scroll/viewport state for one ComputeRange call (no React/Ink).
type RangeInput struct {
	ItemKeys []string
	// HeightCache maps ItemKey → measured row height.
	// Missing key: DefaultEstimate when building offsets; PessimisticHeight when extending coverage.
	HeightCache map[string]int

	ScrollTop    int
	PendingDelta int
	ViewportH    int
	IsSticky     bool
	// ListOrigin: scroll coords vs list-local offsets (listOriginRef in TS).
	ListOrigin int

	// PrevRange optional: mounted-but-unmeasured guard + slide cap.
	PrevRange *Range
	// MountedKeys: keys mounted in PrevRange; used with HeightCache for guard.
	MountedKeys map[string]struct{}

	FastScroll bool

	// MaxMountedItemsOverride, when > 0, replaces MaxMountedItems for this ComputeRange call
	// (TS CLAUDE_CODE_DISABLE_VIRTUAL_SCROLL widens the effective window in gou-demo; not full Ink path).
	MaxMountedItemsOverride int
}

// VirtualScrollResult mirrors src/hooks/useVirtualScroll.ts VirtualScrollResult
// (range, topSpacer, bottomSpacer, offsets — no measureRef / DOM hooks).
type VirtualScrollResult struct {
	Range
	TopSpacer    int
	BottomSpacer int
	Offsets      []int // len n+1, Offsets[0]=0, Offsets[i]=sum heights [0,i)
	TotalHeight  int
}

// BuildOffsets returns prefix sums: out[0]=0, out[i+1]=out[i]+height(itemKeys[i]).
// Missing cache entry uses defaultEstimate (TS: DEFAULT_ESTIMATE).
func BuildOffsets(itemKeys []string, cache map[string]int, defaultEstimate int) []int {
	n := len(itemKeys)
	out := make([]int, n+1)
	for i := 0; i < n; i++ {
		k := itemKeys[i]
		h, ok := cache[k]
		if !ok {
			h = defaultEstimate
		}
		out[i+1] = out[i] + h
	}
	return out
}

// HeightForCoverage returns height for range extension (TS: PESSIMISTIC_HEIGHT when unknown).
func HeightForCoverage(cache map[string]int, key string) int {
	if h, ok := cache[key]; ok {
		return h
	}
	return PessimisticHeight
}

// PruneHeightCache removes entries whose keys are not in live.
func PruneHeightCache(cache map[string]int, live []string) {
	if len(cache) == 0 {
		return
	}
	keep := make(map[string]struct{}, len(live))
	for _, k := range live {
		keep[k] = struct{}{}
	}
	for k := range cache {
		if _, ok := keep[k]; !ok {
			delete(cache, k)
		}
	}
}

// ScaleHeightCache scales cached heights when Columns change (TS resize path).
func ScaleHeightCache(cache map[string]int, oldCols, newCols int) {
	if newCols <= 0 || oldCols == newCols || len(cache) == 0 {
		return
	}
	ratio := float64(oldCols) / float64(newCols)
	for k, h := range cache {
		cache[k] = max(1, int(math.Round(float64(h)*ratio)))
	}
}

func maxMountedCap(in RangeInput) int {
	if in.MaxMountedItemsOverride > 0 {
		return in.MaxMountedItemsOverride
	}
	return MaxMountedItems
}

// ComputeRange computes visible item indices and spacers (simplified vs full TS: no deferred range).
func ComputeRange(in RangeInput) VirtualScrollResult {
	capItems := maxMountedCap(in)
	n := len(in.ItemKeys)
	offsets := BuildOffsets(in.ItemKeys, in.HeightCache, DefaultEstimate)
	totalHeight := 0
	if n > 0 {
		totalHeight = offsets[n]
	}

	var start, end int

	switch {
	case in.ViewportH <= 0 || in.ScrollTop < 0:
		// Cold start: tail slice (COLD_START_COUNT).
		start = max(0, n-ColdStartCount)
		end = n

	case in.IsSticky:
		// Include enough messages from the bottom until suffix [start,n) is at least budget rows.
		// Use totalHeight-offsets[start] (not offsets[start-1]) — the old start-1 form was off-by-one
		// and could stop while the rendered range [start,n) was still far below budget (e.g. huge
		// message just above the window), leaving a mostly empty message pane.
		budget := in.ViewportH + OverscanRows
		start = n
		for start > 0 && totalHeight-offsets[start] < budget {
			start--
		}
		end = n

	default:
		listOrigin := in.ListOrigin
		maxSpanRows := in.ViewportH * 3
		rawLo := min(in.ScrollTop, in.ScrollTop+in.PendingDelta)
		rawHi := max(in.ScrollTop, in.ScrollTop+in.PendingDelta)
		span := rawHi - rawLo
		clampedLo := rawLo
		if span > maxSpanRows {
			if in.PendingDelta < 0 {
				clampedLo = rawHi - maxSpanRows
			} else {
				clampedLo = rawLo
			}
		}
		clampedHi := clampedLo + min(span, maxSpanRows)
		effLo := max(0, clampedLo-listOrigin)
		effHi := clampedHi - listOrigin
		lo := effLo - OverscanRows

		// Binary search: smallest start such that offsets[start+1] > lo.
		start = 0
		l, r := 0, n
		for l < r {
			m := (l + r) >> 1
			if offsets[m+1] <= lo {
				l = m + 1
			} else {
				r = m
			}
			start = l
		}

		// Mounted-but-unmeasured guard (simplified: scan prev [start, prevEnd) if start advanced).
		if in.PrevRange != nil && in.MountedKeys != nil && in.HeightCache != nil {
			p0, p1 := in.PrevRange.Start, in.PrevRange.End
			if p0 < start && p0 < p1 {
				for i := p0; i < min(start, p1); i++ {
					k := in.ItemKeys[i]
					if _, mounted := in.MountedKeys[k]; !mounted {
						continue
					}
					if _, measured := in.HeightCache[k]; !measured {
						start = i
						break
					}
				}
			}
		}

		needed := in.ViewportH + 2*OverscanRows
		maxEnd := min(n, start+capItems)
		coverage := 0
		end = start
		for end < maxEnd &&
			(coverage < needed || offsets[end] < effHi+in.ViewportH+OverscanRows) {
			coverage += HeightForCoverage(in.HeightCache, in.ItemKeys[end])
			end++
		}

		// Expand start upward if coverage still short (TS second loop).
		minStart := max(0, end-capItems)
		coverage = 0
		for i := start; i < end; i++ {
			coverage += HeightForCoverage(in.HeightCache, in.ItemKeys[i])
		}
		for start > minStart && coverage < needed {
			start--
			coverage += HeightForCoverage(in.HeightCache, in.ItemKeys[start])
		}

		if in.FastScroll && in.PrevRange != nil {
			pS, pE := in.PrevRange.Start, in.PrevRange.End
			if start < pS-SlideStep {
				start = pS - SlideStep
			}
			if end > pE+SlideStep {
				end = pE + SlideStep
			}
			if start > end {
				end = min(start+SlideStep, n)
			}
		}
	}

	// Final trim if window too wide (TS: MAX_MOUNTED_ITEMS by viewport position).
	if end-start > capItems {
		mid := (offsets[start] + offsets[end]) / 2
		if in.ScrollTop-in.ListOrigin < mid {
			end = start + capItems
		} else {
			start = end - capItems
		}
	}

	start = clamp(start, 0, n)
	end = clamp(end, start, n)

	topSpacer := 0
	if start < len(offsets) {
		topSpacer = offsets[start]
	}
	bottomSpacer := totalHeight
	if end < len(offsets) {
		bottomSpacer = totalHeight - offsets[end]
	}

	return VirtualScrollResult{
		Range:        Range{Start: start, End: end},
		TopSpacer:    topSpacer,
		BottomSpacer: bottomSpacer,
		Offsets:      offsets,
		TotalHeight:  totalHeight,
	}
}

// ScrollOffsetForIndex returns list-local scroll offset for item i (TS: scrollToIndex).
func ScrollOffsetForIndex(offsets []int, i, listOrigin int) int {
	if i < 0 || i >= len(offsets)-1 {
		return -1
	}
	return offsets[i] + listOrigin
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
