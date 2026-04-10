package sessiontranscript

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestIsLoggableMessage_skipsProgress(t *testing.T) {
	if IsLoggableMessage(types.Message{Type: types.MessageTypeProgress, UUID: "x"}, "external") {
		t.Fatal("progress should not log")
	}
}

func TestCollectReplIDs(t *testing.T) {
	msg := json.RawMessage(`{"content":[{"type":"tool_use","id":"rid1","name":"REPL","input":{}}]}`)
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "a1", Message: msg},
	}
	ids := CollectReplIDs(msgs)
	if _, ok := ids["rid1"]; !ok {
		t.Fatalf("got %#v", ids)
	}
}

func TestCleanMessagesForExternal_stripsREPL(t *testing.T) {
	asst := json.RawMessage(`{"content":[{"type":"tool_use","id":"u1","name":"REPL","input":{}},{"type":"text","text":"hi"}]}`)
	user := json.RawMessage(`{"content":[{"type":"tool_result","tool_use_id":"u1","content":"x"}]}`)
	all := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "a1", Message: asst},
		{Type: types.MessageTypeUser, UUID: "u2", Message: user},
	}
	out := CleanMessagesForLogging(all, all, "external")
	if len(out) != 1 || out[0].Type != types.MessageTypeAssistant {
		t.Fatalf("got %#v", out)
	}
}

func TestIsLoggableMessage_attachmentExternalSkipped(t *testing.T) {
	att := json.RawMessage(`{"type":"image","data":"x"}`)
	if IsLoggableMessage(types.Message{Type: types.MessageTypeAttachment, UUID: "u", Attachment: att}, "external") {
		t.Fatal("external attachment should be skipped by default")
	}
}

func TestIsLoggableMessage_hookAdditionalContextWithEnv(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SAVE_HOOK_ADDITIONAL_CONTEXT", "1")
	att := json.RawMessage(`{"type":"hook_additional_context","text":"x"}`)
	if !IsLoggableMessage(types.Message{Type: types.MessageTypeAttachment, UUID: "u", Attachment: att}, "external") {
		t.Fatal("expected hook_additional_context when env set")
	}
}

func TestIsLoggableMessage_attachmentAnt(t *testing.T) {
	att := json.RawMessage(`{"type":"image"}`)
	if !IsLoggableMessage(types.Message{Type: types.MessageTypeAttachment, UUID: "u", Attachment: att}, "ant") {
		t.Fatal("ant keeps attachments")
	}
}
