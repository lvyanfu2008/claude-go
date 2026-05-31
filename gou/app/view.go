// Command gou-demo is a minimal Bubble Tea full-screen UI: new message renderer + markdown + tool blocks (Phase 4 messagerow).
// Extracted [model.Update] branches: update_streaming.go (query yield / NDJSON), update_layout.go (window resize).
// Layout uses the full terminal size (main buffer by default). Optional GOU_DEMO_ALT_SCREEN=1 uses the alternate buffer (no shell scrollback mixing).
// With GOU_DEMO_LOG=1, trace uses the same path rules as TS debug log (see goc/ccb-engine/debugpath); on TTY without GOU_DEMO_LOG_FILE, trace goes to that file, not stderr.
//
// Run from repo: cd goc && go run ./cmd/gou-demo
//
// Flags: -transcript=file.json (UI or API messages), -replay-cc=events.ndjson, -stream-stdin (pipe NDJSON),
// Real model: [goc/conversation-runtime/query.Query] HTTP streaming parity when ANTHROPIC_API_KEY (or ANTHROPIC_AUTH_TOKEN) is set
// and GOU_QUERY_STREAMING_PARITY=1 or GOU_DEMO_STREAMING_TOOL_EXECUTION=1 (see [query.BuildQueryConfig]).
// When a tool gate returns ask, GOU_QUERY_ASK_STRATEGY=allow auto-allows for headless demo (maps to [toolexecution.ExecutionDeps.AskResolver]).
// GOU_TOOLEXEC_BASH_SANDBOX_1B=1 enables permissions.ts whole-tool ask bypass on Bash when the tool input carries a non-empty command without dangerously_disable_sandbox (see toolexecution.WholeToolAskSkippedForBash1b).
// Go-side init port (subset of TS init.ts): gou-demo runs [goc/claudeinit.Init] (includes [settingsfile.EnsureProjectClaudeEnvOnce]). See docs/plans/go-init-port.md.
// Go local tool parity (streaming parity + [skilltools.ParityToolRunner]): Bash is allowed by default (same as TS); set GOU_DEMO_NO_LOCAL_BASH=1 to disable unless CCB_ENGINE_LOCAL_BASH=1. PowerShell is off unless CCB_ENGINE_LOCAL_POWERSHELL=1 (uses pwsh or powershell.exe). AskUserQuestion auto-picks the first option per question unless GOU_DEMO_NO_ASK_AUTO_FIRST=1. WebFetch is allowed by default; set CCB_ENGINE_DISABLE_WEB_FETCH=1 to block network fetches in the Go runner. See docs/plans/go-tools-parity.md.
//
// System # Language / # Output Style: merged from ~/.harness/settings.go.json and project .harness/settings.go.json / settings.local.json (see settingsfile; project settings.go.json is TS-only). CLAUDE_CODE_LANGUAGE and CLAUDE_CODE_OUTPUT_STYLE_* override when set (non-empty); built-in outputStyle keys Explanatory/Learning use prompts from src/constants/outputStyles.ts (embedded).
// Extra HARNESS.md roots: optional runtimeContext.toolPermissionContext.additionalWorkingDirectories (JSON) and/or GOU_DEMO_EXTRA_CLAUDE_MD_ROOTS / CLAUDE_CODE_EXTRA_CLAUDE_MD_ROOTS (comma or PATH-style list). Paths from runtime/env are always scanned when passed (see [querycontext.ExtraClaudeMdRootsForFetch]); CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1 is only needed for env-only flows in claudemd that do not pass explicit roots.
// Debug log (optional): GOU_DEMO_LOG_FILE=/path/to.log, or GOU_DEMO_LOG=1 (default file path matches TS getDebugLogPath via goc/ccb-engine/debugpath when stderr is TTY). GOU_DEMO_LOG_STDERR=1 forces stderr (may corrupt TUI). Lines are prefixed [gou-demo].
// ToolUseContext dump: CLAUDE_CODE_LOG_TOOL_USE_CONTEXT or GOU_DEMO_LOG_TOOL_USE_CONTEXT = 1|summary|full (with logging enabled) prints JSON after each BuildDemoParams; full includes the entire commands[] snapshot.
// Read/Grep/Glob stream tail: default keeps each tool_use + tool_result as separate rows (avoids looking like history was cleared). Set GOU_DEMO_COLLAPSE_READ_SEARCH_TAIL=1 for TS-style merge into collapsed_read_search (gou/ccbstream/apply.go).
// Prompt: merged one-line Grep/Glob/Read summaries (GOU_DEMO_TOOL_USE_SUMMARY_LINE) wait GOU_DEMO_TOOL_USE_SUMMARY_DELAY_MS after each assistant message first appears (default 2000 ms) while full Search/Read rows are shown; set to 0 to collapse immediately.
//
// Keys: ↑/↓/PgUp/PgDn scroll the message pane, End bottom. Prompt: default Enter send; Alt+Enter or Option+Enter (macOS) newline when the terminal sends Meta; Ctrl+J / LF newline. GOU_DEMO_REPL_ENTER_SUBMITS=0 for chat mode (Enter newline, Alt+Enter send). Shift+↑↓ move line. F2 toggles the slash list; leading "/" (TS) or mid-input " … /tok" shows the list; ↑/↓ move selection; Tab inserts; Enter applies selection and runs submit; input stays in the main field. Ctrl+l forces a full-screen clear + redraw (TS Global app:redraw). Ctrl+o toggles TS-style transcript (frozen tail; / search with n/N when not in dump; search bar Esc clears; ctrl+e show-all expands collapsed/grouped + full tool_result bodies except in dump). In the main prompt, user messages that contain only tool_result / advisor_tool_result blocks are omitted from the list (no "user / ↩ tool_result …" stub row); mixed user rows still fold tool_result bodies to one line + (ctrl+o to expand). Transcript (compact): same omission + tool_result folded on user rows; assistant rows show ⏺+⎿ summaries. Ctrl+e show-all or [ dump shows full blocks. [ (no search bar) enables dump: show-all + plain transcript to scrollback (Printf). v opens frozen transcript in $VISUAL/$EDITOR via temp file (tea.ExecProcess). Transcript pager (search bar closed, not dump): arrows/pgup/pgdn/end, j/k, g, G/shift+g, ctrl+u/d, ctrl+b/f, b, space (full page), ctrl+n/p (line). Esc/q/ctrl+c exit transcript when search bar closed. In prompt mode, q or Esc quit. Columns < 80 use a shorter header/footer (TS REPL isNarrow). Terminal tab title: OSC 0 unless CLAUDE_CODE_DISABLE_TERMINAL_TITLE=1; loading shows a "…" prefix. CLAUDE_CODE_PERMISSION_MODE sets tool permission mode for submits (TS toolPermissionContext.mode).
// Theme: CLAUDE_CODE_THEME=light (after merged settings env) selects a higher-contrast palette; see [theme.InitFromThemeName]. GOU_DEMO_STATUS_LINE=1 shows theme/msg counts above the prompt.
// Message pane: new renderer ([message.VirtualList] in [gou/message]) drives both prompt and transcript screens. Prompt uses [bubbles/viewport] by default (full-document scroll + ctrl+y fold-all); disable with GOU_DEMO_BUBBLES_VIEWPORT=0|false|off|no to render the visible slice directly on top of m.scrollTop.
// Mouse: SGR mouse (cell motion) enables wheel + plain left-drag on the message list when not disabled by env. Set GOU_DEMO_DISABLE_MOUSE_SCROLL=1 to ignore wheel/drag in-app. Mirror TS fullscreen.ts: CLAUDE_CODE_DISABLE_MOUSE=1 or GOU_DEMO_DISABLE_MOUSE=1 omits SGR mouse (keyboard scroll still works), unless GOU_DEMO_DISALLOW_DISABLE_MOUSE=1. One-column TUI scrollbar is on by default when the pane is wide enough; GOU_DEMO_MESSAGE_SCROLLBAR=0|false|off|no or GOU_DEMO_NO_SCROLLBAR=1 turns it off. Alternate screen: opt-in GOU_DEMO_ALT_SCREEN=1 (default main buffer). Bubbles viewport: at-top wheel-up can release mouse for host scrollback; opt out with GOU_DEMO_MSG_HISTORY_MOUSE_RELEASE=0|false|off|no.
// Slash: /name is resolved in-process — disk skills via [goc/slashresolve.ResolveDiskSkill], bundled prompts via [goc/slashresolve.ResolveBundledSkill] (embedded markdown under slashresolve/skills/bundled). Other prompt commands need a disk skill (SkillRoot) or a bundled definition. Unknown names that look like command names and are not root filesystem paths (non-Windows) return TS-style Unknown skill without calling the model; otherwise the line is treated as a normal user prompt.
// MCP skills (scheme-2 R0/R1): -mcp-commands-json=path or GOU_DEMO_MCP_COMMANDS_JSON → JSON array of types.Command merged into Skill/commands (enable FEATURE_MCP_SKILLS=1 for listing).
// MCP tool defs (assembleToolPool): -mcp-tools-json=path or GOU_DEMO_MCP_TOOLS_JSON → JSON array merged into Options.Tools when GOU_DEMO_USE_EMBEDDED_TOOLS_API=1 (see mcpcommands.EnvToolsJSONPath).
//
// Session JSONL (default on): persists via [goc/sessiontranscript] (~/.harness/projects/.../<session>.jsonl). After each successful ProcessUserInput + ApplyBaseResult, maybeRecordTranscript runs so user rows land before streaming yields. Streaming parity wires [query.QueryDeps.OnQueryYield] to RecordTranscript with a growing turn prefix (same as TS recordTranscript(messages)) so parentUuid chains; each yield is deduped by message UUID; turn end still calls maybeRecordTranscript for a full-store sync. File-history-snapshot stubs: default at most one line per session (before the first non-meta user) unless CLAUDE_CODE_DISABLE_FILE_CHECKPOINTING (TS fileHistory off); GOU_DEMO_FILE_HISTORY_SNAPSHOT_EACH_USER=1 restores one stub before every new non-meta user; GOU_DEMO_SKIP_FILE_HISTORY_SNAPSHOT=1 omits stubs. User message UUIDs follow TS (crypto.randomUUID via process-user-input when DemoConfig.uuid is unset). Set GOU_DEMO_SESSION_ID to a UUID or the store gets a random UUID when the default "demo" id is invalid.
// Skill listing follows TS delta (sentSkillNames): later submits omit skills already injected. Set GOU_DEMO_SKILL_LISTING_EVERY_TURN=1 to use a fresh sent map each submit so the full listing is attached every round (debug only; not TS production behavior).
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"goc/ccb-engine/diaglog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"goc/commands"
	"goc/compactservice"
	processuserinput "goc/conversation-runtime/process-user-input"
	"goc/conversation-runtime/query"
	"goc/gou/ccbhydrate"
	"goc/gou/commandqueue"
	"goc/gou/markdown"
	"goc/gou/messagerow"
	"goc/gou/pui"
	"goc/gou/segdiff"
	"goc/gou/textutil"
	"goc/gou/theme"
	"goc/growthbook"
	"goc/hookexec"
	"goc/messagesapi"
	"goc/querycontext"
	"goc/services/autodream"
	"goc/services/extractmemories"
	"goc/services/sessionmemory"
	"goc/sessiontranscript"
	"goc/tools"
	"goc/tools/localtools"
	"goc/tools/skilltools"
	"goc/tools/toolexecution"
	"goc/tools/toolresultpersist"
	"goc/tools/toolsearchwire"
	"goc/types"
)

func (m *model) View() tea.View {
	// When the interactive hooks config menu is active, render it instead of the normal view.
	if m.hooksConfigMenu != nil {
		return m.wrapRootView(m.hooksConfigMenu.View().Content)
	}

	// When the interactive question UI is active, render it instead of the normal view.
	if m.questionUI != nil {
		return m.wrapRootView(m.questionUI.View().Content)
	}

	if m.width == 0 {
		return m.wrapRootView("Loading…")
	}

	vpH := listViewportH(m)
	bodyCols := m.messageBodyColsForLayout()
	useVp := m.msgViewportWanted()
	if useVp {
		m.msgViewportSyncGeometry()
		m.applyMsgViewportContentFromView()
		if m.msgViewportFallback {
			useVp = false
		}
	}

	var b strings.Builder
	var promptLineOffset int
	narrow := m.cols > 0 && m.cols < 80
	plainTitle := replChromeComposeTerminalTitle(m.store.ConversationID, m.queryBusy, m.store.HasStreaming())
	if !gouDemoTerminalTitleDisabled() && plainTitle != m.lastEmittedTitlePlain {
		m.lastEmittedTitlePlain = plainTitle
		if osc := oscSetWindowTitle(plainTitle); osc != "" {
			b.WriteString(osc)
		}
	}
	topBar := replChromeTopBar(narrow)
	if m.uiScreen == gouDemoScreenTranscript {
		topBar = replChromeTranscriptTopBar(narrow)
	}
	title := lipgloss.NewStyle().Bold(true).Render(topBar)
	b.WriteString(title)
	b.WriteByte('\n')

	m.renderMessagePane(&b, vpH, bodyCols, useVp)

	if m.uiScreen != gouDemoScreenTranscript {
		streamRows := m.promptBottomStreamRows()
		if len(streamRows) > 0 {
			b.WriteString(strings.Join(streamRows, "\n"))
			b.WriteByte('\n')
		}
		// Spinner row (when query busy, flush left)
		if m.queryBusy {
			spinner := SpinnerRow(m.spinnerVerb, m.spinnerFrame, m.queryBusyStartedAt, m.spinnerTokens, false, m.lastActivity, m.cols)
			if spinner != "" {
				b.WriteString(spinner)
				b.WriteByte('\n')
			}
		}
		// Task list (inline, after stream rows)
		if m.taskList != nil && m.taskList.isVisible() {
			maxDisplay := m.taskListViewMaxDisplay()
			if tl := m.taskList.view(maxDisplay, m.cols); tl != "" {
				indented := applyMessagePaneGutter(tl, m.width)
				b.WriteString(indented)
				b.WriteByte('\n')
			}
		}
	}
	if s := m.statusLineString(); s != "" {
		b.WriteString(s)
		b.WriteByte('\n')
	}

	promptLineOffset = m.promptAreaLayout(&b)
	out := lipgloss.NewStyle().MaxWidth(m.width).Render(b.String())
	if v := m.permModal.View(m.width); v != "" {
		out = lipgloss.JoinVertical(lipgloss.Left, out, v)
	}
	v := m.wrapRootView(out)
	if runtime.GOOS == "windows" && m.uiScreen == gouDemoScreenPrompt && m.pr.Focused() {
		v.Cursor = tea.NewCursor(
			2+m.pr.CursorDisplayCol(),
			promptLineOffset+m.pr.CursorLine(),
		)
	}
	return v
}

func (m *model) wrapRootView(content string) tea.View {
	v := tea.NewView(content)
	v.AltScreen = gouDemoAltScreenEnabled() && !m.suspendAltScreenForScrollbackDump
	if gouDemoMouseCellMotionEnabled() {
		if m.msgHistoryBrowseMouseOff {
			v.MouseMode = tea.MouseModeNone
		} else {
			v.MouseMode = tea.MouseModeCellMotion
		}
	}
	return v
}

func (m *model) showToolUseCtrlOExpandHint() bool {
	return m.uiScreen == gouDemoScreenPrompt && !m.transcriptDumpMode
}

// userAssistantPairBlankLine is true when the UI inserts one empty line between adjacent
// user and assistant scroll rows (either order).
func userAssistantPairBlankLine(a, b types.Message) bool {
	u, aType := types.MessageTypeUser, types.MessageTypeAssistant
	c := types.MessageTypeCollapsedReadSearch
	return a.Type == u && b.Type == aType || a.Type == c && b.Type == aType
}

// streamGapAfterUserMessage is true when the StreamingText tail should be separated from the
// message list by the same blank line as user↔assistant rows (last scroll message is user).
func streamGapAfterUserMessage(msgView []types.Message) bool {
	return len(msgView) > 0 && msgView[len(msgView)-1].Type == types.MessageTypeUser
}

func userMessageHasPromptText(msg types.Message) bool {
	if msg.Type != types.MessageTypeUser {
		return false
	}
	msg = messagerow.NormalizeMessageJSON(msg)
	if len(msg.Content) == 0 {
		return false
	}
	var blocks []types.MessageContentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return false
	}
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			return true
		}
	}
	return false
}

// userMessageRendersOnlyFoldedToolStubs is true when this user row would render only folded
// tool_result/advisor_tool_result stubs (ToolBodyOmitted), with no other visible segments.
// Matches actual [SegmentsFromMessageOpts] output so unknown content block types or API quirks
// do not leave a lone "↩ tool_result tool_use_id=…" line under assistant ⎿ summaries.
func (m *model) userMessageRendersOnlyFoldedToolStubs(msg types.Message) bool {
	if msg.Type != types.MessageTypeUser {
		return false
	}
	msg = messagerow.NormalizeMessageJSON(msg)
	if len(msg.Content) == 0 {
		return false
	}
	segs := messagerow.SegmentsFromMessageOpts(msg, m.messagerowOpts(msg))
	if len(segs) == 0 {
		return false
	}
	hasFoldedTool := false
	for _, s := range segs {
		switch s.Kind {
		case messagerow.SegTextMarkdown:
			if strings.TrimSpace(s.Text) != "" {
				return false
			}
		case messagerow.SegToolResult, messagerow.SegAdvisorToolResult:
			if !s.ToolBodyOmitted {
				return false
			}
			hasFoldedTool = true
		case messagerow.SegThinking:
			if strings.TrimSpace(s.Text) != "" {
				return false
			}
		default:
			return false
		}
	}
	return hasFoldedTool
}

// skipOmittableToolResultUserRow hides user messages that only render folded tool_result stubs
// (prompt always; transcript unless ctrl+e show-all or dump).
// The assistant tool_use row already shows the summary; omitting avoids duplicate ↩ tool_result lines.
func (m *model) skipFoldedToolResultStubInPrompt(msg types.Message) bool {
	if messagerow.VerboseToolOutputEnabled() {
		return false
	}
	if !m.userMessageRendersOnlyFoldedToolStubs(msg) {
		return false
	}
	if m.uiScreen == gouDemoScreenPrompt {
		return true
	}
	if m.uiScreen == gouDemoScreenTranscript && !m.transcriptShowAll && !m.transcriptDumpMode {
		return true
	}
	return false
}

// userPromptPrefixStyled renders bright "> " for user rows (matches user message body emphasis).
func userPromptPrefixStyled(userMsgRowBg bool) string {
	st := lipgloss.NewStyle().Foreground(theme.UserMessageText()).Bold(true)
	if userMsgRowBg {
		st = st.Background(theme.UserMessageBackground())
	}
	return st.Render(UserPromptPointerGlyph() + " ")
}

// userInputViewWithPromptPrefix prepends the same dim "> " as user rows on the first line of the bottom input.
func userInputViewWithPromptPrefix(m *model) string {
	v := m.pr.View()
	prefix := userPromptPrefixStyled(false)
	lines := strings.Split(v, "\n")
	if len(lines) == 0 {
		return prefix
	}
	lines[0] = prefix + lines[0]
	return strings.Join(lines, "\n")
}

func toolRowLeadPrefix(userRow bool) string {
	glyph := "\u25cf " // ● — TS figures.BLACK_CIRCLE non-darwin
	if runtime.GOOS == "darwin" {
		glyph = "\u23fa " // ⏺ — TS figures.BLACK_CIRCLE on darwin
	}
	return baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(glyph)
}

// prefixToolGlyphFirstLine prepends the dim tool lead (⏺ / ●) to the first line of rendered assistant text.
func prefixToolGlyphFirstLine(body string) string {
	if body == "" {
		return toolRowLeadPrefix(false)
	}
	p := toolRowLeadPrefix(false)
	i := strings.IndexByte(body, '\n')
	if i < 0 {
		return p + body
	}
	return p + body[:i] + body[i:]
}

func toolUseResolved(resolved map[string]struct{}, toolUseID string) bool {
	if resolved == nil || toolUseID == "" {
		return false
	}
	_, ok := resolved[toolUseID]
	return ok
}

// toolUseResolvedForDisplay treats a tool as resolved if it is in the resolved map, or (when detail is on)
// if tool_result payload exists for that id — avoids stale resolved maps skipping ⏺+⎿ stats.
func toolUseResolvedForDisplay(resolved map[string]struct{}, toolResultByID map[string]json.RawMessage, toolUseID string, allowResultPayloadAsResolved bool) bool {
	if toolUseID == "" {
		return false
	}
	if resolved != nil {
		if _, ok := resolved[toolUseID]; ok {
			return true
		}
	}
	if allowResultPayloadAsResolved && toolResultByID != nil {
		raw, ok := toolResultByID[toolUseID]
		if ok && len(raw) > 0 {
			return true
		}
	}
	return false
}

// toolUseSummaryLineResolvedForDisplay is true when every merged tool_use id in a SegToolUseSummaryLine has a result (or resolved map entry).
func toolUseSummaryLineResolvedForDisplay(resolved map[string]struct{}, toolResultByID map[string]json.RawMessage, seg messagerow.Segment, allowResultPayloadAsResolved bool) bool {
	ids := seg.ToolUseIDs
	if len(ids) == 0 {
		return toolUseResolvedForDisplay(resolved, toolResultByID, seg.ToolUseID, allowResultPayloadAsResolved)
	}
	for _, id := range ids {
		if !toolUseResolvedForDisplay(resolved, toolResultByID, id, allowResultPayloadAsResolved) {
			return false
		}
	}
	return true
}

// segmentJoinSeparator inserts an extra blank line after assistant prose before a merged Grep/Glob/Read summary line.
func segmentJoinSeparator(prev, cur messagerow.Segment) string {
	if prev.Kind == messagerow.SegTextMarkdown && strings.TrimSpace(prev.Text) != "" && cur.Kind == messagerow.SegToolUseSummaryLine {
		return "\n\n"
	}
	return "\n"
}

// transcriptAssistantPairBlankLine is true when the UI inserts one empty line between consecutive
// assistant rows in transcript (breathing room before the next ⏺ block).
func transcriptAssistantPairBlankLine(m *model, a, b types.Message) bool {
	if m == nil || m.uiScreen != gouDemoScreenTranscript {
		return false
	}
	return a.Type == types.MessageTypeAssistant && b.Type == types.MessageTypeAssistant
}

// priorNonEmptyAssistantText reports whether any earlier segment is non-empty assistant markdown.
// One ⏺/● marks the start of the assistant "paragraph"; tool title lines after that omit the lead glyph.
func priorNonEmptyAssistantText(segs []messagerow.Segment, idx int) bool {
	for j := 0; j < idx && j < len(segs); j++ {
		if segs[j].Kind == messagerow.SegTextMarkdown && strings.TrimSpace(segs[j].Text) != "" {
			return true
		}
	}
	return false
}

// baseMsgStyle adds the user-message row background so nested lipgloss.Render calls do not reset ANSI and punch holes in the gray bar.
func baseMsgStyle(userRow bool) lipgloss.Style {
	s := lipgloss.NewStyle()
	if userRow {
		s = s.Background(theme.UserMessageBackground())
	}
	return s
}

func logg(kind string, r string) {
	//return
	diaglog.Line("[goc/formatMessageSegments] seg kind=%s out=%s", kind, r)
}

// formatMessageSegments mirrors Message.tsx per-block branches (text→markdown, tool_use/tool_result/thinking).
// assistantLeadGlyph prefixes the first non-empty assistant text segment (TS-style ⏺ before the opening sentence).
// searchHL applies transcript search highlight to visible plain substrings (TS useSearchHighlight).
// showResolvedToolStats enables ⎿ TranscriptResolvedHintExtra for resolved Search/Read when tool_result JSON is available (prompt + transcript).
// userRow: when true, all lipgloss spans use the same row background as styleUserMessageLines (user-authored rows).
func formatMessageSegments(segs []messagerow.Segment, cols int, toolUseCtrlOHint bool, resolved map[string]struct{}, assistantLeadGlyph bool, searchHL string, toolResultByID map[string]json.RawMessage, showResolvedToolStats bool, userRow bool) string {
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
			md := styleMarkdownTokens(markdown.CachedLexer(textForMd), cols, userRow)
			if assistantLeadGlyph && !assistantTextLeadDone && strings.TrimSpace(seg.Text) != "" {
				assistantTextLeadDone = true
				md = prefixToolGlyphFirstLine(md)
			}
			piece = md
			logg("SegTextMarkdown", piece)
		case messagerow.SegToolUse:
			if seg.ToolFacing != "" {
				row1 := ""
				if !priorNonEmptyAssistantText(segs, i) {
					row1 = toolRowLeadPrefix(userRow)
				}
				row1 += baseMsgStyle(userRow).Foreground(theme.ToolUseAccent()).Bold(true).Render(withHL(seg.ToolFacing))
				if p := strings.TrimSpace(seg.ToolParen); p != "" {
					row1 += " (" + withHL(p) + ")"
				}
				var toolLines []string
				toolLines = append(toolLines, row1)
				res := toolUseResolvedForDisplay(resolved, toolResultByID, seg.ToolUseID, showResolvedToolStats)
				if showResolvedToolStats && res {
					var raw json.RawMessage
					if toolResultByID != nil {
						raw = toolResultByID[seg.ToolUseID]
					}
					hint, extra := messagerow.TranscriptResolvedHintExtra(seg.ToolFacing, raw)
					if hint != "" {
						toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(hint))))
						if extra != "" {
							toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("     "+textutil.LinkifyOSC8(withHL(extra))))
						}
					}
				} else if !res {
					if act := strings.TrimSpace(seg.Text); act != "" {
						actLine := baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(withHL(act) + "…")
						if toolUseCtrlOHint {
							actLine += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
						}
						toolLines = append(toolLines, actLine)
					}
					if h := strings.TrimSpace(seg.ToolHint); h != "" {
						toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(h))))
					}
				}
				piece = strings.Join(toolLines, "\n")
			} else {
				line := baseMsgStyle(userRow).Foreground(theme.ToolUseAccent()).Bold(true).Render("⚙ " + withHL(seg.Text))
				if toolUseCtrlOHint {
					line += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
				}
				piece = line
			}
			logg("SegToolUse", piece)
		case messagerow.SegToolResult:
			piece = segdiff.FormatToolResultSegmentForTranscript(seg, userRow, toolUseCtrlOHint, cols, withHL, baseMsgStyle)
			logg("SegToolResult", piece)
		case messagerow.SegThinking:
			body := textutil.LinkifyOSC8(seg.Text)
			piece = baseMsgStyle(userRow).Bold(true).Render("● " + withHL(body))
			logg("SegThinking", piece)
		case messagerow.SegDisplayHint:
			piece = baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
			logg("SegDisplayHint", piece)
		case messagerow.SegServerToolUse:
			if seg.ToolFacing != "" {
				row1 := ""
				if !priorNonEmptyAssistantText(segs, i) {
					row1 = toolRowLeadPrefix(userRow)
				}
				row1 += baseMsgStyle(userRow).Foreground(theme.ServerAccent()).Bold(true).Render(withHL(seg.ToolFacing))
				if p := strings.TrimSpace(seg.ToolParen); p != "" {
					row1 += " (" + withHL(p) + ")"
				}
				var toolLines []string
				toolLines = append(toolLines, row1)
				res := toolUseResolvedForDisplay(resolved, toolResultByID, seg.ToolUseID, showResolvedToolStats)
				if showResolvedToolStats && res {
					var raw json.RawMessage
					if toolResultByID != nil {
						raw = toolResultByID[seg.ToolUseID]
					}
					hint, extra := messagerow.TranscriptResolvedHintExtra(seg.ToolFacing, raw)
					if hint != "" {
						toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(hint))))
						if extra != "" {
							toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("     "+textutil.LinkifyOSC8(withHL(extra))))
						}
					}
				} else if !res {
					if act := strings.TrimSpace(seg.Text); act != "" {
						actLine := baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(withHL(act) + "…")
						if toolUseCtrlOHint {
							actLine += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
						}
						toolLines = append(toolLines, actLine)
					}
					if h := strings.TrimSpace(seg.ToolHint); h != "" {
						toolLines = append(toolLines, baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render("  ⎿  "+textutil.LinkifyOSC8(withHL(h))))
					}
				}
				piece = strings.Join(toolLines, "\n")
			} else {
				line := baseMsgStyle(userRow).Foreground(theme.ServerAccent()).Bold(true).Render("⎈ " + withHL(seg.Text))
				if toolUseCtrlOHint {
					line += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
				}
				piece = line
			}
			logg("SegServerToolUse", piece)
		case messagerow.SegAdvisorToolResult:
			st := baseMsgStyle(userRow).Foreground(theme.AdvisorAccent())
			if seg.IsToolError {
				st = baseMsgStyle(userRow).Foreground(theme.ToolError())
			}
			body := textutil.LinkifyOSC8(seg.Text)
			line := st.Render("✧ " + withHL(body))
			if seg.ToolBodyOmitted && toolUseCtrlOHint {
				line += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
			}
			piece = line
			logg("SegAdvisorToolResult", piece)
		case messagerow.SegGroupedToolUse:
			piece = baseMsgStyle(userRow).Foreground(theme.GroupedAccent()).Bold(true).Render("▦ " + withHL(seg.Text))
			logg("SegGroupedToolUse", piece)
		case messagerow.SegCollapsedReadSearch:
			piece = baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
			logg("SegCollapsedReadSearch", piece)
		case messagerow.SegToolUseSummaryLine:
			line := baseMsgStyle(userRow).Foreground(theme.DimMuted()).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
			if !toolUseSummaryLineResolvedForDisplay(resolved, toolResultByID, seg, showResolvedToolStats) && toolUseCtrlOHint {
				line += baseMsgStyle(userRow).Faint(true).Render(" (ctrl+o to expand)")
			}
			piece = "  " + line
			logg("SegToolUseSummaryLine", piece)
		case messagerow.SegSkillListingAvailable:
			n := seg.Num
			if n < 1 {
				n = 1
			}
			word := "skills"
			if n == 1 {
				word = "skill"
			}
			piece = baseMsgStyle(userRow).Bold(true).Render(strconv.Itoa(n)) + baseMsgStyle(userRow).Render(" "+word+" available")
			logg("SegSkillListingAvailable", piece)
		default:
			piece = baseMsgStyle(userRow).Faint(true).Render(textutil.LinkifyOSC8(withHL(seg.Text)))
			logg("default", piece)
		}
		if piece == "" {
			continue
		}
		if b.Len() > 0 && lastSegIdx >= 0 {
			b.WriteString(segmentJoinSeparator(segs[lastSegIdx], segs[i]))
		}
		b.WriteString(piece)
		lastSegIdx = i
	}
	return strings.TrimSpace(b.String())
}

// styleMarkdownInlineSegments renders paragraph/list_item runs with TS-style inline `code` color
// and strong/emphasis (terminal lipgloss).
func styleMarkdownInlineSegments(segs []markdown.InlineSegment, linePrefix string, userRow bool) string {
	if len(segs) == 0 {
		return ""
	}
	stCode := baseMsgStyle(userRow).Foreground(theme.MarkdownInlineCode())
	var stPlain, stBold, stItalic, stBoldItalic lipgloss.Style
	if userRow {
		ut := theme.UserMessageText()
		stPlain = baseMsgStyle(userRow).Foreground(ut).Bold(true)
		stBold = baseMsgStyle(userRow).Foreground(ut).Bold(true)
		stItalic = baseMsgStyle(userRow).Foreground(ut).Italic(true)
		stBoldItalic = baseMsgStyle(userRow).Foreground(ut).Bold(true).Italic(true)
	} else {
		stPlain = baseMsgStyle(userRow)
		stBold = baseMsgStyle(userRow).Bold(true)
		stItalic = baseMsgStyle(userRow).Italic(true)
		stBoldItalic = baseMsgStyle(userRow).Bold(true).Italic(true)
	}
	var b strings.Builder
	for i, seg := range segs {
		txt := seg.Text
		if i == 0 && linePrefix != "" {
			txt = linePrefix + txt
		}
		if seg.Code {
			b.WriteString(stCode.Render(txt))
			continue
		}
		var st lipgloss.Style
		switch {
		case seg.Bold && seg.Italic:
			st = stBoldItalic
		case seg.Bold:
			st = stBold
		case seg.Italic:
			st = stItalic
		default:
			st = stPlain
		}
		b.WriteString(st.Render(txt))
	}
	return b.String()
}

// headingMarkdownStyle is bold + heading color; level spacing is leading spaces only (no # in output).
func headingMarkdownStyle(userRow bool) lipgloss.Style {
	return baseMsgStyle(userRow).Bold(true).Foreground(theme.MarkdownHeading())
}

// styleMarkdownTokens applies lipgloss to block tokens (mirrors Markdown.tsx roles, terminal-only).
func styleMarkdownTokens(toks []markdown.Token, cols int, userRow bool) string {
	if len(toks) == 0 {
		return ""
	}
	var parts []string
	for _, t := range toks {
		switch t.Type {
		case "heading":
			lv := min(max(t.Level, 1), 6)
			levelPad := strings.Repeat(" ", (lv-1)*2)
			hst := headingMarkdownStyle(userRow)
			if len(t.Segments) > 0 {
				inner := styleMarkdownInlineSegments(t.Segments, "", userRow)
				rendered := hst.Render(inner)
				parts = append(parts, wrapHeadingForMessagePane(rendered, levelPad, cols))
			} else {
				plain := strings.TrimSpace(t.Text)
				wrapped := wrapHeadingForMessagePane(plain, levelPad, cols)
				lines := strings.Split(wrapped, "\n")
				var hb strings.Builder
				for i, ln := range lines {
					if i > 0 {
						hb.WriteByte('\n')
					}
					hb.WriteString(hst.Render(ln))
				}
				parts = append(parts, hb.String())
			}
		case "code":
			// Apply syntax highlighting if highlighter is available
			var highlightedCode string
			if markdownHighlighter != nil {
				highlighted, err := markdownHighlighter.HighlightCode(t.Text, t.Lang)
				if err == nil && highlighted != "" {
					highlightedCode = highlighted
				}
			}

			// If highlighting failed or highlighter is disabled, use plain code
			if highlightedCode == "" {
				cb := "```" + t.Lang + "\n" + t.Text
				if t.Text != "" && !strings.HasSuffix(t.Text, "\n") {
					cb += "\n"
				}
				cb += "```"
				parts = append(parts, baseMsgStyle(userRow).Faint(true).Render(cb))
			} else {
				// For highlighted code, just show the highlighted content without backticks
				parts = append(parts, baseMsgStyle(userRow).Render(highlightedCode))
			}
		case "list_item":
			indent := strings.Repeat(" ", t.ListIndent)
			var prefix string
			if t.ListContinuation {
				prefix = indent + "   "
			} else if t.ListOrdered && t.ListIndex > 0 {
				prefix = indent + fmt.Sprintf("%d. ", t.ListIndex)
			} else {
				prefix = indent + "- "
			}
			if len(t.Segments) > 0 {
				parts = append(parts, styleMarkdownInlineSegments(t.Segments, prefix, userRow))
			} else if userRow {
				parts = append(parts, baseMsgStyle(userRow).Foreground(theme.UserMessageText()).Bold(true).Render(prefix+t.Text))
			} else {
				parts = append(parts, baseMsgStyle(userRow).Render(prefix+t.Text))
			}
		case "blockquote":
			if len(t.Segments) > 0 {
				inner := styleMarkdownInlineSegments(t.Segments, "", userRow)
				pref := "> " + strings.ReplaceAll(inner, "\n", "\n> ")
				parts = append(parts, pref)
			} else if userRow {
				parts = append(parts, baseMsgStyle(userRow).Foreground(theme.UserMessageText()).Italic(true).Bold(true).Render("> "+strings.ReplaceAll(t.Text, "\n", "\n> ")))
			} else {
				parts = append(parts, baseMsgStyle(userRow).Italic(true).Render("> "+strings.ReplaceAll(t.Text, "\n", "\n> ")))
			}
		case "hr":
			parts = append(parts, baseMsgStyle(userRow).Faint(true).Render("---"))
		case "paragraph":
			if len(t.Segments) > 0 {
				parts = append(parts, styleMarkdownInlineSegments(t.Segments, "", userRow))
			} else {
				if userRow {
					parts = append(parts, baseMsgStyle(userRow).Foreground(theme.UserMessageText()).Bold(true).Render(t.Text))
				} else {
					parts = append(parts, t.Text)
				}
			}
		default:
			if userRow {
				parts = append(parts, baseMsgStyle(userRow).Foreground(theme.UserMessageText()).Bold(true).Render(t.Text))
			} else {
				parts = append(parts, t.Text)
			}
		}
	}
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			if toks[i-1].Type == "list_item" {
				b.WriteByte('\n')
			} else {
				b.WriteString("\n\n")
			}
		}
		b.WriteString(part)
	}
	return strings.TrimSpace(b.String())
}

// handleTraditionalScrollKey handles scroll keys when not using viewport (traditional virtual scrolling).
func (m *model) handleTraditionalScrollKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "down":
		m.sticky = false
		m.scrollTop += 1
		return nil
	case "up":
		m.sticky = false
		m.scrollTop = max(0, m.scrollTop-1)
		return nil
	case "pgdown", "space":
		m.sticky = false
		m.scrollTop += listViewportH(m) / 2
		return nil
	case "pgup", "b":
		m.sticky = false
		m.scrollTop = max(0, m.scrollTop-listViewportH(m)/2)
		return nil
	case "end", "G", "shift+g", "ctrl+end":
		m.sticky = true
		m.scrollTop = 1 << 30
		return nil
	case "home", "ctrl+home":
		m.sticky = false
		m.scrollTop = 0
		return nil
	case "ctrl+u":
		m.sticky = false
		m.scrollTop = max(0, m.scrollTop-listViewportH(m)/2)
		return nil
	case "ctrl+d":
		m.sticky = false
		m.scrollTop += listViewportH(m) / 2
		return nil
	case "ctrl+b":
		m.sticky = false
		m.scrollTop = max(0, m.scrollTop-listViewportH(m))
		return nil
	case "ctrl+f":
		m.sticky = false
		m.scrollTop += listViewportH(m)
		return nil
	case "ctrl+n":
		m.sticky = false
		m.scrollTop += 1
		return nil
	case "ctrl+p":
		m.sticky = false
		m.scrollTop = max(0, m.scrollTop-1)
		return nil
	}
	return nil
}
func (m *model) gouSubmitFromPromptText(fullPrompt, line string) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	gouDemoTracef("enter input=%q", previewForTrace(line, 120))
	cwd, _ := os.Getwd()
	toolProjectRoot := resolveToolProjectRoot(cwd)
	mergedLang, mergedOutName, mergedOutPrompt := gouDemoMergedSystemLocale()
	preExp := fullPrompt
	demoCfg := pui.DemoConfig{
		SessionID:           m.store.ConversationID,
		Language:            mergedLang,
		MCPCommandsJSONPath: m.mcpCommandsJSONPath,
		MCPToolsJSONPath:    m.mcpToolsJSONPath,
		PreExpansionInput:   &preExp,
		PermissionMode:      &m.permissionMode,
	}
	if m.tsBridge != nil {
		demoCfg.TSContextBridge = m.tsBridge
	}
	params, err := pui.BuildDemoParams(line, m.store, demoCfg)
	if err != nil {
		gouDemoTracef("BuildDemoParams error: %v", err)
		m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: build params: %v", err)))
		m.rebuildHeightCache()
		m.sticky = true
		m.scrollTop = 1 << 30
		return m, cmd
	}
	if params.RuntimeContext != nil {
		gouDemoLogToolUseContext(params.RuntimeContext)
	}
	params.ProcessSlashCommand = pui.NewSlashResolveProcessSlashCommand(pui.SlashResolveHandlerOptions{
		SessionID:        m.store.ConversationID,
		Store:            m.store,
		ReadFileState:    m.readFileState,
		Cwd:              cwd,
		SessionMemState:  m.sessionMemState,
		GuidancePtr:      &m.lastGuidance,
		UserContextPtr:   &m.lastUserCtx,
		SystemContextPtr: &m.lastSystemCtx,
		SendMsg: func(msg any) {
			m.ccbSend(msg)
		},
	})
	gouDemoTracef("ProcessUserInput start priorMsgs=%d", len(m.store.Messages))
	r, err := processuserinput.ProcessUserInput(context.Background(), params)
	gouDemoTracef("ProcessUserInput end err=%v", err)
	if err != nil {
		m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("processUserInput: %v", err)))
		m.rebuildHeightCache()
		m.sticky = true
		m.scrollTop = 1 << 30
		return m, cmd
	}
	rStore := r
	if r != nil && strings.HasPrefix(line, "/") && r.Execution == nil && !r.ShouldQuery &&
		extractSlashLocalPanelText(r) != "" {
		rStore = slashResultForStoreOmittingPanelDupes(r)
	}
	out := pui.ApplyBaseResult(m.store, rStore, &m.processUserInputBaseResultHandoff)
	gouDemoTracef("after ApplyBaseResult shouldQuery=%v effectiveShouldQuery=%v hadExecutionRequest=%v messagesAppended=%d",
		r != nil && r.ShouldQuery, out.EffectiveShouldQuery, out.HadExecutionRequest, len(rStore.Messages))
	if out.NextInput != "" {
		m.pr.SetValue(out.NextInput)
		m.syncSlashListAfterPrompt()
	}
	m.applySlashResultPanelFromSubmit(line, r, out)
	m.rebuildHeightCache()
	m.sticky = true
	m.scrollTop = 1 << 30
	// Flush user (and any other new rows) before OnQueryYield appends streaming assistant/tool lines so JSONL follows conversation time order.
	m.maybeRecordTranscript()
	if out.EffectiveShouldQuery && !out.HadExecutionRequest {
		usedCCB := false
		var normToolsJSON json.RawMessage
		if params.RuntimeContext != nil {
			normToolsJSON = params.RuntimeContext.ToolUseContext.Options.Tools
		}
		var normToolDefs []struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(normToolsJSON, &normToolDefs)
		toolSpecs := make([]messagesapi.ToolSpec, 0, len(normToolDefs))
		for _, t := range normToolDefs {
			toolSpecs = append(toolSpecs, messagesapi.ToolSpec{Name: t.Name})
		}
		normOpts := messagesapi.OptionsFromEnv()
		if gouDemoEnvTruthy("GOU_DEMO_NON_INTERACTIVE") {
			normOpts.NonInteractive = true
		}
		tryMsgs := func() (json.RawMessage, error) {
			return ccbhydrate.MessagesJSONNormalized(m.store.Messages, toolSpecs, normOpts)
		}
		if m.ccbInline && m.ccbSend != nil {
			baseMsgs, err := tryMsgs()
			if err != nil {
				gouDemoTracef("gou-demo: ccbhydrate.MessagesJSON error: %v", err)
				m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: ccb messages JSON: %v", err)))
				m.rebuildHeightCache()
			} else if len(bytes.TrimSpace(baseMsgs)) < 3 || bytes.Equal(bytes.TrimSpace(baseMsgs), []byte("[]")) {
				gouDemoTracef("gou-demo: empty messages JSON bytes=%d", len(baseMsgs))
				m.store.AppendMessage(pui.SystemNotice("gou-demo: empty chat transcript (cannot call model)"))
				m.rebuildHeightCache()
			} else {
				var toolsJSON json.RawMessage
				if params.RuntimeContext != nil {
					toolsJSON = params.RuntimeContext.ToolUseContext.Options.Tools
				}
				var toolDefs []struct {
					Name string `json:"name"`
				}
				_ = json.Unmarshal(toolsJSON, &toolDefs)
				names := make([]string, 0, len(toolDefs))
				for _, t := range toolDefs {
					names = append(names, t.Name)
				}
				hasSkillTool := false
				skillNm := skilltools.SkillToolName()
				for _, t := range toolDefs {
					if t.Name == skillNm {
						hasSkillTool = true
						break
					}
				}
				skillListing := params.SkillListingCommands
				if len(skillListing) == 0 {
					skillListing = commands.SkillToolCommands(params.Commands)
				}
				discoverNm := strings.TrimSpace(os.Getenv("CLAUDE_CODE_DISCOVER_SKILLS_TOOL_NAME"))
				mainLoopModel := gouDemoQueryMainLoopModel(params)
				m.lastMainLoopModel = mainLoopModel
				gouOpts := commands.GouDemoSystemOpts{
					EnabledToolNames:       commands.EnabledToolNames(names),
					SkillToolCommands:      skillListing,
					ModelID:                mainLoopModel,
					Cwd:                    cwd,
					Language:               mergedLang,
					DiscoverSkillsToolName: discoverNm,
					NonInteractiveSession:  gouDemoEnvTruthy("GOU_DEMO_NON_INTERACTIVE"),
					OutputStyleName:        mergedOutName,
					OutputStylePrompt:      mergedOutPrompt,
				}
				commands.ApplyGouDemoRuntimeEnv(&gouOpts)
				var customSys, appendSys string
				if params.RuntimeContext != nil {
					if p := params.RuntimeContext.ToolUseContext.Options.CustomSystemPrompt; p != nil {
						customSys = strings.TrimSpace(*p)
					}
					if p := params.RuntimeContext.ToolUseContext.Options.AppendSystemPrompt; p != nil {
						appendSys = strings.TrimSpace(*p)
					}
				}
				extraRoots := querycontext.ExtraClaudeMdRootsForFetch(params.RuntimeContext)
				fetchOpts := querycontext.FetchOpts{
					CustomSystemPrompt:  customSys,
					Gou:                 gouOpts,
					ExtraClaudeMdRoots:  extraRoots,
					SessionStartSource:  "startup",
					HooksSessionID:      m.store.ConversationID,
					HooksTranscriptPath: "",
				}
				if m.tsBridge != nil {
					fetchOpts.TSSnapshot = m.tsBridge
				}
				partsRes, errParts := querycontext.FetchSystemPromptParts(context.Background(), fetchOpts)
				var guidance string
				var userCtxReminder string
				if errParts != nil {
					gouDemoTracef("FetchSystemPromptParts: %v (fallback BuildGouDemoSystemPrompt)", errParts)
					m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: system context: %v (using base prompt only)", errParts)))
					m.rebuildHeightCache()
					guidance = commands.BuildGouDemoSystemPrompt(gouOpts)
				} else {
					userCtxReminder = querycontext.FormatUserContextReminder(partsRes.UserContext)
					var base []string
					if customSys != "" {
						base = []string{customSys}
					} else {
						base = slices.Clone(partsRes.DefaultSystemPrompt)
					}
					if appendSys != "" {
						base = append(base, appendSys)
					}
					guidance = strings.Join(base, "\n\n")
				}
				m.lastGuidance = guidance

				listing := ""
				var listingMeta *ccbhydrate.SkillListingMeta
				if !gouDemoEnvTruthy("GOU_DEMO_SKIP_SKILL_LISTING") {
					listingSent := m.skillListingSent
					if gouDemoEnvTruthy("GOU_DEMO_SKILL_LISTING_EVERY_TURN") {
						listingSent = make(map[string]struct{})
					}
					if s, n, initial, ok := commands.AppendSkillListingForAPI(skillListing, hasSkillTool, listingSent, nil); ok {
						listing = s
						listingMeta = &ccbhydrate.SkillListingMeta{SkillCount: n, IsInitial: initial}
					}
				}
				msgsJSON, errL := ccbhydrate.MessagesJSONWithLeadingMeta(m.store.Messages, userCtxReminder, listing, listingMeta, toolSpecs, normOpts)
				if errL != nil {
					gouDemoTracef("gou-demo: MessagesJSONWithLeadingMeta error: %v", errL)
					m.store.AppendMessage(pui.SystemNotice(fmt.Sprintf("gou-demo: skill listing hydrate: %v", errL)))
					m.rebuildHeightCache()
				} else {
					// When dynamic tool loading is active, prepend <available-deferred-tools>
					// so the model knows which tools to discover via ToolSearch.
					msgsBefore := len(msgsJSON)
					if prep := toolsearchwire.PrepareMessagesForWire(msgsJSON, toolsJSON, mainLoopModel, false, false, m.store.Messages); len(prep) > 0 {
						msgsJSON = prep
					}
					// Persist announcement in store for delta tracking across turns.
					if len(msgsJSON) > msgsBefore {
						toolsearchwire.PersistDeferredAnnouncement(m.store, toolsJSON)
					}
					reqID := fmt.Sprintf("turn-%d", time.Now().UnixNano())
					m.store.ClearStreaming()
					m.store.ClearStreamingToolUses()
					// TS: skill_listing attachment is pushed to mutableMessages before callModel (QueryEngine attachment case).
					if strings.TrimSpace(listing) != "" {
						if att, ok := ccbhydrate.SkillListingStoreMessage(listing, listingMeta); ok {
							m.store.AppendMessage(att)
							m.rebuildHeightCache()
						}
					}
					gouDemoTracef("gou-demo model turn start requestID=%s msgsJSONBytes=%d toolsBytes=%d systemBytes=%d",
						reqID, len(msgsJSON), len(toolsJSON), len(guidance))
					cwdAbs, errAbs := filepath.Abs(cwd)
					if errAbs != nil {
						cwdAbs = cwd
					}
					runner := skilltools.ParityToolRunner{
						DemoToolRunner: skilltools.DemoToolRunner{
							Commands:  params.Commands,
							SessionID: m.store.ConversationID,
						},
						WorkDir:          cwdAbs,
						ProjectRoot:      toolProjectRoot,
						ReadFileState:    m.readFileState,
						LocalBashDefault: true,
						AskAutoFirst:     !gouDemoEnvTruthy("GOU_DEMO_NO_ASK_AUTO_FIRST"),
						MainLoopModel:    mainLoopModel,
						Messages:         m.store.Messages,
						MessagesFunc:     func() []types.Message { return m.store.Messages },
						SystemPrompt:     []string{guidance},
						NotificationCallback: func(agentID, toolUseID, outputFile, status, summary, output string, tokenCount, toolUseCount int, durationMs int64) {
							commandqueue.EnqueueAgentNotification(agentID, toolUseID, outputFile, status, summary, output, tokenCount, toolUseCount, durationMs)
							// Notify TUI to check for idle auto-submit
							if send := m.ccbSend; send != nil {
								send(commandQueueNotifyMsg{})
							}
						},
						ProgressCallback: func(msg *types.Message) {
							if msg == nil {
								return
							}
							// Parse agent lifecycle events from progress messages
							if msg.Type == types.MessageTypeProgress && len(msg.Data) > 0 {
								var data struct {
									Type             string   `json:"type"`
									AgentID          string   `json:"agentId"`
									AgentType        string   `json:"agentType"`
									Description      string   `json:"description"`
									Name             string   `json:"name"`
									IsBackground     bool     `json:"isBackground"`
									ParentToolUseID  string   `json:"parentToolUseID"`
									TokenCount       int      `json:"tokenCount"`
									ToolUseCount     int      `json:"toolUseCount"`
									LastActivityDesc string   `json:"lastActivityDesc"`
									Summary          string   `json:"summary"`
									RecentActivities []string `json:"recentActivities"`
									Status           string   `json:"status"`
								}
								if json.Unmarshal(msg.Data, &data) == nil {
									switch data.Type {
									case "agent_registered":
										evictAfter := time.Now().Add(24 * time.Hour)
										task := &AgentTaskState{
											ID:              data.AgentID,
											AgentType:       data.AgentType,
											Description:     data.Description,
											Name:            data.Name,
											Status:          "running",
											StartTime:       time.Now(),
											IsBackground:    data.IsBackground,
											ParentToolUseID: data.ParentToolUseID,
											EvictAfter:      &evictAfter,
										}
										if send := m.ccbSend; send != nil {
											send(AgentRegisteredMsg{Task: task})
										}
										// Still forward for message pane rendering
										if send := m.ccbSend; send != nil {
											send(gouQueryYieldMsg{Message: *msg})
										}
										return
									case "agent_summary":
										if send := m.ccbSend; send != nil {
											send(AgentProgressMsg{
												AgentID: data.AgentID,
												Progress: &AgentTaskProgress{
													Summary:          data.Summary,
													TokenCount:       data.TokenCount,
													ToolUseCount:     data.ToolUseCount,
													LastActivityDesc: data.LastActivityDesc,
													RecentActivities: data.RecentActivities,
												},
											})
										}
										return
									case "agent_completed":
										if send := m.ccbSend; send != nil {
											send(AgentCompletedMsg{AgentID: data.AgentID, Status: "completed"})
										}
										return
									}
								}
							}
							// Default: forward as yield message for the message pane
							if send := m.ccbSend; send != nil {
								send(gouQueryYieldMsg{Message: *msg})
							}
						},
						EditDeps: &localtools.EditDeps{
							OnNotebookEdit: func(absPath, oldString, newString string, replaceAll bool, roots []string, state *localtools.ReadFileState, userModified bool) (string, bool, error) {
								return tools.NotebookEditFromEdit(absPath, oldString, newString, replaceAll, roots)
							},
						},
					}
					if params.RuntimeContext != nil && params.RuntimeContext.ToolPermissionContext != nil {
						runner.ToolPermission = params.RuntimeContext.ToolPermissionContext
					}
					if gouDemoPreferQueryStreamingParity() {
						var userCtx map[string]string
						var systemCtx map[string]string
						if errParts == nil {
							userCtx = gouDemoUserContextMapForQuery(partsRes.UserContext)
							systemCtx = partsRes.SystemContext
						}
						m.lastUserCtx = userCtx
						m.lastSystemCtx = systemCtx
						tcx := types.ToolUseContext{}
						if params.RuntimeContext != nil {
							tcx = params.RuntimeContext.ToolUseContext
						}
						tcx.Options.MainLoopModel = mainLoopModel
						if m.toolResultState != nil {
							tcx.ContentReplacementState = m.toolResultState.ToJSON()
						}
						var trySMCompact compactservice.TrySessionMemoryCompactFn
						if m.sessionMemState != nil {
							sessionID := m.store.ConversationID
							trySMCompact = func(ctx context.Context, messages []types.Message, agentID string, autoCompactThreshold *int) (*compactservice.CompactionResult, error) {
								return sessionmemory.TrySessionMemoryCompaction(
									ctx,
									m.sessionMemState,
									sessionID,
									cwd,
									messages,
									"", // transcriptPath
									autoCompactThreshold,
									func(ctx context.Context, trigger string, model string) ([]types.Message, error) {
										runner := hookexec.SessionStartHookRunner(toolProjectRoot, cwd, sessionID, "")
										res, err := runner(ctx, compactservice.SessionStartHookTrigger(trigger),
											compactservice.SessionStartHookInput{Model: model})
										if err != nil {
											return nil, err
										}
										msgs := make([]types.Message, len(res))
										for i, r := range res {
											msgs[i] = r
										}
										return msgs, nil
									},
									agentID,
									mainLoopModel,
									nil, // planAttachmentProvider
								)
							}
						}
						qdeps := query.ProductionDeps(trySMCompact, func(phase string) {
							if send := m.ccbSend; send != nil {
								send(compactPhaseMsg{Phase: phase})
							}
						})
						te := toolexecution.ExecutionDeps{
							InvokeTool:              runner.Run,
							MainLoopModel:           mainLoopModel,
							ReadToolRoots:           runner.ToolReadMappingRoots(),
							ReadToolMemCWD:          runner.ToolReadMappingMemCWD(),
							MultiMessageToolHandler: skilltools.NewSkillMultiMessageHandler(params.Commands, m.store.ConversationID, nil),
							QueryCanUseTool: func(ctx context.Context, toolName, toolUseID string, input json.RawMessage) (toolexecution.PermissionDecision, error) {
								if toolName == "AskUserQuestion" {
									return toolexecution.AskDecision("Answer questions?"), nil
								}
								return toolexecution.AllowDecision(), nil
							},
						}
						// Opt-in TS permissions.ts 1b: whole-tool alwaysAsk on Bash skipped when input looks sandboxed (see toolexecution.BashSandboxRule1b).
						if gouDemoEnvTruthy("GOU_TOOLEXEC_BASH_SANDBOX_1B") {
							te.SandboxingEnabled = true
							te.AutoAllowBashWholeToolAskWhenSandboxed = true
						}
						// Tool result persistence: enabled by default (mirrors TS), set GOU_DEMO_TOOL_RESULT_PERSIST=0 to disable.
						// Large tool results are saved to disk and replaced with a preview in the tool_result block.
						if !gouDemoEnvFalsy("GOU_DEMO_TOOL_RESULT_PERSIST") {
							te.ToolResultPersistConfig = &toolexecution.ToolResultPersistConfig{
								SessionInfo: toolresultpersist.SessionInfo{
									SessionID: m.store.ConversationID,
									Cwd:       cwd,
								},
								ProcessOptions:          toolresultpersist.DefaultProcessOptions(),
								ContentReplacementState: m.toolResultState,
							}
						}
						m.askAutoFirst = !gouDemoEnvTruthy("GOU_DEMO_NO_ASK_AUTO_FIRST")
						m.installAskResolver(&te, m.askAutoFirst)
						qdeps.ToolexecutionDeps = te
						// Wire tool result budget enforcement when persistence is enabled.
						// The closure captures the live Go *ContentReplacementState so mutations
						// survive across turns (mirrors TS shared ContentReplacementState instance).
						if m.toolResultState != nil {
							statePtr := m.toolResultState
							sessionInfo := te.ToolResultPersistConfig.SessionInfo
							qdeps.ApplyToolResultBudget = func(ctx context.Context, in *query.ToolResultBudgetInput) ([]types.Message, error) {
								return toolresultpersist.ApplyToolResultBudget(
									in.Messages,
									statePtr,
									sessionInfo,
									0,   // use default MaxToolResultsPerMessageChars
									nil, // skipToolNames
								), nil
							}
						}
						// Snapshot matches qp.Messages (TS QueryEngine messages at callModel): includes skill_listing row if appended above.
						msgsForQ := slices.Clone(m.store.Messages)
						// Prepend SessionStart hook_additional_context attachments (e.g. superpowers using-superpowers skill).
						// These are ephemeral — not persisted to store, only injected into the current query messages.
						if partsRes.SessionStartHookMessages != nil {
							msgsForQ = append(slices.Clone(partsRes.SessionStartHookMessages), msgsForQ...)
						}
						// Inject pending background agent notifications as user messages
						// so the model sees them when the next query starts.
						if commandqueue.HasPendingNotifications() {
							for _, cmd := range commandqueue.DrainCommandQueue() {
								content, _ := json.Marshal([]map[string]any{{
									"type": "text",
									"text": "A background agent completed a task:\n" + cmd.Value,
								}})
								msgsForQ = append(msgsForQ, types.Message{
									Type:    "user",
									Content: content,
								})
							}
						}
						if send := m.ccbSend; send != nil {
							qdeps.OnStreamingToolUses = func(ctx context.Context, uses []query.StreamingToolUseLive) error {
								send(gouStreamingToolUsesMsg{Uses: uses})
								return nil
							}
						}
						if m.transcript != nil {
							tr := m.transcript
							// Mirror TS recordTranscript(messages): each yield appends to the same turn prefix so
							// sessiontranscript dedup sees already-recorded user (and prior yields) before new rows.
							turnPrefix := slices.Clone(msgsForQ)
							qdeps.OnQueryYield = func(ctx context.Context, y query.QueryYield) error {
								if y.Message == nil {
									return nil
								}
								turnPrefix = append(turnPrefix, *y.Message)
								_, err := tr.RecordTranscript(ctx, turnPrefix, sessiontranscript.RecordOpts{AllMessages: turnPrefix})
								return err
							}
						}
						qdeps.OnQueryComplete = func(ctx context.Context, qcp query.QueryCompleteParams) {
							extractmemories.Execute(ctx, m.extractMemState, extractmemories.ExtractionParams{
								Messages:       qcp.Messages,
								ToolUseContext: qcp.ToolUseContext,
								SystemPrompt:   qcp.SystemPrompt,
								UserContext:    qcp.UserContext,
								SystemContext:  qcp.SystemContext,
								Cwd:            qcp.Cwd,
								QuerySource:    qcp.QuerySource,
								NewUUID:        query.RandomUUID,
								SkipIndex:      growthbook.IsTenguMothCopse(),
								AppendSystemMessage: func(msg types.Message) {
									if send := m.ccbSend; send != nil {
										send(gouMemoryAppendMsg{Msg: msg})
									}
								},
							})
							_, dreamErr := autodream.Execute(ctx, m.autoDreamState,
								qcp.ToolUseContext, qcp.SystemPrompt,
								qcp.UserContext, qcp.SystemContext,
								qcp.QuerySource, query.RandomUUID,
								commands.ClaudeConfigHome(), qcp.Cwd,
								"", /* memoryDir — let Execute resolve */
								m.store.ConversationID,
							)
							if dreamErr != nil {
								gouDemoTracef("autodream: %v", dreamErr)
							}
							// Session memory extraction (TS sessionMemory post-turn hook).
							m.sessionMemHook(ctx, qcp)
						}
						qdeps.DrainCommandQueue = func() []string {
							var result []string
							for _, cmd := range commandqueue.DrainCommandQueue() {
								result = append(result, cmd.Value)
							}
							return result
						}
						qp := query.QueryParams{
							Messages:        msgsForQ,
							SystemPrompt:    query.AsSystemPrompt([]string{guidance}),
							UserContext:     userCtx,
							SystemContext:   systemCtx,
							ToolUseContext:  tcx,
							QuerySource:     params.QuerySource,
							StreamingParity: true,
							Deps:            &qdeps,
						}
						if params.RuntimeContext != nil && params.RuntimeContext.ToolPermissionContext != nil {
							pc := *params.RuntimeContext.ToolPermissionContext
							types.NormalizeToolPermissionContextData(&pc)
							qp.ToolPermissionContext = &pc
						}
						processuserinput.ApplyQueryHostEnvGates(&qp)
						processuserinput.WireToolexecutionFromProcessUserInput(&qp, params)
						gouDemoTracef("query streaming parity turn requestID=%s storeMsgs=%d toolsBytes=%d",
							reqID, len(m.store.Messages), len(toolsJSON))
						// Save params for auto-submit of bg agent notifications
						qpCopy := qp
						m.lastQueryParams = &qpCopy
						m.beginQuerySpinner()
						m.queryBusy = true
						m.store.ClearStreamingToolUses()
						ctx, cancel := context.WithCancel(context.Background())
						m.queryCancel = cancel
						runQueryStreamingParityTurn(ctx, m.ccbSend, qp)
						usedCCB = true
					} else {
						m.store.AppendMessage(pui.SystemNotice(
							"gou-demo: ccb-engine/localturn was removed. For a real model reply, set ANTHROPIC_API_KEY (or ANTHROPIC_AUTH_TOKEN) and GOU_QUERY_STREAMING_PARITY=1 or GOU_DEMO_STREAMING_TOOL_EXECUTION=1.",
						))
						m.rebuildHeightCache()
					}
				}
			}
		}
		if usedCCB {
			if cmd != nil {
				return m, tea.Batch(cmd, spinnerTickCmd())
			}
			return m, spinnerTickCmd()
		}
		if !m.ccbInline {
			m.store.AppendMessage(pui.SystemNotice(
				"gou-demo: real HTTP / streaming parity is disabled (GOU_DEMO_CCB_INLINE=0). Unset it and set ANTHROPIC_API_KEY (or ANTHROPIC_AUTH_TOKEN) with GOU_QUERY_STREAMING_PARITY=1 or GOU_DEMO_STREAMING_TOOL_EXECUTION=1 for a model reply.",
			))
			m.rebuildHeightCache()
			m.sticky = true
			m.scrollTop = 1 << 30
		}
		if cmd != nil {
			return m, cmd
		}
		return m, nil
	}
	gouDemoTracef("no query path (effectiveShouldQuery=%v hadExecutionRequest=%v)", out.EffectiveShouldQuery, out.HadExecutionRequest)
	return m, cmd
}
