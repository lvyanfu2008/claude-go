package processuserinput

import (
	"encoding/json"
	"strings"

	"goc/types"
)

// ProcessTextPrompt mirrors src/conversation-runtime/processUserInput/processTextPrompt.ts processTextPrompt.
// When logEvent is non-nil, emits tengu_input_prompt with is_negative / is_keep_going (TS: logEvent + userPromptKeywords).
// OTEL / setPromptId / startInteractionSpan are omitted in Go.
func ProcessTextPrompt(
	inputStr string,
	inputBlocks []types.ContentBlockParam,
	imageContentBlocks []types.ContentBlockParam,
	imagePasteIDs []int,
	attachmentMessages []types.Message,
	uuid *string,
	permissionMode *types.PermissionMode,
	isMeta *bool,
	logEvent func(name string, payload map[string]any),
) (ProcessTextPromptResult, error) {
	userPromptText := userPromptTextForTenguInput(inputStr, inputBlocks)
	if logEvent != nil {
		logEvent("tengu_input_prompt", map[string]any{
			"is_negative":    MatchesNegativeKeyword(userPromptText),
			"is_keep_going": MatchesKeepGoingKeyword(userPromptText),
		})
	}

	if len(imageContentBlocks) > 0 {
		textParts := buildTextBlocks(inputStr, inputBlocks)
		content := append(textParts, imageBlocksToAny(imageContentBlocks)...)
		um, err := newUserMessage(content, uuid, metaPtr(isMeta), permissionMode)
		if err != nil {
			return ProcessTextPromptResult{}, err
		}
		if len(imagePasteIDs) > 0 {
			um.ImagePasteIDs = imagePasteIDs
		}
		msgs := append([]types.Message{um}, attachmentMessages...)
		return ProcessTextPromptResult{Messages: msgs, ShouldQuery: true}, nil
	}

	var content any
	if len(inputBlocks) > 0 {
		content = blocksToContentArray(inputBlocks)
	} else {
		content = inputStr
	}
	um, err := newUserMessage(content, uuid, metaPtr(isMeta), permissionMode)
	if err != nil {
		return ProcessTextPromptResult{}, err
	}
	msgs := append([]types.Message{um}, attachmentMessages...)
	return ProcessTextPromptResult{Messages: msgs, ShouldQuery: true}, nil
}

// userPromptTextForTenguInput mirrors processTextPrompt.ts userPromptText (first text block for array input).
func userPromptTextForTenguInput(inputStr string, inputBlocks []types.ContentBlockParam) string {
	if len(inputBlocks) == 0 {
		return inputStr
	}
	for _, b := range inputBlocks {
		if b.Type == "text" {
			return b.Text
		}
	}
	return ""
}

func metaPtr(isMeta *bool) *bool {
	if isMeta != nil && *isMeta {
		t := true
		return &t
	}
	return nil
}

func buildTextBlocks(inputStr string, inputBlocks []types.ContentBlockParam) []any {
	if len(inputBlocks) > 0 {
		return blocksToContentArray(inputBlocks)
	}
	s := strings.TrimSpace(inputStr)
	if s == "" {
		return nil
	}
	return []any{map[string]any{"type": "text", "text": s}}
}

func blocksToContentArray(blocks []types.ContentBlockParam) []any {
	out := make([]any, 0, len(blocks))
	for _, b := range blocks {
		m := map[string]any{"type": b.Type, "text": b.Text}
		if len(b.Source) > 0 {
			m["source"] = json.RawMessage(b.Source)
		}
		out = append(out, m)
	}
	return out
}

func imageBlocksToAny(blocks []types.ContentBlockParam) []any {
	return blocksToContentArray(blocks)
}
