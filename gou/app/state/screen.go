package state

type ScreenMode int

const (
	ScreenPrompt    ScreenMode = iota
	ScreenTranscript
)

type Screen struct {
	Mode                              ScreenMode
	Frozen                            interface{}
	ShowAll                           bool
	DumpMode                          bool
	SuspendAltScreenForScrollbackDump bool
	PromptSavedScrollTop              int
	PromptSavedSticky                 bool
	EditorBusy                        bool
	EditorStatus                      string
	EditorGen                         int
	SearchOpen                        bool
	SearchQuery                       string
	SearchHits                        []int
	SearchCursor                      int
}
