package main

import "goc/gou/ccbstream"

// ccbStreamEventNeedsFullHeightRebuild is false for assistant_delta: [ccbstream.Apply]
// only appends to store.StreamingText; message list keys and persisted row bodies are unchanged.
// Prompt View renders streaming markdown outside virtual-scroll keys (see main.go View).
func ccbStreamEventNeedsFullHeightRebuild(ev ccbstream.StreamEvent) bool {
	return ev.Type != "assistant_delta"
}
