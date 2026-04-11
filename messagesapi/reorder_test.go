package messagesapi

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func userMsg(text string) types.Message {
	raw, _ := json.Marshal(text)
	return types.Message{Type: types.MessageTypeUser, Content: raw}
}

func asstMsg() types.Message {
	raw, _ := json.Marshal("x")
	return types.Message{Type: types.MessageTypeAssistant, Content: raw}
}

func attMsg() types.Message {
	att, _ := json.Marshal(map[string]string{"type": "skill_listing", "content": "z"})
	return types.Message{Type: types.MessageTypeAttachment, Attachment: att}
}

func TestReorderAttachmentsForAPI_trailingAttAfterLastUserStaysAfterUser(t *testing.T) {
	// [ctx, u1, asst, u2, att] — trailing attachment must follow u2, not sit between asst and u2.
	msgs := []types.Message{
		userMsg("ctx"),
		userMsg("first"),
		asstMsg(),
		userMsg("second"),
		attMsg(),
	}
	out := reorderAttachmentsForAPI(msgs)
	if len(out) != len(msgs) {
		t.Fatalf("len got %d want %d", len(out), len(msgs))
	}
	// After reverse, expect ... asst, u2, att in chronological order.
	want := []int{0, 1, 2, 3, 4}
	for j, wi := range want {
		if out[j].Type != msgs[wi].Type {
			t.Fatalf("idx %d type got %s want %s", j, out[j].Type, msgs[wi].Type)
		}
	}
}

func TestReorderAttachmentsForAPI_singleUserAfterAssistantStillBubbles(t *testing.T) {
	// [asst, u, att] — one user after assistant; attachment bubbles to assistant (forward: asst, att, u).
	msgs := []types.Message{
		asstMsg(),
		userMsg("only"),
		attMsg(),
	}
	out := reorderAttachmentsForAPI(msgs)
	if len(out) != 3 {
		t.Fatalf("len %d", len(out))
	}
	if out[0].Type != types.MessageTypeAssistant || out[1].Type != types.MessageTypeAttachment || out[2].Type != types.MessageTypeUser {
		t.Fatalf("want [assistant, attachment, user], got types %s %s %s", out[0].Type, out[1].Type, out[2].Type)
	}
}
