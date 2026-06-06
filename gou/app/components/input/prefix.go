package input

import (
	"strings"

	"charm.land/lipgloss/v2"

	"goc/gou/theme"
)

// UserPromptPointerGlyph is the character before user-authored text in the message
// list and input area (same line, one space after).
func UserPromptPointerGlyph() string {
	return ">"
}

// UserPromptPrefixStyled renders bright "> " for user rows (matches user message body emphasis).
func UserPromptPrefixStyled(userMsgRowBg bool) string {
	st := lipgloss.NewStyle().Foreground(theme.UserMessageText()).Bold(true)
	if userMsgRowBg {
		st = st.Background(theme.UserMessageBackground())
	}
	return st.Render(UserPromptPointerGlyph() + " ")
}

// UserInputViewWithPromptPrefix prepends the same dim "> " as user rows on the first line of the bottom input.
func UserInputViewWithPromptPrefix(deps Deps) string {
	v := deps.InputView()
	prefix := UserPromptPrefixStyled(false)
	lines := strings.Split(v, "\n")
	if len(lines) == 0 {
		return prefix
	}
	lines[0] = prefix + lines[0]
	return strings.Join(lines, "\n")
}
