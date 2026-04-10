// Package transcript loads conversation rows from JSON files (UI-shaped or API-shaped).
package transcript

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"goc/types"
)

// LoadFile detects format and returns types.Message rows for gou Store.
//
// Supported:
//   - JSON array of UI messages (types.Message with type, uuid, content/message, …)
//   - JSON array of API messages [{ "role":"user"|"assistant", "content": ... }]
func LoadFile(path string) ([]types.Message, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DecodeJSON(data)
}

// DecodeJSON same as LoadFile but from bytes.
func DecodeJSON(data []byte) ([]types.Message, error) {
	data = trimBOM(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("transcript: empty file")
	}
	// Probe first element.
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("transcript: expected JSON array: %w", err)
	}
	if len(arr) == 0 {
		return nil, fmt.Errorf("transcript: empty array")
	}
	var head map[string]any
	if err := json.Unmarshal(arr[0], &head); err != nil {
		return nil, err
	}
	if _, ok := head["type"]; ok {
		var out []types.Message
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, fmt.Errorf("transcript: UI messages: %w", err)
		}
		return out, nil
	}
	if _, ok := head["role"]; ok {
		return fromAPIMessagesArray(data)
	}
	return nil, fmt.Errorf("transcript: first object has neither \"type\" nor \"role\"")
}

func fromAPIMessagesArray(data []byte) ([]types.Message, error) {
	var api []types.APIMessage
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, fmt.Errorf("transcript: API messages: %w", err)
	}
	out := make([]types.Message, 0, len(api))
	for i, m := range api {
		mt := types.MessageTypeUser
		switch m.Role {
		case "assistant":
			mt = types.MessageTypeAssistant
		case "user":
			mt = types.MessageTypeUser
		default:
			mt = types.MessageTypeSystem
		}
		inner, err := json.Marshal(struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}{Role: m.Role, Content: m.Content})
		if err != nil {
			return nil, err
		}
		out = append(out, types.Message{
			Type:    mt,
			UUID:    fmt.Sprintf("api-%06d", i),
			Message: inner,
		})
	}
	return out, nil
}

func trimBOM(b []byte) []byte {
	return []byte(strings.TrimPrefix(string(b), "\ufeff"))
}
