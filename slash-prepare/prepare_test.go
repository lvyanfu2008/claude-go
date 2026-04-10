package slashprepare

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPrepare_searchArgs(t *testing.T) {
	r := Prepare(StdinRequest{Input: "/search foo bar"})
	if r.Reject != nil {
		t.Fatalf("unexpected reject: %+v", r.Reject)
	}
	if r.CommandName != "search" || r.Args != "foo bar" || r.IsMcp {
		t.Fatalf("got %+v", r)
	}
}

func TestPrepare_mcpForm(t *testing.T) {
	r := Prepare(StdinRequest{Input: "/mcp:tool (MCP) arg1 arg2"})
	if r.Reject != nil {
		t.Fatalf("unexpected reject: %+v", r.Reject)
	}
	if r.CommandName != "mcp:tool (MCP)" || r.Args != "arg1 arg2" || !r.IsMcp {
		t.Fatalf("got %+v", r)
	}
}

func TestPrepare_doubleSpaceArgs(t *testing.T) {
	// Mirrors TS: split(" ") leaves empty segment → args " b"
	r := Prepare(StdinRequest{Input: "/a  b"})
	if r.Reject != nil {
		t.Fatalf("unexpected reject: %+v", r.Reject)
	}
	if r.CommandName != "a" || r.Args != " b" || r.IsMcp {
		t.Fatalf("got %+v", r)
	}
}

func TestPrepare_notSlashReject(t *testing.T) {
	r := Prepare(StdinRequest{Input: "hello"})
	if r.Reject == nil || r.Reject.Reason != errSlashForm {
		t.Fatalf("got %+v", r)
	}
}

func TestPrepare_onlySlashReject(t *testing.T) {
	r := Prepare(StdinRequest{Input: "/"})
	if r.Reject == nil || r.Reject.Reason != errSlashForm {
		t.Fatalf("got %+v", r)
	}
}

func TestPrepare_invalidUTF8Reject(t *testing.T) {
	r := Prepare(StdinRequest{Input: string([]byte{0xff, 0xfe, 0xfd})})
	if r.Reject == nil || r.Reject.Reason == "" {
		t.Fatalf("got %+v", r)
	}
}

func TestPrepare_longWarning(t *testing.T) {
	long := "/" + strings.Repeat("a", maxInputRunes)
	r := Prepare(StdinRequest{Input: long})
	if r.Reject != nil {
		t.Fatalf("unexpected reject")
	}
	if len(r.Warnings) == 0 {
		t.Fatal("expected warning")
	}
}

func TestRun_validJSON(t *testing.T) {
	out, err := Run([]byte(`{"input":"/ls"}`))
	if err != nil {
		t.Fatal(err)
	}
	var res Result
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatal(err)
	}
	if res.CommandName != "ls" || res.Reject != nil {
		t.Fatalf("got %+v", res)
	}
}
