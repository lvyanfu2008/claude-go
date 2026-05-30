package ink

type PatchKind int

const (
	PatchNone PatchKind = iota
	PatchCreate
	PatchUpdate
	PatchDelete
	PatchReplace
)

type Patch struct {
	Kind    PatchKind
	Index   int
	NewNode *VNode
}

type Reconciler struct{}

func (r *Reconciler) Diff(oldRoot, newRoot *VNode) []Patch {
	if oldRoot == nil {
		return []Patch{{Kind: PatchCreate, NewNode: newRoot}}
	}
	if oldRoot.Type != newRoot.Type || oldRoot.Key != newRoot.Key {
		return []Patch{{Kind: PatchReplace, NewNode: newRoot}}
	}
	return diffChildren(oldRoot.Children, newRoot.Children)
}

func diffChildren(oldKids, newKids []VNode) []Patch {
	var patches []Patch
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

	for i, newV := range newKids {
		if oldIdx, ok := oldByKey[newV.Key]; ok {
			oldV := &oldKids[oldIdx]
			if oldV.Type != newV.Type {
				patches = append(patches, Patch{Kind: PatchReplace, Index: i, NewNode: &newV})
			} else if propsChanged(oldV, &newV) {
				patches = append(patches, Patch{Kind: PatchUpdate, Index: i, NewNode: &newV})
			}
			childPatches := diffChildren(oldV.Children, newV.Children)
			patches = append(patches, childPatches...)
		} else {
			patches = append(patches, Patch{Kind: PatchCreate, Index: i, NewNode: &newV})
		}
	}

	for _, oldV := range oldKids {
		if oldV.Key == "" {
			continue
		}
		if _, ok := newByKey[oldV.Key]; !ok {
			for i := range oldKids {
				if oldKids[i].Key == oldV.Key {
					patches = append(patches, Patch{Kind: PatchDelete, Index: i})
					break
				}
			}
		}
	}
	return patches
}

func propsChanged(oldV, newV *VNode) bool {
	return oldV.Props.GetString("content") != newV.Props.GetString("content") ||
		oldV.Props.GetBool("bold") != newV.Props.GetBool("bold") ||
		oldV.Props.GetBool("dim") != newV.Props.GetBool("dim") ||
		oldV.Props.GetInt("width") != newV.Props.GetInt("width") ||
		oldV.Props.GetInt("height") != newV.Props.GetInt("height") ||
		oldV.Props.GetInt("flexGrow") != newV.Props.GetInt("flexGrow")
}
