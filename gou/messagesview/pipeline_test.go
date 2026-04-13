package messagesview

import (
	"encoding/json"
	"fmt"
	"testing"

	"goc/types"
)

func TestMessagesForScrollList_dropsProgressAndNullAttach(t *testing.T) {
	meta := true
	hookOK, _ := json.Marshal(map[string]any{"type": "hook_success"})
	msgs := []types.Message{
		{Type: types.MessageTypeUser, UUID: "u1", Content: []byte(`[{"type":"text","text":"hi"}]`)},
		{Type: types.MessageTypeProgress, UUID: "p1", Data: json.RawMessage(`{}`)},
		{Type: types.MessageTypeAttachment, UUID: "a1", Attachment: hookOK},
		{Type: types.MessageTypeUser, UUID: "u_meta", Content: []byte(`[{"type":"text","text":"x"}]`), IsMeta: &meta},
	}
	got := MessagesForScrollList(msgs, ScrollListOpts{TranscriptMode: true, ShowAllInTranscript: true, VirtualScrollEnabled: true})
	if len(got) != 1 || got[0].UUID != "u1" {
		t.Fatalf("got %v", uuids(got))
	}
}

func TestShouldShowUserMessage_transcriptOnlyUserHiddenInPrompt(t *testing.T) {
	tr := true
	msg := types.Message{Type: types.MessageTypeUser, UUID: "u", Content: []byte(`[]`), IsVisibleInTranscriptOnly: &tr}
	if !ShouldShowUserMessage(msg, true) {
		t.Fatal("visible in transcript")
	}
	if ShouldShowUserMessage(msg, false) {
		t.Fatal("hidden on prompt screen")
	}
}

func TestMessagesForScrollList_transcriptTailWhenVirtualOff(t *testing.T) {
	var msgs []types.Message
	for i := 0; i < 35; i++ {
		msgs = append(msgs, types.Message{
			Type:    types.MessageTypeUser,
			UUID:    fmt.Sprintf("u%d", i),
			Content: []byte(`[{"type":"text","text":"x"}]`),
		})
	}
	got := MessagesForScrollList(msgs, ScrollListOpts{
		TranscriptMode:       true,
		ShowAllInTranscript:  false,
		VirtualScrollEnabled: false,
	})
	if len(got) != MaxTranscriptMessagesWithoutVirtualScroll {
		t.Fatalf("len=%d want %d", len(got), MaxTranscriptMessagesWithoutVirtualScroll)
	}
	if got[0].UUID != "u5" {
		t.Fatalf("first kept uuid=%s want u5 (last 30 of u0..u34)", got[0].UUID)
	}
}

func TestMessagesForScrollList_noTailWhenVirtualOn(t *testing.T) {
	var msgs []types.Message
	for i := 0; i < 35; i++ {
		msgs = append(msgs, types.Message{
			Type:    types.MessageTypeUser,
			UUID:    fmt.Sprintf("u%d", i),
			Content: []byte(`[{"type":"text","text":"x"}]`),
		})
	}
	got := MessagesForScrollList(msgs, ScrollListOpts{
		TranscriptMode:       true,
		ShowAllInTranscript:  false,
		VirtualScrollEnabled: true,
	})
	if len(got) != 35 {
		t.Fatalf("len=%d want full list when virtual on", len(got))
	}
}
