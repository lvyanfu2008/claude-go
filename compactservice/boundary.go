package compactservice

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"goc/types"
)

// CompactTrigger mirrors the union `'manual' | 'auto'` on CompactMetadata.trigger in TS.
type CompactTrigger string

const (
	CompactTriggerManual CompactTrigger = "manual"
	CompactTriggerAuto   CompactTrigger = "auto"
)

// PreservedSegmentMeta mirrors the { preservedKept, preservedTail } annotation that
// annotateBoundaryWithPreservedSegment stamps onto compactMetadata in TS.
// Used by partial compact to record how many tail messages survived compaction.
type PreservedSegmentMeta struct {
	// Kept is the number of most-recent messages retained verbatim in the recent tail.
	Kept int `json:"kept"`
	// Total is the number of messages summarized (pre-compact conversation size).
	Total int `json:"total"`
}

// CompactMetadata mirrors CompactMetadata on SystemCompactBoundaryMessage in TS.
// Field order follows TS; unknown future fields should be added rather than embedded
// as json.RawMessage to preserve parity diagnostics.
type CompactMetadata struct {
	Trigger           CompactTrigger        `json:"trigger"`
	PreTokens         int                   `json:"preTokens"`
	UserContext       string                `json:"userContext,omitempty"`
	MessagesSummarized *int                 `json:"messagesSummarized,omitempty"`
	// PreservedSegment is written by AnnotateBoundaryWithPreservedSegment for partial-compact paths.
	PreservedSegment  *PreservedSegmentMeta `json:"preservedSegment,omitempty"`
}

// newUUID generates a RFC-4122 v4 UUID. Hosts override via Deps.NewUUID to match the
// parent query.NewUUID path (deterministic tests / custom UUID providers).
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%s",
		uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
		uint16(b[4])<<8|uint16(b[5]),
		uint16(b[6])<<8|uint16(b[7]),
		uint16(b[8])<<8|uint16(b[9]),
		hex.EncodeToString(b[10:16]),
	)
}

// nowRFC3339 matches toIsoString() semantics (UTC, nano precision).
func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// CreateCompactBoundaryMessage mirrors createCompactBoundaryMessage in utils/messages.ts:
//
//	export function createCompactBoundaryMessage(
//	  trigger: 'manual' | 'auto',
//	  preTokens: number,
//	  lastPreCompactMessageUuid?: UUID,
//	  userContext?: string,
//	  messagesSummarized?: number,
//	): SystemCompactBoundaryMessage
//
// The returned message has type:"system", subtype:"compact_boundary", content:"Conversation compacted",
// level:"info", isMeta:false, UUID generated, logicalParentUuid set iff lastPreCompactMessageUuid provided.
func CreateCompactBoundaryMessage(
	trigger CompactTrigger,
	preTokens int,
	lastPreCompactMessageUUID string,
	userContext string,
	messagesSummarized *int,
) (types.Message, error) {
	meta := CompactMetadata{
		Trigger:            trigger,
		PreTokens:          preTokens,
		UserContext:        userContext,
		MessagesSummarized: messagesSummarized,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return types.Message{}, fmt.Errorf("compactservice: marshal metadata: %w", err)
	}

	content, err := json.Marshal("Conversation compacted")
	if err != nil {
		return types.Message{}, fmt.Errorf("compactservice: marshal content: %w", err)
	}

	ts := nowRFC3339()
	subtype := "compact_boundary"
	level := "info"
	isMeta := false

	m := types.Message{
		Type:            types.MessageTypeSystem,
		UUID:            newUUID(),
		Subtype:         &subtype,
		Level:           &level,
		Timestamp:       &ts,
		IsMeta:          &isMeta,
		Content:         json.RawMessage(content),
		CompactMetadata: json.RawMessage(metaJSON),
	}
	if lastPreCompactMessageUUID != "" {
		u := lastPreCompactMessageUUID
		m.LogicalParentUUID = &u
	}
	return m, nil
}

// IsCompactBoundaryMessage mirrors isCompactBoundaryMessage in TS.
func IsCompactBoundaryMessage(m types.Message) bool {
	return m.Type == types.MessageTypeSystem && m.Subtype != nil && *m.Subtype == "compact_boundary"
}

// FindLastCompactBoundaryIndex mirrors findLastCompactBoundaryIndex / findLastCompactBoundaryMessageIndex in TS.
// Returns -1 when no boundary is present.
func FindLastCompactBoundaryIndex(messages []types.Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if IsCompactBoundaryMessage(messages[i]) {
			return i
		}
	}
	return -1
}

// AnnotateBoundaryWithPreservedSegment mirrors annotateBoundaryWithPreservedSegment in TS.
// Finds the most recent compact boundary and stamps PreservedSegmentMeta onto its CompactMetadata.
// Returns a new slice with the updated boundary; messages slice is not mutated.
func AnnotateBoundaryWithPreservedSegment(messages []types.Message, kept, total int) []types.Message {
	idx := FindLastCompactBoundaryIndex(messages)
	if idx == -1 {
		return messages
	}
	updated := make([]types.Message, len(messages))
	copy(updated, messages)
	b := updated[idx]
	var meta CompactMetadata
	if len(b.CompactMetadata) > 0 {
		_ = json.Unmarshal(b.CompactMetadata, &meta)
	}
	meta.PreservedSegment = &PreservedSegmentMeta{Kept: kept, Total: total}
	if raw, err := json.Marshal(meta); err == nil {
		b.CompactMetadata = raw
	}
	updated[idx] = b
	return updated
}
