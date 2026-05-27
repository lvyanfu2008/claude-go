package sessiontranscript

import (
	"encoding/json"

	"goc/types"
)

// ApplySnipRemovals mirrors applySnipRemovals in sessionStorage.ts. It removes
// messages whose UUIDs appear in any snip_boundary's removedUuids, keeping the
// boundary messages themselves.
func ApplySnipRemovals(messages []types.Message) []types.Message {
	removedSet := make(map[string]bool)
	for _, m := range messages {
		if m.Type != types.MessageTypeSystem || m.Subtype == nil || *m.Subtype != "snip_boundary" {
			continue
		}
		for _, uuid := range parseSnipRemovedUuids(m.CompactMetadata) {
			removedSet[uuid] = true
		}
	}
	if len(removedSet) == 0 {
		return messages
	}
	filtered := make([]types.Message, 0, len(messages))
	for _, m := range messages {
		if removedSet[m.UUID] {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

type snipBoundaryMeta struct {
	RemovedUuids []string `json:"removedUuids"`
}

func parseSnipRemovedUuids(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var meta snipBoundaryMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil
	}
	return meta.RemovedUuids
}
