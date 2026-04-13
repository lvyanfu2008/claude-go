// SearchReadSummaryText mirrors src/utils/collapseReadSearch.ts getSearchReadSummaryText
// (search/read/list/repl + user memory + team memory counts). Used for collapsed_read_search UI.
package messagerow

import (
	"fmt"
	"strings"

	"goc/types"
)

// CtrlOToExpandHint is the static transcript-expand hint (gou-demo: ctrl+o opens TS-style transcript screen; hint still matches Ink CtrlOToExpand wording).
const CtrlOToExpandHint = " (ctrl+o to expand)"

// SearchReadSummaryText builds a comma-separated summary like TS getSearchReadSummaryText.
// Pass nil team* pointers when those counts are absent (JSON omitempty).
func SearchReadSummaryText(
	isActive bool,
	searchCount, readCount, listCount, replCount int,
	memoryReadCount, memorySearchCount, memoryWriteCount int,
	teamMemoryReadCount, teamMemorySearchCount, teamMemoryWriteCount *int,
) string {
	var parts []string

	if memoryReadCount > 0 {
		parts = append(parts, memoryReadSummaryPart(isActive, len(parts) == 0, memoryReadCount))
	}
	if memorySearchCount > 0 {
		parts = append(parts, memorySearchSummaryPart(isActive, len(parts) == 0))
	}
	if memoryWriteCount > 0 {
		parts = append(parts, memoryWriteSummaryPart(isActive, len(parts) == 0, memoryWriteCount))
	}
	appendTeamMemorySummaryParts(&parts, isActive, teamMemoryReadCount, teamMemorySearchCount, teamMemoryWriteCount)

	if searchCount > 0 {
		parts = append(parts, searchSummaryPart(isActive, len(parts) == 0, searchCount))
	}
	if readCount > 0 {
		parts = append(parts, readSummaryPart(isActive, len(parts) == 0, readCount))
	}
	if listCount > 0 {
		parts = append(parts, listSummaryPart(isActive, len(parts) == 0, listCount))
	}
	if replCount > 0 {
		parts = append(parts, replSummaryPart(isActive, replCount))
	}

	text := strings.Join(parts, ", ")
	if isActive && text != "" {
		return text + "…"
	}
	return text
}

// SearchReadSummaryTextFromMessage reads counts from types.Message (collapsed_read_search row).
func SearchReadSummaryTextFromMessage(isActive bool, msg types.Message) string {
	return SearchReadSummaryText(
		isActive,
		msg.SearchCount, msg.ReadCount, msg.ListCount, msg.ReplCount,
		msg.MemoryReadCount, msg.MemorySearchCount, msg.MemoryWriteCount,
		msg.TeamMemoryReadCount, msg.TeamMemorySearchCount, msg.TeamMemoryWriteCount,
	)
}

func memoryReadSummaryPart(isActive, firstInLine bool, memoryReadCount int) string {
	verb := pickVerb(isActive, firstInLine, "Recalling", "recalling", "Recalled", "recalled")
	noun := "memories"
	if memoryReadCount == 1 {
		noun = "memory"
	}
	return fmt.Sprintf("%s %d %s", verb, memoryReadCount, noun)
}

func memorySearchSummaryPart(isActive, firstInLine bool) string {
	return pickVerb(isActive, firstInLine, "Searching", "searching", "Searched", "searched") + " memories"
}

func memoryWriteSummaryPart(isActive, firstInLine bool, memoryWriteCount int) string {
	verb := pickVerb(isActive, firstInLine, "Writing", "writing", "Wrote", "wrote")
	noun := "memories"
	if memoryWriteCount == 1 {
		noun = "memory"
	}
	return fmt.Sprintf("%s %d %s", verb, memoryWriteCount, noun)
}

func appendTeamMemorySummaryParts(parts *[]string, isActive bool, tr, ts, tw *int) {
	teamReadCount := 0
	if tr != nil {
		teamReadCount = *tr
	}
	teamSearchCount := 0
	if ts != nil {
		teamSearchCount = *ts
	}
	teamWriteCount := 0
	if tw != nil {
		teamWriteCount = *tw
	}
	if teamReadCount > 0 {
		verb := pickVerb(isActive, len(*parts) == 0, "Recalling", "recalling", "Recalled", "recalled")
		noun := "memories"
		if teamReadCount == 1 {
			noun = "memory"
		}
		*parts = append(*parts, fmt.Sprintf("%s %d team %s", verb, teamReadCount, noun))
	}
	if teamSearchCount > 0 {
		verb := pickVerb(isActive, len(*parts) == 0, "Searching", "searching", "Searched", "searched")
		*parts = append(*parts, verb+" team memories")
	}
	if teamWriteCount > 0 {
		verb := pickVerb(isActive, len(*parts) == 0, "Writing", "writing", "Wrote", "wrote")
		noun := "memories"
		if teamWriteCount == 1 {
			noun = "memory"
		}
		*parts = append(*parts, fmt.Sprintf("%s %d team %s", verb, teamWriteCount, noun))
	}
}

func searchSummaryPart(isActive, firstInLine bool, searchCount int) string {
	verb := pickVerb(isActive, firstInLine, "Searching for", "searching for", "Searched for", "searched for")
	noun := "patterns"
	if searchCount == 1 {
		noun = "pattern"
	}
	return fmt.Sprintf("%s %d %s", verb, searchCount, noun)
}

func readSummaryPart(isActive, firstInLine bool, readCount int) string {
	verb := pickVerb(isActive, firstInLine, "Reading", "reading", "Read", "read")
	noun := "files"
	if readCount == 1 {
		noun = "file"
	}
	return fmt.Sprintf("%s %d %s", verb, readCount, noun)
}

func listSummaryPart(isActive, firstInLine bool, listCount int) string {
	verb := pickVerb(isActive, firstInLine, "Listing", "listing", "Listed", "listed")
	noun := "directories"
	if listCount == 1 {
		noun = "directory"
	}
	return fmt.Sprintf("%s %d %s", verb, listCount, noun)
}

func replSummaryPart(isActive bool, replCount int) string {
	verb := "REPL'd"
	if isActive {
		verb = "REPL'ing"
	}
	noun := "times"
	if replCount == 1 {
		noun = "time"
	}
	return fmt.Sprintf("%s %d %s", verb, replCount, noun)
}

func pickVerb(isActive, firstInLine bool, activeFirst, activeRest, doneFirst, doneRest string) string {
	if isActive {
		if firstInLine {
			return activeFirst
		}
		return activeRest
	}
	if firstInLine {
		return doneFirst
	}
	return doneRest
}
