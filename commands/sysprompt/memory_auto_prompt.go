package sysprompt

import (
	"goc/commands/featuregates"
	"goc/memdir"
)

// BuildAutoMemoryPrompt mirrors loadMemoryPrompt() (src/memdir/memdir.ts).
// Thin wrapper that resolves feature gates and delegates to memdir.BuildAutoMemoryPrompt.
func BuildAutoMemoryPrompt(o GouDemoSystemOpts) string {
	return memdir.BuildAutoMemoryPrompt(memdir.AutoMemoryPromptOpts{
		Cwd:                    o.Cwd,
		MemorySkipIndex:        o.MemorySkipIndex,
		KairosActive:           featuregates.Feature("KAIROS") && o.KairosActive,
		TeamMemActive:          featuregates.Feature("TEAMMEM"),
		EmbeddedSearchTools:    o.EmbeddedSearchTools,
		ReplModeEnabled:        o.ReplModeEnabled,
		MemorySearchPastContext: o.MemorySearchPastContext,
	})
}
