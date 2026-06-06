package state

type Layout struct {
	Width         int
	Height        int
	Cols          int
	MsgBodyCols   int
	MsgScrollbarW int
	TitleH        int
	StreamH       int
}

func NewLayout() *Layout {
	return &Layout{
		TitleH:  1,
		StreamH: 4,
	}
}
