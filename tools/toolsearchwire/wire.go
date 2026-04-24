// Package toolsearchwire is a thin facade over goc/internal/toolsearch for callers that want a stable tools/* import (e.g. query streaming).
package toolsearchwire

import (
	"encoding/json"

	"goc/internal/toolsearch"
)

// WireToolsJSON delegates to package toolsearch under goc/internal (same BuildWireConfig + ApplyWire as streaming parity HTTP payloads).
func WireToolsJSON(toolsJSON json.RawMessage, modelID string, hasPendingMcp, openAICompat bool, discoveryMsgsJSON json.RawMessage) (json.RawMessage, error) {
	return toolsearch.WireToolsJSON(toolsJSON, modelID, hasPendingMcp, openAICompat, discoveryMsgsJSON)
}
