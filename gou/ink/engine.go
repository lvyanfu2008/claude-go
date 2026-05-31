package ink

import (
	"goc/gou/ink/core"
	"goc/gou/ink/layout"
	"goc/gou/ink/render"
	"goc/gou/ink/store"
	"goc/gou/ink/vdom"
	"goc/gou/theme"
)

// RenderEngine orchestrates all five layers of the rendering pipeline:
// core (terminal I/O), vdom (virtual DOM + fiber reconciliation),
// layout (flexbox + scroll), render (screen buffer + pixel-aware diff),
// and store (reactive state).  It owns the main event loop and drives
// frame-by-frame updates through the complete pipeline.
type RenderEngine struct {
	Terminal *core.Terminal
	Store    *store.Store
	Theme    *theme.Palette
	RootComp vdom.Component

	reconciler *vdom.FiberReconciler
	prevVTree  *vdom.VNode
	prevScreen *render.Screen
	firstFrame bool

	keyParser  *core.KeyboardParser
	diffEngine *render.DiffEngine

	quitCh chan struct{}
}

// NewEngine creates a RenderEngine.  The terminal, store, palette, and root
// component must all be provided (terminal may be nil in tests that do not
// call Run).
func NewEngine(term *core.Terminal, st *store.Store, pal *theme.Palette, root vdom.Component) *RenderEngine {
	return &RenderEngine{
		Terminal:   term,
		Store:      st,
		Theme:      pal,
		RootComp:   root,
		reconciler: &vdom.FiberReconciler{},
		keyParser:  core.NewKeyboardParser(),
		diffEngine: render.NewDiffEngine(),
		firstFrame: true,
		quitCh:     make(chan struct{}),
	}
}

// Run enters the main event loop.  It initialises the terminal, wires up the
// store's render callback, renders the first frame, then processes terminal
// input, resize events, and the quit signal until the read channel closes or
// Quit is called.
func (e *RenderEngine) Run() error {
	if err := e.Terminal.Init(); err != nil {
		return err
	}
	defer e.Terminal.Shutdown()

	e.Store.SetOnRender(e.render)
	go e.Store.RunRenderLoop()

	e.render() // first frame

	for {
		select {
		case raw, ok := <-e.Terminal.Read():
			if !ok {
				return nil
			}
			e.handleInput(raw)
		case <-e.Terminal.ResizeCh():
			e.handleResize()
		case <-e.quitCh:
			return nil
		}
	}
}

func (e *RenderEngine) handleInput(raw []byte) {
	switch {
	case core.IsMouseEvent(raw):
		// Mouse events: routed to store for component access
		ev, ok := core.DecodeMouse(raw)
		if ok {
			_ = ev // TODO: route to component handler
		}
	case core.IsBracketedPasteStart(raw):
		// Bracketed paste: accumulate content (future)
	default:
		key := e.keyParser.Parse(raw)
		e.onKey(key)
	}
}

func (e *RenderEngine) onKey(key core.ParsedKey) {
	switch {
	case key.Key == "c" && key.Mod&core.Ctrl != 0:
		e.Quit()
	case key.Key == "o" && key.Mod&core.Ctrl != 0:
		e.toggleTranscript()
	case key.Key == "e" && key.Mod&core.Ctrl != 0:
		cur := e.Store.GetMeta("transcriptShowAll")
		if cur == "1" {
			e.Store.SetMeta("transcriptShowAll", "0")
		} else {
			e.Store.SetMeta("transcriptShowAll", "1")
		}
	case key.Key == "[" && key.Mod == 0 && e.Store.GetMeta("transcriptSearchQuery") == "":
		e.Store.SetMeta("transcriptShowAll", "1")
		e.Store.SetMeta("uiScreen", "transcript")
	case key.Key == "esc":
		e.Quit()
	case key.Key == "enter":
		e.submitInput()
	case key.Key == "up":
		e.scrollMessages(-1)
	case key.Key == "down":
		e.scrollMessages(1)
	case key.Key == "pgup":
		e.scrollMessages(-e.Store.Height() / 2)
	case key.Key == "pgdn":
		e.scrollMessages(e.Store.Height() / 2)
	case key.Key == "end":
		e.scrollToBottom()
	case key.Key == "backspace":
		e.deleteBeforeCursor()
	case key.Key == "left":
		e.moveCursor(-1)
	case key.Key == "right":
		e.moveCursor(1)
	case key.Key == "home":
		e.Store.SetCursorPos(0)
	case key.Key == "delete":
		e.deleteAfterCursor()
	default:
		if len(key.Runes) > 0 {
			e.insertRunes(key.Runes)
		}
	}
}

// submitInput appends the current input as a user message and clears the input.
func (e *RenderEngine) submitInput() {
	val := e.Store.InputValue()
	if val == "" {
		return
	}
	msgs := e.Store.GetMessages()
	msgs = append(msgs, vdom.Message{
		UUID: "user-" + itoa(len(msgs)),
		Type: "user",
		ContentBlocks: []vdom.ContentBlock{{Type: "text", Content: val}},
	})
	e.Store.SetMessages(msgs)
	e.Store.SetInputValue("")
	e.Store.SetCursorPos(0)
}

func (e *RenderEngine) insertRunes(runes []rune) {
	val := []rune(e.Store.InputValue())
	pos := e.Store.CursorPos()
	if pos > len(val) {
		pos = len(val)
	}
	val = append(val[:pos], append(runes, val[pos:]...)...)
	e.Store.SetInputValue(string(val))
	e.Store.SetCursorPos(pos + len(runes))
}

func (e *RenderEngine) deleteBeforeCursor() {
	val := []rune(e.Store.InputValue())
	pos := e.Store.CursorPos()
	if pos > 0 && len(val) > 0 {
		val = append(val[:pos-1], val[pos:]...)
		e.Store.SetInputValue(string(val))
		e.Store.SetCursorPos(pos - 1)
	}
}

func (e *RenderEngine) deleteAfterCursor() {
	val := []rune(e.Store.InputValue())
	pos := e.Store.CursorPos()
	if pos < len(val) {
		val = append(val[:pos], val[pos+1:]...)
		e.Store.SetInputValue(string(val))
	}
}

func (e *RenderEngine) moveCursor(delta int) {
	pos := e.Store.CursorPos() + delta
	maxPos := len([]rune(e.Store.InputValue()))
	if pos < 0 {
		pos = 0
	}
	if pos > maxPos {
		pos = maxPos
	}
	e.Store.SetCursorPos(pos)
}

func (e *RenderEngine) scrollMessages(delta int) {
	newTop := e.Store.ScrollTop() + delta
	if newTop < 0 {
		newTop = 0
	}
	e.Store.SetScrollTop(newTop)
}

func (e *RenderEngine) scrollToBottom() {
	e.Store.SetScrollTop(1<<31 - 1) // max int, clamped by renderer
}

func (e *RenderEngine) toggleTranscript() {
	cur := e.Store.GetMeta("uiScreen")
	if cur == "transcript" {
		e.Store.SetMeta("uiScreen", "prompt")
	} else {
		e.Store.SetMeta("uiScreen", "transcript")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func (e *RenderEngine) handleResize() {
	e.prevScreen = nil // force full repaint
	w, h := e.Terminal.Size()
	if w > 0 {
		e.Store.SetWidth(w)
	}
	if h > 0 {
		e.Store.SetHeight(h)
	}
	e.Store.ScheduleRender()
}

// render runs one complete frame through the full pipeline:
// component → reconcile → layout → rasterize → diff → write.
func (e *RenderEngine) render() {
	w, h := e.Terminal.Size()
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	// Sync terminal dimensions into store so components see current size.
	e.Store.SetWidth(w)
	e.Store.SetHeight(h)

	// Terminal tab title
	if e.Terminal != nil {
		title := "gou-demo"
		if e.Store.IsLoading() || e.Store.StreamingText() != "" {
			title = "… " + title
		}
		e.writeTermTitle(title)
	}

	ctx := &vdom.Context{
		Theme:    e.Theme,
		Store:    e.Store,
		Schedule: e.Store.ScheduleRender,
	}
	newTree := e.RootComp(ctx, vdom.Props{})

	e.reconciler.Reconcile(e.prevVTree, &newTree)
	e.prevVTree = &newTree

	layout.ComputeLayout(&newTree, vdom.Constraints{MinW: 0, MaxW: w, MinH: 0, MaxH: h})

	cur := render.NewScreen(w, h)
	render.Rasterize(&newTree, cur)

	if e.firstFrame {
		e.Terminal.Write([]byte(render.EraseDisplay()))
		e.Terminal.Write([]byte(render.CursorTo(0, 0)))
		e.firstFrame = false
	}

	output := e.diffEngine.Generate(e.prevScreen, cur)
	if output != "" {
		e.Terminal.Write([]byte(output))
	}

	// Save current screen for next frame diff
	if e.prevScreen == nil || e.prevScreen.Width != w || e.prevScreen.Height != h {
		e.prevScreen = render.NewScreen(w, h)
	} else {
		e.prevScreen.Clear()
	}
	for y := 0; y < h && y < cur.Height; y++ {
		copy(e.prevScreen.Cells[y][:w], cur.Cells[y][:w])
	}
}

func (e *RenderEngine) writeTermTitle(title string) {
	// OSC 0 sets both icon name and window title
	e.Terminal.Write([]byte("\x1b]0;" + title + "\a"))
}

// Quit signals the main event loop to exit cleanly.  Safe to call multiple
// times.
func (e *RenderEngine) Quit() {
	select {
	case <-e.quitCh:
	default:
		close(e.quitCh)
	}
}
