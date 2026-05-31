package compo

import (
	"goc/gou/ink"
	"strconv"
	"strings"
)

func TranscriptScreen(ctx *ink.Context, msgs []ink.Message, searchQuery string, showAll bool) ink.VNode {
	children := make([]ink.VNode, 0)
	matchCount := 0
	for _, msg := range msgs {
		if searchQuery != "" {
			if !messageContains(msg, searchQuery) {
				continue
			}
			matchCount++
		}
		children = append(children, MessageRow(ctx, msg))
	}
	headerText := "TRANSCRIPT — / search  Esc close  ctrl+e show all"
	if searchQuery != "" {
		headerText = "Search: " + searchQuery + " (" + strconv.Itoa(matchCount) + " matches)"
	}
	return ink.VNode{
		Type: "Box", Key: "transcript-screen",
		Props: ink.Props{"direction": "column"},
		Children: append([]ink.VNode{
			{Type: "Text", Props: ink.Props{"content": headerText, "dim": true}},
		}, children...),
	}
}

func messageContains(msg ink.Message, query string) bool {
	for _, b := range msg.ContentBlocks {
		if strings.Contains(strings.ToLower(b.Content), strings.ToLower(query)) {
			return true
		}
		if strings.Contains(strings.ToLower(b.Name), strings.ToLower(query)) {
			return true
		}
	}
	return false
}
