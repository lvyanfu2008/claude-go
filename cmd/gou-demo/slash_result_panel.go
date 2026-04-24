package main

import (
	"encoding/json"
	"strings"

	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/gou/layout"
	"goc/gou/pui"
	"goc/gou/theme"

	"charm.land/lipgloss/v2"

	"goc/types"
)

func slashResultPanelMaxBodyLines(termHeight int) int {
	return min(12, max(4, termHeight/4))
}

// slashResultPanelLayout returns separator, wrapped body lines, and hint when the panel should show.
func (m *model) slashResultPanelLayout() (sep string, body []string, hint string) {
	if m.slashResultPanel == nil || m.uiScreen != gouDemoScreenPrompt {
		return "", nil, ""
	}
	raw := strings.TrimSpace(*m.slashResultPanel)
	if raw == "" {
		return "", nil, ""
	}
	cols := m.cols
	if cols < 1 {
		cols = 40
	}
	wrapped := layout.WrapForViewport(raw, cols)
	fullLines := strings.Split(wrapped, "\n")
	maxB := slashResultPanelMaxBodyLines(m.height)
	truncated := len(fullLines) > maxB
	lines := fullLines
	if truncated {
		lines = fullLines[:maxB]
		if len(lines) > 0 {
			lines[len(lines)-1] = strings.TrimRight(lines[len(lines)-1], " \t") + "…"
		}
	}
	rule := strings.Repeat("─", cols)
	sep = lipgloss.NewStyle().Faint(true).Foreground(theme.DimMuted()).Width(cols).Render(rule)
	hint = lipgloss.NewStyle().Faint(true).Foreground(theme.DimMuted()).Render("Esc 关闭")
	return sep, lines, hint
}

func (m *model) slashResultPanelChromeExtra() int {
	_, body, _ := m.slashResultPanelLayout()
	if len(body) == 0 {
		return 0
	}
	// Separator + body + hint line
	return 1 + len(body) + 1
}

func (m *model) slashResultPanelViewBlock() string {
	sep, body, hint := m.slashResultPanelLayout()
	if len(body) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(sep)
	b.WriteByte('\n')
	for i, ln := range body {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(ln)
	}
	b.WriteByte('\n')
	b.WriteString(hint)
	return b.String()
}

// slashResultForStoreOmittingPanelDupes drops informational system rows that are shown in the bottom
// slash panel so they are not duplicated in the scrollable message list above the input.
func slashResultForStoreOmittingPanelDupes(r *processuserinput.ProcessUserInputBaseResult) *processuserinput.ProcessUserInputBaseResult {
	if r == nil || len(r.Messages) == 0 {
		return r
	}
	const inf = "informational"
	var kept []types.Message
	for i := range r.Messages {
		msg := r.Messages[i]
		if msg.Type == types.MessageTypeSystem && msg.Subtype != nil && *msg.Subtype == inf {
			var s string
			if json.Unmarshal(msg.Content, &s) == nil && strings.TrimSpace(s) != "" {
				continue
			}
		}
		kept = append(kept, msg)
	}
	if len(kept) == len(r.Messages) {
		return r
	}
	out := *r
	out.Messages = kept
	return &out
}

func extractSlashLocalPanelText(r *processuserinput.ProcessUserInputBaseResult) string {
	if r == nil || len(r.Messages) == 0 {
		return ""
	}
	const inf = "informational"
	var parts []string
	for i := range r.Messages {
		msg := r.Messages[i]
		if msg.Type != types.MessageTypeSystem {
			continue
		}
		if msg.Subtype == nil || *msg.Subtype != inf {
			continue
		}
		var s string
		if json.Unmarshal(msg.Content, &s) != nil || strings.TrimSpace(s) == "" {
			continue
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n\n")
}

func (m *model) applySlashResultPanelFromSubmit(line string, r *processuserinput.ProcessUserInputBaseResult, out pui.ApplyProcessUserInputBaseResultOutcome) {
	if m.uiScreen != gouDemoScreenPrompt {
		m.slashResultPanel = nil
		return
	}
	if strings.HasPrefix(line, "/") && r != nil && !out.HadExecutionRequest && !out.EffectiveShouldQuery && len(r.Messages) > 0 {
		if txt := extractSlashLocalPanelText(r); txt != "" {
			m.slashResultPanel = ptrCloneString(txt)
			return
		}
	}
	m.slashResultPanel = nil
}

func ptrCloneString(s string) *string {
	p := new(string)
	*p = s
	return p
}

func (m *model) clearSlashResultPanel() {
	m.slashResultPanel = nil
}
