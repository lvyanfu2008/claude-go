package messagesview

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestReorderMessagesInUI_toolUseThenPreThenResultThenPost(t *testing.T) {
	assistantBlocks := []types.MessageContentBlock{
		{Type: "tool_use", ID: "call_1", Name: "Read", Input: json.RawMessage(`{}`)},
	}
	assistantContent, err := json.Marshal(assistantBlocks)
	if err != nil {
		t.Fatal(err)
	}
	preAtt, _ := json.Marshal(map[string]any{
		"type": "hook", "hookEvent": "PreToolUse", "toolUseId": "call_1", "detail": "pre",
	})
	postAtt, _ := json.Marshal(map[string]any{
		"type": "hook", "hookEvent": "PostToolUse", "tool_use_id": "call_1", "detail": "post",
	})
	resultBlocks := []types.MessageContentBlock{
		{Type: "tool_result", ToolUseID: "call_1", Content: json.RawMessage(`"done"`)},
	}
	resultContent, err := json.Marshal(resultBlocks)
	if err != nil {
		t.Fatal(err)
	}
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "a1", Content: assistantContent},
		{Type: types.MessageTypeUser, UUID: "u_res", Content: resultContent},
		{Type: types.MessageTypeAttachment, UUID: "att_pre", Attachment: preAtt},
		{Type: types.MessageTypeAttachment, UUID: "att_post", Attachment: postAtt},
		{Type: types.MessageTypeUser, UUID: "u_tail", Content: []byte(`[{"type":"text","text":"tail"}]`)},
	}
	got := ReorderMessagesInUI(msgs)
	if len(got) != 5 {
		t.Fatalf("len=%d want 5: %+v", len(got), uuids(got))
	}
	if got[0].UUID != "a1" || got[1].UUID != "att_pre" || got[2].UUID != "u_res" || got[3].UUID != "att_post" || got[4].UUID != "u_tail" {
		t.Fatalf("order got %v want a1, att_pre, u_res, att_post, u_tail", uuids(got))
	}
}

func TestReorderMessagesInUI_apiErrorsMiddleDroppedFinalKept(t *testing.T) {
	st := func(s string) *string { return &s }
	msgs := []types.Message{
		{Type: types.MessageTypeUser, UUID: "u1", Content: []byte(`[{"type":"text","text":"hi"}]`)},
		{Type: types.MessageTypeSystem, UUID: "e1", Subtype: st("api_error")},
		{Type: types.MessageTypeSystem, UUID: "e2", Subtype: st("api_error")},
		{Type: types.MessageTypeUser, UUID: "u2", Content: []byte(`[{"type":"text","text":"after"}]`)},
		{Type: types.MessageTypeSystem, UUID: "e3", Subtype: st("api_error")},
	}
	got := ReorderMessagesInUI(msgs)
	if len(got) != 3 {
		t.Fatalf("len=%d want [u1,u2,e3] only: %v", len(got), uuids(got))
	}
	if got[0].UUID != "u1" || got[1].UUID != "u2" || got[2].UUID != "e3" {
		t.Fatalf("got %v", uuids(got))
	}
	if got[2].Subtype == nil || *got[2].Subtype != "api_error" {
		t.Fatalf("last system row should be api_error, got subtype=%v", got[2].Subtype)
	}
}

func TestReorderMessagesInUI_orphanHookAttachmentDropped(t *testing.T) {
	preAtt, _ := json.Marshal(map[string]any{
		"type": "hook", "hookEvent": "PreToolUse", "toolUseId": "missing_id", "detail": "orphan",
	})
	msgs := []types.Message{
		{Type: types.MessageTypeAttachment, UUID: "orph", Attachment: preAtt},
	}
	got := ReorderMessagesInUI(msgs)
	if len(got) != 0 {
		t.Fatalf("orphan hook only: want empty, got %v", uuids(got))
	}
}

func TestReorderMessagesInUI_orphanHookBeforeUserKeepsUserOnly(t *testing.T) {
	preAtt, _ := json.Marshal(map[string]any{
		"type": "hook", "hookEvent": "PreToolUse", "toolUseId": "ghost", "detail": "orphan",
	})
	msgs := []types.Message{
		{Type: types.MessageTypeAttachment, UUID: "orph", Attachment: preAtt},
		{Type: types.MessageTypeUser, UUID: "u1", Content: []byte(`[{"type":"text","text":"ok"}]`)},
	}
	got := ReorderMessagesInUI(msgs)
	if len(got) != 1 || got[0].UUID != "u1" {
		t.Fatalf("got %v", uuids(got))
	}
}

func uuids(ms []types.Message) []string {
	out := make([]string, len(ms))
	for i := range ms {
		out[i] = ms[i].UUID
	}
	return out
}
