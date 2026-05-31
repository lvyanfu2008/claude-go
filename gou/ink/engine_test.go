package ink

import (
	"testing"

	"goc/gou/theme"
)

func TestEngineCreate(t *testing.T) {
	st := NewStore()
	pal := theme.ActivePalette()
	eng := NewEngine(nil, st, pal, func(ctx *Context, p Props) VNode {
		return VNode{Type: "Text", Props: Props{"content": "test"}}
	})
	if eng == nil {
		t.Fatal("expected engine")
	}
}

func TestEngineQuit(t *testing.T) {
	st := NewStore()
	pal := theme.ActivePalette()
	eng := NewEngine(nil, st, pal, func(ctx *Context, p Props) VNode {
		return VNode{Type: "Text"}
	})
	eng.Quit()
	select {
	case <-eng.quitCh:
		// Expected
	default:
		t.Fatal("quitCh should be closed")
	}
}
