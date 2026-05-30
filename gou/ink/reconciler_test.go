package ink

import "testing"

func TestReconciler_noChange(t *testing.T) {
	r := &Reconciler{}
	old := &VNode{Type: "Box", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "hello"}},
	}}
	new := &VNode{Type: "Box", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "hello"}},
	}}
	if len(r.Diff(old, new)) != 0 {
		t.Error("expected 0 patches")
	}
}

func TestReconciler_createChild(t *testing.T) {
	r := &Reconciler{}
	old := &VNode{Type: "Box", Children: []VNode{}}
	new := &VNode{Type: "Box", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "hello"}},
	}}
	if len(r.Diff(old, new)) == 0 {
		t.Error("expected Create patch")
	}
}

func TestReconciler_deleteChild(t *testing.T) {
	r := &Reconciler{}
	old := &VNode{Type: "Box", Children: []VNode{
		{Type: "Text", Key: "a"},
	}}
	new := &VNode{Type: "Box", Children: []VNode{}}
	found := false
	for _, p := range r.Diff(old, new) {
		if p.Kind == PatchDelete {
			found = true
		}
	}
	if !found {
		t.Error("expected Delete patch")
	}
}

func TestReconciler_updateContent(t *testing.T) {
	r := &Reconciler{}
	old := &VNode{Type: "Box", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "old"}},
	}}
	new := &VNode{Type: "Box", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "new"}},
	}}
	found := false
	for _, p := range r.Diff(old, new) {
		if p.Kind == PatchUpdate {
			found = true
		}
	}
	if !found {
		t.Error("expected Update patch")
	}
}
