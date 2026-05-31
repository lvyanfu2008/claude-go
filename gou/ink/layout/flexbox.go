package layout

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"goc/gou/ink/vdom"
)

// ComputeLayout walks the VNode tree post-order and computes LayoutResult for each node.
func ComputeLayout(node *vdom.VNode, constraints vdom.Constraints) {
	switch node.Type {
	case "Text":
		layoutText(node, constraints)
	case "Box":
		layoutBox(node, constraints)
	case "ScrollBox":
		layoutScrollBox(node, constraints)
	default:
		node.Layout = vdom.LayoutResult{W: 0, H: 0}
	}
}

func layoutText(node *vdom.VNode, c vdom.Constraints) {
	content := node.Props.GetString("content")
	if content == "" {
		node.Layout = vdom.LayoutResult{}
		return
	}

	maxW := c.MaxW
	if maxW <= 0 {
		maxW = 80
	}

	var lines []string
	for _, line := range splitLinesANSI(content) {
		wrapped := wordWrapANSI(line, maxW)
		lines = append(lines, wrapped...)
	}

	node.Layout = vdom.LayoutResult{
		W: maxLineWidth(lines, maxW),
		H: len(lines),
	}
}

func splitLinesANSI(s string) []string {
	var lines []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func wordWrapANSI(line string, maxW int) []string {
	if maxW <= 0 {
		return []string{line}
	}
	var result []string
	var current strings.Builder
	currentW := 0
	i := 0
	runes := []rune(line)

	for i < len(runes) {
		r := runes[i]
		if r == '\x1b' {
			// skip ANSI escape
			current.WriteRune(r)
			i++
			for i < len(runes) && runes[i] != 'm' {
				current.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				current.WriteRune(runes[i]) // 'm'
				i++
			}
			continue
		}
		rw := runewidth.RuneWidth(r)
		if r == ' ' && currentW >= maxW {
			result = append(result, current.String())
			current.Reset()
			currentW = 0
			i++
			continue
		}
		if currentW+rw > maxW && current.Len() > 0 {
			result = append(result, current.String())
			current.Reset()
			currentW = 0
		}
		current.WriteRune(r)
		currentW += rw
		i++
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

func maxLineWidth(lines []string, maxW int) int {
	w := 0
	for _, l := range lines {
		lw := runewidth.StringWidth(stripANSIStr(l))
		if lw > w {
			w = lw
		}
	}
	if w > maxW {
		w = maxW
	}
	return w
}

func stripANSIStr(s string) string {
	var result []rune
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result = append(result, r)
	}
	return string(result)
}

// layoutBox: full flexbox
func layoutBox(node *vdom.VNode, c vdom.Constraints) {
	direction := node.Props.GetString("direction")
	if direction == "" {
		direction = "column"
	}

	w := node.Props.GetInt("width")
	if w <= 0 {
		w = c.MaxW
	}
	pad := node.Props.GetInt("padding")
	gap := node.Props.GetInt("gap")
	innerW := w - 2*pad

	// STEP 1: Compute all children preferred sizes
	for i := range node.Children {
		childC := vdom.Constraints{
			MinW: node.Children[i].Props.GetInt("minWidth"),
			MaxW: innerW,
		}
		if cw := node.Children[i].Props.GetInt("width"); cw > 0 {
			childC.MinW = cw
			childC.MaxW = cw
		}
		ComputeLayout(&node.Children[i], childC)
	}

	// STEP 2: Resolve main axis (flexGrow/flexShrink)
	totalChildMain := 0
	for i := range node.Children {
		totalChildMain += childMainSize(&node.Children[i], direction)
	}
	totalGap := 0
	if len(node.Children) > 1 {
		totalGap = (len(node.Children) - 1) * gap
	}

	if direction == "row" {
		free := innerW - totalChildMain - totalGap
		if free > 0 {
			distributeGrow(node, free, direction)
		} else if free < 0 {
			distributeShrink(node, -free, direction)
		}
		totalChildMain = 0
		for i := range node.Children {
			totalChildMain += childMainSize(&node.Children[i], direction)
		}
	}

	// STEP 3: Resolve cross axis
	maxCross := 0
	for i := range node.Children {
		cs := childCrossSize(&node.Children[i], direction)
		if cs > maxCross {
			maxCross = cs
		}
	}
	align := node.Props.GetString("alignItems")
	for i := range node.Children {
		applyCrossStretch(&node.Children[i], align, direction, maxCross)
	}

	// STEP 4: Determine container height, apply column flexGrow
	var h int
	if direction == "column" {
		h = totalChildMain + totalGap + 2*pad
		if c.MaxH > 0 && h < c.MaxH && hasFlexGrowChild(node) {
			h = c.MaxH
		}
		extraH := h - 2*pad - totalChildMain - totalGap
		if extraH > 0 {
			distributeGrow(node, extraH, direction)
			totalChildMain = 0
			for i := range node.Children {
				totalChildMain += childMainSize(&node.Children[i], direction)
			}
		}
	} else {
		h = maxCross + 2*pad
	}

	justify := node.Props.GetString("justifyContent")
	if justify == "" {
		justify = "start"
	}

	freeMain := h - 2*pad - totalChildMain - totalGap
	var mainOffset int
	switch justify {
	case "center":
		mainOffset = pad + freeMain/2
	case "end":
		mainOffset = pad + freeMain
	case "between":
		mainOffset = pad
	default:
		mainOffset = pad
	}

	cur := mainOffset
	for i := range node.Children {
		child := &node.Children[i]
		if direction == "column" {
			child.Layout.Y = cur
			child.Layout.X = pad + crossAxisOffset(child, align, direction, innerW)
		} else {
			child.Layout.X = cur
			child.Layout.Y = pad + crossAxisOffset(child, align, direction, maxCross)
		}
		cur += childMainSize(child, direction)
		if justify == "between" && len(node.Children) > 1 && i < len(node.Children)-1 {
			cur += freeMain / (len(node.Children) - 1)
		}
		cur += gap
	}

	node.Layout = vdom.LayoutResult{W: w, H: h}
}

func layoutScrollBox(node *vdom.VNode, c vdom.Constraints) {
	w := node.Props.GetInt("width")
	if w <= 0 {
		w = c.MaxW
	}

	innerW := w
	contentH := 0
	for i := range node.Children {
		child := &node.Children[i]
		ComputeLayout(child, vdom.Constraints{MinW: 0, MaxW: innerW})
		child.Layout.X = 0
		child.Layout.Y = contentH
		contentH += child.Layout.H
	}

	h := node.Props.GetInt("height")
	if h <= 0 {
		h = c.MaxH
	}
	if h <= 0 {
		h = contentH // natural height from children
	}

	visStart := 0
	visEnd := len(node.Children)

	node.Layout = vdom.LayoutResult{
		W:            w,
		H:            h,
		ContentH:     contentH,
		OverflowTop:  0,
		VisibleRange: [2]int{visStart, visEnd},
	}
}

func childMainSize(child *vdom.VNode, direction string) int {
	if direction == "column" {
		return child.Layout.H
	}
	return child.Layout.W
}

func childCrossSize(child *vdom.VNode, direction string) int {
	if direction == "column" {
		return child.Layout.W
	}
	return child.Layout.H
}

func distributeGrow(node *vdom.VNode, free int, direction string) {
	totalGrow := 0
	for i := range node.Children {
		totalGrow += node.Children[i].Props.GetInt("flexGrow")
	}
	if totalGrow == 0 {
		return
	}
	for i := range node.Children {
		grow := node.Children[i].Props.GetInt("flexGrow")
		if grow == 0 {
			continue
		}
		delta := free * grow / totalGrow
		if direction == "column" {
			node.Children[i].Layout.H += delta
		} else {
			node.Children[i].Layout.W += delta
		}
	}
}

func distributeShrink(node *vdom.VNode, overflow int, direction string) {
	totalShrink := 0
	for i := range node.Children {
		sh := node.Children[i].Props.GetInt("flexShrink")
		if sh == 0 {
			sh = 1
		}
		totalShrink += sh
	}
	if totalShrink == 0 {
		return
	}
	for i := range node.Children {
		sh := node.Children[i].Props.GetInt("flexShrink")
		if sh == 0 {
			sh = 1
		}
		delta := overflow * sh / totalShrink
		if direction == "column" {
			node.Children[i].Layout.H -= delta
			if node.Children[i].Layout.H < 0 {
				node.Children[i].Layout.H = 0
			}
		} else {
			node.Children[i].Layout.W -= delta
			if node.Children[i].Layout.W < 0 {
				node.Children[i].Layout.W = 0
			}
		}
	}
}

func applyCrossStretch(child *vdom.VNode, align, direction string, maxCross int) {
	if align == "stretch" {
		if direction == "column" {
			child.Layout.W = maxCross
		} else {
			child.Layout.H = maxCross
		}
	}
}

func crossAxisOffset(child *vdom.VNode, align, direction string, available int) int {
	childCross := childCrossSize(child, direction)
	switch align {
	case "center":
		return (available - childCross) / 2
	case "end":
		return available - childCross
	default:
		return 0
	}
}

func hasFlexGrowChild(node *vdom.VNode) bool {
	for i := range node.Children {
		if node.Children[i].Props.GetInt("flexGrow") > 0 {
			return true
		}
	}
	return false
}
