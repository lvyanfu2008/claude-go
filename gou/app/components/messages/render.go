package messages

import (
	"encoding/json"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"goc/gou/app/config"
	"goc/gou/markdown"
	"goc/gou/messagerow"
	"goc/gou/segdiff"
	"goc/gou/textutil"
	"goc/gou/theme"
	"goc/types"
)

// Renderer orchestrates message pane rendering for gou-demo.
// It holds a deps interface for model-level state and a highlighter for code
// syntax highlighting.
type Renderer struct {
	deps        Deps
	highlighter *markdown.Highlighter
}

// Deps provides model-level dependencies needed by the Renderer.
type Deps interface {
	// MessagerowOpts returns rendering options for a given message.
	MessagerowOpts(msg types.Message) *messagerow.RenderOpts

	// ShowToolUseCtrlOExpandHint returns true when the "(ctrl+o to expand)" hint
	// should be shown on tool_use rows.
	ShowToolUseCtrlOExpandHint() bool

	// ResolvedToolIDs returns the set of tool_use IDs that already have results.
	ResolvedToolIDs() map[string]struct{}

	// ScreenIsTranscript reports whether the current screen mode is transcript.
	ScreenIsTranscript() bool

	// ScreenShowAll reports whether "show all" is active (ctrl+e).
	ScreenShowAll() bool

	// ScreenDumpMode reports whether dump mode is active.
	ScreenDumpMode() bool
}

// NewRenderer creates a new Renderer with the given dependencies and highlighter.
func NewRenderer(deps Deps, hl *markdown.Highlighter) *Renderer {
	return &Renderer{
		deps:        deps,
		highlighter: hl,
	}
}

// Highlighter returns the syntax highlighter instance.
func (r *Renderer) Highlighter() *markdown.Highlighter {
	return r.highlighter
}

// FormatMessageSegments mirrors Message.tsx per-block branches
// (text->markdown, tool_use/tool_result/thinking).
// assistantLeadGlyph prefixes the first non-empty assistant text segment (TS-style
// ⏺ before the opening sentence).
// searchHL applies transcript search highlight to visible plain substrings
// (TS useSearchHighlight).
// showResolvedToolStats enables ⎿ TranscriptResolvedHintExtra for resolved Search/Read
// when tool_result JSON is available (prompt + transcript).
// userRow: when true, all lipgloss spans use the same row background as
// styleUserMessageLines (user-authored rows).
func (r *Renderer) FormatMessageSegments(
	segs []messagerow.Segment,
	cols int,
	toolUseCtrlOHint bool,
	resolved map[string]struct{},
	assistantLeadGlyph bool,
	searchHL string,
	toolResultByID map[string]json.RawMessage,
	showResolvedToolStats bool,
	userRow bool,
) string {
	hlSt := transcriptSearchHLStyle()
	withHL := func(s string) string {
		if strings.TrimSpace(searchHL) == "" {
			return s
		}
		return highlightSearchPlain(s, searchHL, hlSt)
	}
	var b strings.Builder
	var lastSegIdx int = -1
	assistantTextLeadDone := false
	for i, seg := range segs {
		var piece string
		switch seg.Kind {
		case messagerow.SegTextMarkdown:
			textForMd := seg.Text
			if strings.TrimSpace(searchHL) != "" {
				textForMd = highlightSearchPlain(seg.Text, searchHL, hlSt)
			}
			md := StyleMarkdownTokens(r.highlighter, markdown.CachedLexer(textForMd), cols, userRow)
			if assistantLeadGlyph && !assistantTextLeadDone && strings.TrimSpace(seg.Text) != "" {
				assistantTextLeadDone = true
				md = PrefixToolGlyphFirstLine(md)
			}
			piece = md
		case messagerow.SegToolUse:
			if seg.ToolFacing != "" {
				row1 := ""
				if !PriorNonEmptyAssistantText(segs, i) {
					row1 = ToolRowLeadPrefix(userRow)
				}
				row1 += BaseMsgStyle(userRow).Foreground(theme.ToolUseAccent()).Bold(true).Render(withHL(seg.ToolFacing))
				if p := strings.TrimSpace(seg.ToolParen); p != "" {
					row1 += " (" + withHL(p) + ")"
				}
				var toolLines []string
				toolLines = append(toolLines, row1)
				res := ToolUseResolvedForDisplay(resolved, toolResultByID, seg.ToolUseID, showResolvedToolStats)
				if showResolvedToolStats && res {
					var raw json.RawMessage
					if toolResultByID != nil {
						raw = toolResultByID[seg.ToolUseID]
					}
					hint, extra := messagerow.TranscriptResolvedHintExtra(seg.ToolFacing, raw)
					if hint != "" {
						toolLines = append(toolLines, BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(hint))))
						if extra != "" {
							toolLines = append(toolLines, BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("     "+textutil.LinkifyOSC8(withHL(extra))))
						}
					}
				} else if !res {
					if act := strings.TrimSpace(seg.Text); act != "" {
						actLine := BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(withHL(act) + "…")
						if toolUseCtrlOHint {
							actLine += BaseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
						}
						toolLines = append(toolLines, actLine)
					}
					if h := strings.TrimSpace(seg.ToolHint); h != "" {
						toolLines = append(toolLines, BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(h))))
					}
				}
				piece = strings.Join(toolLines, "\n")
			} else {
				line := BaseMsgStyle(userRow).Foreground(theme.ToolUseAccent()).Bold(true).Render("⚙ " + withHL(seg.Text))
				if toolUseCtrlOHint {
					line += BaseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
				}
				piece = line
			}
		case messagerow.SegToolResult:
			piece = segdiff.FormatToolResultSegmentForTranscript(seg, userRow, toolUseCtrlOHint, cols, withHL, BaseMsgStyle)
		case messagerow.SegThinking:
			body := textutil.LinkifyOSC8(seg.Text)
			piece = BaseMsgStyle(userRow).Bold(true).Render("● " + withHL(body))
		case messagerow.SegDisplayHint:
			piece = BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
		case messagerow.SegServerToolUse:
			if seg.ToolFacing != "" {
				row1 := ""
				if !PriorNonEmptyAssistantText(segs, i) {
					row1 = ToolRowLeadPrefix(userRow)
				}
				row1 += BaseMsgStyle(userRow).Foreground(theme.ServerAccent()).Bold(true).Render(withHL(seg.ToolFacing))
				if p := strings.TrimSpace(seg.ToolParen); p != "" {
					row1 += " (" + withHL(p) + ")"
				}
				var toolLines []string
				toolLines = append(toolLines, row1)
				res := ToolUseResolvedForDisplay(resolved, toolResultByID, seg.ToolUseID, showResolvedToolStats)
				if showResolvedToolStats && res {
					var raw json.RawMessage
					if toolResultByID != nil {
						raw = toolResultByID[seg.ToolUseID]
					}
					hint, extra := messagerow.TranscriptResolvedHintExtra(seg.ToolFacing, raw)
					if hint != "" {
						toolLines = append(toolLines, BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(hint))))
						if extra != "" {
							toolLines = append(toolLines, BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("     "+textutil.LinkifyOSC8(withHL(extra))))
						}
					}
				} else if !res {
					if act := strings.TrimSpace(seg.Text); act != "" {
						actLine := BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(withHL(act) + "…")
						if toolUseCtrlOHint {
							actLine += BaseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
						}
						toolLines = append(toolLines, actLine)
					}
					if h := strings.TrimSpace(seg.ToolHint); h != "" {
						toolLines = append(toolLines, BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(h))))
					}
				}
				piece = strings.Join(toolLines, "\n")
			} else {
				line := BaseMsgStyle(userRow).Foreground(theme.ServerAccent()).Bold(true).Render("⎈ " + withHL(seg.Text))
				if toolUseCtrlOHint {
					line += BaseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
				}
				piece = line
			}
		case messagerow.SegAdvisorToolResult:
			st := BaseMsgStyle(userRow).Foreground(theme.AdvisorAccent())
			if seg.IsToolError {
				st = BaseMsgStyle(userRow).Foreground(theme.ToolError())
			}
			body := textutil.LinkifyOSC8(seg.Text)
			line := st.Render("✧ " + withHL(body))
			if seg.ToolBodyOmitted && toolUseCtrlOHint {
				line += BaseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
			}
			piece = line
		case messagerow.SegGroupedToolUse:
			piece = BaseMsgStyle(userRow).Foreground(theme.GroupedAccent()).Bold(true).Render("▦ " + withHL(seg.Text))
		case messagerow.SegCollapsedReadSearch:
			piece = BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
		case messagerow.SegToolUseSummaryLine:
			line := BaseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
			if !ToolUseSummaryLineResolvedForDisplay(resolved, toolResultByID, seg.ToolUseIDs, seg.ToolUseID, showResolvedToolStats) && toolUseCtrlOHint {
				line += BaseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
			}
			piece = "  " + line
		case messagerow.SegSkillListingAvailable:
			n := seg.Num
			if n < 1 {
				n = 1
			}
			word := "skills"
			if n == 1 {
				word = "skill"
			}
			piece = BaseMsgStyle(userRow).Bold(true).Render(strconv.Itoa(n)) + BaseMsgStyle(userRow).Render(" "+word+" available")
		default:
			piece = BaseMsgStyle(userRow).Faint(true).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
		}
		if piece == "" {
			continue
		}
		if b.Len() > 0 && lastSegIdx >= 0 {
			b.WriteString(SegmentJoinSeparator(segs[lastSegIdx], segs[i]))
		}
		b.WriteString(piece)
		lastSegIdx = i
	}
	return strings.TrimSpace(b.String())
}

// RenderMessageRow renders a single message row (legacy path).
// Returns the rendered string, or empty if the message should be skipped.
func (r *Renderer) RenderMessageRow(msg types.Message, cols int, searchHL string) string {
	ropts := r.deps.MessagerowOpts(msg)
	if SkipFoldedToolResultStubInPrompt(
		msg,
		messagerow.VerboseToolOutputEnabled(),
		ropts,
		r.deps.ScreenIsTranscript() == false,
		r.deps.ScreenIsTranscript(),
		r.deps.ScreenShowAll(),
		r.deps.ScreenDumpMode(),
	) {
		return ""
	}

	segs := messagerow.SegmentsFromMessageOpts(msg, ropts)
	var header string
	if msg.Type != types.MessageTypeAttachment {
		switch msg.Type {
		case types.MessageTypeUser:
			// No "user" title row: "> " on the first body line only.
		case types.MessageTypeAssistant:
			// No "assistant" title row — body starts directly.
		case types.MessageTypeCollapsedReadSearch, types.MessageTypeGroupedToolUse:
			// Same as assistant — no raw type label.
		default:
			header = lipgloss.NewStyle().Bold(true).Foreground(theme.MessageTypeColor(msg.Type)).Render(string(msg.Type))
		}
	}

	body := r.FormatMessageSegments(
		segs,
		cols,
		r.deps.ShowToolUseCtrlOExpandHint(),
		r.deps.ResolvedToolIDs(),
		msg.Type == types.MessageTypeAssistant,
		searchHL,
		messagerow.CollectToolResultContentByToolUseID(nil),
		true,
		msg.Type == types.MessageTypeUser,
	)

	body = withUserPromptPointerIfNeeded(msg, body)
	body = withCollapsedSpaceIfNeeded(msg, body)
	block := body
	if header != "" {
		block = header + "\n" + body
	}
	wrapped := applyMessagePaneGutter(block, cols)
	rows := strings.Split(wrapped, "\n")
	maxRows := 1000000 // effectively unlimited
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	if msg.Type == types.MessageTypeUser {
		return "\n" + styleUserMessageLines(rows, cols) + "\n"
	}
	return strings.Join(rows, "\n")
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// withUserPromptPointerIfNeeded prepends dim "> " before the first body line of
// user messages (same line as text).
func withUserPromptPointerIfNeeded(msg types.Message, body string) string {
	if msg.Type != types.MessageTypeUser || !UserMessageHasPromptText(msg) || body == "" {
		return body
	}
	prefix := userPromptPrefixStyled(true)
	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		return prefix
	}
	lines[0] = prefix + lines[0]
	return strings.Join(lines, "\n")
}

// userPromptPrefixStyled renders bright "> " for user rows (matches user message
// body emphasis).
func userPromptPrefixStyled(userMsgRowBg bool) string {
	st := lipgloss.NewStyle().Foreground(theme.UserMessageText()).Bold(true)
	if userMsgRowBg {
		st = st.Background(theme.UserMessageBackground())
	}
	return st.Render("> ")
}

// styleUserMessageLines applies a full-width gray background per row (ANSI-safe;
// lipgloss pads to cols).
func styleUserMessageLines(rows []string, cols int) string {
	st := lipgloss.NewStyle().Background(theme.UserMessageBackground()).Width(cols)
	var b strings.Builder
	for i, ln := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(st.Render(ln))
	}
	return b.String()
}

// withCollapsedSpaceIfNeeded adds leading space for collapsed message types.
func withCollapsedSpaceIfNeeded(msg types.Message, body string) string {
	if msg.Type != types.MessageTypeCollapsedReadSearch || body == "" {
		return body
	}
	return "  " + body
}

// applyMessagePaneGutter delegates to config.ApplyMessagePaneGutter.
func applyMessagePaneGutter(block string, cols int) string {
	return config.ApplyMessagePaneGutter(block, cols)
}

// transcriptSearchHLStyle returns the lipgloss style for transcript search
// highlight matches (terminal TS useSearchHighlight parity).
func transcriptSearchHLStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color("58")).Foreground(lipgloss.Color("230"))
}

// highlightSearchPlain wraps case-insensitive needle matches in hl.
func highlightSearchPlain(s, needle string, hl lipgloss.Style) string {
	needle = strings.TrimSpace(needle)
	if needle == "" || s == "" {
		return s
	}
	lowS := strings.ToLower(s)
	lowN := strings.ToLower(needle)
	if lowN == "" {
		return s
	}
	var b strings.Builder
	cur := 0
	for cur < len(s) {
		rel := strings.Index(lowS[cur:], lowN)
		if rel < 0 {
			b.WriteString(s[cur:])
			break
		}
		idx := cur + rel
		b.WriteString(s[cur:idx])
		end := idx + len(lowN)
		if end > len(s) {
			end = len(s)
		}
		b.WriteString(hl.Render(s[idx:end]))
		cur = end
	}
	return b.String()
}
