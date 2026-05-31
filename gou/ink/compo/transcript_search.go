package compo

import "goc/gou/ink"

func TranscriptSearchBar(ctx *ink.Context, query string) ink.VNode {
	return ink.VNode{
		Type: "Box", Key: "transcript-search",
		Props: ink.Props{"direction": "row"},
		Children: []ink.VNode{
			{Type: "Text", Props: ink.Props{"content": "Search: ", "dim": true}},
			{Type: "Text", Props: ink.Props{"content": query, "bold": true}},
		},
	}
}
