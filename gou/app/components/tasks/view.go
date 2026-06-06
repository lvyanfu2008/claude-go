package tasks

import (
	"strings"
)

// No separate view struct needed for now — the task list rendering is handled by
// the taskListModel in the app package, which provides TaskListView through Deps.
// RenderTaskList in layout.go uses that method.
//
// This file reserves the convention for future extensions (e.g., a separate
// TasksRenderer struct analogous to messages.Renderer or input.InputRenderer).

// indentBlock applies gutter indentation to the given block (2-space prefix per line).
func indentBlock(block string) string {
	if block == "" {
		return ""
	}
	lines := strings.Split(block, "\n")
	for i, ln := range lines {
		if ln != "" {
			lines[i] = "  " + ln
		}
	}
	return strings.Join(lines, "\n")
}
