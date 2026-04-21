package messagerow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatWriteEditToolResultBodyIfApplicable_updatePatch(t *testing.T) {
	raw := json.RawMessage(`{
	  "type": "update",
	  "filePath": "src/a.go",
	  "content": "x",
	  "structuredPatch": [
	    {
	      "oldStart": 1, "oldLines": 1, "newStart": 1, "newLines": 2,
	      "lines": ["-old", "+new", "+more"]
	    }
	  ]
	}`)
	got, ok := FormatWriteEditToolResultBodyIfApplicable(raw)
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.Contains(got, "src/a.go") || !strings.Contains(got, "@@ -1,1 +1,2 @@") {
		t.Fatalf("unexpected diff header/body:\n%s", got)
	}
	if !strings.Contains(got, "-old") || !strings.Contains(got, "+new") {
		t.Fatalf("missing hunk lines:\n%s", got)
	}
}

func TestFormatWriteEditToolResultBodyIfApplicable_editNoType(t *testing.T) {
	raw := json.RawMessage(`{
	  "filePath": "b.txt",
	  "structuredPatch": [
	    {
	      "oldStart": 0, "oldLines": 0, "newStart": 1, "newLines": 1,
	      "lines": ["+hello"]
	    }
	  ]
	}`)
	got, ok := FormatWriteEditToolResultBodyIfApplicable(raw)
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.Contains(got, "+++ b.txt") || !strings.Contains(got, "+hello") {
		t.Fatalf("got:\n%s", got)
	}
}

func TestFormatWriteEditToolResultBodyIfApplicable_create(t *testing.T) {
	raw := json.RawMessage(`{
	  "type": "create",
	  "filePath": "new.txt",
	  "content": "line1\nline2",
	  "structuredPatch": []
	}`)
	got, ok := FormatWriteEditToolResultBodyIfApplicable(raw)
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.Contains(got, "new file") || !strings.Contains(got, "+line1") {
		t.Fatalf("got:\n%s", got)
	}
}

func TestFormatWriteEditToolResultBodyIfApplicable_doubleEncoded(t *testing.T) {
	inner := `{"type":"update","filePath":"x.go","structuredPatch":[{"oldStart":1,"oldLines":1,"newStart":1,"newLines":1,"lines":["-a","+b"]}]}`
	outer, err := json.Marshal(inner)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := FormatWriteEditToolResultBodyIfApplicable(json.RawMessage(outer))
	if !ok {
		t.Fatal("expected ok after unwrap")
	}
	if !strings.Contains(got, "+b") {
		t.Fatalf("got:\n%s", got)
	}
}

func TestIndentedWriteEditDiffLinesFromToolResultJSON(t *testing.T) {
	raw := `{"filePath":"f","structuredPatch":[{"oldStart":1,"oldLines":1,"newStart":1,"newLines":1,"lines":["-x"]}]}`
	lines, ok := IndentedWriteEditDiffLinesFromToolResultJSON(raw)
	if !ok || len(lines) < 2 {
		t.Fatalf("lines=%v ok=%v", lines, ok)
	}
	var found bool
	for _, ln := range lines {
		if strings.Contains(ln, "-x") {
			found = true
			if !strings.HasPrefix(ln, "  ") {
				t.Fatalf("expected 2-space indent, got %q", ln)
			}
		}
	}
	if !found {
		t.Fatalf("missing indented -x line in %#v", lines)
	}
}
