// Package toolsearchwire exposes tool-list shaping for non-engine callers (e.g. query streaming)
// without importing ccb-engine/internal/... from outside the ccb-engine module subtree.
package toolsearchwire

import (
	"encoding/json"

	"goc/ccb-engine/internal/toolsearch"
)

// WireToolsJSON delegates to internal/toolsearch (same BuildWireConfig + ApplyWire as CCB Session.RunTurn).
func WireToolsJSON(toolsJSON json.RawMessage, modelID string, hasPendingMcp, openAICompat bool, discoveryMsgsJSON json.RawMessage) (json.RawMessage, error) {
	return toolsearch.WireToolsJSON(toolsJSON, modelID, hasPendingMcp, openAICompat, discoveryMsgsJSON)
}
