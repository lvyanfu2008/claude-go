// Attachment rows mirror TS AttachmentMessage.tsx (subset used by gou-demo store).

package messagerow

import (
	"encoding/json"
	"strings"

	"goc/types"
)

// skillListingEntryLines counts formatted listing lines like "- name: desc" (TS formatCommandsWithinBudget).
func skillListingEntryLines(inner string) int {
	n := 0
	for _, line := range strings.Split(inner, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "-") {
			n++
		}
	}
	return n
}

func segmentsFromAttachment(msg types.Message) []Segment {
	if len(msg.Attachment) == 0 {
		return nil
	}
	var att struct {
		Type       string `json:"type"`
		Content    string `json:"content"`
		SkillCount int    `json:"skillCount"`
		IsInitial  bool   `json:"isInitial"`
	}
	if err := json.Unmarshal(msg.Attachment, &att); err != nil || strings.TrimSpace(att.Type) == "" {
		return []Segment{{Kind: SegTextMarkdown, Text: "[attachment · invalid JSON]"}}
	}
	switch att.Type {
	case "skill_listing":
		if att.IsInitial {
			return nil
		}
		n := att.SkillCount
		if n <= 0 {
			n = skillListingEntryLines(att.Content)
		}
		if n <= 0 {
			return nil
		}
		return []Segment{{Kind: SegSkillListingAvailable, Num: n}}
	default:
		// TS default: null render for most types; show a one-line hint for unknowns in this terminal port.
		return []Segment{{Kind: SegDisplayHint, Text: "attachment · " + att.Type}}
	}
}
