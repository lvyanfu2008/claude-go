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

// PreservedSegmentMeta mirrors the preservedSegment annotation that
// annotateBoundaryWithPreservedSegment stamps onto compactMetadata in TS:
//
//	preservedSegment: {
//	  headUuid: keep[0].uuid,
//	  anchorUuid,
//	  tailUuid: keep.at(-1).uuid,
//	}
//
// Used by partial compact and SM compaction to record which tail messages survived.
type PreservedSegmentMeta struct {
	HeadUuid   string `json:"headUuid"`
	AnchorUuid string `json:"anchorUuid"`
	TailUuid   string `json:"tailUuid"`
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
	// PreCompactDiscoveredTools mirrors TS preCompactDiscoveredTools — stamped by
	// session-memory compaction to carry discovered tool names across the boundary.
	PreCompactDiscoveredTools []string `json:"preCompactDiscoveredTools,omitempty"`
}

// NewUUID generates a RFC-4122 v4 UUID. Exported for use by packages that need
// a standalone UUID generator without wiring Deps (e.g. sessionmemory).
func NewUUID() string {
	return newUUID()
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
//
// TS signature:
//
//	export function annotateBoundaryWithPreservedSegment(
//	  boundary: SystemCompactBoundaryMessage,
//	  anchorUuid: UUID,
//	  messagesToKeep: readonly Message[] | undefined,
//	): SystemCompactBoundaryMessage
//
// Returns boundary unchanged if messagesToKeep is empty/nil.
func AnnotateBoundaryWithPreservedSegment(boundary types.Message, anchorUuid string, messagesToKeep []types.Message) types.Message {
	if len(messagesToKeep) == 0 {
		return boundary
	}
	var meta CompactMetadata
	if len(boundary.CompactMetadata) > 0 {
		_ = json.Unmarshal(boundary.CompactMetadata, &meta)
	}
	meta.PreservedSegment = &PreservedSegmentMeta{
		HeadUuid:   messagesToKeep[0].UUID,
		AnchorUuid: anchorUuid,
		TailUuid:   messagesToKeep[len(messagesToKeep)-1].UUID,
	}
	if raw, err := json.Marshal(meta); err == nil {
		boundary.CompactMetadata = json.RawMessage(raw)
	}
	return boundary
}
