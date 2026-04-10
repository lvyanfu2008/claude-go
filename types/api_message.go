// APIMessage is one Anthropic API message { role, content } for hydrate / transcript import.
// content is raw JSON (string or array of blocks).
package types

import "encoding/json"

// APIMessage mirrors ccb-engine anthropic.Message JSON for files that store API-shaped history.
type APIMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}
