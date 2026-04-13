package messagerow

import (
	"encoding/json"
	"testing"

	"goc/types"
)

func TestSegmentsFromAttachment_skillListing_hiddenWhenInitial(t *testing.T) {
	att, _ := json.Marshal(map[string]any{
		"type":       "skill_listing",
		"content":    "- a: x\n- b: y",
		"skillCount": 2,
		"isInitial":  true,
	})
	msg := types.Message{Type: types.MessageTypeAttachment, Attachment: att}
	segs := SegmentsFromMessage(msg)
	if len(segs) != 0 {
		t.Fatalf("TS hides initial batch: got %+v", segs)
	}
}

func TestSegmentsFromAttachment_skillListing_showsCount(t *testing.T) {
	att, _ := json.Marshal(map[string]any{
		"type":       "skill_listing",
		"content":    "- a: x\n- b: y",
		"skillCount": 2,
		"isInitial":  false,
	})
	msg := types.Message{Type: types.MessageTypeAttachment, Attachment: att}
	segs := SegmentsFromMessage(msg)
	if len(segs) != 1 || segs[0].Kind != SegSkillListingAvailable || segs[0].Num != 2 {
		t.Fatalf("%+v", segs)
	}
}

func TestSegmentsFromAttachment_skillListing_infersCountFromContent(t *testing.T) {
	att, _ := json.Marshal(map[string]any{
		"type":    "skill_listing",
		"content": "- one: a\n- two: b",
	})
	msg := types.Message{Type: types.MessageTypeAttachment, Attachment: att}
	segs := SegmentsFromMessage(msg)
	if len(segs) != 1 || segs[0].Num != 2 {
		t.Fatalf("%+v", segs)
	}
}
