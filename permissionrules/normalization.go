package permissionrules

import "strings"

// Claude.ai server names are prefixed with this string (src/services/mcp/normalization.ts).
const claudeAIServerPrefix = "claude.ai "

// NormalizeNameForMCP mirrors normalizeNameForMCP in src/services/mcp/normalization.ts.
func NormalizeNameForMCP(name string) string {
	normalized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, name)
	if strings.HasPrefix(name, claudeAIServerPrefix) {
		for strings.Contains(normalized, "__") {
			normalized = strings.ReplaceAll(normalized, "__", "_")
		}
		normalized = strings.Trim(normalized, "_")
	}
	return normalized
}
