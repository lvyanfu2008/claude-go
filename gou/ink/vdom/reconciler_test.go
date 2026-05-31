package vdom

import "testing"

func TestReconcilerSameTypeNoChange(t *testing.T) {
	old := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "hello"}},
	}}
	new := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "hello"}},
	}}
	r := &FiberReconciler{}
	root := r.Reconcile(old, new)
	if root.effectTag != NoEffect {
		t.Errorf("expected NoEffect, got %v", root.effectTag)
	}
}

func TestReconcilerContentChange(t *testing.T) {
	old := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "hello"}},
	}}
	new := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "world"}},
	}}
	r := &FiberReconciler{}
	root := r.Reconcile(old, new)
	hasEffect := false
	walkEffects(root, func(f *Fiber) {
		if f.effectTag != NoEffect {
			hasEffect = true
		}
	})
	if !hasEffect {
		t.Error("expected effect from content change")
	}
}

func TestReconcilerNewChild(t *testing.T) {
	old := &VNode{Type: "Box", Key: "root", Children: []VNode{}}
	new := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "new"}},
	}}
	r := &FiberReconciler{}
	root := r.Reconcile(old, new)
	hasPlacement := false
	walkEffects(root, func(f *Fiber) {
		if f.effectTag == Placement {
			hasPlacement = true
		}
	})
	if !hasPlacement {
		t.Error("expected Placement effect for new child")
	}
}

func TestReconcilerRemoveChild(t *testing.T) {
	old := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "x"}},
	}}
	new := &VNode{Type: "Box", Key: "root", Children: []VNode{}}
	r := &FiberReconciler{}
	root := r.Reconcile(old, new)
	hasDeletion := false
	walkEffects(root, func(f *Fiber) {
		if f.effectTag == Deletion {
			hasDeletion = true
		}
	})
	if !hasDeletion {
		t.Error("expected Deletion effect for removed child")
	}
}

func TestReconcilerTypeChangeReplaces(t *testing.T) {
	old := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "x"}},
	}}
	new := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Box", Key: "a", Props: Props{"direction": "row"}},
	}}
	r := &FiberReconciler{}
	root := r.Reconcile(old, new)
	hasReplace := false
	walkEffects(root, func(f *Fiber) {
		if f.effectTag == Replacement {
			hasReplace = true
		}
	})
	if !hasReplace {
		t.Error("expected Replacement effect for type change")
	}
}

func walkEffects(f *Fiber, fn func(*Fiber)) {
	if f.effectTag != NoEffect {
		fn(f)
	}
	for c := f.child; c != nil; c = c.sibling {
		walkEffects(c, fn)
	}
}
