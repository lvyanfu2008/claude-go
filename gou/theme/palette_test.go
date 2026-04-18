package theme

import (
	"image/color"
	"testing"

	"goc/types"
)

func TestInitFromThemeName_lightChangesUserColor(t *testing.T) {
	t.Cleanup(func() { InitFromThemeName("") })
	InitFromThemeName("")
	c0 := MessageTypeColor(types.MessageTypeUser)
	InitFromThemeName("light")
	c1 := MessageTypeColor(types.MessageTypeUser)
	if colorsEqual(c0, c1) {
		t.Fatal("expected different user colors for light theme")
	}
	if ActiveTheme() != "light" {
		t.Fatalf("active %q", ActiveTheme())
	}
	InitFromThemeName("")
	if ActiveTheme() != "default" {
		t.Fatalf("reset active %q", ActiveTheme())
	}
}

func colorsEqual(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}
