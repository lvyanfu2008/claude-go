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
			Type  string         `json:"type"`
			Block map[string]any `json:"content_block"`
			Delta map[string]any `json:"delta"`
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

func TestDSMLRealMangledSample(t *testing.T) {
	// Real-world sample captured from a production subagent: two Bash invokes
	// with spaces around markers, and a proper wrapper close tag.
	chunks := []string{
		"查找工程中的X开头的类文件",
		"< | DSML | tool_calls>",
		"< | DSML | invoke name=\"Bash\">",
		"< | DSML | parameter name=\"command\" string=\"true\">/usr/bin/find /d/work/www-auth-v6/cdcs-auth-fin -name \"X0180*\" -o -name \"X0885*\" -type f 2>/dev/null</ | DSML |parameter>",
		"< | DSML | parameter name=\"description\" string=\"true\">搜索 X0180 和 X0885 开头的所有文件</ | DSML | parameter>",
		"</ | DSML | invoke>",
		"< | DSML | invoke name=\"Bash\">",
		"< | DSML | parameter name=\"command\" string=\"true\">ls -d /d/work/www-auth-v6/cdcs-auth-fin/*/src/main/java/com/cebbank/*/</ | DSML | parameter>",
		"< | DSML | parameter name=\"description\" string=\"true\">列出所有模块下的包结构</ | DSML | parameter>",
		"</ | DSML | invoke>",
		"</ | DSML | tool_calls>",
	}
	_, names, inputs := collectDSMLEmission(t, chunks)
	if len(names) != 2 || names[0] != "Bash" || names[1] != "Bash" {
		t.Fatalf("expected two Bash tool_use, got %v", names)
	}
	if len(inputs) != 2 {
		t.Fatalf("expected two input_json_delta, got %d", len(inputs))
	}
	var args1 map[string]any
	if err := json.Unmarshal([]byte(inputs[0]), &args1); err != nil {
		t.Fatalf("bad args JSON: %v", err)
	}
	if cmd, _ := args1["command"].(string); !strings.Contains(cmd, "/usr/bin/find") {
		t.Fatalf("command arg should contain the find command, got %q", cmd)
	}
	if desc, _ := args1["description"].(string); !strings.Contains(desc, "X0180") {
		t.Fatalf("description arg should contain X0180, got %q", desc)
	}
	var args2 map[string]any
	if err := json.Unmarshal([]byte(inputs[1]), &args2); err != nil {
		t.Fatalf("bad args2 JSON: %v", err)
	}
	if cmd, _ := args2["command"].(string); !strings.Contains(cmd, "ls -d") {
		t.Fatalf("second command arg should be ls -d, got %q", cmd)
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

func TestDSMLParameterWithoutStringAttr(t *testing.T) {
	// Per ai-dynamo/vLLM, the string attribute may be omitted entirely;
	// the value is then treated as a string.
	chunks := []string{
		"<|DSML|tool_calls>",
		"<|DSML|invoke name=\"configure\">",
		"<|DSML|parameter name=\"mode\">quickly</|DSML|parameter>",
		"</|DSML|invoke>",
		"</|DSML|tool_calls>",
	}
	_, names, inputs := collectDSMLEmission(t, chunks)
	if len(names) != 1 || names[0] != "configure" {
		t.Fatalf("expected tool_use configure, got %v", names)
	}
	if len(inputs) == 0 {
		t.Fatal("expected input")
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(inputs[0]), &args); err != nil {
		t.Fatalf("bad args JSON: %v", err)
	}
	if args["mode"] != "quickly" {
		t.Fatalf("mode should be the string \"quickly\", got %v", args["mode"])
	}
}
