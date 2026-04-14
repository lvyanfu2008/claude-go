package main

import (
	"encoding/base64"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// gouDemoCopyStatusClearMsg clears the ephemeral copy status line (Bubble Tea tick).
type gouDemoCopyStatusClearMsg struct{}

func tmuxPassthroughClipboard(seq string) string {
	esc := "\x1b"
	st := esc + "\\"
	if strings.TrimSpace(os.Getenv("TMUX")) != "" {
		return esc + "Ptmux;" + strings.ReplaceAll(seq, esc, esc+esc) + st
	}
	if strings.TrimSpace(os.Getenv("STY")) != "" {
		return esc + "P" + seq + st
	}
	return seq
}

// osc52ClipboardSequence returns ESC ] 52 ; c ; <base64> BEL (TS ink/termio/osc).
func osc52ClipboardSequence(text string) string {
	b64 := base64.StdEncoding.EncodeToString([]byte(text))
	return "\x1b" + "]52;c;" + b64 + "\a"
}

func copyNativeFireAndForget(text string) {
	go func(t string) {
		switch runtime.GOOS {
		case "darwin":
			c := exec.Command("pbcopy")
			c.Stdin = strings.NewReader(t)
			_ = c.Run()
		case "linux":
			c := exec.Command("wl-copy")
			c.Stdin = strings.NewReader(t)
			if c.Run() == nil {
				return
			}
			c2 := exec.Command("xclip", "-selection", "clipboard")
			c2.Stdin = strings.NewReader(t)
			if c2.Run() == nil {
				return
			}
			c3 := exec.Command("xsel", "--clipboard", "--input")
			c3.Stdin = strings.NewReader(t)
			_ = c3.Run()
		case "windows":
			c := exec.Command("clip")
			c.Stdin = strings.NewReader(t)
			_ = c.Run()
		}
	}(text)
}

func tmuxLoadBufferFireAndForget(text string) {
	if strings.TrimSpace(os.Getenv("TMUX")) == "" {
		return
	}
	args := []string{"load-buffer", "-w", "-"}
	if strings.TrimSpace(os.Getenv("LC_TERMINAL")) == "iTerm2" {
		args = []string{"load-buffer", "-"}
	}
	c := exec.Command("tmux", args...)
	c.Stdin = strings.NewReader(text)
	_ = c.Start()
}

func gouDemoClipboardPathKind() int {
	if runtime.GOOS == "darwin" && strings.TrimSpace(os.Getenv("SSH_CONNECTION")) == "" {
		return 0 // native
	}
	if strings.TrimSpace(os.Getenv("TMUX")) != "" {
		return 1 // tmux buffer
	}
	return 2 // osc52
}

// gouDemoKeyIsCtrlC matches Ctrl+C / ETX for in-app copy (some stacks use KeyRunes for byte 3).
func gouDemoKeyIsCtrlC(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyBreak {
		return true
	}
	if msg.Type == tea.KeyRunes && !msg.Paste && len(msg.Runes) == 1 && msg.Runes[0] == '\x03' {
		return true
	}
	return false
}

// selectionCopyToClipboardCmd mirrors TS setClipboard and go-tui/main/test_ignore.go: atotto/clipboard first,
// then native exec fallbacks, tmux load-buffer, and OSC 52 via tea.Printf when not on alt-screen (Printf is a no-op there).
func (m *model) selectionCopyToClipboardCmd(text string) tea.Cmd {
	if text == "" {
		return nil
	}
	atOK := clipboard.WriteAll(text) == nil
	if !atOK {
		copyNativeFireAndForget(text)
	}
	tmuxLoadBufferFireAndForget(text)
	seq := osc52ClipboardSequence(text)
	if strings.TrimSpace(os.Getenv("TMUX")) != "" {
		seq = tmuxPassthroughClipboard(seq)
	}
	n := utf8.RuneCountInString(text)
	if atOK {
		m.copyStatus = "copied " + strconv.Itoa(n) + " chars"
	} else {
		switch gouDemoClipboardPathKind() {
		case 0:
			m.copyStatus = "copied " + strconv.Itoa(n) + " chars"
		case 1:
			m.copyStatus = "copied " + strconv.Itoa(n) + " chars · tmux: prefix + ]"
		default:
			m.copyStatus = "sent " + strconv.Itoa(n) + " chars (OSC 52)"
		}
	}
	tick := tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return gouDemoCopyStatusClearMsg{}
	})
	if gouDemoAltScreenEnabled() {
		return tick
	}
	return tea.Batch(
		tea.Printf("%s", seq),
		tick,
	)
}
