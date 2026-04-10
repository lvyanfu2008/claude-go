package claudemd

import (
	"regexp"
	"strings"
)

var htmlCommentSpan = regexp.MustCompile(`<!--[\s\S]*?-->`)

// StripHTMLCommentsFenceAware removes `<!-- ... -->` outside fenced code blocks (``` / ~~~).
func StripHTMLCommentsFenceAware(content string) (out string, stripped bool) {
	lines := strings.Split(content, "\n")
	var b strings.Builder
	inFence := false
	fenceMarker := ""
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if inFence {
			b.WriteString(line)
			b.WriteByte('\n')
			if isFenceEnd(trim, fenceMarker) {
				inFence = false
				fenceMarker = ""
			}
			continue
		}
		if fm, ok := fenceStart(trim); ok {
			inFence = true
			fenceMarker = fm
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}
		newLine := htmlCommentSpan.ReplaceAllStringFunc(line, func(s string) string {
			stripped = true
			return ""
		})
		b.WriteString(newLine)
		b.WriteByte('\n')
	}
	out = strings.TrimSuffix(b.String(), "\n")
	if strings.HasSuffix(content, "\n") && out != "" {
		out += "\n"
	}
	return out, stripped
}

func fenceStart(trim string) (marker string, ok bool) {
	if strings.HasPrefix(trim, "```") && len(trim) >= 3 {
		return "```", true
	}
	if strings.HasPrefix(trim, "~~~") && len(trim) >= 3 {
		return "~~~", true
	}
	return "", false
}

func isFenceEnd(trim, marker string) bool {
	if marker == "" {
		return false
	}
	return strings.HasPrefix(trim, marker) && strings.TrimSpace(trim) == marker
}
