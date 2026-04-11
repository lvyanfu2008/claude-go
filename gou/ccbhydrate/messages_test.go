package ccbhydrate

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/messagesapi"
	"goc/types"
)

// userAPIContentPlainText returns user message content after NormalizeMessagesForAPI projection
// (JSON string or [{type:text,text:...},...]).
func userAPIContentPlainText(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []map[string]any
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatalf("content: %s", string(raw))
	}
	var parts []string
	for _, b := range blocks {
		if typ, _ := b["type"].(string); typ == "text" {
			parts = append(parts, stringFromAny(b["text"]))
		}
	}
	return strings.Join(parts, "\n")
}

func stringFromAny(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	default:
		b, _ := json.Marshal(x)
		var s string
		_ = json.Unmarshal(b, &s)
		return s
	}
}

func TestMessagesJSONWithLeadingMeta_preservesAssistantBetweenUsers(t *testing.T) {
	u1, _ := json.Marshal("first")
	u2, _ := json.Marshal("second")
	rawA, _ := json.Marshal("asst")
	msgs := []types.Message{
		{Type: types.MessageTypeUser, UUID: "1", Content: u1},
		{Type: types.MessageTypeAssistant, UUID: "2", Content: rawA},
		{Type: types.MessageTypeUser, UUID: "3", Content: u2},
	}
	ctx := "CTX"
	listing := "SKILL"
	out, err := MessagesJSONWithLeadingMeta(msgs, ctx, listing, nil, messagesapi.DefaultOptions())
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
	if len(arr) != 3 {
		t.Fatalf("want [user, assistant, user], got %d: %+v", len(arr), arr)
	}
	if arr[0].Role != "user" || arr[1].Role != "assistant" || arr[2].Role != "user" {
		t.Fatalf("roles: %+v", arr)
	}
	u0 := userAPIContentPlainText(t, arr[0].Content)
	if !strings.Contains(u0, "first") || !strings.Contains(u0, "CTX") {
		t.Fatalf("first user: %q", u0)
	}
	if userAPIContentPlainText(t, arr[1].Content) != "asst" {
		t.Fatalf("assistant body")
	}
	u2t := userAPIContentPlainText(t, arr[2].Content)
	if !strings.Contains(u2t, "second") || !strings.Contains(u2t, "SKILL") {
		t.Fatalf("last user: %q", u2t)
	}
	// TS processTextPrompt + normalize: client text blocks precede merged attachment (skill) text.
	if strings.Index(u2t, "second") >= strings.Index(u2t, "SKILL") {
		t.Fatalf("want skill listing after client text (TS attachment order), got %q", u2t)
	}
}

func TestMessagesJSONWithSkillListing_insertsAfterLastUser(t *testing.T) {
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
	if len(arr) != 1 || arr[0].Role != "user" {
		t.Fatalf("want 1 merged user message, got %+v", arr)
	}
	got := userAPIContentPlainText(t, arr[0].Content)
	want := "hi\n" + listing
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMessagesJSONWithLeadingMeta_tsOrder_contextThenUserThenSkill(t *testing.T) {
	rawText, _ := json.Marshal("hi")
	msgs := []types.Message{
		{Type: types.MessageTypeUser, UUID: "1", Content: rawText},
	}
	ctx := "<system-reminder>\nctx\n</system-reminder>"
	listing := "<system-reminder>\nskills\n</system-reminder>"
	out, err := MessagesJSONWithLeadingMeta(msgs, ctx, listing, nil, messagesapi.DefaultOptions())
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
	if len(arr) != 1 || arr[0].Role != "user" {
		t.Fatalf("want 1 merged user (TS normalizeMessagesForAPI), got len=%d", len(arr))
	}
	got := userAPIContentPlainText(t, arr[0].Content)
	want := ctx + "\n" + "hi\n" + listing
	if got != want {
		t.Fatalf("got %q want %q", got, want)
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
	if len(arr) != 1 || arr[0].Role != "user" {
		t.Fatalf("want 1 merged user, got %+v", arr)
	}
	s := userAPIContentPlainText(t, arr[0].Content)
	want := "<system-reminder>\nctx\n</system-reminder>\nhi"
	if s != want {
		t.Fatalf("got %q want %q", s, want)
	}
}

func TestInsertUserMessageAfterLastUserJSON_afterUserBeforeAssistant(t *testing.T) {
	rawU, _ := json.Marshal("user line")
	rawA, _ := json.Marshal("assistant line")
	msgs := []types.Message{
		{Type: types.MessageTypeUser, UUID: "1", Content: rawU},
		{Type: types.MessageTypeAssistant, UUID: "2", Content: rawA},
	}
	base, err := MessagesJSONNormalized(msgs, nil, messagesapi.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	out, err := InsertUserMessageAfterLastUserJSON(base, "<system-reminder>\nskills\n</system-reminder>")
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
		t.Fatalf("len=%d want 2 (merged user+listing, assistant)", len(arr))
	}
	if arr[0].Role != "user" || arr[1].Role != "assistant" {
		t.Fatalf("roles %+v", arr)
	}
	u := userAPIContentPlainText(t, arr[0].Content)
	want := "user line\n<system-reminder>\nskills\n</system-reminder>"
	if u != want {
		t.Fatalf("got %q want %q", u, want)
	}
}

func TestInsertUserMessageAfterLastUserJSON_noUserAppends(t *testing.T) {
	rawA, _ := json.Marshal("only assistant")
	msgs := []types.Message{{Type: types.MessageTypeAssistant, UUID: "1", Content: rawA}}
	base, err := MessagesJSONNormalized(msgs, nil, messagesapi.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	out, err := InsertUserMessageAfterLastUserJSON(base, "meta")
	if err != nil {
		t.Fatal(err)
	}
	var arr []struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(out, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 || arr[0].Role != "assistant" || arr[1].Role != "user" {
		t.Fatalf("%+v", arr)
	}
}

func TestMergeConsecutiveUserMessagesJSON_skipsNonStringPair(t *testing.T) {
	// Two adjacent users but second content is not a JSON string — do not merge.
	arr := []apiMessage{
		{Role: "user", Content: json.RawMessage(`"a"`)},
		{Role: "user", Content: json.RawMessage(`[{"type":"text","text":"b"}]`)},
	}
	raw, err := json.Marshal(arr)
	if err != nil {
		t.Fatal(err)
	}
	out, err := MergeConsecutiveUserMessagesJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	var got []apiMessage
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
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
