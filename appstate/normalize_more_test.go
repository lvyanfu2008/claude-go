package appstate

import (
	"testing"

	"goc/types"
)

func TestNormalizeAppState_settingsToolPermissionInitialMessage(t *testing.T) {
	msg := types.Message{}
	a := AppState{
		InitialMessage: &InitialMessage{Message: msg},
	}
	NormalizeAppState(&a)
	if len(a.Settings) == 0 {
		t.Fatal("settings")
	}
	if string(a.Settings) != "{}" {
		t.Fatalf("settings: %s", a.Settings)
	}
	if len(a.ToolPermissionContext.AlwaysAllowRules) == 0 {
		t.Fatal("tool permission rules")
	}
	if a.InitialMessage.AllowedPrompts == nil {
		t.Fatal("allowedPrompts slice")
	}
}
