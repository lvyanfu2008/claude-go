package bashprepare

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPrepare_trimAndCommand(t *testing.T) {
	r := Prepare(StdinRequest{Input: "  echo hi  ", Shell: "bash"})
	if r.Reject != nil {
		t.Fatalf("unexpected reject: %+v", r.Reject)
	}
	if r.Command != "echo hi" {
		t.Fatalf("command %q", r.Command)
	}
}

func TestPrepare_emptyReject(t *testing.T) {
	r := Prepare(StdinRequest{Input: "   \t\n  "})
	if r.Reject == nil || r.Reject.Reason == "" {
		t.Fatalf("want reject, got %+v", r)
	}
}

func TestPrepare_nullByteReject(t *testing.T) {
	r := Prepare(StdinRequest{Input: "foo\x00bar"})
	if r.Reject == nil {
		t.Fatal("expected reject")
	}
}

func TestPrepare_invalidUTF8Reject(t *testing.T) {
	r := Prepare(StdinRequest{Input: string([]byte{0xff, 0xfe, 0xfd})})
	if r.Reject == nil || r.Reject.Reason == "" {
		t.Fatalf("want reject for invalid UTF-8, got %+v", r)
	}
}

func TestPrepare_longWarning(t *testing.T) {
	long := strings.Repeat("a", maxCommandRunes+1)
	r := Prepare(StdinRequest{Input: long})
	if r.Reject != nil {
		t.Fatalf("unexpected reject")
	}
	if len(r.Warnings) == 0 {
		t.Fatal("expected warning for long command")
	}
}

func TestRun_validJSON(t *testing.T) {
	out, err := Run([]byte(`{"input":"ls","shell":"bash"}`))
	if err != nil {
		t.Fatal(err)
	}
	var res Result
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatal(err)
	}
	if res.Command != "ls" {
		t.Fatalf("got %q", res.Command)
	}
}
