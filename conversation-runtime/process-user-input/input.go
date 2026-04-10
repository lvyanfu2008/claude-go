package processuserinput

import (
	"bytes"
	"encoding/json"

	"goc/types"
	"goc/utils"
)

func parseInput(raw json.RawMessage) (text string, blocks []types.ContentBlockParam, isString bool, err error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", nil, true, nil
	}
	if raw[0] == '"' {
		var s string
		err = json.Unmarshal(raw, &s)
		return s, nil, true, err
	}
	err = json.Unmarshal(raw, &blocks)
	return "", blocks, false, err
}

func blocksHaveImage(blocks []types.ContentBlockParam) bool {
	for _, b := range blocks {
		if b.Type == "image" {
			return true
		}
	}
	return false
}

func pastedHasImagePaste(m map[string]utils.PastedContent) bool {
	for _, v := range m {
		if isValidImagePaste(v) {
			return true
		}
	}
	return false
}
