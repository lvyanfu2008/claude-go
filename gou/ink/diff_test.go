package ink

import (
	"image/color"
	"testing"

	"goc/gou/ink/render"
)

func TestDiffEngine_noChange(t *testing.T) {
	s1 := render.NewScreen(3, 2)
	s2 := render.NewScreen(3, 2)
	d := render.DiffEngine{}
	out := d.Generate(s1, s2)
	if len(out) > 10 {
		t.Errorf("no-change diff should be minimal, got %d bytes: %q", len(out), out)
	}
}

func TestDiffEngine_singleCell(t *testing.T) {
	s1 := render.NewScreen(3, 1)
	s2 := render.NewScreen(3, 1)
	s2.Put(1, 0, 'X', render.CellStyle{Bold: true})
	d := render.DiffEngine{}
	out := d.Generate(s1, s2)
	if len(out) == 0 {
		t.Error("expected output for changed cell")
	}
}

func TestDiffEngine_sameStyleBatch(t *testing.T) {
	s1 := render.NewScreen(5, 1)
	s2 := render.NewScreen(5, 1)
	style := render.CellStyle{FG: color.RGBA{255, 0, 0, 255}}
	s2.PutString(1, 0, "ABC", style)
	d := render.DiffEngine{}
	out := d.Generate(s1, s2)
	t.Logf("diff output: %q", out)
}
