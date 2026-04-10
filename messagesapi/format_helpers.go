package messagesapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// formatFileSize mirrors src/utils/format.ts formatFileSize.
func formatFileSize(sizeInBytes int64) string {
	kb := float64(sizeInBytes) / 1024
	if kb < 1 {
		return fmt.Sprintf("%d bytes", sizeInBytes)
	}
	if kb < 1024 {
		return trimTrailingZero(fmt.Sprintf("%.1fKB", kb))
	}
	mb := kb / 1024
	if mb < 1024 {
		return trimTrailingZero(fmt.Sprintf("%.1fMB", mb))
	}
	gb := mb / 1024
	return trimTrailingZero(fmt.Sprintf("%.1fGB", gb))
}

func trimTrailingZero(s string) string {
	return strings.TrimSuffix(s, ".0")
}

// shellQuoteSingle mirrors shell-quote for a single path token (directory attachment ls).
func shellQuoteSingle(path string) string {
	if path == "" {
		return "''"
	}
	if !strings.ContainsAny(path, " \t\n\"'\\$`") {
		return path
	}
	return "'" + strings.ReplaceAll(path, `'`, `'\''`) + "'"
}

// deriveShortMessageId mirrors src/utils/messageIdUtils.ts.
func deriveShortMessageId(uuidStr string) string {
	hex := strings.ReplaceAll(uuidStr, "-", "")
	if len(hex) < 10 {
		hex = hex + strings.Repeat("0", 10-len(hex))
	}
	hex = hex[:10]
	n, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return "0"
	}
	s := strconv.FormatUint(n, 36)
	if len(s) > 6 {
		return s[:6]
	}
	return s
}

func jsonStringify(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
