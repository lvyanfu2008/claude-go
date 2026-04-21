package message

import (
	"strings"
	"testing"
)

func TestToolResultTextPartsFromContent_string(t *testing.T) {
	p := toolResultTextPartsFromContent(`{"x":1}`)
	if len(p) != 1 || p[0] != `{"x":1}` {
		t.Fatalf("got %#v", p)
	}
}

func TestToolResultTextPartsFromContent_array(t *testing.T) {
	raw := `{"filePath":"a.go","structuredPatch":[{"oldStart":1,"oldLines":1,"newStart":1,"newLines":1,"lines":["-x","+y"]}]}`
	arr := []interface{}{
		map[string]interface{}{"type": "text", "text": raw},
	}
	p := toolResultTextPartsFromContent(arr)
	if len(p) != 1 || p[0] != raw {
		t.Fatalf("got %#v", p)
	}
}

func TestWriteEditDiffLinesFromToolResultBlock_arrayContent(t *testing.T) {
	raw := `{"filePath":"a.go","structuredPatch":[{"oldStart":1,"oldLines":1,"newStart":1,"newLines":1,"lines":["-x","+y"]}]}`
	block := map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{"type": "text", "text": raw},
		},
	}
	lines, ok := writeEditDiffLinesFromToolResultBlock(block)
	if !ok || len(lines) < 2 {
		t.Fatalf("ok=%v lines=%v", ok, lines)
	}
	found := false
	for _, ln := range lines {
		if strings.Contains(ln, "+y") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected +y in lines: %#v", lines)
	}
}

func TestWriteEditDiffLinesFromToolResultBlock_promptPrefersDiff(t *testing.T) {
	raw := `{"type":"create","filePath":"n.txt","content":"hi\n","structuredPatch":[]}`
	block := map[string]interface{}{"content": raw}
	lines, ok := writeEditDiffLinesFromToolResultBlock(block)
	if !ok || len(lines) == 0 {
		t.Fatalf("ok=%v lines=%v", ok, lines)
	}
	if !strings.Contains(lines[0], "+++") && !strings.Contains(lines[0], "new file") {
		t.Fatalf("expected create diff header, first line=%q", lines[0])
	}
}
