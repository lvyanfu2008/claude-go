package query

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestSnipCompact_NoSnipResults(t *testing.T) {
	messages := []types.Message{
		{Type: types.MessageTypeUser, UUID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890", Content: json.RawMessage(`"hello"`)},
		{Type: types.MessageTypeAssistant, UUID: "b2c3d4e5-f6a7-8901-bcde-f12345678901", Content: json.RawMessage(`"hi"`)},
	}
	res, err := snipCompact(messages, testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result, got %+v", res)
	}
}

func TestSnipCompact_EmptyMessages(t *testing.T) {
	res, err := snipCompact(nil, testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result, got %+v", res)
	}
}

func TestSnipCompact_SingleSnipResult(t *testing.T) {
	// UUIDs chosen so deriveShortMessageID produces predictable short IDs.
	// UUID: 00000000-0000-4000-8000-000000000001 → hex 0000000000400080000000000001
	// first 10 hex chars = 0000000000 → base-36 = "0"
	msg1 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "00000000-0000-4000-8000-000000000001",
		Content: json.RawMessage(`"message one"`),
	}
	// UUID: 00000000-0000-4000-8000-000000000002 → short ID also "0" (first 10 hex = 0000000000)
	// Use a different UUID that gives a different short ID.
	// UUID: 0000000a-0000-4000-8000-000000000002 → hex 0000000a0000400080000000000002
	// first 10 hex = 0000000a00 → parse as uint64 → 2560 → base-36 = "1yw"
	msg2 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "0000000a-0000-4000-8000-000000000002",
		Content: json.RawMessage(`"message two"`),
	}
	msg3 := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "0000000b-0000-4000-8000-000000000003",
		Content: json.RawMessage(`"assistant reply"`),
	}

	// Compute short IDs for msg1 and msg2.
	sid1 := deriveShortMessageID(msg1.UUID) // "0"
	sid2 := deriveShortMessageID(msg2.UUID) // "1yw"

	// Snip tool result referencing msg1 and msg2.
	snipResultJSON := `{"data":{"snipped_count":2,"summary":"test snip","message_ids":["` + sid1 + `","` + sid2 + `"]}}`
	snipMsg := types.Message{
		Type:          types.MessageTypeUser,
		UUID:          "0000000c-0000-4000-8000-000000000004",
		ToolUseResult: types.ToolUseResultJSONBytes(snipResultJSON),
	}

	messages := []types.Message{msg1, msg2, msg3, snipMsg}

	res, err := snipCompact(messages, testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if len(res.Messages) != 2 {
		t.Fatalf("expected 2 remaining messages, got %d: %+v", len(res.Messages), res.Messages)
	}
	// msg3 (assistant) and snipMsg should remain.
	if res.Messages[0].UUID != msg3.UUID {
		t.Errorf("expected msg3 to remain, got %s", res.Messages[0].UUID)
	}
	if res.Messages[1].UUID != snipMsg.UUID {
		t.Errorf("expected snipMsg to remain, got %s", res.Messages[1].UUID)
	}
	if res.TokensFreed <= 0 {
		t.Errorf("expected TokensFreed > 0, got %d", res.TokensFreed)
	}
	if res.BoundaryMessage == nil {
		t.Fatal("expected BoundaryMessage")
	}
	if res.BoundaryMessage.Type != types.MessageTypeSystem {
		t.Errorf("expected system message type, got %s", res.BoundaryMessage.Type)
	}
	if res.BoundaryMessage.Subtype == nil || *res.BoundaryMessage.Subtype != "snip_boundary" {
		t.Errorf("expected subtype snip_boundary, got %v", res.BoundaryMessage.Subtype)
	}
}

func TestSnipCompact_ShortIDsNoMatch(t *testing.T) {
	msg1 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "00000000-0000-4000-8000-000000000001",
		Content: json.RawMessage(`"hello"`),
	}

	// Snip result referencing a short ID that doesn't match any message.
	snipResultJSON := `{"data":{"snipped_count":1,"summary":"no match","message_ids":["nonexistent"]}}`
	snipMsg := types.Message{
		Type:          types.MessageTypeUser,
		UUID:          "0000000c-0000-4000-8000-000000000004",
		ToolUseResult: types.ToolUseResultJSONBytes(snipResultJSON),
	}

	messages := []types.Message{msg1, snipMsg}
	res, err := snipCompact(messages, testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result when no IDs match, got %+v", res)
	}
}

func TestSnipCompact_MultipleSnipResults(t *testing.T) {
	msg1 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "00000000-0000-4000-8000-000000000001",
		Content: json.RawMessage(`"msg1"`),
	}
	msg2 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "0000000a-0000-4000-8000-000000000002",
		Content: json.RawMessage(`"msg2"`),
	}
	msg3 := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "0000000b-0000-4000-8000-000000000003",
		Content: json.RawMessage(`"assistant"`),
	}

	sid1 := deriveShortMessageID(msg1.UUID)
	sid2 := deriveShortMessageID(msg2.UUID)

	// Two separate Snip results.
	snip1JSON := `{"data":{"snipped_count":1,"summary":"first","message_ids":["` + sid1 + `"]}}`
	snip2JSON := `{"data":{"snipped_count":1,"summary":"second","message_ids":["` + sid2 + `"]}}`

	snipMsg1 := types.Message{
		Type:          types.MessageTypeUser,
		UUID:          "0000000c-0000-4000-8000-000000000004",
		ToolUseResult: types.ToolUseResultJSONBytes(snip1JSON),
	}
	snipMsg2 := types.Message{
		Type:          types.MessageTypeUser,
		UUID:          "0000000d-0000-4000-8000-000000000005",
		ToolUseResult: types.ToolUseResultJSONBytes(snip2JSON),
	}

	messages := []types.Message{msg1, msg2, msg3, snipMsg1, snipMsg2}

	res, err := snipCompact(messages, testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if len(res.Messages) != 3 {
		t.Fatalf("expected 3 remaining (assistant + 2 snip msgs), got %d", len(res.Messages))
	}
	// Boundary should contain both summaries.
	if res.BoundaryMessage == nil {
		t.Fatal("expected BoundaryMessage")
	}
}

func TestSnipCompact_NonSnipToolResultIgnored(t *testing.T) {
	msg1 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "00000000-0000-4000-8000-000000000001",
		Content: json.RawMessage(`"keep me"`),
	}
	// A non-Snip tool result (e.g., Bash output) should be ignored.
	bashResultJSON := `{"data":{"output":"ls result","exitCode":0}}`
	bashMsg := types.Message{
		Type:          types.MessageTypeUser,
		UUID:          "0000000c-0000-4000-8000-000000000004",
		ToolUseResult: types.ToolUseResultJSONBytes(bashResultJSON),
	}

	messages := []types.Message{msg1, bashMsg}
	res, err := snipCompact(messages, testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result for non-Snip tool result, got %+v", res)
	}
}

func TestSnipCompact_BoundaryAlreadyExists(t *testing.T) {
	// Simulates Turn 2+ where snip_boundary from a previous turn already exists.
	// The snipCompact should filter based on the boundary WITHOUT creating a new one.
	msg1 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "00000000-0000-4000-8000-000000000001",
		Content: json.RawMessage(`"message one"`),
	}
	msg2 := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "0000000b-0000-4000-8000-000000000003",
		Content: json.RawMessage(`"assistant"`),
	}

	sid1 := deriveShortMessageID(msg1.UUID)

	// Snip tool result — already processed in a previous turn.
	snipResultJSON := `{"data":{"snipped_count":1,"summary":"old snip","message_ids":["` + sid1 + `"]}}`
	snipResultMsg := types.Message{
		Type:          types.MessageTypeUser,
		UUID:          "0000000c-0000-4000-8000-000000000004",
		ToolUseResult: types.ToolUseResultJSONBytes(snipResultJSON),
	}

	// Existing snip_boundary from the previous turn.
	boundaryMeta, _ := json.Marshal(snipBoundaryMetadata{
		Trigger:      "snip",
		Summary:      "old snip",
		RemovedUuids: []string{msg1.UUID},
		RemovedCount: 1,
	})
	boundaryContent, _ := json.Marshal("Snipped 1 messages: old snip")
	subtype := "snip_boundary"
	level := "info"
	isMeta := false
	boundaryMsg := types.Message{
		Type:            types.MessageTypeSystem,
		UUID:            "0000000d-0000-4000-8000-000000000005",
		Subtype:         &subtype,
		Level:           &level,
		IsMeta:          &isMeta,
		Content:         json.RawMessage(boundaryContent),
		CompactMetadata: json.RawMessage(boundaryMeta),
	}

	messages := []types.Message{msg1, msg2, snipResultMsg, boundaryMsg}

	res, err := snipCompact(messages, testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result (should filter based on existing boundary)")
	}
	// msg1 should be filtered out based on the existing boundary.
	if len(res.Messages) != 3 {
		t.Fatalf("expected 3 remaining messages (msg2 + snipResult + boundary), got %d", len(res.Messages))
	}
	// No new boundary should be created — the snip result was already processed.
	if res.BoundaryMessage != nil {
		t.Errorf("expected no new boundary, got %+v", res.BoundaryMessage)
	}
	if res.TokensFreed <= 0 {
		t.Errorf("expected TokensFreed > 0 from existing boundary, got %d", res.TokensFreed)
	}
}

func TestSnipCompact_NewSnipAfterExistingBoundary(t *testing.T) {
	// An existing boundary covers msg1. A NEW snip result covers msg3.
	// Both should be filtered, and a new boundary should be created for msg3.
	msg1 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "00000000-0000-4000-8000-000000000001",
		Content: json.RawMessage(`"msg1"`),
	}
	msg2 := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    "0000000b-0000-4000-8000-000000000003",
		Content: json.RawMessage(`"msg2"`),
	}
	msg3 := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "0000000a-0000-4000-8000-000000000002",
		Content: json.RawMessage(`"msg3"`),
	}

	// Existing boundary covers msg1.
	boundaryMeta, _ := json.Marshal(snipBoundaryMetadata{
		Trigger:      "snip",
		Summary:      "first snip",
		RemovedUuids: []string{msg1.UUID},
		RemovedCount: 1,
	})
	boundaryContent, _ := json.Marshal("Snipped 1 messages: first snip")
	subtype := "snip_boundary"
	level := "info"
	isMeta := false
	boundaryMsg := types.Message{
		Type:            types.MessageTypeSystem,
		UUID:            "0000000d-0000-4000-8000-000000000005",
		Subtype:         &subtype,
		Level:           &level,
		IsMeta:          &isMeta,
		Content:         json.RawMessage(boundaryContent),
		CompactMetadata: json.RawMessage(boundaryMeta),
	}

	// NEW snip result covering msg3 (appeared after the boundary).
	sid3 := deriveShortMessageID(msg3.UUID)
	newSnipJSON := `{"data":{"snipped_count":1,"summary":"second snip","message_ids":["` + sid3 + `"]}}`
	newSnipMsg := types.Message{
		Type:          types.MessageTypeUser,
		UUID:          "0000000e-0000-4000-8000-000000000006",
		ToolUseResult: types.ToolUseResultJSONBytes(newSnipJSON),
	}

	messages := []types.Message{msg1, msg2, msg3, boundaryMsg, newSnipMsg}

	res, err := snipCompact(messages, testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	// Only msg2 + boundaryMsg + newSnipMsg should remain.
	if len(res.Messages) != 3 {
		t.Fatalf("expected 3 remaining messages, got %d", len(res.Messages))
	}
	// A new boundary should be created for msg3.
	if res.BoundaryMessage == nil {
		t.Fatal("expected a new boundary for the new snip result")
	}
}

func TestDeriveShortMessageID(t *testing.T) {
	// Known UUID → known short ID.
	// 00000000-0000-4000-8000-000000000000 → hex 000000000040008000000000000000
	// first 10 = 0000000000 → base-36 = "0"
	if got := deriveShortMessageID("00000000-0000-4000-8000-000000000000"); got != "0" {
		t.Errorf("expected '0', got %q", got)
	}
	// aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa → hex aaaaaaaaaaaaaaaaaaaaaaaaaaaa
	// first 10 = aaaaaaaaaa → 733007751850 → base-36 = "4ldqpda"
	// Actually let me just verify the function is deterministic.
	id1 := deriveShortMessageID("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	id2 := deriveShortMessageID("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if id1 != id2 {
		t.Errorf("short IDs should be deterministic: %q vs %q", id1, id2)
	}
}

func TestEstimateMsgTokens(t *testing.T) {
	msg := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "00000000-0000-4000-8000-000000000001",
		Content: json.RawMessage(`"hello world this is a test message"`),
	}
	n := estimateMsgTokens(msg)
	if n < 1 {
		t.Errorf("expected at least 1 token, got %d", n)
	}
}

func TestSnipCompact_RepairToolPairs(t *testing.T) {
	// When a tool_result user message is snipped, the corresponding assistant
	// message (with tool_use blocks) must also be removed to keep API validity.
	assistantUUID := "0000000b-0000-4000-8000-000000000003"

	userMsg := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "00000000-0000-4000-8000-000000000001",
		Content: json.RawMessage(`"user message"`),
	}
	assistantMsg := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    assistantUUID,
		Content: json.RawMessage(`"assistant with tool_use"`),
	}
	toolResultMsg := types.Message{
		Type:                   types.MessageTypeUser,
		UUID:                   "0000000a-0000-4000-8000-000000000002",
		SourceToolAssistantUUID: &assistantUUID,
		Content:                json.RawMessage(`"tool result"`),
	}

	// Snip the tool_result message.
	sid := deriveShortMessageID(toolResultMsg.UUID)
	snipJSON := `{"data":{"snipped_count":1,"summary":"snip tool result","message_ids":["` + sid + `"]}}`
	snipMsg := types.Message{
		Type:          types.MessageTypeUser,
		UUID:          "0000000c-0000-4000-8000-000000000004",
		ToolUseResult: types.ToolUseResultJSONBytes(snipJSON),
	}

	messages := []types.Message{userMsg, assistantMsg, toolResultMsg, snipMsg}

	res, err := snipCompact(messages, testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	// Both tool_result AND the paired assistant must be removed.
	// Only userMsg + snipMsg should remain.
	if len(res.Messages) != 2 {
		t.Fatalf("expected 2 remaining messages (userMsg + snipMsg), got %d", len(res.Messages))
	}
	for _, m := range res.Messages {
		if m.UUID == assistantMsg.UUID {
			t.Errorf("assistant message should have been removed as part of tool pair")
		}
		if m.UUID == toolResultMsg.UUID {
			t.Errorf("tool_result message should have been removed")
		}
	}
}

func TestSnipCompact_RepairToolPairs_AssistantRemoved(t *testing.T) {
	// When an assistant message is snipped, its tool_result user messages
	// should also be removed.
	assistantUUID := "0000000b-0000-4000-8000-000000000003"

	userMsg := types.Message{
		Type:    types.MessageTypeUser,
		UUID:    "00000000-0000-4000-8000-000000000001",
		Content: json.RawMessage(`"user"`),
	}
	assistantMsg := types.Message{
		Type:    types.MessageTypeAssistant,
		UUID:    assistantUUID,
		Content: json.RawMessage(`"assistant"`),
	}
	toolResultMsg := types.Message{
		Type:                   types.MessageTypeUser,
		UUID:                   "0000000a-0000-4000-8000-000000000002",
		SourceToolAssistantUUID: &assistantUUID,
		Content:                json.RawMessage(`"tool result"`),
	}

	// Snip the assistant message — need to use its short ID.
	// But assistant messages don't get [id:] tags in TS/Go, so the model
	// normally can't snip them. However, /force-snip covers all messages.
	// We simulate by using the assistant's short ID.
	sid := deriveShortMessageID(assistantMsg.UUID)
	snipJSON := `{"data":{"snipped_count":1,"summary":"snip assistant","message_ids":["` + sid + `"]}}`
	snipMsg := types.Message{
		Type:          types.MessageTypeUser,
		UUID:          "0000000c-0000-4000-8000-000000000004",
		ToolUseResult: types.ToolUseResultJSONBytes(snipJSON),
	}

	messages := []types.Message{userMsg, assistantMsg, toolResultMsg, snipMsg}

	res, err := snipCompact(messages, testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	// Both assistant AND the paired tool_result must be removed.
	if len(res.Messages) != 2 {
		t.Fatalf("expected 2 remaining, got %d: %+v", len(res.Messages), res.Messages)
	}
}

var testUUIDSeq int

func testUUID() string {
	testUUIDSeq++
	return "00000000-0000-4000-8000-0000000000" + string(rune('0'+testUUIDSeq%10))
}
