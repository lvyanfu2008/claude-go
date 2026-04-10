package messagesapi

import (
	"fmt"

	"goc/types"
)

// ImageSizeError mirrors src/utils/imageValidation.ts ImageSizeError.
type ImageSizeError struct {
	Index int
	Size  int
	Max   int
}

func (e *ImageSizeError) Error() string {
	return fmt.Sprintf("image at index %d: base64 length %d exceeds API limit %d", e.Index, e.Size, e.Max)
}

// validateImagesForAPI mirrors src/utils/imageValidation.ts validateImagesForAPI.
func validateImagesForAPI(messages []types.Message) error {
	var firstErr *ImageSizeError
	imageIndex := 0
	for _, msg := range messages {
		if msg.Type != types.MessageTypeUser {
			continue
		}
		inner, err := getInner(&msg)
		if err != nil {
			continue
		}
		blocks, err := parseContentArrayOrString(inner.Content)
		if err != nil {
			continue
		}
		for _, block := range blocks {
			if t, _ := block["type"].(string); t != "image" {
				continue
			}
			src, ok := block["source"].(map[string]any)
			if !ok {
				continue
			}
			st, _ := src["type"].(string)
			if st != "base64" {
				continue
			}
			data, _ := src["data"].(string)
			if data == "" {
				continue
			}
			imageIndex++
			if len(data) > apiImageMaxBase64Size {
				firstErr = &ImageSizeError{Index: imageIndex, Size: len(data), Max: apiImageMaxBase64Size}
			}
		}
	}
	if firstErr != nil {
		return firstErr
	}
	return nil
}

