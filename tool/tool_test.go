package tool

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestToolMatchesName(t *testing.T) {
	if !ToolMatchesName("Read", []string{"Cat"}, "Cat") {
		t.Fatal("alias should match")
	}
	if ToolMatchesName("Read", nil, "Write") {
		t.Fatal("should not match")
	}
}

func TestFindToolSpecByName(t *testing.T) {
	specs := []types.ToolSpec{
		{Name: "Grep", Aliases: []string{"grep"}},
		{Name: "Glob"},
	}
	if p := FindToolSpecByName(specs, "grep"); p == nil || p.Name != "Grep" {
		t.Fatalf("got %+v", p)
	}
	if FindToolSpecByName(specs, "None") != nil {
		t.Fatal("expected nil")
	}
}

func TestFilterToolProgressMessages(t *testing.T) {
	hookData, _ := json.Marshal(map[string]string{"type": "hook_progress"})
	toolData, _ := json.Marshal(map[string]string{"type": "bash_progress"})
	msgs := []types.Message{
		{Type: types.MessageTypeProgress, Data: hookData},
		{Type: types.MessageTypeProgress, Data: toolData},
	}
	out := FilterToolProgressMessages(msgs)
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
}

func TestEmptyToolPermissionContextData(t *testing.T) {
	ctx := types.EmptyToolPermissionContextData()
	if ctx.Mode != types.PermissionDefault || ctx.IsBypassPermissionsModeAvailable {
		t.Fatalf("%+v", ctx)
	}
}
