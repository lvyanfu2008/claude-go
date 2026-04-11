package query

import (
	"encoding/json"

	"goc/types"
)

// QueryYield is one item yielded from Query (TS AsyncGenerator union element).
// At most one non-nil pointer field should be set.
type QueryYield struct {
	StreamEvent    json.RawMessage
	RequestStart   json.RawMessage
	Message        *types.Message
	Tombstone      json.RawMessage
	ToolUseSummary json.RawMessage
	Terminal       *Terminal
}
