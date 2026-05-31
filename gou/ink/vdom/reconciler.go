package vdom

// EffectTag classifies the type of DOM operation a fiber represents.
type EffectTag int

const (
	NoEffect    EffectTag = iota
	Placement             // new node inserted
	Update                // props changed on existing node
	Deletion              // node removed
	Replacement           // node type or key changed (unmount + mount)
)

// FiberReconciler implements a key-based diffing algorithm that produces a
// Fiber tree annotated with effect tags.  It mirrors React's coarse-grained
// reconciliation strategy: same position + same key + same type => update in
// place; different type or key => replacement; unmatched old children =>
// deletion; unmatched new children => placement.
type FiberReconciler struct{}

// Reconcile computes the diff between oldRoot and newRoot and returns a
// Fiber tree annotated with the minimal set of effects needed to bring the
// rendered output in line with newRoot.
func (r *FiberReconciler) Reconcile(oldRoot, newRoot *VNode) *Fiber {
	if oldRoot == nil {
		// Entire tree is new.
		f := &Fiber{vnode: newRoot, effectTag: Placement}
		r.buildFiberTree(f, newRoot)
		return f
	}
	if oldRoot.Type != newRoot.Type || oldRoot.Key != newRoot.Key {
		// Root changed type or key — full replacement.
		f := &Fiber{vnode: newRoot, effectTag: Replacement}
		r.buildFiberTree(f, newRoot)
		return f
	}
	// Same root — diff children.
	root := &Fiber{vnode: newRoot}
	r.reconcileChildren(root, oldRoot.Children, newRoot.Children)
	return root
}

// buildFiberTree constructs a flat Fiber tree (child + sibling pointers) from
// a VNode tree, tagging every fiber with Placement.
func (r *FiberReconciler) buildFiberTree(fiber *Fiber, vnode *VNode) {
	for i := range vnode.Children {
		child := &Fiber{
			vnode:       &vnode.Children[i],
			effectTag:   Placement,
			returnFiber: fiber,
		}
		if i == 0 {
			fiber.child = child
		} else {
			s := fiber.child
			for s.sibling != nil {
				s = s.sibling
			}
			s.sibling = child
		}
		r.buildFiberTree(child, &vnode.Children[i])
	}
}

// reconcileChildren diffs oldKids against newKids by key, producing child
// fibers with the appropriate effect tags.  Children without keys always
// produce new placements.
func (r *FiberReconciler) reconcileChildren(parent *Fiber, oldKids, newKids []VNode) {
	// Build key -> index lookups.
	oldByKey := make(map[string]int)
	for i, v := range oldKids {
		if v.Key != "" {
			oldByKey[v.Key] = i
		}
	}
	newByKey := make(map[string]int)
	for i, v := range newKids {
		if v.Key != "" {
			newByKey[v.Key] = i
		}
	}

	var prevSibling *Fiber
	for i, newV := range newKids {
		var childFiber *Fiber

		if oldIdx, ok := oldByKey[newV.Key]; ok && newV.Key != "" {
			// Matched by key.
			oldV := &oldKids[oldIdx]
			if oldV.Type != newV.Type {
				// Type changed — replace.
				childFiber = &Fiber{vnode: &newV, effectTag: Replacement, returnFiber: parent}
				r.buildFiberTree(childFiber, &newV)
			} else {
				// Same type — may need update.
				childFiber = &Fiber{vnode: &newV, returnFiber: parent}
				if propsChanged(oldV, &newV) {
					childFiber.effectTag = Update
				}
				r.reconcileChildren(childFiber, oldV.Children, newV.Children)
			}
		} else {
			// No key match (or unkeyed) — new child.
			childFiber = &Fiber{vnode: &newV, effectTag: Placement, returnFiber: parent}
			r.buildFiberTree(childFiber, &newV)
		}

		if i == 0 {
			parent.child = childFiber
		} else {
			prevSibling.sibling = childFiber
		}
		prevSibling = childFiber
	}

	// Mark deletions for old keyed children not present in new.
	for i, oldV := range oldKids {
		if oldV.Key == "" {
			continue
		}
		if _, ok := newByKey[oldV.Key]; !ok {
			del := &Fiber{
				vnode:       &oldKids[i],
				effectTag:   Deletion,
				returnFiber: parent,
				deleted:     true,
			}
			if parent.child == nil {
				parent.child = del
			} else {
				s := parent.child
				for s.sibling != nil {
					s = s.sibling
				}
				s.sibling = del
			}
		}
	}
}

// propsChanged reports whether any layout- or content-relevant prop differs
// between oldV and newV.  It checks a fixed set of known keys.
func propsChanged(oldV, newV *VNode) bool {
	if oldV.Props == nil && newV.Props == nil {
		return false
	}
	if oldV.Props == nil || newV.Props == nil {
		return true
	}
	keys := []string{
		"content", "bold", "dim", "italic", "underline",
		"width", "height", "flexGrow", "direction", "stickyBottom",
		"color", "bg", "padding", "gap",
		"alignItems", "justifyContent", "minWidth",
	}
	for _, k := range keys {
		if oldV.Props[k] != newV.Props[k] {
			return true
		}
	}
	return false
}
