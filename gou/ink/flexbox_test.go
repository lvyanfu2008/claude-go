package ink

import "testing"

func TestLayoutText_simple(t *testing.T) {
	node := &VNode{Type: "Text", Props: Props{"content": "hello"}}
	ComputeLayout(node, Constraints{MinW: 0, MaxW: 80})
	if node.Layout.W < 1 || node.Layout.H != 1 {
		t.Errorf("expected w>0, h=1; got w=%d, h=%d", node.Layout.W, node.Layout.H)
	}
}

func TestLayoutText_wrap(t *testing.T) {
	node := &VNode{Type: "Text", Props: Props{"content": "hello world this is a long line"}}
	ComputeLayout(node, Constraints{MinW: 0, MaxW: 10})
	if node.Layout.H < 2 {
		t.Errorf("expected multi-line wrap, got h=%d", node.Layout.H)
	}
}

func TestLayoutBox_childrenPositioned(t *testing.T) {
	node := &VNode{Type: "Box", Props: Props{"width": 80},
		Children: []VNode{
			{Type: "Text", Props: Props{"content": "line1"}},
			{Type: "Text", Props: Props{"content": "line2"}},
		},
	}
	ComputeLayout(node, Constraints{MinW: 0, MaxW: 80})
	if node.Children[0].Layout.Y >= node.Children[1].Layout.Y {
		t.Error("child 1 should be above child 2")
	}
	if node.Layout.H < 2 {
		t.Errorf("box height >= 2, got %d", node.Layout.H)
	}
}

func TestLayoutBox_rowDirection(t *testing.T) {
	node := &VNode{Type: "Box", Props: Props{"width": 60, "direction": "row", "gap": 1},
		Children: []VNode{
			{Type: "Text", Props: Props{"content": "A"}},
			{Type: "Text", Props: Props{"content": "BB"}},
		},
	}
	ComputeLayout(node, Constraints{MinW: 0, MaxW: 80})
	if node.Children[0].Layout.X >= node.Children[1].Layout.X {
		t.Error("in row direction, child1.x < child2.x")
	}
}

func TestLayoutBox_flexGrow(t *testing.T) {
	node := &VNode{Type: "Box", Props: Props{"width": 80, "direction": "row"},
		Children: []VNode{
			{Type: "Text", Props: Props{"content": "A", "flexGrow": 1}},
			{Type: "Text", Props: Props{"content": "B", "flexGrow": 1}},
		},
	}
	ComputeLayout(node, Constraints{MinW: 0, MaxW: 80})
	if node.Children[0].Layout.W < 5 || node.Children[1].Layout.W < 5 {
		t.Errorf("grow children should have width, got w1=%d w2=%d",
			node.Children[0].Layout.W, node.Children[1].Layout.W)
	}
}

func TestLayoutScrollBox_contentHeight(t *testing.T) {
	node := &VNode{Type: "ScrollBox", Props: Props{"width": 80, "height": 10},
		Children: []VNode{
			{Type: "Text", Props: Props{"content": "a"}},
			{Type: "Text", Props: Props{"content": "b"}},
			{Type: "Text", Props: Props{"content": "c"}},
		},
	}
	ComputeLayout(node, Constraints{MinW: 0, MaxW: 80, MaxH: 10})
	if node.Layout.ContentH <= 0 {
		t.Error("expected positive ContentH")
	}
	if node.Layout.H != 10 {
		t.Errorf("expected viewport height 10, got %d", node.Layout.H)
	}
}
