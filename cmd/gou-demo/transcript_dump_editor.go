package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"goc/gou/layout"
	"goc/gou/messagerow"
	"goc/gou/messagesview"
	"goc/types"
)

// TS REPL.tsx: [ → dumpMode + showAll; plain dump to scrollback (Printf).
// v → renderMessagesToPlainText (strip trailing spaces per line), temp file, openFileInExternalEditor.

type gouTranscriptEditorPrepMsg struct {
	Gen    int
	Path   string
	Err    error
	Editor string
	Bin    string
	Args   []string
}

type gouTranscriptEditorExecDoneMsg struct {
	Gen int
	Err error
}

type gouTranscriptEditorClearStatusMsg struct {
	Gen int
}

var transcriptTrailingSpaceRE = regexp.MustCompile(`(?m)[ \t]+$`)

func stripTranscriptExportTrailingSpaces(s string) string {
	return transcriptTrailingSpaceRE.ReplaceAllString(s, "")
}

func exportTranscriptWidth(m *model) int {
	w := m.cols - 6
	if w < 80 {
		w = max(80, m.width-6)
	}
	if w < 80 {
		w = 80
	}
	return w
}

func parseExternalEditorFromEnv() (bin string, args []string, display string) {
	ed := strings.TrimSpace(os.Getenv("VISUAL"))
	if ed == "" {
		ed = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if ed == "" {
		return "", nil, ""
	}
	parts := strings.Fields(ed)
	if len(parts) == 0 {
		return "", nil, ed
	}
	return parts[0], parts[1:], ed
}

func transcriptExportPlain(m *model, wrapCols int) string {
	if wrapCols < 1 {
		wrapCols = 80
	}
	// Respect current screen mode for filters (prompt vs transcript).
	isInTranscript := m.uiScreen == gouDemoScreenTranscript
	showAll := m.transcriptShowAll || m.transcriptDumpMode

	raw := slices.Clone(m.store.Messages)
	// VirtualScrollEnabled must be true to avoid 30-message truncation in maybeTranscriptTail.
	msgView := messagesview.MessagesForScrollList(raw, messagesview.ScrollListOpts{
		TranscriptMode:       isInTranscript,
		ShowAllInTranscript:  showAll,
		VirtualScrollEnabled: true,
		ResolvedToolUseIDs:   m.resolvedToolIDs,
	})

	var b strings.Builder
	for i := range msgView {
		if i > 0 {
			b.WriteByte('\n')
		}
		msg := messagerow.NormalizeMessageJSON(msgView[i])
		if !isInTranscript && m.skipFoldedToolResultStubInPrompt(msg) {
			continue
		}

		b.WriteString(string(msg.Type))
		b.WriteByte('\n')
		b.WriteString(transcriptPlainBodyFromMessage(m, msg, wrapCols))
	}
	// ... streaming tools ...
	// Only export streaming tools if we are in transcript mode (matches UI).
	if isInTranscript {
		for _, group := range m.transcriptStreamingToolsForView() {
			if !group.IsGroup {
				tu := group.Single
				b.WriteByte('\n')
				b.WriteString(string(types.MessageTypeAssistant))
				b.WriteByte('\n')
				line := "⚙ " + tu.Name + " · streaming"
				if s := strings.TrimSpace(tu.UnparsedInput); s != "" {
					line += "\n" + s
				}
				b.WriteString(strings.TrimRight(layout.WrapForViewport(line, wrapCols), "\n"))
			} else {
				b.WriteByte('\n')
				b.WriteString(string(types.MessageTypeAssistant))
				b.WriteByte('\n')
				summary := messagerow.SearchReadSummaryText(true, group.SearchCount, group.ReadCount, group.ListCount, 0, 0, 0, 0, 0, nil, nil, nil)
				line := "⏺ " + summary + messagerow.CtrlOToExpandHint
				for _, item := range group.Items {
					path := extractPartialJSONField(item.UnparsedInput, "file_path")
					if path == "" {
						path = extractPartialJSONField(item.UnparsedInput, "path")
					}
					if path == "" {
						path = extractPartialJSONField(item.UnparsedInput, "pattern")
					}
					if path == "" {
						path = "..."
					}
					line += "\n  ⎿  " + path
				}
				b.WriteString(strings.TrimRight(layout.WrapForViewport(line, wrapCols), "\n"))
			}
		}
	}
	return stripTranscriptExportTrailingSpaces(b.String())
}

func transcriptPlainBodyFromMessage(m *model, msg types.Message, wrapCols int) string {
	opts := m.messagerowOpts(msg)
	segs := messagerow.SegmentsFromMessageOpts(msg, opts)

	// Use a simplified plain-text version of formatMessageSegments symbols.
	var parts []string
	assistantTextLeadDone := false
	for _, seg := range segs {
		var piece string
		switch seg.Kind {
		case messagerow.SegTextMarkdown:
			md := seg.Text
			if msg.Type == types.MessageTypeAssistant && !assistantTextLeadDone && strings.TrimSpace(seg.Text) != "" {
				assistantTextLeadDone = true
				glyph := "● "
				if runtime.GOOS == "darwin" {
					glyph = "⏺ "
				}
				md = glyph + md
			}
			piece = md
		case messagerow.SegToolUse:
			if seg.ToolFacing != "" {
				piece = "⚙ " + seg.ToolFacing
				if p := strings.TrimSpace(seg.ToolParen); p != "" {
					piece += " (" + p + ")"
				}
				if act := strings.TrimSpace(seg.Text); act != "" {
					piece += "\n" + act + "…"
				}
			} else {
				piece = "⚙ " + seg.Text
			}
		case messagerow.SegToolResult:
			piece = "↩ " + seg.Text
		case messagerow.SegThinking:
			piece = "● " + seg.Text
		case messagerow.SegServerToolUse:
			if seg.ToolFacing != "" {
				piece = "⎈ " + seg.ToolFacing
			} else {
				piece = "⎈ " + seg.Text
			}
		case messagerow.SegAdvisorToolResult:
			piece = "✧ " + seg.Text
		case messagerow.SegGroupedToolUse:
			piece = "▦ " + seg.Text
		case messagerow.SegToolUseSummaryLine:
			piece = "  " + seg.Text
		default:
			piece = seg.Text
		}
		if piece != "" {
			parts = append(parts, piece)
		}
	}
	raw := strings.Join(parts, "\n")
	if msg.Type == types.MessageTypeUser {
		// Prepend "> " to match UI.
		lines := strings.Split(raw, "\n")
		for i, ln := range lines {
			lines[i] = "> " + ln
		}
		raw = strings.Join(lines, "\n")
	}
	return strings.TrimRight(layout.WrapForViewport(raw, wrapCols), "\n")
}

func transcriptBracketDumpScrollbackCmd(plain string) tea.Cmd {
	plain = strings.TrimSuffix(plain, "\n")
	if plain == "" {
		return nil
	}
	return tea.Printf("%s", plain+"\n")
}

func scheduleTranscriptEditorStatusClear(gen int) tea.Cmd {
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return gouTranscriptEditorClearStatusMsg{Gen: gen}
	})
}

func (m *model) transcriptEditorPrepCmd(gen int) tea.Cmd {
	w := exportTranscriptWidth(m)
	return func() tea.Msg {
		text := transcriptExportPlain(m, w)
		path := filepath.Join(os.TempDir(), fmt.Sprintf("gou-transcript-%d.txt", time.Now().UnixMilli()))
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			return gouTranscriptEditorPrepMsg{Gen: gen, Err: err}
		}
		bin, args, display := parseExternalEditorFromEnv()
		return gouTranscriptEditorPrepMsg{
			Gen:    gen,
			Path:   path,
			Editor: display,
			Bin:    bin,
			Args:   args,
		}
	}
}

func editorExecCommand(bin string, args []string, path string) *exec.Cmd {
	argv := append(slicesCloneStrings(args), path)
	return exec.Command(bin, argv...)
}

func slicesCloneStrings(s []string) []string {
	out := make([]string, len(s))
	copy(out, s)
	return out
}

func (m *model) handleTranscriptEditorChainMsg(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case gouTranscriptEditorPrepMsg:
		if msg.Gen != m.transcriptEditorGen {
			m.transcriptEditorBusy = false
			return nil
		}
		if msg.Err != nil {
			m.transcriptEditorBusy = false
			m.transcriptEditorStatus = fmt.Sprintf("render failed: %v", msg.Err)
			return scheduleTranscriptEditorStatusClear(m.transcriptEditorGen)
		}
		if msg.Bin == "" {
			m.transcriptEditorBusy = false
			m.transcriptEditorStatus = fmt.Sprintf("wrote %s · no $VISUAL/$EDITOR set", msg.Path)
			return scheduleTranscriptEditorStatusClear(m.transcriptEditorGen)
		}
		m.transcriptEditorStatus = fmt.Sprintf("opening %s", msg.Path)
		c := editorExecCommand(msg.Bin, msg.Args, msg.Path)
		return tea.ExecProcess(c, func(err error) tea.Msg {
			return gouTranscriptEditorExecDoneMsg{Gen: msg.Gen, Err: err}
		})
	case gouTranscriptEditorExecDoneMsg:
		if msg.Gen != m.transcriptEditorGen {
			return nil
		}
		m.transcriptEditorBusy = false
		if msg.Err != nil {
			m.transcriptEditorStatus = fmt.Sprintf("editor: %v", msg.Err)
		}
		return scheduleTranscriptEditorStatusClear(m.transcriptEditorGen)
	case gouTranscriptEditorClearStatusMsg:
		if msg.Gen != m.transcriptEditorGen {
			return nil
		}
		m.transcriptEditorStatus = ""
		return nil
	default:
		return nil
	}
}
