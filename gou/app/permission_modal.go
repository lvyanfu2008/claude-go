package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"goc/tools/toolexecution"
)

// permissionModalModel handles the permission allow/deny dialog as a sub-model.
type permissionModalModel struct {
	active   bool
	toolName string
	input    string
	replyCh  chan permissionAskReply
}

func newPermissionModalModel() *permissionModalModel {
	return &permissionModalModel{}
}

func (pm *permissionModalModel) Activate(toolName, input string, replyCh chan permissionAskReply) {
	pm.active = true
	pm.toolName = toolName
	pm.input = input
	pm.replyCh = replyCh
}

func (pm *permissionModalModel) Dismiss() {
	pm.active = false
}

func (pm *permissionModalModel) IsActive() bool {
	return pm.active
}

func (pm *permissionModalModel) Update(msg tea.KeyPressMsg) {
	if !pm.active {
		return
	}
	switch msg.String() {
	case "y", "Y", "enter", "space":
		pm.active = false
		reply := permissionAskReply{dec: toolexecution.AllowDecision()}
		if pm.replyCh != nil {
			select {
			case pm.replyCh <- reply:
			default:
			}
		}
	case "n", "N", "esc", "q":
		pm.active = false
		reply := permissionAskReply{dec: toolexecution.DenyDecision("denied by user")}
		if pm.replyCh != nil {
			select {
			case pm.replyCh <- reply:
			default:
			}
		}
	case "d", "D":
		pm.active = false
		reply := permissionAskReply{dec: toolexecution.DenyDecision("always denied")}
		if pm.replyCh != nil {
			select {
			case pm.replyCh <- reply:
			default:
			}
		}
	}
}

func (pm *permissionModalModel) View(width int) string {
	if !pm.active {
		return ""
	}
	inputPreview := pm.input
	if len(inputPreview) > 400 {
		inputPreview = inputPreview[:400] + "..."
	}
	title := "─── Allow " + pm.toolName + "? " + strings.Repeat("─", max(1, width-len(pm.toolName)-18))
	body := inputPreview
	hint := "Y allow  N deny  D always deny  Esc"
	return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(title + "\n" + body + "\n" + hint)
}
