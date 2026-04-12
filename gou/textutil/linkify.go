package textutil

import (
	"regexp"
	"strings"
)

var urlPattern = regexp.MustCompile(`https?://[^\s\x1b"<>]+`)

// LinkifyOSC8 wraps http(s) URLs in OSC 8 hyperlinks (same idea as TS linkify in terminal).
// Safe to apply to plain tool output; avoid running on strings that already contain OSC sequences you need to preserve.
func LinkifyOSC8(s string) string {
	return urlPattern.ReplaceAllStringFunc(s, func(u string) string {
		u = strings.TrimRight(u, ".,;:!?)")
		// OSC 8: \e]8;;url\e\\  text  \e]8;;\e\\
		return "\x1b]8;;" + u + "\x1b\\" + u + "\x1b]8;;\x1b\\"
	})
}
