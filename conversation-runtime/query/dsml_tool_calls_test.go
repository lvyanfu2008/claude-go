package query

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/anthropicmessages"
)

// collectEmittedTypes replays chunks through an adapter and returns the
// emitted event types plus any tool_use names and input_json partials.
func collectDSMLEmission(t *testing.T, chunks []string) (types []string, toolNames []string, inputs []string) {
	t.Helper()
	ad := newOpenAIStreamAdapter("deepseek-v4-flash")
	record := func(ev anthropicmessages.MessageStreamEvent) error {
		var m struct {
			Type     string         `json:"type"`
			Block    map[string]any `json:"content_block"`
			Delta    map[string]any `json:"delta"`
		}
		_ = json.Unmarshal(ev.Raw, &m)
		types = append(types, m.Type)
		// Record the inner delta type so text_delta vs input_json_delta is visible.
		if dt, _ := m.Delta["type"].(string); dt != "" {
			types = append(types, dt)
		}
		if m.Type == "content_block_start" {
			if t2, _ := m.Block["type"].(string); t2 == "tool_use" {
				if n, _ := m.Block["name"].(string); n != "" {
					toolNames = append(toolNames, n)
				}
			}
		}
		if m.Type == "content_block_delta" {
			if dt, _ := m.Delta["type"].(string); dt == "input_json_delta" {
				if pj, _ := m.Delta["partial_json"].(string); pj != "" {
					inputs = append(inputs, pj)
				}
			}
		}
		return nil
	}
	for _, c := range chunks {
		chunk, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{{"delta": map[string]any{"content": c}}},
		})
		if err := ad.HandleChunk(chunk, record); err != nil {
			t.Fatal(err)
		}
	}
	if err := ad.FlushOpenBlocks(record); err != nil {
		t.Fatal(err)
	}
	return
}

func TestDSMLToolCallsStreaming(t *testing.T) {
	// DeepSeek emits DSML tool calls as text deltas, split across chunks.
	chunks := []string{
		"好的,我来查找:",
		"<|DSML|tool_calls>",
		"<|DSML|invoke name=\"Glob\">",
		"<|DSML|parameter name=\"pattern\" string=\"true\">**/mapper/*.xml</|DSML|parameter>",
		"<|DSML|parameter name=\"limit\" string=\"false\">5</|DSML|parameter>",
		"</|DSML|invoke>",
		"</|DSML|tool_calls>",
		"完成",
	}
	types, names, inputs := collectDSMLEmission(t, chunks)

	// The raw DSML markers must never leak into text; only surrounding prose may.
	joined := strings.Join(types, ",")
	if strings.Contains(joined, "DSML") {
		t.Fatalf("DSML markers must not leak as text, got %q", joined)
	}
	// A tool_use must be parsed.
	if len(names) != 1 || names[0] != "Glob" {
		t.Fatalf("expected tool_use Glob, got %v", names)
	}
	// Arguments must round-trip with correct types.
	if len(inputs) == 0 {
		t.Fatal("expected input_json_delta with tool arguments")
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(inputs[0]), &args); err != nil {
		t.Fatalf("bad args JSON: %v", err)
	}
	if args["pattern"] != "**/mapper/*.xml" {
		t.Fatalf("pattern should stay string, got %v", args["pattern"])
	}
	if v, ok := args["limit"].(float64); !ok || v != 5 {
		t.Fatalf("limit should be number 5, got %v", args["limit"])
	}
}

func TestDSMLFullWidthBars(t *testing.T) {
	// DeepSeek V4 uses full-width vertical bars.
	chunks := []string{
		"<｜DSML｜tool_calls>",
		"<｜DSML｜invoke name=\"Read\">",
		"<｜DSML｜parameter name=\"file_path\" string=\"true\">D:/x.txt</｜DSML｜parameter>",
		"</｜DSML｜invoke>",
		"</｜DSML｜tool_calls>",
	}
	_, names, inputs := collectDSMLEmission(t, chunks)
	if len(names) != 1 || names[0] != "Read" {
		t.Fatalf("expected tool_use Read, got %v", names)
	}
	if len(inputs) == 0 || !strings.Contains(inputs[0], "D:/x.txt") {
		t.Fatalf("expected file_path arg, got %v", inputs)
	}
}

func TestDSMLUnclosedBlockFlushedAsText(t *testing.T) {
	// If the stream ends mid-block, the buffer must be forwarded as text (no data loss).
	chunks := []string{
		"<|DSML|tool_calls>",
		"<|DSML|invoke name=\"Glob\">",
		"unfinished...",
	}
	types, names, _ := collectDSMLEmission(t, chunks)
	if len(names) != 0 {
		t.Fatalf("unclosed block should not produce a tool_use, got %v", names)
	}
	if !strings.Contains(strings.Join(types, ","), "text_delta") {
		t.Fatalf("unclosed DSML should be flushed as text, got %q", types)
	}
}

func TestDSMLDoesNotBreakPlainText(t *testing.T) {
	// Ordinary text (no DSML markers) must flow through unchanged.
	chunks := []string{"你好", "世界"}
	types, names, _ := collectDSMLEmission(t, chunks)
	if len(names) != 0 {
		t.Fatalf("plain text should not parse tool_use, got %v", names)
	}
	if !strings.Contains(strings.Join(types, ","), "text_delta") {
		t.Fatalf("plain text should emit text_delta, got %q", types)
	}
}

func TestDSMLStraySpacesAroundMarkers(t *testing.T) {
	// Some models emit stray spaces around DSML markers, e.g.
	// "<| DSML |invoke name=...>" — these must still parse as tool calls.
	chunks := []string{
		"<| DSML |tool_calls>",
		"<| DSML |invoke name=\"Glob\">",
		"<| DSML |parameter name=\"pattern\" string=\"true\">**/x*.go</| DSML |parameter>",
		"</| DSML |invoke>",
		"</| DSML |tool_calls>",
	}
	_, names, inputs := collectDSMLEmission(t, chunks)
	if len(names) != 1 || names[0] != "Glob" {
		t.Fatalf("expected tool_use Glob despite stray spaces, got %v", names)
	}
	if len(inputs) == 0 || !strings.Contains(inputs[0], "**/x*.go") {
		t.Fatalf("expected pattern arg, got %v", inputs)
	}
}
