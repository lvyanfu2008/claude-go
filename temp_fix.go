package message

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"goc/ccb-engine/diaglog"
	"goc/types"
)

// renderTextBlock renders a text block.
func (r *UserMessageRenderer) renderTextBlock(block map[string]interface{}, ctx *RenderContext) ([]string, error) {
	text, _ := block["text"].(string)

	// Check for special message types
	if strings.Contains(text, "<bash-input>") {
		return r.renderBashInput(text, ctx)
	}
	if strings.Contains(text, "<bash-stdout") || strings.Contains(text, "<bash-stderr") {
		return r.renderBashOutput(text, ctx)
	}
	if strings.Contains(text, "<local-command-stdout") || strings.Contains(text, "<local-command-stderr") {
		return r.renderLocalCommandOutput(text, ctx)
	}

	// Regular user prompt
	lines := renderMarkdown(text, getContainerWidth(ctx), ctx.Theme, ctx.Highlighter)

	// Create lipgloss style for user messages: gray background, bold font
	userStyle := lipgloss.NewStyle().
		Background(ctx.Theme.UserMessageBackground).
		Foreground(ctx.Theme.UserMessageText).
		Bold(true)

	// Apply styling to each line including prefix
	for i, line := range lines {
		// Add prefix first, then apply styling to the entire line
		if i == 0 {
			line = "  > " + line
		} else {
			line = "    " + line
		}
		lines[i] = userStyle.Render(line)
	}

	return lines, nil
}