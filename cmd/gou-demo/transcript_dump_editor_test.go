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

func TestExportTranscriptWidth_atLeast80(t *testing.T) {
	t.Parallel()
	m := &model{cols: 70, width: 70}
	if w := exportTranscriptWidth(m); w < 80 {
		t.Fatalf("expected min width 80, got %d", w)
	}
}

func TestTranscriptBracketDumpScrollbackCmd_empty(t *testing.T) {
	t.Parallel()
	cmd := transcriptBracketDumpScrollbackCmd("")
	if cmd != nil {
		t.Fatal("expected nil cmd for empty plain body")
	}
}

func TestTranscriptBracketDumpScrollbackCmd_plainUsesPrintf(t *testing.T) {
	t.Parallel()
	cmd := transcriptBracketDumpScrollbackCmd("hello")
	if cmd == nil {
		t.Fatal("expected printf cmd")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("expected non-nil message from printf cmd")
	}
}
