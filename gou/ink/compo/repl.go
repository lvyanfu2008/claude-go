package compo

import "goc/gou/ink"

// REPL renders the full REPL layout: messages, status line, and prompt.
// In transcript mode (uiScreen="transcript"), it shows the frozen transcript view.
func REPL(ctx *ink.Context, p ink.Props) ink.VNode {
	uiScreen := ctx.Store.GetMeta("uiScreen")
	searchQuery := ctx.Store.GetMeta("transcriptSearchQuery")
	showAll := ctx.Store.GetMeta("transcriptShowAll") == "1"

	if uiScreen == "transcript" {
		return TranscriptScreen(ctx, ctx.Store.GetMessages(), searchQuery, showAll)
	}

	// Build message rows as direct VirtualList children so each row
	// has its own height entry in VirtualScrollState.
	msgRows := MessageRows(ctx)
	msgCount := len(msgRows)
	if msgCount == 0 {
		msgCount = 1 // ensure at least one row so VirtualScrollState is valid
	}

	viewportH := ctx.Store.Height() - 3 // status line + prompt area
	if viewportH < 5 {
		viewportH = 5
	}

	vs := ink.NewVirtualScrollState(msgCount, viewportH)
	vs.ScrollTop = ctx.Store.ScrollTop()
	vs.StickyBottom = ctx.Store.GetMeta("stickyBottom") != "0"

	kids := []ink.VNode{
		{
			Type: "VirtualList", Key: "messages-scroll",
			Props: ink.Props{"stickyBottom": true, "flexGrow": 1, "virtualScroll": vs},
			Children: msgRows,
		},
		StatusLine(ctx),
		Spinner(ctx, ctx.Store.IsLoading(), "Thinking..."),
		PromptInput(ctx),
	}
	return ink.VNode{
		Type: "Box", Key: "repl",
		Props: ink.Props{"direction": "column"},
		Children: kids,
	}
}
