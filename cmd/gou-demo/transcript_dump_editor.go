package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"goc/gou/layout"
	"goc/gou/messagerow"
	"goc/types"
)

// TS REPL.tsx: [ → dumpMode + showAll; unwrap from alt-screen + plain dump to scrollback.
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
	n := m.transcriptEffectiveN()
	opts := &messagerow.RenderOpts{ShowAllInTranscript: true}
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		msg := messagerow.NormalizeMessageJSON(m.store.Messages[i])
		b.WriteString(string(msg.Type))
		b.WriteByte('\n')
		b.WriteString(transcriptPlainBodyFromMessage(msg, opts, wrapCols))
	}
	return stripTranscriptExportTrailingSpaces(b.String())
}

func transcriptPlainBodyFromMessage(msg types.Message, opts *messagerow.RenderOpts, wrapCols int) string {
	segs := messagerow.SegmentsFromMessageOpts(msg, opts)
	var parts []string
	for _, seg := range segs {
		if seg.Text != "" {
			parts = append(parts, seg.Text)
		}
	}
	raw := strings.Join(parts, "\n")
	return strings.TrimRight(layout.WrapForViewport(raw, wrapCols), "\n")
}

func transcriptBracketDumpScrollbackCmd(plain string, programUsesAltScreen bool) tea.Cmd {
	plain = strings.TrimSuffix(plain, "\n")
	if plain == "" {
		if programUsesAltScreen {
			return func() tea.Msg { return tea.ExitAltScreen() }
		}
		return nil
	}
	out := plain + "\n"
	if !programUsesAltScreen {
		return tea.Printf("%s", out)
	}
	return tea.Sequence(
		func() tea.Msg { return tea.ExitAltScreen() },
		tea.Printf("%s", out),
	)
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
