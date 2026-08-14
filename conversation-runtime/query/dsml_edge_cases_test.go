package query

import (
	"encoding/json"
	"strings"
	"testing"

	"goc/anthropicmessages"
)

// collectDSMLEvents replays chunks through an adapter and returns emitted
// event types, tool_use names, input_json partials, and concatenated text.
func collectDSMLEvents(t *testing.T, chunks []string) (types []string, toolNames []string, inputs []string, text string) {
	t.Helper()
	ad := newOpenAIStreamAdapter("deepseek-v4-flash")
	var sb strings.Builder
	record := func(ev anthropicmessages.MessageStreamEvent) error {
		var m struct {
			Type  string         `json:"type"`
			Block map[string]any `json:"content_block"`
			Delta map[string]any `json:"delta"`
		}
		_ = json.Unmarshal(ev.Raw, &m)
		types = append(types, m.Type)
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
			if dt, _ := m.Delta["type"].(string); dt == "text_delta" {
				if txt, _ := m.Delta["text"].(string); txt != "" {
					sb.WriteString(txt)
				}
			}
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
	text = sb.String()
	return
}

func TestDSMLPipeInsideParameterValueSurvives(t *testing.T) {
	// Shell commands contain pipes (ps aux | grep x). Normalization must not
	// destroy a pipe inside a parameter VALUE.
	chunks := []string{
		"<|DSML|tool_calls>",
		"<|DSML|invoke name=\"Bash\">",
		"<|DSML|parameter name=\"command\" string=\"true\">ps aux | grep claude</|DSML|parameter>",
		"</|DSML|invoke>",
		"</|DSML|tool_calls>",
	}
	_, names, inputs, _ := collectDSMLEvents(t, chunks)
	if len(names) != 1 || names[0] != "Bash" {
		t.Fatalf("expected tool_use Bash, got %v", names)
	}
	if len(inputs) == 0 {
		t.Fatal("expected input_json_delta")
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(inputs[0]), &args); err != nil {
		t.Fatalf("bad args JSON: %v", err)
	}
	if cmd, _ := args["command"].(string); cmd != "ps aux | grep claude" {
		t.Fatalf("command must keep its pipe, got %q", cmd)
	}
}

func TestDSMLTrailingTextAfterWrapperClose(t *testing.T) {
	// Prose after </|DSML|tool_calls> must be emitted as text and markers
	// must not leak into text output.
	chunks := []string{
		"好的,我来查找:",
		"<|DSML|tool_calls>",
		"<|DSML|invoke name=\"Glob\">",
		"<|DSML|parameter name=\"pattern\" string=\"true\">**/x*.go</|DSML|parameter>",
		"</|DSML|invoke>",
		"</|DSML|tool_calls>",
		"完成",
	}
	_, names, _, text := collectDSMLEvents(t, chunks)
	if len(names) != 1 || names[0] != "Glob" {
		t.Fatalf("expected tool_use Glob, got %v", names)
	}
	if !strings.Contains(text, "完成") {
		t.Fatalf("trailing prose after wrapper close must be emitted as text, got %q", text)
	}
	if strings.Contains(text, "DSML") {
		t.Fatalf("DSML markers must not leak as text, got %q", text)
	}
}

func TestDSMLWrapperStartTagSplitAcrossChunks(t *testing.T) {
	// A start tag split across chunks must not leak either half as text,
	// and the block must still parse as a tool call.
	chunks := []string{
		"<|DSML|tool_call",
		"s>",
		"<|DSML|invoke name=\"Glob\">",
		"<|DSML|parameter name=\"pattern\" string=\"true\">**/x*.go</|DSML|parameter>",
		"</|DSML|invoke>",
		"</|DSML|tool_calls>",
	}
	_, names, _, text := collectDSMLEvents(t, chunks)
	if len(names) != 1 || names[0] != "Glob" {
		t.Fatalf("expected tool_use Glob, got %v", names)
	}
	if strings.Contains(text, "DSML") || strings.Contains(text, "tool_call") {
		t.Fatalf("split start tag must not leak as text, got %q", text)
	}
}

func TestDSMLWrapperMissingCloseFlushedCleanly(t *testing.T) {
	// Some models omit the wrapper close tag entirely (stream stops after the
	// last </DSML invoke>). Tail-flush must not emit raw markers as text.
	chunks := []string{
		"<|DSML|tool_calls>",
		"<|DSML|invoke name=\"Bash\">",
		"<|DSML|parameter name=\"command\" string=\"true\">ls -la</|DSML|parameter>",
		"</|DSML|invoke>",
	}
	_, names, inputs, text := collectDSMLEvents(t, chunks)
	if len(names) != 1 || names[0] != "Bash" {
		t.Fatalf("expected tool_use Bash, got %v", names)
	}
	if len(inputs) == 0 || !strings.Contains(inputs[0], "ls -la") {
		t.Fatalf("expected command arg, got %v", inputs)
	}
	if strings.Contains(text, "DSML") {
		t.Fatalf("wrapper close tag must not leak as text, got %q", text)
	}
}

func TestDSMLXMLEntitiesDecodedInValue(t *testing.T) {
	// Values may carry XML-escaped shell syntax (&lt; &gt; &amp;). They must
	// decode so the tool receives the real command, not escaped text.
	chunks := []string{
		"<|DSML|tool_calls>",
		"<|DSML|invoke name=\"Bash\">",
		"<|DSML|parameter name=\"command\" string=\"true\">ls &amp;&amp; echo ok &gt; /tmp/a.txt</|DSML|parameter>",
		"</|DSML|invoke>",
		"</|DSML|tool_calls>",
	}
	_, names, inputs, _ := collectDSMLEvents(t, chunks)
	if len(names) != 1 || names[0] != "Bash" {
		t.Fatalf("expected tool_use Bash, got %v", names)
	}
	if len(inputs) == 0 {
		t.Fatal("expected input")
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(inputs[0]), &args); err != nil {
		t.Fatalf("bad args JSON: %v", err)
	}
	if cmd, _ := args["command"].(string); cmd != "ls && echo ok > /tmp/a.txt" {
		t.Fatalf("entities must decode to the real command, got %q", cmd)
	}
}

func TestDSMLStringAttrCaseInsensitive(t *testing.T) {
	// The string attribute value may vary in case; only "false" (any case)
	// coerces the value away from string.
	chunks := []string{
		"<|DSML|tool_calls>",
		"<|DSML|invoke name=\"configure\">",
		"<|DSML|parameter name=\"n\" string=\"False\">5</|DSML|parameter>",
		"<|DSML|parameter name=\"label\" string=\"TRUE\">keep</|DSML|parameter>",
		"</|DSML|invoke>",
		"</|DSML|tool_calls>",
	}
	_, names, inputs, _ := collectDSMLEvents(t, chunks)
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
	if v, ok := args["n"].(float64); !ok || v != 5 {
		t.Fatalf("string=\"False\" must coerce to number 5, got %v", args["n"])
	}
	if args["label"] != "keep" {
		t.Fatalf("string=\"TRUE\" must stay a string, got %v", args["label"])
	}
}

func TestDSMLInvokeWithoutParams(t *testing.T) {
	// An invoke with no parameter children is valid: it emits a tool_use with
	// empty input and does not break the stream.
	chunks := []string{
		"<|DSML|tool_calls>",
		"<|DSML|invoke name=\"list_files\">",
		"</|DSML|invoke>",
		"</|DSML|tool_calls>",
	}
	_, names, _, text := collectDSMLEvents(t, chunks)
	if len(names) != 1 || names[0] != "list_files" {
		t.Fatalf("expected tool_use list_files, got %v", names)
	}
	if strings.Contains(text, "DSML") {
		t.Fatalf("markers must not leak, got %q", text)
	}
}

func TestDSMLSecondBlockAfterCompletedOne(t *testing.T) {
	// A fully closed block resets DSML state; a later second block must parse,
	// and plain text between the blocks must be emitted as text.
	chunks := []string{
		"<|DSML|tool_calls>",
		"<|DSML|invoke name=\"Glob\">",
		"<|DSML|parameter name=\"pattern\" string=\"true\">**/a*.go</|DSML|parameter>",
		"</|DSML|invoke>",
		"</|DSML|tool_calls>",
		"中间说明",
		"<|DSML|tool_calls>",
		"<|DSML|invoke name=\"Read\">",
		"<|DSML|parameter name=\"file_path\" string=\"true\">/tmp/x</|DSML|parameter>",
		"</|DSML|invoke>",
		"</|DSML|tool_calls>",
	}
	_, names, inputs, text := collectDSMLEvents(t, chunks)
	if len(names) != 2 || names[0] != "Glob" || names[1] != "Read" {
		t.Fatalf("expected Glob then Read, got %v", names)
	}
	if len(inputs) != 2 {
		t.Fatalf("expected two inputs, got %d", len(inputs))
	}
	if !strings.Contains(text, "中间说明") {
		t.Fatalf("text between blocks must be emitted, got %q", text)
	}
}

func TestDSMLProseAndStartTagSameChunk(t *testing.T) {
	// Streaming: prose and the DSML start tag arrive in the SAME chunk.
	// Prose must be emitted as text, then the block parsed as a tool call.
	chunks := []string{
		"好的,我来查找:<|DSML|tool_calls>",
		"<|DSML|invoke name=\"Glob\">",
		"<|DSML|parameter name=\"pattern\" string=\"true\">**/x*.go</|DSML|parameter>",
		"</|DSML|invoke>",
		"</|DSML|tool_calls>",
	}
	_, names, _, text := collectDSMLEvents(t, chunks)
	if len(names) != 1 || names[0] != "Glob" {
		t.Fatalf("expected tool_use Glob, got %v", names)
	}
	if !strings.Contains(text, "好的") {
		t.Fatalf("same-chunk prose must be emitted as text, got %q", text)
	}
	if strings.Contains(text, "DSML") {
		t.Fatalf("markers must not leak, got %q", text)
	}
}

func TestReproNonStreamDSMLBlock(t *testing.T) {
	// DeepSeek non-stream response: content holds prose THEN a full DSML block.
	content := "┆ Listing all project source files\nNow let me read the core Java files.\n\n" +
		"< | DSML | tool_calls>\n" +
		"< | DSML | invoke name=\"PowerShell\">\n" +
		"< | DSML | parameter name=\"command\" string=\"true\">Get-Content \"src\\main\\java\\com\\cebbank\\cdcs\\customer\\merge\\BathCustomerMergeApplication.java\"\n-Raw</ | DSML | parameter>\n" +
		"< | DSML | parameter name=\"description\" string=\"true\">Read main application class</ | DSML | parameter>\n" +
		"</ | DSML | invoke>\n" +
		"</ | DSML | tool_calls>"
	body, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{
			"message":       map[string]any{"content": content, "role": "assistant"},
			"finish_reason": "tool_calls",
		}},
	})
	var names []string
	var textSB strings.Builder
	emit := func(ev anthropicmessages.MessageStreamEvent) error {
		var m struct {
			Type  string         `json:"type"`
			Block map[string]any `json:"content_block"`
			Delta map[string]any `json:"delta"`
		}
		_ = json.Unmarshal(ev.Raw, &m)
		if m.Type == "content_block_start" {
			if t2, _ := m.Block["type"].(string); t2 == "tool_use" {
				if n, _ := m.Block["name"].(string); n != "" {
					names = append(names, n)
				}
			}
		}
		if m.Type == "content_block_delta" {
			if dt, _ := m.Delta["type"].(string); dt == "text_delta" {
				if txt, _ := m.Delta["text"].(string); txt != "" {
					textSB.WriteString(txt)
				}
			}
		}
		return nil
	}
	if err := ReplayOpenAINonStreamChatResponse(body, "deepseek-v4", emit); err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "PowerShell" {
		t.Fatalf("expected 1 PowerShell tool_use, got %v", names)
	}
	if !strings.Contains(textSB.String(), "Listing all project") {
		t.Fatalf("prose before block lost, text=%q", textSB.String())
	}
	if strings.Contains(textSB.String(), "DSML") {
		t.Fatalf("DSML markers leaked, text=%q", textSB.String())
	}
}

const reproBlock = `< | DSML | tool_calls>
< | DSML | invoke name="PowerShell">
< | DSML | parameter name="command" string="true">Get-Content "src\main\java\com\cebbank\cdcs\customer\merge\BathCustomerMergeApplication.java"
-Raw</ | DSML | parameter>
< | DSML | parameter name="description" string="true">Read main application class</ | DSML | parameter>
</ | DSML | invoke>
< | DSML | invoke name="PowerShell">
< | DSML | parameter name="command" string="true">Get-Content
"src\main\java\com\cebbank\cdcs\customer\merge\handler\BathAuthCustomerMergeJobHandler.java" -Raw</ | DSML | parameter>
< | DSML | parameter name="description" string="true">Read job handler class</ | DSML | parameter>
</ | DSML | invoke>
< | DSML | invoke name="PowerShell">
< | DSML | parameter name="command" string="true">Get-Content "src\main\java\com\cebbank\cdcs\customer\merge\service\AuthCustomerMergeService.java"
-Raw</ | DSML | parameter>
< | DSML | parameter name="description" string="true">Read merge service class</ | DSML | parameter>
</ | DSML | invoke>
</ | DSML | tool_calls>`

// Real SSE frames are dozens of chars; split at 60 and 12 runes per chunk to
// cover both realistic granularities, including mid-word start-tag splits.
func TestReproRealChunkSize(t *testing.T) {
	for _, n := range []int{60, 12} {
		chunks := splitChunksSmall(reproBlock, n)
		_, names, inputs, text := collectDSMLEvents(t, chunks)
		t.Logf("chunk=%d names=%v inputs=%d", n, names, len(inputs))
		if len(names) != 3 || names[0] != "PowerShell" {
			t.Fatalf("chunk=%d: expected 3 PowerShell, got %v", n, names)
		}
		if strings.Contains(text, "DSML") {
			t.Fatalf("chunk=%d: markers leaked into text: %q", n, text)
		}
		if len(inputs) != 3 {
			t.Fatalf("chunk=%d: expected 3 inputs, got %d", n, len(inputs))
		}
		var args map[string]any
		if err := json.Unmarshal([]byte(inputs[0]), &args); err != nil {
			t.Fatalf("chunk=%d: bad args: %v", n, err)
		}
		cmd, _ := args["command"].(string)
		if !strings.Contains(cmd, "BathCustomerMergeApplication.java") {
			t.Fatalf("chunk=%d: command missing file, got %q", n, cmd)
		}
		if !strings.Contains(cmd, "Get-Content") {
			t.Fatalf("chunk=%d: command missing Get-Content, got %q", n, cmd)
		}
	}
}

func splitChunksSmall(s string, n int) []string {
	rs := []rune(s)
	var out []string
	for i := 0; i < len(rs); i += n {
		end := i + n
		if end > len(rs) {
			end = len(rs)
		}
		out = append(out, string(rs[i:end]))
	}
	return out
}

func TestDSMLProseMentionsDSMLWithoutBlock(t *testing.T) {
	// Ordinary text that merely MENTIONS "DSML" (no actual markers) must still
	// flow through as text — the strong signal is the "<DSML" token, not the
	// bare word.
	chunks := []string{
		"DSML is a tool-call markup format used by DeepSeek.",
		"这里提到 DSML 但没有真实标记。",
	}
	_, names, _, text := collectDSMLEvents(t, chunks)
	if len(names) != 0 {
		t.Fatalf("mention without markers must not parse tool_use, got %v", names)
	}
	if !strings.Contains(text, "DSML is a tool-call") {
		t.Fatalf("mention must be emitted as text, got %q", text)
	}
}

func TestDSMLProseCrossChunkFragment(t *testing.T) {
	// Subagent outputs prose, then a start tag SPLIT across chunks ("tool_call"
	// then "s>"), then the invoke. The split tag must not leak as text and the
	// prose must still be emitted.
	chunks := []string{
		"        Let me use bash to read the files:\n\n        < | DSML | tool_call",
		"s>\n        < | DSML | invoke name=\"Bash\">\n",
		"        < | DSML | parameter name=\"command\" string=\"true\">cat x.txt</ | DSML | parameter>\n",
		"        </ | DSML | invoke>\n        </ | DSML | tool_calls>",
	}
	_, names, inputs, text := collectDSMLEvents(t, chunks)
	if len(names) != 1 || names[0] != "Bash" {
		t.Fatalf("expected tool_use Bash, got %v", names)
	}
	if len(inputs) != 1 || !strings.Contains(inputs[0], "cat x.txt") {
		t.Fatalf("expected command arg, got %v", inputs)
	}
	if !strings.Contains(text, "Let me use bash to read the files:") {
		t.Fatalf("prose must be emitted as text, got %q", text)
	}
	if strings.Contains(text, "DSML") {
		t.Fatalf("split start tag must not leak, got %q", text)
	}
}
