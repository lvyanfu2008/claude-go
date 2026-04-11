package messagesapi

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/types"
)

func TestNormalizeMessagesForAPI_compactAllTextUserContent_defaultOffTSBlocks(t *testing.T) {
	t.Parallel()
	// Default off: no collapseAllTextUserContentBlocks; a later user row after assistant keeps multiple text siblings.
	u1 := mustJSON(t, map[string]any{"role": "user", "content": "lead"})
	u2 := mustJSON(t, map[string]any{
		"role": "user",
		"content": []map[string]any{
			{"type": "text", "text": "x"},
			{"type": "text", "text": "y"},
		},
	})
	asst := mustJSON(t, map[string]any{"role": "assistant", "content": []map[string]any{{"type": "text", "text": "ok"}}})
	msgs := []types.Message{
		{Type: types.MessageTypeUser, UUID: "1", Message: u1},
		{Type: types.MessageTypeAssistant, UUID: "a", Message: asst},
		{Type: types.MessageTypeUser, UUID: "2", Message: u2},
	}
	out, err := NormalizeMessagesForAPI(msgs, nil, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("len=%d", len(out))
	}
	inner2, _ := getInner(&out[2])
	blocks, err := parseContentArrayOrString(inner2.Content)
	if err != nil || len(blocks) != 2 {
		t.Fatalf("second user blocks=%v err=%v", blocks, err)
	}
}

func TestNormalizeMessagesForAPI_singleUser_allTextSiblingsPreservedTS(t *testing.T) {
	t.Parallel()
	a := mustJSON(t, map[string]any{
		"role": "user",
		"content": []map[string]any{
			{"type": "text", "text": "x"},
			{"type": "text", "text": "y"},
		},
	})
	msgs := []types.Message{{Type: types.MessageTypeUser, UUID: "1", Message: a}}
	out, err := NormalizeMessagesForAPI(msgs, nil, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	inner, _ := getInner(&out[0])
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil || len(blocks) != 2 {
		t.Fatalf("TS leaves sibling text blocks on one user row; blocks=%v err=%v", blocks, err)
	}
	if blocks[0]["text"] != "x" || blocks[1]["text"] != "y" {
		t.Fatalf("blocks=%v", blocks)
	}
}

// Hydrate/import can set top-level Content while Message omits content; ensureInnerFromContent must
// backfill inner so syncTopLevelContent does not wipe blocks (no TS collapse of siblings by default).
func TestNormalizeMessagesForAPI_ensureInner_topLevelContentThreeBlocks(t *testing.T) {
	t.Parallel()
	rawBlocks, err := json.Marshal([]map[string]any{
		{"type": "text", "text": "<system-reminder>\nctx\n</system-reminder>\n\n"},
		{"type": "text", "text": "hi"},
		{"type": "text", "text": "<system-reminder>\nskills\n</system-reminder>"},
	})
	if err != nil {
		t.Fatal(err)
	}
	badInner := mustJSON(t, map[string]any{"role": "user"})
	msgs := []types.Message{
		{Type: types.MessageTypeUser, UUID: "1", Message: badInner, Content: json.RawMessage(rawBlocks)},
	}
	out, err := NormalizeMessagesForAPI(msgs, nil, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	inner, _ := getInner(&out[0])
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil || len(blocks) != 3 {
		t.Fatalf("want 3 text blocks (TS-style siblings), got blocks=%v err=%v", blocks, err)
	}
}

func TestNormalizeMessagesForAPI_compactAllTextUserContent_optInCollapses(t *testing.T) {
	t.Parallel()
	a := mustJSON(t, map[string]any{
		"role": "user",
		"content": []map[string]any{
			{"type": "text", "text": "x"},
			{"type": "text", "text": "y"},
		},
	})
	msgs := []types.Message{{Type: types.MessageTypeUser, UUID: "1", Message: a}}
	out, err := NormalizeMessagesForAPI(msgs, nil, Options{CompactAllTextUserContent: true})
	if err != nil {
		t.Fatal(err)
	}
	inner, _ := getInner(&out[0])
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil || len(blocks) != 1 {
		t.Fatalf("blocks=%v err=%v", blocks, err)
	}
	tx, _ := blocks[0]["text"].(string)
	if tx != "xy" {
		t.Fatalf("got %q", tx)
	}
}

func TestNormalizeMessagesForAPI_dropsVirtualUser(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal("hi")
	v := true
	msgs := []types.Message{
		{Type: types.MessageTypeUser, UUID: "1", Content: raw},
		{Type: types.MessageTypeUser, UUID: "2", Content: raw, IsVirtual: &v},
	}
	out, err := NormalizeMessagesForAPI(msgs, nil, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].UUID != "1" {
		t.Fatalf("uuid=%q", out[0].UUID)
	}
}

func TestNormalizeToolInputForAPI_exitPlanStripsPlan(t *testing.T) {
	t.Parallel()
	in := map[string]any{"plan": "x", "planFilePath": "/p", "ok": true}
	out := normalizeToolInputForAPI(exitPlanModeV2ToolName, in)
	m, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("%T", out)
	}
	if _, ok := m["plan"]; ok {
		t.Fatal("plan still present")
	}
	if m["ok"] != true {
		t.Fatal(m)
	}
}

func TestNormalizeMessagesForAPI_mergeAssistantSameMessageID(t *testing.T) {
	t.Parallel()
	inner1 := mustJSON(t, map[string]any{
		"role": "assistant",
		"id":   "mid",
		"content": []map[string]any{
			{"type": "text", "text": "a"},
		},
	})
	inner2 := mustJSON(t, map[string]any{
		"role": "assistant",
		"id":   "mid",
		"content": []map[string]any{
			{"type": "text", "text": "b"},
		},
	})
	msgs := []types.Message{
		{Type: types.MessageTypeAssistant, UUID: "u1", Message: inner1},
		{Type: types.MessageTypeAssistant, UUID: "u2", Message: inner2},
	}
	out, err := NormalizeMessagesForAPI(msgs, nil, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d want 1 merged", len(out))
	}
	inner, _ := getInner(&out[0])
	blocks, err := parseContentArrayOrString(inner.Content)
	if err != nil || len(blocks) != 2 {
		t.Fatalf("blocks=%v err=%v", blocks, err)
	}
}

func TestNormalizeAttachment_teammateMailbox_agentSwarms(t *testing.T) {
	t.Parallel()
	att, err := json.Marshal(map[string]any{
		"type": "teammate_mailbox",
		"messages": []map[string]any{
			{"from": "alice", "text": "hello", "timestamp": "t1"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := normalizeAttachmentForAPI(att, Options{AgentSwarmsEnabled: true}, func() string { return "u1" })
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	inner, _ := getInner(&out[0])
	var s string
	if err := json.Unmarshal(inner.Content, &s); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "teammate-message") || !strings.Contains(s, "alice") || !strings.Contains(s, "hello") {
		t.Fatalf("got %q", s)
	}
}

func TestNormalizeAttachment_teammateMailbox_swarmOffSkips(t *testing.T) {
	t.Parallel()
	att, _ := json.Marshal(map[string]any{
		"type":     "teammate_mailbox",
		"messages": []map[string]any{{"from": "a", "text": "x", "timestamp": "t"}},
	})
	out, err := normalizeAttachmentForAPI(att, DefaultOptions(), func() string { return "u" })
	if err != nil || len(out) != 0 {
		t.Fatalf("out=%v err=%v", out, err)
	}
}

func TestNormalizeAttachment_skillDiscovery_flag(t *testing.T) {
	t.Parallel()
	att, _ := json.Marshal(map[string]any{
		"type": "skill_discovery",
		"skills": []map[string]any{
			{"name": "n", "description": "d"},
		},
	})
	if out, _ := normalizeAttachmentForAPI(att, DefaultOptions(), func() string { return "u" }); len(out) != 0 {
		t.Fatalf("expected skip without flag, got %d", len(out))
	}
	out, err := normalizeAttachmentForAPI(att, Options{ExperimentalSkillSearch: true}, func() string { return "u" })
	if err != nil || len(out) != 1 {
		t.Fatalf("len=%d err=%v", len(out), err)
	}
}

func TestNormalizeAttachment_skillListing(t *testing.T) {
	t.Parallel()
	att, err := json.Marshal(map[string]any{
		"type":    "skill_listing",
		"content": "- foo: bar",
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := normalizeAttachmentForAPI(att, DefaultOptions(), func() string { return "fixed-uuid" })
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	inner, _ := getInner(&out[0])
	var s string
	if err := json.Unmarshal(inner.Content, &s); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "Skill tool") || !strings.Contains(s, "foo") {
		t.Fatalf("content: %q", s)
	}
}

func TestDeriveShortMessageId_stable(t *testing.T) {
	t.Parallel()
	u := "550e8400-e29b-41d4-a716-446655440000"
	a := deriveShortMessageId(u)
	b := deriveShortMessageId(u)
	if a != b || len(a) == 0 {
		t.Fatalf("%q %q", a, b)
	}
}

func TestNormalizeAttachment_planMode_fullV2_default(t *testing.T) {
	t.Parallel()
	att, _ := json.Marshal(map[string]any{
		"type":         "plan_mode",
		"planFilePath": "/tmp/plan.md",
		"planExists":   false,
		"reminderType": "full",
	})
	out, err := normalizeAttachmentForAPI(att, DefaultOptions(), func() string { return "u" })
	if err != nil || len(out) != 1 {
		t.Fatalf("len=%d err=%v", len(out), err)
	}
	inner, _ := getInner(&out[0])
	var s string
	if err := json.Unmarshal(inner.Content, &s); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "### Phase 1: Initial Understanding") {
		snippet := s
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		t.Fatalf("missing Phase 1: %q", snippet)
	}
	if !strings.Contains(s, "Launch up to 3 Explore agents") {
		t.Fatal("expected default explore count 3")
	}
	if !strings.Contains(s, "Begin with a **Context** section") {
		t.Fatal("expected control Phase 4")
	}
	if strings.Contains(s, "Iterative Planning Workflow") {
		t.Fatal("full V2 should not use interview template")
	}
}

func TestNormalizeAttachment_planMode_interviewPhase(t *testing.T) {
	t.Parallel()
	att, _ := json.Marshal(map[string]any{
		"type":         "plan_mode",
		"planFilePath": "/p.md",
		"planExists":   true,
	})
	out, err := normalizeAttachmentForAPI(att, Options{PlanModeInterviewPhase: true}, func() string { return "u" })
	if err != nil || len(out) != 1 {
		t.Fatalf("len=%d err=%v", len(out), err)
	}
	inner, _ := getInner(&out[0])
	var s string
	_ = json.Unmarshal(inner.Content, &s)
	if !strings.Contains(s, "## Iterative Planning Workflow") {
		t.Fatalf("expected interview workflow")
	}
	if strings.Contains(s, "### Phase 1: Initial Understanding") {
		t.Fatal("interview should not include 5-phase Phase 1")
	}
}

func TestNormalizeAttachment_planMode_sparse_workflowWording(t *testing.T) {
	t.Parallel()
	att, _ := json.Marshal(map[string]any{
		"type":         "plan_mode",
		"planFilePath": "/p.md",
		"reminderType": "sparse",
	})
	out, err := normalizeAttachmentForAPI(att, DefaultOptions(), func() string { return "u" })
	if err != nil || len(out) != 1 {
		t.Fatalf("len=%d err=%v", len(out), err)
	}
	inner, _ := getInner(&out[0])
	var s string
	_ = json.Unmarshal(inner.Content, &s)
	if !strings.Contains(s, "Follow 5-phase workflow.") {
		t.Fatalf("sparse default: %q", s)
	}
	out2, _ := normalizeAttachmentForAPI(att, Options{PlanModeInterviewPhase: true}, func() string { return "u2" })
	inner2, _ := getInner(&out2[0])
	var s2 string
	_ = json.Unmarshal(inner2.Content, &s2)
	if !strings.Contains(s2, "Follow iterative workflow:") {
		t.Fatalf("sparse interview: %q", s2)
	}
}

func TestNormalizeAttachment_planMode_phase4Trim(t *testing.T) {
	t.Parallel()
	att, _ := json.Marshal(map[string]any{
		"type":         "plan_mode",
		"planFilePath": "/p.md",
		"planExists":   false,
	})
	opts := Options{PlanPhase4Variant: "trim"}
	out, err := normalizeAttachmentForAPI(att, opts, func() string { return "u" })
	if err != nil || len(out) != 1 {
		t.Fatal(err)
	}
	inner, _ := getInner(&out[0])
	var s string
	_ = json.Unmarshal(inner.Content, &s)
	if !strings.Contains(s, "One-line **Context**") {
		t.Fatalf("expected trim Phase 4: %q", s)
	}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
