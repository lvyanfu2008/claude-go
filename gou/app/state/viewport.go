package state

import "charm.land/bubbles/v2/viewport"

type Viewport struct {
	Enabled               bool
	Model                 viewport.Model
	LastGeom              string
	LastContentSig        string
	NeedResizeContent     bool
	FoldAll               bool
	FoldRev               int
	HistoryBrowseMouseOff bool
}
