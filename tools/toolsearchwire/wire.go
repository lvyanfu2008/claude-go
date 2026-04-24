// Package toolsearchwire exposes tool-list shaping for non-engine callers (e.g. query streaming)
// without importing goc/internal/toolsearch from packages that should stay narrow.
package toolsearchwire

import (
	"encoding/json"

	"goc/internal/toolsearch"
)

// WireToolsJSON delegates to package toolsearch under goc/internal (same BuildWireConfig + ApplyWire as streaming parity HTTP payloads).
func WireToolsJSON(toolsJSON json.RawMessage, modelID string, hasPendingMcp, openAICompat bool, discoveryMsgsJSON json.RawMessage) (json.RawMessage, error) {
	return toolsearch.WireToolsJSON(toolsJSON, modelID, hasPendingMcp, openAICompat, discoveryMsgsJSON)
}
