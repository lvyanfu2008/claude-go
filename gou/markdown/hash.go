package markdown

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashContent mirrors src/utils/hash.ts hashContent Node fallback (sha256 hex).
func HashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
