package ccbstream

import (
	"bufio"
	"encoding/json"
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

// Msg wraps StreamEvent for Bubble Tea (Update switch).
type Msg StreamEvent

// Feed reads NDJSON lines from r and sends Msg to p. Run in a goroutine.
// For keyboard + stdin pipe on Unix, create the program with tea.WithInput(open("/dev/tty")).
func Feed(r io.Reader, p *tea.Program) {
	go func() {
		s := bufio.NewScanner(r)
		// Long lines (tool payloads)
		const max = 1024 * 1024
		buf := make([]byte, 0, 64*1024)
		s.Buffer(buf, max)
		for s.Scan() {
			line := s.Bytes()
			if len(line) == 0 {
				continue
			}
			var ev StreamEvent
			if err := json.Unmarshal(line, &ev); err != nil || ev.Type == "" {
				continue
			}
			p.Send(Msg(ev))
		}
	}()
}
