package state

import "time"

type MessageTracking struct {
	FirstShownAt            map[string]time.Time
	LastAssistantContentLen map[string]int
	RebuildHeightCacheCalls int
}
