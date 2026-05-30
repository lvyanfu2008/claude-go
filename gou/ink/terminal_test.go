package ink

import (
	"os"
	"testing"
)

func TestTerminalInitShutdown(t *testing.T) {
	if os.Getenv("GOU_INK_TEST_TERMINAL") != "1" {
		t.Skip("set GOU_INK_TEST_TERMINAL=1 to run terminal tests")
	}
	term := NewTerminal()
	err := term.Init()
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	w, h := term.Size()
	if w <= 0 || h <= 0 {
		t.Errorf("expected positive terminal size, got %dx%d", w, h)
	}
	term.Shutdown()
}

func TestTerminalResizeChannel(t *testing.T) {
	term := NewTerminal()
	if term.ResizeEvents() == nil {
		t.Error("ResizeEvents should return non-nil channel")
	}
}
