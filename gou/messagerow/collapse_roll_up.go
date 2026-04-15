// CollapseReadSearchTail merges a trailing run of Read/Grep/Glob (+ collapsible Bash) tool_use + tool_result pairs
// into one collapsed_read_search row (TS collapseReadSearchGroups tail subset; no memory/MCP/hooks).
package messagerow

import (
	"sort"
	"strings"

	"goc/types"
)

// CollapseReadSearchTail replaces the longest trailing suffix of
// [assistant: single tool_use][user: single tool_result] pairs (Read, Grep, Glob, Bash when TS-classified search/read/list, or any Bash when GOU_DEMO_COLLAPSE_ALL_BASH=1)
// with one collapsed_read_search message. No-op if fewer than one complete pair.
func CollapseReadSearchTail(msgs *[]types.Message) {
	if msgs == nil || len(*msgs) < 2 {
		return
	}
	slice := *msgs
	suffixStart := len(slice)
	for n := len(slice); n >= 2; {
		u := slice[n-1]
		a := slice[n-2]
		if u.Type != types.MessageTypeUser || a.Type != types.MessageTypeAssistant {
			break
		}
		tu, okA := assistantSingleToolUse(a)
		if !okA || !canRollupAssistantToolPair(tu) {
			break
		}
		tr, okU := userSingleToolResult(u)
		if !okU || tr.ToolUseID == "" || tu.ID == "" || tr.ToolUseID != tu.ID {
			break
		}
		suffixStart = n - 2
		n -= 2
	}
	if suffixStart >= len(slice) {
		return
	}
	tail := slice[suffixStart:]
	if len(tail) < 2 {
		return
	}
	collapsed := buildCollapsedReadSearch(tail)
	prefix := slice[:suffixStart]
	out := make([]types.Message, 0, len(prefix)+1)
	out = append(out, prefix...)
	out = append(out, collapsed)
	*msgs = out
}

// canRollupAssistantToolPair mirrors TS getToolSearchOrReadInfo collapsible; optional GOU_DEMO_COLLAPSE_ALL_BASH=1 matches TS fullscreen (any Bash rolls up).
func canRollupAssistantToolPair(tu types.MessageContentBlock) bool {
	name := strings.TrimSpace(tu.Name)
	switch name {
	case "Read", "Grep", "Glob":
		return true
	case "Bash", "BashZog":
		if CollapseAllBashFromEnv() {
			return true
		}
		m := decodeToolInputMap(tu.Input)
		cmd, _ := m["command"].(string)
		isS, isR, isL := IsSearchOrReadBashCommand(cmd)
		return isS || isR || isL
	default:
		return false
	}
}

func buildCollapsedReadSearch(tail []types.Message) types.Message {
	first := tail[0]
	nested := make([]types.Message, len(tail))
	copy(nested, tail)

	searchCount := 0
	listCount := 0
	bashCount := 0
	readPathSet := make(map[string]struct{})
	readOpCount := 0
	var searchArgs []string
	var latestHint *string

	for i := 0; i+1 < len(tail); i += 2 {
		a := tail[i]
		tu, _ := assistantSingleToolUse(a)
		m := decodeToolInputMap(tu.Input)
		switch strings.TrimSpace(tu.Name) {
		case "Read":
			fp := strFromMap(m, "file_path")
			if fp != "" {
				d := DisplayPathForActivity(fp)
				readPathSet[d] = struct{}{}
				latestHint = strPtr(d)
			} else {
				readOpCount++
			}
		case "Grep", "Glob":
			searchCount++
			pat := strFromMap(m, "pattern")
			if pat != "" {
				searchArgs = append(searchArgs, pat)
				latestHint = strPtr(`"` + pat + `"`)
			}
		case "Bash", "BashZog":
			cmd := strFromMap(m, "command")
			isS, isR, isL := IsSearchOrReadBashCommand(cmd)
			// TS collapseReadSearchGroups: isList before isSearch before read; generic Bash only in fullscreen (here: CollapseAllBashFromEnv).
			if isL {
				listCount++
			} else if isS {
				searchCount++
			} else if isR {
				readOpCount++
			} else if CollapseAllBashFromEnv() {
				bashCount++
			}
			if strings.TrimSpace(cmd) != "" {
				latestHint = strPtr(commandAsHintBody(cmd))
			}
		}
	}

	readCount := len(readPathSet)
	if readCount == 0 {
		readCount = readOpCount
	}
	readPaths := make([]string, 0, len(readPathSet))
	for p := range readPathSet {
		readPaths = append(readPaths, p)
	}
	sort.Strings(readPaths)

	dm := first
	out := types.Message{
		Type:              types.MessageTypeCollapsedReadSearch,
		UUID:              "collapsed-" + first.UUID,
		SearchCount:       searchCount,
		ReadCount:         readCount,
		ListCount:         listCount,
		ReadFilePaths:     readPaths,
		SearchArgs:        searchArgs,
		LatestDisplayHint: latestHint,
		Messages:          nested,
		DisplayMessage:    &dm,
	}
	if bashCount > 0 {
		bc := bashCount
		out.BashCount = &bc
	}
	if first.Timestamp != nil {
		ts := *first.Timestamp
		out.Timestamp = &ts
	}
	return out
}

func strPtr(s string) *string { return &s }
