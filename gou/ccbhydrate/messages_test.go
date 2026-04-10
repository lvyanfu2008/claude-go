package ccbhydrate

import (
	"encoding/json"
	"testing"

	"goc/messagesapi"
	"goc/types"
)

func TestMessagesJSONWithSkillListing_prependsUser(t *testing.T) {
	rawText, _ := json.Marshal("hi")
	msgs := []types.Message{
		{Type: types.MessageTypeUser, UUID: "1", Content: rawText},
	}
	listing := "<system-reminder>\nx\n</system-reminder>"
	out, err := MessagesJSONWithSkillListing(msgs, listing, nil, messagesapi.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	var arr []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 {
		t.Fatalf("len=%d", len(arr))
	}
	if arr[0].Role != "user" {
		t.Fatal(arr[0].Role)
	}
	var s string
	if err := json.Unmarshal(arr[0].Content, &s); err != nil {
		t.Fatal(err)
	}
	if s != listing {
		t.Fatalf("got %q", s)
	}
}

func TestMessagesJSONWithSkillListing_emptyListing(t *testing.T) {
	rawText, _ := json.Marshal("hi")
	msgs := []types.Message{{Type: types.MessageTypeUser, UUID: "1", Content: rawText}}
	a, _ := MessagesJSONNormalized(msgs, nil, messagesapi.DefaultOptions())
	b, err := MessagesJSONWithSkillListing(msgs, "", nil, messagesapi.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("a=%s b=%s", a, b)
	}
}

func TestPrependUserMessageJSON_prepends(t *testing.T) {
	rawText, _ := json.Marshal("hi")
	msgs := []types.Message{
		{Type: types.MessageTypeUser, UUID: "1", Content: rawText},
	}
	base, err := MessagesJSONNormalized(msgs, nil, messagesapi.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	out, err := PrependUserMessageJSON(base, "<system-reminder>\nctx\n</system-reminder>")
	if err != nil {
		t.Fatal(err)
	}
	var arr []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 || arr[0].Role != "user" {
		t.Fatalf("%+v", arr)
	}
	var s string
	if err := json.Unmarshal(arr[0].Content, &s); err != nil {
		t.Fatal(err)
	}
	if s != "<system-reminder>\nctx\n</system-reminder>" {
		t.Fatalf("got %q", s)
	}
}

func TestPrependUserMessageJSON_emptyTextNoop(t *testing.T) {
	rawText, _ := json.Marshal("hi")
	msgs := []types.Message{{Type: types.MessageTypeUser, UUID: "1", Content: rawText}}
	base, _ := MessagesJSONNormalized(msgs, nil, messagesapi.DefaultOptions())
	out, err := PrependUserMessageJSON(base, "   ")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(base) {
		t.Fatalf("got %s want %s", out, base)
	}
}
