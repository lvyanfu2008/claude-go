package layout

import (
	"testing"

	"goc/gou/ink/vdom"
)

func TestLayoutText_simple(t *testing.T) {
	node := &vdom.VNode{Type: "Text", Props: vdom.Props{"content": "hello"}}
	ComputeLayout(node, vdom.Constraints{MinW: 0, MaxW: 80})
	if node.Layout.W < 1 || node.Layout.H != 1 {
		t.Errorf("expected w>0, h=1; got w=%d, h=%d", node.Layout.W, node.Layout.H)
	}
}

func TestLayoutText_wrap(t *testing.T) {
	node := &vdom.VNode{Type: "Text", Props: vdom.Props{"content": "hello world this is a long line"}}
	ComputeLayout(node, vdom.Constraints{MinW: 0, MaxW: 10})
	if node.Layout.H < 2 {
		t.Errorf("expected multi-line wrap, got h=%d", node.Layout.H)
	}
}

func TestLayoutBox_childrenPositioned(t *testing.T) {
	node := &vdom.VNode{Type: "Box", Props: vdom.Props{"width": 80},
		Children: []vdom.VNode{
			{Type: "Text", Props: vdom.Props{"content": "line1"}},
			{Type: "Text", Props: vdom.Props{"content": "line2"}},
		},
	}
	ComputeLayout(node, vdom.Constraints{MinW: 0, MaxW: 80})
	if node.Children[0].Layout.Y >= node.Children[1].Layout.Y {
		t.Error("child 1 should be above child 2")
	}
	if node.Layout.H < 2 {
		t.Errorf("box height >= 2, got %d", node.Layout.H)
	}
}

func TestLayoutBox_rowDirection(t *testing.T) {
	node := &vdom.VNode{Type: "Box", Props: vdom.Props{"width": 60, "direction": "row", "gap": 1},
		Children: []vdom.VNode{
			{Type: "Text", Props: vdom.Props{"content": "A"}},
			{Type: "Text", Props: vdom.Props{"content": "BB"}},
		},
	}
	ComputeLayout(node, vdom.Constraints{MinW: 0, MaxW: 80})
	if node.Children[0].Layout.X >= node.Children[1].Layout.X {
		t.Error("in row direction, child1.x < child2.x")
	}
}

func TestLayoutBox_flexGrow(t *testing.T) {
	node := &vdom.VNode{Type: "Box", Props: vdom.Props{"width": 80, "direction": "row"},
		Children: []vdom.VNode{
			{Type: "Text", Props: vdom.Props{"content": "A", "flexGrow": 1}},
			{Type: "Text", Props: vdom.Props{"content": "B", "flexGrow": 1}},
		},
	}
	ComputeLayout(node, vdom.Constraints{MinW: 0, MaxW: 80})
	if node.Children[0].Layout.W < 5 || node.Children[1].Layout.W < 5 {
		t.Errorf("grow children should have width, got w1=%d w2=%d",
			node.Children[0].Layout.W, node.Children[1].Layout.W)
	}
}

func TestLayoutScrollBox_contentHeight(t *testing.T) {
	node := &vdom.VNode{Type: "ScrollBox", Props: vdom.Props{"width": 80, "height": 10},
		Children: []vdom.VNode{
			{Type: "Text", Props: vdom.Props{"content": "a"}},
			{Type: "Text", Props: vdom.Props{"content": "b"}},
			{Type: "Text", Props: vdom.Props{"content": "c"}},
		},
	}
	ComputeLayout(node, vdom.Constraints{MinW: 0, MaxW: 80, MaxH: 10})
	if node.Layout.ContentH <= 0 {
		t.Error("expected positive ContentH")
	}
	if node.Layout.H != 10 {
		t.Errorf("expected viewport height 10, got %d", node.Layout.H)
	}
}
