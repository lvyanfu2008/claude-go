// Package textutil holds small string helpers for gou TUI rendering (e.g. OSC 8 URLs).
package textutil

import "runtime"

// AssistantBullet returns the BLACK_CIRCLE glyph matching TS figures.ts:
// ⏺ on macOS, ● on other platforms.
func AssistantBullet() string {
	if runtime.GOOS == "darwin" {
		return "⏺ "
	}
	return "● "
}
