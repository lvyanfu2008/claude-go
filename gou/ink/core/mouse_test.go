package core

import "testing"

func TestDecodeMousePress(t *testing.T) {
	ev, ok := DecodeMouse([]byte("\x1b[<0;10;20M"))
	if !ok {
		t.Fatal("expected mouse event")
	}
	if ev.Type != MousePress {
		t.Errorf("expected Press, got %v", ev.Type)
	}
	if ev.X != 10 || ev.Y != 20 {
		t.Errorf("expected (10,20), got (%d,%d)", ev.X, ev.Y)
	}
	if ev.Button != 0 {
		t.Errorf("expected button 0, got %d", ev.Button)
	}
}

func TestDecodeMouseRelease(t *testing.T) {
	ev, ok := DecodeMouse([]byte("\x1b[<0;5;10m"))
	if !ok {
		t.Fatal("expected mouse event")
	}
	if ev.Type != MouseRelease {
		t.Errorf("expected Release, got %v", ev.Type)
	}
}

func TestDecodeMouseWheelUp(t *testing.T) {
	ev, ok := DecodeMouse([]byte("\x1b[<64;10;5M"))
	if !ok {
		t.Fatal("expected mouse event")
	}
	if ev.Type != MouseWheel {
		t.Errorf("expected Wheel, got %v", ev.Type)
	}
	if ev.Button != -1 {
		t.Errorf("expected button -1 (up), got %d", ev.Button)
	}
}

func TestDecodeMouseWheelDown(t *testing.T) {
	ev, ok := DecodeMouse([]byte("\x1b[<65;10;5M"))
	if !ok {
		t.Fatal("expected mouse event")
	}
	if ev.Button != 1 {
		t.Errorf("expected button 1 (down), got %d", ev.Button)
	}
}

func TestDecodeNonMouse(t *testing.T) {
	_, ok := DecodeMouse([]byte("\x1b[A"))
	if ok {
		t.Error("up arrow should not be a mouse event")
	}
	_, ok = DecodeMouse([]byte("hello"))
	if ok {
		t.Error("text should not be a mouse event")
	}
}

func TestIsMouseEvent(t *testing.T) {
	if !IsMouseEvent([]byte("\x1b[<0;10;20M")) {
		t.Error("SGR mouse should be detected")
	}
	if IsMouseEvent([]byte("\x1b[A")) {
		t.Error("CSI up should not be mouse")
	}
}

func TestIsBracketedPaste(t *testing.T) {
	if !IsBracketedPasteStart([]byte("\x1b[200~")) {
		t.Error("bracketed paste start not detected")
	}
	if !IsBracketedPasteEnd([]byte("\x1b[201~")) {
		t.Error("bracketed paste end not detected")
	}
}
