package main

import "testing"

func TestStripTranscriptExportTrailingSpaces(t *testing.T) {
	in := "hello  \nworld\t\n"
	got := stripTranscriptExportTrailingSpaces(in)
	want := "hello\nworld\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
