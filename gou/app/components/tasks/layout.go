// Package tasks provides task list rendering (inline task list view and layout
// calculations) for the gou-demo TUI.
package tasks

import (
	state "goc/gou/app/state"
)

// Deps provides model-level dependencies for task layout and rendering.
type Deps interface {
	LayoutHeight() int
	ScreenMode() state.ScreenMode
	TaskListIsVisible() bool
	TaskListView(maxDisplay, cols int) string
}

// TaskListViewMaxDisplay matches the line budget for View() task list after stream rows.
func TaskListViewMaxDisplay(deps Deps) int {
	h := deps.LayoutHeight()
	if h <= 10 {
		return 0
	}
	return min(10, max(3, h-14))
}

// TaskListViewReservedRows is the vertical space between the message pane and the status line
// that the task list can occupy.
func TaskListViewReservedRows(deps Deps) int {
	if deps.ScreenMode() == state.ScreenTranscript {
		return 0
	}
	if !deps.TaskListIsVisible() {
		return 0
	}
	md := TaskListViewMaxDisplay(deps)
	if md == 0 {
		return 2 // header + " … +…" (task_list.view maxDisplay=0)
	}
	return 2 + md
}

// RenderTaskList renders the inline task list (including gutter).
func RenderTaskList(deps Deps) string {
	if !deps.TaskListIsVisible() {
		return ""
	}
	maxDisplay := TaskListViewMaxDisplay(deps)
	return deps.TaskListView(maxDisplay, deps.LayoutHeight())
}
