package ink

import (
	"image/color"
	"strings"
	"testing"
)

func TestStyleToSGR_bold(t *testing.T) {
	s := CellStyle{Bold: true}
	result := styleToSGR(s)
	if !strings.Contains(result, "1") {
		t.Errorf("expected bold (1) in SGR, got %q", result)
	}
}

func TestStyleToSGR_color(t *testing.T) {
	s := CellStyle{FG: color.RGBA{255, 0, 0, 255}}
	result := styleToSGR(s)
	if !strings.Contains(result, "38;2;255;0;0") {
		t.Errorf("expected RGB fg in SGR, got %q", result)
	}
}

func TestStyleToSGR_empty(t *testing.T) {
	s := CellStyle{}
	result := styleToSGR(s)
	if result != "" {
		t.Errorf("expected empty SGR for empty style, got %q", result)
	}
}

func TestStyleToSGR_combined(t *testing.T) {
	s := CellStyle{Bold: true, Dim: true, FG: color.RGBA{0, 255, 0, 255}}
	result := styleToSGR(s)
	if !strings.Contains(result, "1;2;") {
		t.Errorf("expected bold+dim in SGR, got %q", result)
	}
}
