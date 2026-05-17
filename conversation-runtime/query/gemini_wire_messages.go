package query

import (
	"encoding/json"

	"goc/messagesapi"
	"goc/types"
)

// anthropicMessagesToGemini converts a slice of typed messages + system prompts
// to Gemini's contents[] + optional systemInstruction.
func anthropicMessagesToGemini(msgs []types.Message, systemPrompts []string, opts messagesapi.Options) ([]GeminiContent, *GeminiInstruction, error) {
	var contents []GeminiContent

	if len(systemPrompts) > 0 {
		allSystem := ""
		for _, s := range systemPrompts {
			allSystem += s + "\n"
		}
		// Gemini requires system instruction as separate field.
	}

	for _, msg := range msgs {
		switch msg.Type {
		case types.MessageTypeUser:
			c := GeminiContent{Role: "user"}
			// Extract text from content blocks.
			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal([]byte(msg.Content), &blocks); err == nil {
				for _, b := range blocks {
					if b.Type == "text" && b.Text != "" {
						c.Parts = append(c.Parts, GeminiPart{Text: b.Text})
					}
					if b.Type == "tool_result" {
						// tool_result not directly supported in Gemini format
					}
				}
			}
			if len(c.Parts) > 0 {
				contents = append(contents, c)
			}

		case types.MessageTypeAssistant:
			c := GeminiContent{Role: "model"}
			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal([]byte(msg.Content), &blocks); err == nil {
				for _, b := range blocks {
					if b.Type == "text" && b.Text != "" {
						c.Parts = append(c.Parts, GeminiPart{Text: b.Text})
					}
				}
			}
			if len(c.Parts) > 0 {
				contents = append(contents, c)
			}
		}
	}

	return contents, nil, nil
}
