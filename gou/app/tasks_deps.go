package app

import (
	"goc/gou/app/components/tasks"
	state "goc/gou/app/state"
)

// tasksDeps adapts *model to tasks.Deps.
type tasksDeps struct {
	m *model
}

func (d tasksDeps) LayoutHeight() int           { return d.m.Layout.Height }
func (d tasksDeps) ScreenMode() state.ScreenMode { return d.m.Screen.Mode }

func (d tasksDeps) TaskListIsVisible() bool {
	if d.m.Agent.TaskList == nil {
		return false
	}
	return d.m.Agent.TaskList.(*taskListModel).isVisible()
}

func (d tasksDeps) TaskListView(maxDisplay, cols int) string {
	if d.m.Agent.TaskList == nil {
		return ""
	}
	return d.m.Agent.TaskList.(*taskListModel).view(maxDisplay, cols)
}

// Compile-time check
var _ tasks.Deps = tasksDeps{}
