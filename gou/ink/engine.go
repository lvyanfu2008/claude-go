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
	case key.Key == "enter":
		e.Store.ScheduleRender()
	case key.Key == "up":
	case key.Key == "down":
	case key.Key == "esc":
	default:
		if len(key.Runes) > 0 {
			_ = key.Runes
		}
	}
}

func (e *RenderEngine) handleResize() {
	e.prevScreen = nil // force full repaint
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

	ctx := &vdom.Context{
		Theme:    e.Theme,
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

// Quit signals the main event loop to exit cleanly.  Safe to call multiple
// times.
func (e *RenderEngine) Quit() {
	select {
	case <-e.quitCh:
	default:
		close(e.quitCh)
	}
}
