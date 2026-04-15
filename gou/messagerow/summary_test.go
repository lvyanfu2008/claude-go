package messagerow

import "testing"

func TestSearchReadSummaryText_readOnly(t *testing.T) {
	got := SearchReadSummaryText(false, 0, 2, 0, 0, 0, 0, 0, 0, nil, nil, nil)
	if got != "Read 2 files" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchReadSummaryText_searchAndRead(t *testing.T) {
	got := SearchReadSummaryText(false, 1, 2, 0, 0, 0, 0, 0, 0, nil, nil, nil)
	if got != "Searched for 1 pattern, Read 2 files" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchReadSummaryText_activeTrailingEllipsis(t *testing.T) {
	got := SearchReadSummaryText(true, 1, 0, 0, 0, 0, 0, 0, 0, nil, nil, nil)
	if got != "Searching for 1 pattern…" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchReadSummaryText_listRepl(t *testing.T) {
	got := SearchReadSummaryText(false, 0, 0, 2, 3, 0, 0, 0, 0, nil, nil, nil)
	if got != "Listed 2 directories, REPL'd 3 times" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchReadSummaryText_memory(t *testing.T) {
	got := SearchReadSummaryText(false, 0, 0, 0, 0, 0, 1, 0, 0, nil, nil, nil)
	if got != "Recalled 1 memory" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchReadSummaryText_teamMemory(t *testing.T) {
	tr := 2
	got := SearchReadSummaryText(false, 0, 0, 0, 0, 0, 0, 0, 0, &tr, nil, nil)
	if got != "Recalled 2 team memories" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchReadSummaryText_empty(t *testing.T) {
	got := SearchReadSummaryText(false, 0, 0, 0, 0, 0, 0, 0, 0, nil, nil, nil)
	if got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchReadSummaryText_bash(t *testing.T) {
	got := SearchReadSummaryText(false, 0, 0, 0, 0, 2, 0, 0, 0, nil, nil, nil)
	if got != "Ran 2 bash commands" {
		t.Fatalf("got %q", got)
	}
	gotActive := SearchReadSummaryText(true, 0, 0, 0, 0, 1, 0, 0, 0, nil, nil, nil)
	if gotActive != "Running 1 bash command…" {
		t.Fatalf("got %q", gotActive)
	}
}
