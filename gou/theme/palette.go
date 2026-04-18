package theme

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"goc/types"
)

// Palette holds terminal colors for gou TUI (subset of TS design-system roles).
type Palette struct {
	User      color.Color
	Assistant color.Color
	Default   color.Color
	ToolUse   color.Color
	ToolMuted color.Color
	ToolError color.Color
	Advisor   color.Color
	Grouped   color.Color
	Collapsed color.Color
	Server    color.Color
	Heading   color.Color
	InlineCode color.Color // markdown `inline` spans — light blue (ANSI 256)
	// UserMessageBackground fills rows behind user-authored text in the gou-demo message list.
	UserMessageBackground color.Color
	// UserMessageText is the primary foreground for user-authored prose (bright; Bold applied in gou-demo).
	UserMessageText color.Color
}

var (
	activePalette  = defaultPalette()
	activeThemeKey = "default"
)

func defaultPalette() Palette {
	return Palette{
		User:                  lipgloss.Color("39"),
		Assistant:             lipgloss.Color("141"),
		Default:               lipgloss.Color("245"),
		ToolUse:               lipgloss.Color("213"),
		ToolMuted:             lipgloss.Color("245"),
		ToolError:             lipgloss.Color("196"),
		Advisor:               lipgloss.Color("183"),
		Grouped:               lipgloss.Color("226"),
		Collapsed:             lipgloss.Color("114"),
		Server:                lipgloss.Color("99"),
		Heading:               lipgloss.Color("255"), // markdown # / ## / ### — bright white (matches user body emphasis)
		InlineCode:            lipgloss.Color("117"), // light sky blue on dark bg
		UserMessageBackground: lipgloss.Color("236"),
		UserMessageText:       lipgloss.Color("255"),
	}
}

// lightPalette uses higher-contrast ANSI256 picks (rough TS "light" terminal feel).
func lightPalette() Palette {
	return Palette{
		User:                  lipgloss.Color("25"),
		Assistant:             lipgloss.Color("55"),
		Default:               lipgloss.Color("240"),
		ToolUse:               lipgloss.Color("92"),
		ToolMuted:             lipgloss.Color("241"),
		ToolError:             lipgloss.Color("124"),
		Advisor:               lipgloss.Color("96"),
		Grouped:               lipgloss.Color("130"),
		Collapsed:             lipgloss.Color("64"),
		Server:                lipgloss.Color("54"),
		Heading:               lipgloss.Color("24"), // primary text on light bg (matches UserMessageText)
		InlineCode:            lipgloss.Color("39"), // dodger blue on light bg (distinct from User 25)
		UserMessageBackground: lipgloss.Color("252"),
		UserMessageText:       lipgloss.Color("24"),
	}
}

func paletteForName(name string) Palette {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "light":
		return lightPalette()
	default:
		return defaultPalette()
	}
}

// InitFromThemeName selects the active lipgloss palette (call after merged settings env, e.g. CLAUDE_CODE_THEME).
func InitFromThemeName(name string) {
	activeThemeKey = ParseThemeName(name)
	activePalette = paletteForName(activeThemeKey)
}

// ActiveTheme returns the normalized theme key last passed to [InitFromThemeName].
func ActiveTheme() string {
	return activeThemeKey
}

// MessageTypeColor returns the role header color from the active palette.
func MessageTypeColor(mt types.MessageType) color.Color {
	switch mt {
	case types.MessageTypeUser:
		return activePalette.User
	case types.MessageTypeAssistant:
		return activePalette.Assistant
	default:
		return activePalette.Default
	}
}

// ToolError returns the active tool error color.
func ToolError() color.Color {
	return activePalette.ToolError
}

// ToolWarning is unchanged across palettes (attention).
func ToolWarning() color.Color {
	return lipgloss.Color("214")
}

// DimMuted returns secondary / tool_result default tone.
func DimMuted() color.Color {
	return activePalette.ToolMuted
}

// ToolUseAccent is the tool_use block accent.
func ToolUseAccent() color.Color {
	return activePalette.ToolUse
}

// AdvisorAccent is advisor_tool_result default (non-error).
func AdvisorAccent() color.Color {
	return activePalette.Advisor
}

// GroupedAccent is grouped_tool_use accent.
func GroupedAccent() color.Color {
	return activePalette.Grouped
}

// CollapsedAccent is collapsed_read_search accent.
func CollapsedAccent() color.Color {
	return activePalette.Collapsed
}

// ServerAccent is server_tool_use accent.
func ServerAccent() color.Color {
	return activePalette.Server
}

// MarkdownHeading is ATX heading (# … ###) foreground in markdown (bold applied in gou-demo).
func MarkdownHeading() color.Color {
	return activePalette.Heading
}

// MarkdownInlineCode is inline `code` span color in markdown body styling.
func MarkdownInlineCode() color.Color {
	return activePalette.InlineCode
}

// UserMessageBackground is the full-width row fill behind user message text in the gou-demo message list.
func UserMessageBackground() color.Color {
	return activePalette.UserMessageBackground
}

// UserMessageText returns the bright foreground for user-authored message body text in gou-demo.
func UserMessageText() color.Color {
	return activePalette.UserMessageText
}
