package claudemd

import (
	"fmt"
	"strings"
)

// MAXSanitizedLength matches src/utils/sessionStoragePortable.ts MAX_SANITIZED_LENGTH.
const MAXSanitizedLength = 200

// Djb2Hash matches src/utils/hash.ts djb2Hash (UTF-8 code points; TS uses UTF-16 — ASCII paths match).
func Djb2Hash(s string) int32 {
	var hash int32
	for _, r := range s {
		hash = ((hash << 5) - hash + int32(r)) | 0
	}
	return hash
}

// SanitizePath matches sessionStoragePortable.ts sanitizePath.
func SanitizePath(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	sanitized := b.String()
	if len(sanitized) <= MAXSanitizedLength {
		return sanitized
	}
	h := Djb2Hash(name)
	if h < 0 {
		h = -h
	}
	return fmt.Sprintf("%s-%s", sanitized[:MAXSanitizedLength], formatBase36(uint64(h)))
}

func formatBase36(n uint64) string {
	if n == 0 {
		return "0"
	}
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%36]
		n /= 36
	}
	return string(buf[i:])
}
