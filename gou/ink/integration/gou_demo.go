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
	pal := loadPalette()

	// Define atoms for app state
	messagesAtom := ink.DefineAtom(store, "messages", []ink.Message{})
	streamingTextAtom := ink.DefineAtom(store, "streamingText", "")
	inputValueAtom := ink.DefineAtom(store, "inputValue", "")
	cursorPosAtom := ink.DefineAtom(store, "cursorPos", 0)
	isLoadingAtom := ink.DefineAtom(store, "isLoading", false)
	scrollTopAtom := ink.DefineAtom(store, "scrollTop", 0)
	termWAtom := ink.DefineAtom(store, "termWidth", 80)
	termHAtom := ink.DefineAtom(store, "termHeight", 24)

	// Suppress unused warnings — atoms are used by components via context
	_ = messagesAtom
	_ = streamingTextAtom
	_ = inputValueAtom
	_ = cursorPosAtom
	_ = isLoadingAtom
	_ = scrollTopAtom
	_ = termWAtom
	_ = termHAtom

	engine := ink.NewEngine(term, store, pal, compo.REPL)

	if err := engine.Terminal.Init(); err != nil {
		panic(err)
	}
	defer engine.Terminal.Shutdown()

	// Seed welcome message
	store.SetOnRender(func() {})

	if err := engine.Run(); err != nil {
		panic(err)
	}
	return true
}

func loadPalette() *theme.Palette {
	pal := theme.ActivePalette()
	if pal == nil {
		pal = &theme.Palette{}
	}
	return pal
}
