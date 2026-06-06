package state

type Scroll struct {
	Top          int
	PendingDelta int
	Sticky       bool
	HeightCache  map[string]int
}

func NewScroll() *Scroll {
	return &Scroll{
		Sticky:      true,
		HeightCache: make(map[string]int),
	}
}
