package compo

import (
	"goc/gou/ink"
	"strings"
)

func SlashPicker(ctx *ink.Context, commands []string, filter string, selected int) ink.VNode {
	children := make([]ink.VNode, 0)
	for i, cmd := range commands {
		if filter != "" && !strings.Contains(strings.ToLower(cmd), strings.ToLower(filter)) {
			continue
		}
		style := ink.Props{"content": "/" + cmd}
		if i == selected {
			style["bold"] = true
		}
		children = append(children, ink.VNode{Type: "Text", Key: cmd, Props: style})
	}
	return ink.VNode{
		Type: "Box", Key: "slash-picker",
		Props: ink.Props{"direction": "column"},
		Children: children,
	}
}
