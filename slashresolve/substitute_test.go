package slashresolve

import (
	"strings"
	"testing"
)

func TestParseArguments_quoted(t *testing.T) {
	got := ParseArguments(`foo "hello world" baz`)
	if len(got) != 3 || got[1] != "hello world" {
		t.Fatalf("got %#v", got)
	}
}

func TestSubstituteArguments_namedAndIndexed(t *testing.T) {
	content := "Hi $who and $ARGUMENTS[1] and $0"
	names := []string{"who"}
	got := SubstituteArguments(content, "x y", false, names)
	if !strings.Contains(got, "x") || !strings.Contains(got, "y") {
		t.Fatalf("got %q", got)
	}
}

func TestSubstituteArguments_appendWhenNoPlaceholder(t *testing.T) {
	got := SubstituteArguments("plain", "a b", true, nil)
	if !strings.Contains(got, "ARGUMENTS: a b") {
		t.Fatalf("got %q", got)
	}
}

func TestParseArgumentNames_filtersNumeric(t *testing.T) {
	got := ParseArgumentNames("foo 1 bar")
	if len(got) != 2 || got[0] != "foo" || got[1] != "bar" {
		t.Fatalf("got %#v", got)
	}
}
