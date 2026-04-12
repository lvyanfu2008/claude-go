package toolsearch

import (
	"bytes"
	"encoding/json"

	"goc/ccb-engine/internal/anthropic"
)

// WireToolsJSON returns tools[] after the same BuildWireConfig + ApplyWire pass as
// ccb-engine Session.RunTurn (mirrors claude.ts filteredTools for the HTTP payload).
// discoveryMsgsJSON is the messages array wire shape (e.g. from [ccbhydrate.MessagesJSONNormalized]);
// it is unmarshalled into []anthropic.Message so [ExtractDiscoveredToolNames] can re-include deferred tools.
func WireToolsJSON(toolsJSON json.RawMessage, modelID string, hasPendingMcp, openAICompat bool, discoveryMsgsJSON json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(toolsJSON)) == 0 || string(bytes.TrimSpace(toolsJSON)) == "null" {
		return toolsJSON, nil
	}
	var tools []anthropic.ToolDefinition
	if err := json.Unmarshal(toolsJSON, &tools); err != nil {
		return nil, err
	}
	var msgs []anthropic.Message
	if len(bytes.TrimSpace(discoveryMsgsJSON)) > 0 && string(bytes.TrimSpace(discoveryMsgsJSON)) != "null" {
		_ = json.Unmarshal(discoveryMsgsJSON, &msgs)
	}
	cfg := BuildWireConfig(modelID, tools, hasPendingMcp, openAICompat)
	wired := ApplyWire(tools, msgs, cfg)
	out, err := json.Marshal(wired)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}
