package ink

import (
	"fmt"
	"strings"

	"goc/gou/theme"
)

type RenderEngine struct {
	Terminal   *Terminal
	Store      *Store
	Theme      *theme.Palette
	RootComp   Component
	Reconciler *Reconciler

	prevVTree   *VNode
	lastMsgRows int
}

func NewEngine(term *Terminal, store *Store, pal *theme.Palette, root Component) *RenderEngine {
	return &RenderEngine{
		Terminal:   term,
		Store:      store,
		Theme:      pal,
		RootComp:   root,
		Reconciler: &Reconciler{},
	}
}

func (e *RenderEngine) Render() {
	w, termH := e.Terminal.Size()
	e.Store.mu.Lock()
	e.Store.Width = w
	e.Store.Height = termH
	e.Store.mu.Unlock()

	ctx := &Context{
		Theme:    e.Theme,
		Store:    e.Store,
		schedule: e.Store.ScheduleRender,
	}
	newTree := e.RootComp(ctx, Props{})

	e.Reconciler.Diff(e.prevVTree, &newTree)
	e.prevVTree = &newTree

	// Layout: messages take natural height, prompt area at bottom.
	ComputeLayout(&newTree, Constraints{MinW: 0, MaxW: w, MinH: 0, MaxH: termH})

	// Rasterize to find content extents.
	tmp := NewScreen(w, 2048)
	Rasterize(&newTree, tmp)

	// Find where messages end and footer (status + prompt) begins.
	// Messages are in the ScrollBox, footer is StatusLine + PromptInput.
	// The footer always starts at the second-to-last 3 rows of content.
	last := lastNonEmptyRow(tmp)
	totalH := last + 1
	if totalH == 0 {
		totalH = 1
	}

	// Footer is the last 3 rows: statusLine, border, prompt text.
	footerH := 3
	if totalH < footerH {
		footerH = totalH
	}
	msgH := totalH - footerH

	// Build message buffer and footer buffer.
	msgRows := NewScreen(w, msgH)
	for row := 0; row < msgH; row++ {
		copy(msgRows.Cells[row], tmp.Cells[row])
	}
	footerRows := NewScreen(w, footerH)
	for row := 0; row < footerH; row++ {
		copy(footerRows.Cells[row], tmp.Cells[msgH+row])
	}

	if e.lastMsgRows == 0 {
		// First frame: write messages from current cursor position,
		// then position footer at bottom of terminal.
		for row := 0; row < msgH; row++ {
			line := renderLineANSI(msgRows.Cells[row])
			e.Terminal.Write([]byte(line))
			e.Terminal.Write([]byte("\x1b[0K\r\n"))
		}
		// Position footer at absolute bottom of terminal.
		e.writeFooterAtBottom(footerRows, termH, footerH)
	} else {
		// Update messages in-place, reposition footer.
		// Rewind to start of messages.
		if e.lastMsgRows > 0 {
			e.Terminal.Write([]byte(fmt.Sprintf("\x1b[%dA", e.lastMsgRows)))
		}
		// Write messages.
		for row := 0; row < msgH; row++ {
			line := renderLineANSI(msgRows.Cells[row])
			e.Terminal.Write([]byte(line))
			e.Terminal.Write([]byte("\x1b[0K\r\n"))
		}
		// Clear leftover message rows if messages shrank.
		if msgH < e.lastMsgRows {
			for i := msgH; i < e.lastMsgRows; i++ {
				e.Terminal.Write([]byte("\x1b[0K\r\n"))
			}
			e.Terminal.Write([]byte(fmt.Sprintf("\x1b[%dA", e.lastMsgRows-msgH)))
		}
		// Reposition and rewrite footer.
		e.writeFooterAtBottom(footerRows, termH, footerH)
	}
	e.lastMsgRows = msgH

	// Position terminal cursor at prompt input.
	cursorY := termH - 2
	cursorX := 2 + e.Store.CursorPos
	if cursorY < 0 {
		cursorY = 0
	}
	e.Terminal.Write([]byte(cursorTo(cursorY, cursorX)))
}

func (e *RenderEngine) writeFooterAtBottom(footer *Screen, termH, footerH int) {
	for row := 0; row < footerH; row++ {
		y := termH - footerH + row
		line := renderLineANSI(footer.Cells[row])
		e.Terminal.Write([]byte(cursorTo(y, 0)))
		e.Terminal.Write([]byte(line))
		e.Terminal.Write([]byte("\x1b[0K"))
	}
}

func renderLineANSI(cells []TermCell) string {
	if len(cells) == 0 {
		return ""
	}
	var buf strings.Builder
	prevStyle := CellStyle{}
	inStyle := false
	for _, c := range cells {
		if !c.Style.Equals(prevStyle) {
			if inStyle {
				buf.WriteString(sgrReset())
			}
			sgr := styleToSGR(c.Style)
			if sgr != "" {
				buf.WriteString(sgr)
				inStyle = true
			}
			prevStyle = c.Style
		}
		if c.Rune == 0 {
			buf.WriteByte(' ')
		} else {
			buf.WriteRune(c.Rune)
		}
	}
	if inStyle {
		buf.WriteString(sgrReset())
	}
	return strings.TrimRight(buf.String(), " ")
}

func (e *RenderEngine) Start() error {
	if err := e.Terminal.Init(); err != nil {
		return err
	}
	e.Store.SetOnRender(e.Render)
	go e.Store.RunRenderLoop()
	return nil
}

func (e *RenderEngine) Stop() {
	e.Store.Stop()
	e.Terminal.Shutdown()
}

func lastNonEmptyRow(s *Screen) int {
	for y := len(s.Cells) - 1; y >= 0; y-- {
		for _, c := range s.Cells[y] {
			if c.Rune != 0 {
				return y
			}
		}
	}
	return 0
}
