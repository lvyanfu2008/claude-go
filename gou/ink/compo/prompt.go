package compo

import "goc/gou/ink"

// PromptInput renders the input prompt line.
func PromptInput(ctx *ink.Context) ink.VNode {
	prompt := "> "
	if ctx.Store.InputValue != "" {
		prompt += ctx.Store.InputValue
	}
	return ink.VNode{
		Type: "Text", Key: "prompt",
		Props: ink.Props{"content": prompt, "bold": true},
	}
}
