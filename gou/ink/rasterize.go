package ink

import "image/color"

func Rasterize(node *VNode, screen *Screen) {
	rasterizeNode(node, screen, 0, 0)
}

// rasterizeNode walks the VNode tree and renders each node at its screen position,
// accumulating offsets from parent containers so that Layout.X/Y (which are relative
// to the parent) are translated to absolute screen coordinates.
func rasterizeNode(node *VNode, screen *Screen, offX, offY int) {
	absX := node.Layout.X + offX
	absY := node.Layout.Y + offY

	switch node.Type {
	case "Text":
		rasterizeTextAt(node, screen, absX, absY)
	case "Box":
		for i := range node.Children {
			rasterizeNode(&node.Children[i], screen, absX, absY)
		}
	case "ScrollBox":
		visStart := node.Layout.VisibleRange[0]
		visEnd := node.Layout.VisibleRange[1]
		if visEnd > len(node.Children) {
			visEnd = len(node.Children)
		}
		for i := visStart; i < visEnd; i++ {
			rasterizeNode(&node.Children[i], screen, absX, absY)
		}
	}
}

// rasterizeTextAt renders a Text node at the given absolute screen position (offX, offY).
func rasterizeTextAt(node *VNode, screen *Screen, offX, offY int) {
	content := node.Props.GetString("content")
	if content == "" {
		return
	}

	style := CellStyle{
		Bold:      node.Props.GetBool("bold"),
		Dim:       node.Props.GetBool("dim"),
		Italic:    node.Props.GetBool("italic"),
		Underline: node.Props.GetBool("underline"),
	}

	if c, ok := node.Props["color"]; ok {
		style.FG = toColor(c)
	}
	if c, ok := node.Props["bg"]; ok {
		style.BG = toColor(c)
	}

	lines := splitLinesANSI(content)
	x := offX
	y := offY

	for _, line := range lines {
		wrapped := wordWrapANSI(line, screen.Width-x)
		for i, wl := range wrapped {
			if y+i >= screen.Height || y+i < 0 {
				continue
			}
			screen.PutString(x, y+i, stripANSIStr(wl), style)
		}
		y += len(wrapped)
	}
}

func toColor(v interface{}) color.Color {
	if c, ok := v.(color.Color); ok {
		return c
	}
	return nil
}
