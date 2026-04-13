package toolinput

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"goc/ccb-engine/internal/anthropic"
	"goc/internal/jsonschemavalidate"
	"goc/internal/toolrefine"
)

//go:embed testdata/zodparity/testfixtures
var zodGoldenFixtures embed.FS

//go:embed testdata/zodparity/expected.jsonl
var zodGoldenExpected string

type zodGoldenLine struct {
	Tool      string `json:"tool"`
	Fixture   string `json:"fixture"`
	ZodAccept bool   `json:"zodAccept"`
}

// TestJSONSchemaAndRefinesMatchZodGoldens ensures Go validation (export JSON Schema + toolrefine)
// agrees with Zod safeParse goldens from claude-code/scripts/zod-parity-goldens.ts.
func TestJSONSchemaAndRefinesMatchZodGoldens(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(zodGoldenExpected))
	n := 0
	for scanner.Scan() {
		n++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var gl zodGoldenLine
		if err := json.Unmarshal(line, &gl); err != nil {
			t.Fatalf("line %d: %v", n, err)
		}
		path := fmt.Sprintf("testdata/zodparity/testfixtures/%s/%s.json", gl.Tool, gl.Fixture)
		raw, err := zodGoldenFixtures.ReadFile(path)
		if err != nil {
			t.Fatalf("line %d read %s: %v", n, path, err)
		}
		schema, ok := anthropic.InputSchemaFromTSAPIExport(gl.Tool)
		if !ok {
			t.Fatalf("line %d: no export schema for tool %q", n, gl.Tool)
		}
		err = jsonschemavalidate.Validate(gl.Tool, schema, raw)
		if err == nil {
			err = toolrefine.AfterJSONSchema(gl.Tool, raw)
		}
		goAccept := err == nil
		if goAccept != gl.ZodAccept {
			t.Fatalf("line %d tool=%s fixture=%s: zodAccept=%v goAccept=%v goErr=%v", n, gl.Tool, gl.Fixture, gl.ZodAccept, goAccept, err)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}
