package ink

import (
	"strings"

	"goc/gou/theme"
)

type RenderEngine struct {
	Terminal   *Terminal
	Store      *Store
	Theme      *theme.Palette
	RootComp   Component
	Reconciler *Reconciler
	Diff       *DiffEngine

	prevVTree  *VNode
	prevScreen *Screen
	currScreen *Screen
}

func NewEngine(term *Terminal, store *Store, pal *theme.Palette, root Component) *RenderEngine {
	return &RenderEngine{
		Terminal:   term,
		Store:      store,
		Theme:      pal,
		RootComp:   root,
		Reconciler: &Reconciler{},
		Diff:       &DiffEngine{},
	}
}

func (e *RenderEngine) Render() {
	w, h := e.Terminal.Size()
	e.Store.mu.Lock()
	e.Store.Width = w
	e.Store.Height = h
	e.Store.mu.Unlock()

	ctx := &Context{
		Theme:    e.Theme,
		Store:    e.Store,
		schedule: e.Store.ScheduleRender,
	}
	newTree := e.RootComp(ctx, Props{})

	_ = e.Reconciler.Diff(e.prevVTree, &newTree)
	e.prevVTree = &newTree

	ComputeLayout(&newTree, Constraints{MinW: 0, MaxW: w, MinH: 0, MaxH: h})

	e.currScreen = NewScreen(w, h)
	Rasterize(&newTree, e.currScreen)

	if e.prevScreen == nil {
		e.Terminal.Write([]byte(eraseDisplay()))
		e.Terminal.Write([]byte(cursorTo(0, 0)))
		for row := 0; row < h && row < len(e.currScreen.Cells); row++ {
			line := renderLineANSI(e.currScreen.Cells[row])
			e.Terminal.Write([]byte(line))
			if row < h-1 {
				e.Terminal.Write([]byte("\r\n"))
			}
		}
	} else {
		output := e.Diff.Generate(e.prevScreen, e.currScreen)
		e.Terminal.Write([]byte(output))
	}
	e.prevScreen = e.currScreen
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
	return buf.String()
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
