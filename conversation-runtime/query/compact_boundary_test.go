package query

import (
	"testing"

	"goc/types"
)

func TestMessagesAfterCompactBoundary_none(t *testing.T) {
	ms := []types.Message{
		{Type: types.MessageTypeUser, UUID: "1"},
	}
	got := MessagesAfterCompactBoundary(ms, CompactBoundaryOpts{})
	if len(got) != 1 || got[0].UUID != "1" {
		t.Fatalf("%#v", got)
	}
}

func TestMessagesAfterCompactBoundary_withMarker(t *testing.T) {
	sub := "compact_boundary"
	ms := []types.Message{
		{Type: types.MessageTypeUser, UUID: "old"},
		{Type: types.MessageTypeSystem, UUID: "b", Subtype: &sub},
		{Type: types.MessageTypeUser, UUID: "new"},
	}
	got := MessagesAfterCompactBoundary(ms, CompactBoundaryOpts{})
	if len(got) != 2 || got[0].UUID != "b" || got[1].UUID != "new" {
		t.Fatalf("%#v", got)
	}
}
