package compo

import "goc/gou/ink"

func Scrollbar(totalH, viewportH, scrollTop, width int) ink.VNode {
	if totalH <= viewportH {
		return ink.VNode{Type: "Text"}
	}
	thumbH := viewportH * viewportH / totalH
	if thumbH < 1 {
		thumbH = 1
	}
	thumbY := scrollTop * viewportH / totalH
	cells := make([]string, viewportH)
	for i := range cells {
		if i >= thumbY && i < thumbY+thumbH {
			cells[i] = "█"
		} else {
			cells[i] = "░"
		}
	}
	kids := make([]ink.VNode, len(cells))
	for i, c := range cells {
		kids[i] = ink.VNode{Type: "Text", Props: ink.Props{"content": c, "dim": true}}
	}
	return ink.VNode{
		Type: "Box", Key: "scrollbar",
		Props: ink.Props{"direction": "column", "width": width},
		Children: kids,
	}
}
