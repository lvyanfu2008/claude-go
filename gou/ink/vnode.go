package ink

import "goc/gou/theme"

type VNode struct {
	Type     string
	Key      string
	Props    Props
	Children []VNode
	Layout   LayoutResult
}

type Props map[string]interface{}

func (p Props) GetString(key string) string {
	if v, ok := p[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (p Props) GetInt(key string) int {
	if v, ok := p[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return 0
}

func (p Props) GetBool(key string) bool {
	if v, ok := p[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func (p Props) Get(key string) interface{} {
	return p[key]
}

type LayoutResult struct {
	X, Y         int
	W, H         int
	ContentH     int
	OverflowTop  int
	VisibleRange [2]int
}

type Constraints struct {
	MinW, MaxW int
	MinH, MaxH int
}

func Unbounded() Constraints {
	return Constraints{MinW: 0, MaxW: 1<<31 - 1, MinH: 0, MaxH: 1<<31 - 1}
}

type Context struct {
	Theme    *theme.Palette
	Store    *Store
	schedule func()
}

type Component func(ctx *Context, props Props) VNode
