package integration

import (
	"os"

	"goc/gou/ink"
	"goc/gou/ink/compo"
	"goc/gou/theme"
)

// RunNewEngine starts the new rendering engine. Activated by GOU_USE_NEW_ENGINE=1.
// Returns true if it handled execution (caller should return).
func RunNewEngine() bool {
	if os.Getenv("GOU_USE_NEW_ENGINE") != "1" {
		return false
	}

	term := ink.NewTerminal()
	store := ink.NewStore()

	pal := theme.ActivePalette()
	if pal == nil {
		pal = &theme.Palette{}
	}

	engine := ink.NewEngine(term, store, pal, compo.REPL)
	if err := engine.Start(); err != nil {
		panic(err)
	}
	defer engine.Stop()

	// Seed some test data so the screen is visible
	store.AppendMessage(ink.Message{
		UUID: "welcome", Type: "system",
		ContentBlocks: []ink.ContentBlock{{Type: "informational", Content: "Welcome to the new terminal engine! GOU_USE_NEW_ENGINE=1"}},
	})
	store.AppendMessage(ink.Message{
		UUID: "u1", Type: "user",
		ContentBlocks: []ink.ContentBlock{{Type: "text", Content: "Hello from the new rendering engine"}},
	})
	store.AppendMessage(ink.Message{
		UUID: "a1", Type: "assistant",
		ContentBlocks: []ink.ContentBlock{{Type: "text", Content: "This is rendered with the **new** Go+Ink terminal engine."}},
	})

	// Run until SIGINT
	select {}
}
