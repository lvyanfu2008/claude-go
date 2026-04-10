// Mirrors src/constants/querySource.ts QuerySource (TS stub is any).
package types

import "encoding/json"

// QuerySource is an opaque label or object passed through analytics; shape is not fixed in TS.
type QuerySource json.RawMessage

// MarshalJSON passes through or null when empty.
func (q QuerySource) MarshalJSON() ([]byte, error) {
	if len(q) == 0 {
		return []byte("null"), nil
	}
	return q, nil
}

// UnmarshalJSON stores raw bytes.
func (q *QuerySource) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*q = nil
		return nil
	}
	*q = QuerySource(append(json.RawMessage(nil), data...))
	return nil
}
