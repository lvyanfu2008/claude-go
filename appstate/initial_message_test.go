package appstate

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestInitialMessage_roundTrip(t *testing.T) {
	mode := types.PermissionPlan
	clear := true
	im := &InitialMessage{
		Message: types.Message{
			Type: types.MessageTypeUser,
			UUID: "u1",
		},
		ClearContext: &clear,
		Mode:         &mode,
		AllowedPrompts: []AllowedPrompt{
			{Tool: "Bash", Prompt: "run tests"},
		},
	}
	b, err := json.Marshal(im)
	if err != nil {
		t.Fatal(err)
	}
	var back InitialMessage
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Message.UUID != "u1" || back.Message.Type != types.MessageTypeUser {
		t.Fatalf("%+v", back.Message)
	}
	if back.Mode == nil || *back.Mode != types.PermissionPlan {
		t.Fatal("mode")
	}
	if len(back.AllowedPrompts) != 1 || back.AllowedPrompts[0].Prompt != "run tests" {
		t.Fatalf("%+v", back.AllowedPrompts)
	}
}
