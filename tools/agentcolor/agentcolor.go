// Package agentcolor mirrors src/tools/AgentTool/agentColorManager.ts — per-agent-type
// color assignment and theme-key resolution for UI display.
//
// Color names map to Ink theme keys (e.g. "red" → "red_FOR_SUBAGENTS_ONLY") so the
// rendered color respects the active theme. The global color map is thread-safe and
// lives for the lifetime of the process — no persistence across restarts.
package agentcolor

import "sync"

// ColorName is one of the predefined agent colors.
type ColorName string

const (
	ColorRed    ColorName = "red"
	ColorBlue   ColorName = "blue"
	ColorGreen  ColorName = "green"
	ColorYellow ColorName = "yellow"
	ColorPurple ColorName = "purple"
	ColorOrange ColorName = "orange"
	ColorPink   ColorName = "pink"
	ColorCyan   ColorName = "cyan"
)

// AllColors is the definitive list of available agent colors, in order.
var AllColors = []ColorName{
	ColorRed,
	ColorBlue,
	ColorGreen,
	ColorYellow,
	ColorPurple,
	ColorOrange,
	ColorPink,
	ColorCyan,
}

// ThemeColorKey returns the Ink theme key for a ColorName.
// These keys are used as Ink TextProps['color'] values and respect the active theme.
func (c ColorName) ThemeColorKey() string {
	return string(c) + "_FOR_SUBAGENTS_ONLY"
}

// ThemeColorKeyFor returns the theme key for a raw color string.
// Returns empty string if the string is not a known ColorName.
func ThemeColorKeyFor(color string) string {
	for _, c := range AllColors {
		if string(c) == color {
			return c.ThemeColorKey()
		}
	}
	return ""
}

// IsValidColorName returns true if the string is a known color name.
func IsValidColorName(s string) bool {
	for _, c := range AllColors {
		if string(c) == s {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------
// Global agent-color map (thread-safe)
// ---------------------------------------------------------------

type colorMap struct {
	mu    sync.RWMutex
	store map[string]ColorName // agentType -> color
}

var global = &colorMap{store: make(map[string]ColorName)}

// ResetColorsForTest clears the global color map (for tests only).
func ResetColorsForTest() {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.store = make(map[string]ColorName)
}

// GetAgentColorName returns the ColorName assigned to agentType, or nil.
func GetAgentColorName(agentType string) *ColorName {
	global.mu.RLock()
	defer global.mu.RUnlock()
	c, ok := global.store[agentType]
	if !ok {
		return nil
	}
	return &c
}

// GetThemeColorKey returns the Ink theme key for agentType.
// Returns nil (undefined) for "general-purpose" or unassigned agents.
// Mirrors TS getAgentColor — the primary lookup used by UI rendering.
func GetThemeColorKey(agentType string) *string {
	if agentType == "general-purpose" {
		return nil
	}

	global.mu.RLock()
	c, ok := global.store[agentType]
	global.mu.RUnlock()
	if !ok {
		return nil
	}
	tk := c.ThemeColorKey()
	return &tk
}

// SetAgentColor assigns or removes a color for agentType.
// A nil color deletes the entry. Only valid ColorName values are stored.
// Mirrors TS setAgentColor.
func SetAgentColor(agentType string, color *ColorName) {
	global.mu.Lock()
	defer global.mu.Unlock()

	if color == nil {
		delete(global.store, agentType)
		return
	}
	// Only store valid colors (matches TS guard: AGENT_COLORS.includes(color))
	for _, c := range AllColors {
		if c == *color {
			global.store[agentType] = *color
			return
		}
	}
}

// SetAgentColorName is a convenience wrapper that takes a raw string.
// Returns false if the string is not a valid color name.
func SetAgentColorName(agentType, color string) bool {
	if !IsValidColorName(color) {
		return false
	}
	c := ColorName(color)
	SetAgentColor(agentType, &c)
	return true
}

// DeleteAgentColor removes any color assignment for agentType.
func DeleteAgentColor(agentType string) {
	global.mu.Lock()
	defer global.mu.Unlock()
	delete(global.store, agentType)
}

// ---------------------------------------------------------------
// Bulk operations
// ---------------------------------------------------------------

// InitAgentColors bulk-initializes the color map from agent definitions.
// Mirrors TS loadAgentsDir.ts where active agents have their color set
// during agent definition loading (setAgentColor for each agent with a color).
func InitAgentColors(agents []AgentColorSetter) {
	global.mu.Lock()
	defer global.mu.Unlock()

	for _, a := range agents {
		if a.Color == "" {
			continue
		}
		for _, c := range AllColors {
			if string(c) == a.Color {
				global.store[a.AgentType] = c
				break
			}
		}
	}
}

// AgentColorSetter describes an agent definition that may carry a color.
type AgentColorSetter struct {
	AgentType string
	Color     string
}

// ---------------------------------------------------------------
// Default color for non-colored agents (TS: cyan_FOR_SUBAGENTS_ONLY)
// ---------------------------------------------------------------

const DefaultAgentThemeColor = "cyan_FOR_SUBAGENTS_ONLY"
