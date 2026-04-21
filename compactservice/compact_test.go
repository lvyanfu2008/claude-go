package compactservice

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"goc/types"
)

// ------ prompt.go ------

func TestGetCompactPrompt_ContainsTSParitySections(t *testing.T) {
	p := GetCompactPrompt("")
	mustContainAll(t, p,
		"CRITICAL: Respond with TEXT ONLY.",
		"1. Primary Request and Intent",
		"9. Optional Next Step",
		"REMINDER: Do NOT call any tools.",
	)
}

func TestGetCompactPrompt_AppendsCustomInstructions(t *testing.T) {
	p := GetCompactPrompt("focus on db")
	if !strings.Contains(p, "Additional Instructions:\nfocus on db") {
		t.Fatalf("custom instructions missing: %q", p)
	}
	// Trailer must remain after custom instructions.
	if strings.Index(p, "focus on db") >= strings.Index(p, "REMINDER:") {
		t.Fatalf("trailer ordering wrong")
	}
}

func TestGetPartialCompactPrompt_DirectionBranches(t *testing.T) {
	fromP := GetPartialCompactPrompt("", PartialCompactDirectionFrom)
	upTo := GetPartialCompactPrompt("", PartialCompactDirectionUpTo)
	if !strings.Contains(fromP, "RECENT portion") {
		t.Fatalf("from prompt missing recent-portion framing")
	}
	if !strings.Contains(upTo, "continuing session") {
		t.Fatalf("up_to prompt missing continuation framing")
	}
}

func TestFormatCompactSummary_StripsAnalysisAndRewritesSummary(t *testing.T) {
	raw := "<analysis>\ndraft notes\n</analysis>\n\n<summary>\n- point 1\n- point 2\n</summary>\n"
	got := FormatCompactSummary(raw)
	if strings.Contains(got, "<analysis>") || strings.Contains(got, "draft notes") {
		t.Fatalf("analysis not stripped: %q", got)
	}
	if !strings.HasPrefix(got, "Summary:\n") {
		t.Fatalf("summary header missing: %q", got)
	}
	if !strings.Contains(got, "- point 1") {
		t.Fatalf("summary content missing")
	}
}

func TestGetCompactUserSummaryMessage_Suppress(t *testing.T) {
	body := GetCompactUserSummaryMessage("<summary>s</summary>", CompactUserSummaryOpts{
		SuppressFollowUpQuestions: true,
		TranscriptPath:            "/tmp/session.jsonl",
	})
	mustContainAll(t, body,
		"This session is being continued",
		"read the full transcript at: /tmp/session.jsonl",
		"Continue the conversation from where it left off",
	)
}

// ------ boundary.go ------

func TestCreateCompactBoundaryMessage(t *testing.T) {
	last := "pre-uuid"
	m, err := CreateCompactBoundaryMessage(CompactTriggerAuto, 123_456, last, "user ctx", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !IsCompactBoundaryMessage(m) {
		t.Fatalf("want boundary, got %+v", m)
	}
	if m.LogicalParentUUID == nil || *m.LogicalParentUUID != last {
		t.Fatalf("logicalParentUuid mismatch: %v", m.LogicalParentUUID)
	}
	var meta CompactMetadata
	if err := json.Unmarshal(m.CompactMetadata, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Trigger != CompactTriggerAuto || meta.PreTokens != 123_456 || meta.UserContext != "user ctx" {
		t.Fatalf("meta mismatch: %+v", meta)
	}
}

// ------ grouping.go ------

func TestGroupMessagesByApiRound(t *testing.T) {
	mk := func(typ types.MessageType, id string) types.Message {
		m := types.Message{Type: typ, UUID: id}
		if typ == types.MessageTypeAssistant {
			m.Message = json.RawMessage(`{"role":"assistant","id":"` + id + `","content":[]}`)
			m.MessageID = strPtr(id)
		} else {
			m.Message = json.RawMessage(`{"role":"user","content":"x"}`)
		}
		return m
	}
	msgs := []types.Message{
		mk(types.MessageTypeUser, "u1"),
		mk(types.MessageTypeAssistant, "A"),
		mk(types.MessageTypeUser, "r1"),     // tool_result in round A
		mk(types.MessageTypeAssistant, "A"), // same round (split assistant chunk)
		mk(types.MessageTypeAssistant, "B"), // new round
		mk(types.MessageTypeUser, "r2"),
	}
	groups := GroupMessagesByApiRound(msgs)
	// TS parity: preamble → group 0, each new assistant id starts a new group.
	// [u1] | [A, r1, A(split)] | [B, r2]
	if len(groups) != 3 {
		t.Fatalf("want 3 groups, got %d", len(groups))
	}
	if len(groups[0]) != 1 || len(groups[1]) != 3 || len(groups[2]) != 2 {
		t.Fatalf("group sizes: %d %d %d", len(groups[0]), len(groups[1]), len(groups[2]))
	}
}

// ------ strip.go ------

func TestStripImagesFromMessages(t *testing.T) {
	userJSON := `{"role":"user","content":[{"type":"image","source":{"type":"base64","data":"..."}},{"type":"text","text":"hi"}]}`
	m := types.Message{Type: types.MessageTypeUser, UUID: "u", Message: json.RawMessage(userJSON)}
	out := StripImagesFromMessages([]types.Message{m})
	if got := string(out[0].Message); !strings.Contains(got, `"[image]"`) || strings.Contains(got, `"base64"`) {
		t.Fatalf("image not stripped: %s", got)
	}
}

func TestStripReinjectedAttachments_ExperimentalGate(t *testing.T) {
	m := types.Message{Type: types.MessageTypeAttachment, Attachment: json.RawMessage(`{"type":"skill_discovery"}`)}
	if len(StripReinjectedAttachments([]types.Message{m}, false)) != 1 {
		t.Fatalf("default should pass-through")
	}
	if len(StripReinjectedAttachments([]types.Message{m}, true)) != 0 {
		t.Fatalf("experimental should strip")
	}
}

// ------ tokens.go ------

func TestRoughTokenCountEstimation_Text(t *testing.T) {
	got := RoughTokenCountEstimation("abcd") // 4 chars / 4 bytes per token = 1
	if got != 1 {
		t.Fatalf("want 1, got %d", got)
	}
	if got := RoughTokenCountEstimation(strings.Repeat("x", 400)); got != 100 {
		t.Fatalf("want 100, got %d", got)
	}
}

func TestTokenCountWithEstimation_UsesLastAssistantUsage(t *testing.T) {
	user := types.Message{
		Type: types.MessageTypeUser, UUID: "u",
		Message: json.RawMessage(`{"role":"user","content":"abcd"}`),
	}
	asst := types.Message{
		Type: types.MessageTypeAssistant, UUID: "a",
		Message:   json.RawMessage(`{"role":"assistant","id":"R1","content":[],"usage":{"input_tokens":100,"output_tokens":50}}`),
		MessageID: strPtr("R1"),
	}
	later := types.Message{
		Type: types.MessageTypeUser, UUID: "u2",
		Message: json.RawMessage(`{"role":"user","content":"efgh"}`),
	}
	total := TokenCountWithEstimation([]types.Message{user, asst, later})
	// usage 150 + rough('efgh') = 150 + round(4/4) = 151
	if total != 151 {
		t.Fatalf("want 151, got %d", total)
	}
}

// ------ auto_compact.go ------

func TestIsAutoCompactEnabled_KillSwitch(t *testing.T) {
	t.Setenv("DISABLE_COMPACT", "1")
	if IsAutoCompactEnabled() {
		t.Fatal("should be disabled")
	}
	t.Setenv("DISABLE_COMPACT", "")
	t.Setenv("DISABLE_AUTO_COMPACT", "true")
	if IsAutoCompactEnabled() {
		t.Fatal("should be disabled via DISABLE_AUTO_COMPACT")
	}
	t.Setenv("DISABLE_AUTO_COMPACT", "0")
	if !IsAutoCompactEnabled() {
		t.Fatal("should be enabled")
	}
}

func TestCalculateTokenWarningState_AboveThresholds(t *testing.T) {
	state := CalculateTokenWarningState(170_000, "claude-sonnet-4", nil, CompactThresholds{})
	// default effective window = 100_000 - 20_000 = 80_000
	// autoCompactThreshold = 80_000 - 13_000 = 67_000
	// warning/error = 67_000 - 20_000 = 47_000
	if !state.IsAboveAutoCompactThreshold {
		t.Fatalf("want above auto-compact at 170k")
	}
	if !state.IsAboveWarningThreshold || !state.IsAboveErrorThreshold {
		t.Fatalf("want warning+error at 170k")
	}
	// 3% left rough
	if state.PercentLeft < 0 || state.PercentLeft > 100 {
		t.Fatalf("percent left out of range: %d", state.PercentLeft)
	}
}

func TestShouldAutoCompact_QuerySourceGuard(t *testing.T) {
	in := ShouldAutoCompactInput{
		Messages:    []types.Message{{Type: types.MessageTypeUser, Message: json.RawMessage(`{"role":"user","content":"x"}`)}},
		Model:       "m",
		QuerySource: "compact",
	}
	if ShouldAutoCompact(in) {
		t.Fatal("must not compact while inside compact")
	}
	in.QuerySource = "session_memory"
	if ShouldAutoCompact(in) {
		t.Fatal("must not compact while inside session_memory")
	}
}

// ------ ptl.go ------

func TestTruncateHeadForPTLRetry_NoGaps(t *testing.T) {
	msgs := []types.Message{
		{Type: types.MessageTypeUser, Message: json.RawMessage(`{"role":"user","content":"hi"}`)},
	}
	if _, ok := TruncateHeadForPTLRetry(msgs, types.Message{}); ok {
		t.Fatal("single-group input should not truncate")
	}
}

func TestFormatCompactSummary_Idempotent(t *testing.T) {
	s1 := FormatCompactSummary("<analysis>a</analysis>\n<summary>body</summary>")
	s2 := FormatCompactSummary(s1)
	if s1 != s2 {
		t.Fatalf("not idempotent:\n1=%q\n2=%q", s1, s2)
	}
}

// ------ CompactConversation end-to-end with injected summarizer ------

func TestCompactConversation_EndToEnd(t *testing.T) {
	msgs := []types.Message{
		{
			Type: types.MessageTypeUser, UUID: "u1",
			Message: json.RawMessage(`{"role":"user","content":"hello"}`),
		},
		{
			Type: types.MessageTypeAssistant, UUID: "a1",
			Message:   json.RawMessage(`{"role":"assistant","id":"R1","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":5,"output_tokens":2}}`),
			MessageID: strPtr("R1"),
		},
	}
	deps := Deps{
		Summarize: func(_ context.Context, in SummaryStreamInput) (SummaryStreamResult, error) {
			if len(in.Messages) == 0 {
				t.Fatal("summarizer received empty messages")
			}
			asst := types.Message{
				Type: types.MessageTypeAssistant, UUID: "summary-u",
				Message:   json.RawMessage(`{"role":"assistant","id":"S1","content":[{"type":"text","text":"<analysis>x</analysis>\n<summary>compacted</summary>"}],"usage":{"input_tokens":100,"output_tokens":20}}`),
				MessageID: strPtr("S1"),
			}
			return SummaryStreamResult{AssistantMessage: asst, Usage: &TokenUsage{InputTokens: 100, OutputTokens: 20}}, nil
		},
	}
	result, err := CompactConversation(context.Background(), msgs, deps, CompactOptions{
		SuppressFollowUpQuestions: true,
		IsAutoCompact:             true,
		Model:                     "claude-sonnet-4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !IsCompactBoundaryMessage(result.BoundaryMarker) {
		t.Fatalf("no boundary")
	}
	if len(result.SummaryMessages) != 1 {
		t.Fatalf("want 1 summary message")
	}
	var envelope struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(result.SummaryMessages[0].Message, &envelope); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(envelope.Content, "Summary:\ncompacted") {
		t.Fatalf("summary content missing: %q", envelope.Content)
	}
	if result.PreCompactTokenCount <= 0 {
		t.Fatalf("preCompactTokenCount must be >0")
	}
	if result.CompactionUsage == nil || result.CompactionUsage.InputTokens != 100 {
		t.Fatalf("usage not threaded: %+v", result.CompactionUsage)
	}
}

func TestCompactConversation_RetriesOnPromptTooLong(t *testing.T) {
	// Two rounds so truncateHeadForPTLRetry has something to drop.
	msgs := []types.Message{
		{Type: types.MessageTypeUser, UUID: "u1", Message: json.RawMessage(`{"role":"user","content":"first"}`)},
		{Type: types.MessageTypeAssistant, UUID: "a1", Message: json.RawMessage(`{"role":"assistant","id":"A","content":[{"type":"text","text":"hi"}]}`), MessageID: strPtr("A")},
		{Type: types.MessageTypeUser, UUID: "u2", Message: json.RawMessage(`{"role":"user","content":"second"}`)},
		{Type: types.MessageTypeAssistant, UUID: "a2", Message: json.RawMessage(`{"role":"assistant","id":"B","content":[{"type":"text","text":"bye"}]}`), MessageID: strPtr("B")},
	}
	calls := 0
	deps := Deps{
		Summarize: func(_ context.Context, in SummaryStreamInput) (SummaryStreamResult, error) {
			calls++
			if calls == 1 {
				errDetails := "prompt is too long: 200000 tokens > 180000 maximum"
				trueP := true
				asst := types.Message{
					Type: types.MessageTypeAssistant, UUID: "err",
					Message:           json.RawMessage(`{"role":"assistant","id":"E","content":[{"type":"text","text":"Prompt is too long"}],"errorDetails":"` + errDetails + `"}`),
					IsApiErrorMessage: &trueP,
				}
				return SummaryStreamResult{AssistantMessage: asst}, nil
			}
			asst := types.Message{
				Type: types.MessageTypeAssistant, UUID: "ok",
				Message:   json.RawMessage(`{"role":"assistant","id":"S","content":[{"type":"text","text":"<summary>ok</summary>"}],"usage":{"input_tokens":1,"output_tokens":1}}`),
				MessageID: strPtr("S"),
			}
			return SummaryStreamResult{AssistantMessage: asst, Usage: &TokenUsage{InputTokens: 1, OutputTokens: 1}}, nil
		},
	}
	result, err := CompactConversation(context.Background(), msgs, deps, CompactOptions{Model: "claude-sonnet-4"})
	if err != nil {
		t.Fatal(err)
	}
	if calls < 2 {
		t.Fatalf("expected at least 2 summarizer calls; got %d", calls)
	}
	if len(result.SummaryMessages) != 1 {
		t.Fatalf("want 1 summary")
	}
}

// -------- attachment extraction tests --------

func TestExtractReadFilePaths_Basic(t *testing.T) {
	// Assistant message with Read tool_use
	assistantMsg := types.Message{
		Type: types.MessageTypeAssistant,
		UUID: "a1",
		Message: mustJSON(map[string]any{
			"role": "assistant",
			"content": []any{
				map[string]any{
					"type": "tool_use",
					"name": "Read",
					"input": map[string]any{
						"file_path": "/foo/bar.txt",
					},
				},
				map[string]any{
					"type": "tool_use",
					"name": "Read",
					"input": map[string]any{
						"file_path": "/baz/qux.go",
					},
				},
			},
		}),
	}
	// Another assistant with a Read already in preserved
	assistantMsg2 := types.Message{
		Type: types.MessageTypeAssistant,
		UUID: "a2",
		Message: mustJSON(map[string]any{
			"role": "assistant",
			"content": []any{
				map[string]any{
					"type": "tool_use",
					"name": "Read",
					"input": map[string]any{
						"file_path": "/preserved/file.txt",
					},
				},
			},
		}),
	}
	// Preserved message with same Read
	preservedMsg := types.Message{
		Type: types.MessageTypeAssistant,
		UUID: "p1",
		Message: mustJSON(map[string]any{
			"role": "assistant",
			"content": []any{
				map[string]any{
					"type": "tool_use",
					"name": "Read",
					"input": map[string]any{
						"file_path": "/preserved/file.txt",
					},
				},
			},
		}),
	}

	messages := []types.Message{assistantMsg, assistantMsg2}
	preserved := []types.Message{preservedMsg}

	result := ExtractReadFilePaths(messages, preserved, 10)

	// Should have 2 files (not 3, since /preserved/file.txt is excluded)
	if len(result) != 2 {
		t.Fatalf("want 2 files, got %d: %+v", len(result), result)
	}
	paths := make(map[string]bool)
	for _, r := range result {
		paths[r.Path] = true
	}
	if !paths["/foo/bar.txt"] || !paths["/baz/qux.go"] {
		t.Fatalf("missing expected paths: %+v", result)
	}
	if paths["/preserved/file.txt"] {
		t.Fatal("should not include preserved file")
	}
}

func TestExtractInvokedSkills_Basic(t *testing.T) {
	attMsg := types.Message{
		Type: types.MessageTypeAttachment,
		UUID: "att1",
		Attachment: mustJSON(map[string]any{
			"type": "invoked_skills",
			"skills": []any{
				map[string]any{
					"name":    "skill-a",
					"path":    "/skills/a.md",
					"content": "Skill A content",
				},
				map[string]any{
					"name":    "skill-b",
					"path":    "/skills/b.md",
					"content": "Skill B content",
				},
			},
		}),
	}
	// A later invoked_skills with skill-a updated
	attMsg2 := types.Message{
		Type: types.MessageTypeAttachment,
		UUID: "att2",
		Attachment: mustJSON(map[string]any{
			"type": "invoked_skills",
			"skills": []any{
				map[string]any{
					"name":    "skill-a",
					"path":    "/skills/a.md",
					"content": "Skill A updated content",
				},
			},
		}),
	}

	messages := []types.Message{attMsg, attMsg2}
	result := ExtractInvokedSkills(messages)

	if len(result) != 2 {
		t.Fatalf("want 2 skills, got %d", len(result))
	}
	// skill-a should have updated content (later occurrence wins)
	var skillA *ExtractedSkill
	for i := range result {
		if result[i].Name == "skill-a" {
			skillA = &result[i]
			break
		}
	}
	if skillA == nil {
		t.Fatal("skill-a not found")
	}
	if skillA.Content != "Skill A updated content" {
		t.Fatalf("skill-a content not updated: %q", skillA.Content)
	}
}

// -------- helpers --------

func mustContainAll(t *testing.T, hay string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(hay, n) {
			t.Fatalf("missing %q in %q", n, hay)
		}
	}
}

func strPtr(s string) *string { return &s }

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
