package processuserinput

import (
	"fmt"
	"os"
	"path/filepath"

	"goc/utils"
)

// storeImages mirrors TS storeImages from imageStore.ts.
// Writes pasted image contents to disk so Claude can reference the file path
// (for manipulation with CLI tools, uploading to PRs, etc.).
// Returns a map of pasted content ID → stored file path.
func storeImages(pastedContents map[string]utils.PastedContent) (map[int]string, error) {
	if len(pastedContents) == 0 {
		return nil, nil
	}

	// Use temp directory for image cache (simpler than ~/.claude/image-cache/)
	tmpDir, err := os.MkdirTemp("", "claude-pasted-images-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir for images: %w", err)
	}

	paths := make(map[int]string, len(pastedContents))
	for idStr, pc := range pastedContents {
		// Parse the string-keyed ID back to int
		var id int
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			id = pc.ID
		}
		if pc.Type != "image" || pc.Content == "" {
			continue
		}

		ext := detectImageFormatFromMagic([]byte(pc.Content))
		// Strip "image/" prefix for extension
		extStr := "png"
		switch ext {
		case "image/jpeg":
			extStr = "jpg"
		case "image/png":
			extStr = "png"
		case "image/gif":
			extStr = "gif"
		case "image/webp":
			extStr = "webp"
		}

		fileName := fmt.Sprintf("%d.%s", id, extStr)
		filePath := filepath.Join(tmpDir, fileName)

		if err := os.WriteFile(filePath, []byte(pc.Content), 0644); err != nil {
			// Non-fatal: log but continue
			continue
		}
		paths[id] = filePath
	}

	return paths, nil
}
