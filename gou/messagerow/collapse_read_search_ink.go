// Ink-parity implementation of claude-code/src/utils/collapseReadSearch.ts collapseReadSearchGroups (state machine).
package messagerow

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"goc/types"
)

const maxHintCharsInk = 300 // TS MAX_HINT_CHARS

func commandAsHintInk(command string) string {
	lines := strings.Split(command, "\n")
	var parts []string
	for _, ln := range lines {
		s := strings.TrimSpace(strings.Join(strings.Fields(ln), " "))
		if s == "" {
			continue
		}
		parts = append(parts, s)
	}
	cleaned := "$ " + strings.Join(parts, "\n")
	if utf8.RuneCountInString(cleaned) > maxHintCharsInk {
		return string([]rune(cleaned)[:maxHintCharsInk-1]) + "…"
	}
	return cleaned
}

func rawMessageContent(msg types.Message) json.RawMessage {
	if len(msg.Content) > 0 {
		return msg.Content
	}
	var env struct {
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(msg.Message, &env) == nil && len(env.Content) > 0 {
		return env.Content
	}
	return nil
}

func assistantContentBlocks(msg types.Message) []types.MessageContentBlock {
	raw := rawMessageContent(msg)
	if len(raw) == 0 {
		return nil
	}
	var blocks []types.MessageContentBlock
	if json.Unmarshal(raw, &blocks) != nil {
		return nil
	}
	return blocks
}

func allContentBlocks(msg types.Message) []types.MessageContentBlock {
	return assistantContentBlocks(msg)
}

func firstContentBlock(msg types.Message) (types.MessageContentBlock, bool) {
	blocks := assistantContentBlocks(msg)
	if len(blocks) == 0 {
		return types.MessageContentBlock{}, false
	}
	return blocks[0], true
}

func attachmentType(msg types.Message) string {
	if len(msg.Attachment) == 0 {
		return ""
	}
	var head struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(msg.Attachment, &head) != nil {
		return ""
	}
	return head.Type
}

func getCollapsibleToolInfoGo(msg types.Message) (name string, input map[string]any, info searchOrReadResult, ok bool) {
	switch msg.Type {
	case types.MessageTypeAssistant:
		b, okb := firstContentBlock(msg)
		if !okb || b.Type != "tool_use" || strings.TrimSpace(b.Name) == "" {
			return "", nil, searchOrReadResult{}, false
		}
		in := decodeToolInputMap(b.Input)
		info := getToolSearchOrReadInfoGo(strings.TrimSpace(b.Name), in)
		if !info.isCollapsible && !info.isREPL {
			return "", nil, searchOrReadResult{}, false
		}
		return strings.TrimSpace(b.Name), in, info, true
	case types.MessageTypeGroupedToolUse:
		if len(msg.Messages) == 0 {
			return "", nil, searchOrReadResult{}, false
		}
		b, okb := firstContentBlock(msg.Messages[0])
		if !okb || b.Type != "tool_use" {
			return "", nil, searchOrReadResult{}, false
		}
		in := decodeToolInputMap(b.Input)
		tn := strings.TrimSpace(msg.ToolName)
		info := getToolSearchOrReadInfoGo(tn, in)
		if !info.isCollapsible && !info.isREPL {
			return "", nil, searchOrReadResult{}, false
		}
		return tn, in, info, true
	default:
		return "", nil, searchOrReadResult{}, false
	}
}

func isTextBreakerInk(msg types.Message) bool {
	if msg.Type != types.MessageTypeAssistant {
		return false
	}
	b, ok := firstContentBlock(msg)
	if !ok || b.Type != "text" {
		return false
	}
	return strings.TrimSpace(b.Text) != ""
}

func isNonCollapsibleToolUseInk(msg types.Message) bool {
	switch msg.Type {
	case types.MessageTypeAssistant:
		b, ok := firstContentBlock(msg)
		if !ok || b.Type != "tool_use" {
			return false
		}
		name := strings.TrimSpace(b.Name)
		in := decodeToolInputMap(b.Input)
		return !isToolSearchOrReadGo(name, in)
	case types.MessageTypeGroupedToolUse:
		if len(msg.Messages) == 0 {
			return false
		}
		b, ok := firstContentBlock(msg.Messages[0])
		if !ok || b.Type != "tool_use" {
			return false
		}
		name := strings.TrimSpace(msg.ToolName)
		in := decodeToolInputMap(b.Input)
		return !isToolSearchOrReadGo(name, in)
	default:
		return false
	}
}

func isPreToolHookSummaryMsg(msg types.Message) bool {
	if msg.Type != types.MessageTypeSystem {
		return false
	}
	if msg.Subtype == nil || *msg.Subtype != "stop_hook_summary" {
		return false
	}
	if msg.HookLabel == nil || *msg.HookLabel != "PreToolUse" {
		return false
	}
	return true
}

func shouldSkipMessageInk(msg types.Message) bool {
	if msg.Type == types.MessageTypeAssistant {
		b, ok := firstContentBlock(msg)
		if ok && (b.Type == "thinking" || b.Type == "redacted_thinking") {
			return true
		}
	}
	if msg.Type == types.MessageTypeAttachment {
		return true
	}
	if msg.Type == types.MessageTypeSystem {
		return true
	}
	return false
}

func isCollapsibleToolUseInk(msg types.Message) bool {
	_, _, _, ok := getCollapsibleToolInfoGo(msg)
	return ok
}

func isCollapsibleToolResultInk(msg types.Message, toolUseIDs map[string]struct{}) bool {
	if msg.Type != types.MessageTypeUser || len(toolUseIDs) == 0 {
		return false
	}
	blocks := allContentBlocks(msg)
	var toolResults []types.MessageContentBlock
	for _, c := range blocks {
		if c.Type == "tool_result" && strings.TrimSpace(c.ToolUseID) != "" {
			toolResults = append(toolResults, c)
		}
	}
	if len(toolResults) == 0 {
		return false
	}
	for _, r := range toolResults {
		if _, ok := toolUseIDs[r.ToolUseID]; !ok {
			return false
		}
	}
	return true
}

func getToolUseIdsFromMessageInk(msg types.Message) []string {
	switch msg.Type {
	case types.MessageTypeAssistant:
		b, ok := firstContentBlock(msg)
		if ok && b.Type == "tool_use" && strings.TrimSpace(b.ID) != "" {
			return []string{strings.TrimSpace(b.ID)}
		}
	case types.MessageTypeGroupedToolUse:
		var ids []string
		for _, m := range msg.Messages {
			blocks := assistantContentBlocks(m)
			for _, b := range blocks {
				if b.Type == "tool_use" && strings.TrimSpace(b.ID) != "" {
					ids = append(ids, strings.TrimSpace(b.ID))
				}
			}
		}
		return ids
	}
	return nil
}

func countToolUsesInk(msg types.Message) int {
	if msg.Type == types.MessageTypeGroupedToolUse {
		return len(msg.Messages)
	}
	return 1
}

func getFilePathsFromReadMessageInk(msg types.Message) []string {
	var paths []string
	collect := func(m types.Message) {
		b, ok := firstContentBlock(m)
		if !ok || b.Type != "tool_use" || strings.TrimSpace(b.Name) != "Read" {
			return
		}
		in := decodeToolInputMap(b.Input)
		fp := strFromMap(in, "file_path")
		if strings.TrimSpace(fp) != "" {
			paths = append(paths, fp)
		}
	}
	switch msg.Type {
	case types.MessageTypeAssistant:
		collect(msg)
	case types.MessageTypeGroupedToolUse:
		for _, m := range msg.Messages {
			collect(m)
		}
	}
	return paths
}

// groupAccumulatorInk mirrors TS GroupAccumulator.
type groupAccumulatorInk struct {
	messages              []types.Message
	searchCount           int
	readFilePaths         map[string]struct{}
	readOperationCount    int
	listCount             int
	toolUseIds            map[string]struct{}
	memorySearchCount     int
	memoryReadFilePaths   map[string]struct{}
	memoryWriteCount      int
	nonMemSearchArgs      []string
	latestDisplayHint     *string
	mcpCallCount          int
	mcpServerNames        map[string]struct{}
	bashCount             int
	bashCommands          map[string]string
	commits               []types.GitCommitEntry
	pushes                []types.GitPushEntry
	branches              []types.GitBranchEntry
	prs                   []types.GitPrEntry
	gitOpBashCount        int
	hookTotalMs           int64
	hookCount             int
	hookInfos             []types.StopHookInfo
	relevantMemories      []types.MemoryAttachment
}

func createEmptyGroupInk() *groupAccumulatorInk {
	g := &groupAccumulatorInk{
		readFilePaths:       make(map[string]struct{}),
		toolUseIds:          make(map[string]struct{}),
		memoryReadFilePaths: make(map[string]struct{}),
		mcpServerNames:      make(map[string]struct{}),
	}
	if CollapseAllBashFromEnv() {
		g.bashCommands = make(map[string]string)
	}
	return g
}

func strPtrInk(s string) *string { return &s }

func appendToolUseContributionInk(msg types.Message, toolName string, input map[string]any, toolInfo searchOrReadResult, g *groupAccumulatorInk) {
	if g == nil {
		return
	}
	if toolInfo.isMemoryWrite {
		n := countToolUsesInk(msg)
		g.memoryWriteCount += n
		return
	}
	if toolInfo.isAbsorbedSilently {
		return
	}
	if toolInfo.mcpServerName != "" {
		n := countToolUsesInk(msg)
		g.mcpCallCount += n
		g.mcpServerNames[toolInfo.mcpServerName] = struct{}{}
		if q, ok := input["query"].(string); ok && strings.TrimSpace(q) != "" {
			g.latestDisplayHint = strPtrInk(`"` + strings.TrimSpace(q) + `"`)
		}
		return
	}
	if CollapseAllBashFromEnv() && toolInfo.isBash {
		n := countToolUsesInk(msg)
		g.bashCount += n
		cmd := strFromMap(input, "command")
		if strings.TrimSpace(cmd) != "" {
			g.latestDisplayHint = strPtrInk(commandAsHintInk(cmd))
			for _, id := range getToolUseIdsFromMessageInk(msg) {
				g.bashCommands[id] = cmd
			}
		}
		return
	}
	if toolInfo.isList {
		g.listCount += countToolUsesInk(msg)
		cmd := strFromMap(input, "command")
		if strings.TrimSpace(cmd) != "" {
			g.latestDisplayHint = strPtrInk(commandAsHintInk(cmd))
		}
		return
	}
	if toolInfo.isSearch {
		n := countToolUsesInk(msg)
		g.searchCount += n
		if isMemorySearchGo(input) {
			g.memorySearchCount += n
		} else {
			pat := strFromMap(input, "pattern")
			if pat == "" {
				pat = strFromMap(input, "glob")
			}
			if pat != "" {
				g.nonMemSearchArgs = append(g.nonMemSearchArgs, pat)
				g.latestDisplayHint = strPtrInk(`"` + pat + `"`)
			}
		}
		return
	}
	// Reads (and bash read classification)
	paths := getFilePathsFromReadMessageInk(msg)
	for _, filePath := range paths {
		g.readFilePaths[filePath] = struct{}{}
		if isAutoManagedMemoryFileGo(filePath) {
			g.memoryReadFilePaths[filePath] = struct{}{}
		} else {
			g.latestDisplayHint = strPtrInk(DisplayPathForActivity(filePath))
		}
	}
	if len(paths) == 0 {
		g.readOperationCount += countToolUsesInk(msg)
		cmd := strFromMap(input, "command")
		if strings.TrimSpace(cmd) != "" {
			g.latestDisplayHint = strPtrInk(commandAsHintInk(cmd))
		}
	}
}

func scanBashResultForGitOpsInk(msg types.Message, g *groupAccumulatorInk) {
	if g == nil || g.bashCommands == nil || len(g.bashCommands) == 0 || !CollapseAllBashFromEnv() {
		return
	}
	var out struct {
		Stdout string `json:"stdout"`
		Stderr string `json:"stderr"`
	}
	_ = json.Unmarshal(msg.ToolUseResult, &out)
	combined := out.Stdout + "\n" + out.Stderr
	for _, c := range allContentBlocks(msg) {
		if c.Type != "tool_result" || strings.TrimSpace(c.ToolUseID) == "" {
			continue
		}
		cmd := g.bashCommands[c.ToolUseID]
		if cmd == "" {
			continue
		}
		ce, pe, be, pr := detectGitOperationGo(cmd, combined)
		if ce != nil {
			g.commits = append(g.commits, *ce)
		}
		if pe != nil {
			g.pushes = append(g.pushes, *pe)
		}
		if be != nil {
			g.branches = append(g.branches, *be)
		}
		if pr != nil {
			g.prs = append(g.prs, *pr)
		}
		if ce != nil || pe != nil || be != nil || pr != nil {
			g.gitOpBashCount++
		}
	}
}

func absorbRelevantMemoriesInk(msg types.Message, g *groupAccumulatorInk) bool {
	if g == nil || len(msg.Attachment) == 0 || attachmentType(msg) != "relevant_memories" {
		return false
	}
	var env struct {
		Memories []types.MemoryAttachment `json:"memories"`
	}
	if json.Unmarshal(msg.Attachment, &env) != nil {
		return false
	}
	g.relevantMemories = append(g.relevantMemories, env.Memories...)
	return true
}

func createCollapsedGroupInk(group *groupAccumulatorInk) types.Message {
	first := group.messages[0]
	totalReadCount := group.readOperationCount
	if len(group.readFilePaths) > 0 {
		totalReadCount = len(group.readFilePaths)
	}
	toolMemoryReadCount := len(group.memoryReadFilePaths)
	memoryReadCount := toolMemoryReadCount + len(group.relevantMemories)
	nonMemReadPaths := make([]string, 0)
	for p := range group.readFilePaths {
		if _, mem := group.memoryReadFilePaths[p]; mem {
			continue
		}
		nonMemReadPaths = append(nonMemReadPaths, DisplayPathForActivity(p))
	}
	sort.Strings(nonMemReadPaths)
	searchCount := group.searchCount - group.memorySearchCount
	if searchCount < 0 {
		searchCount = 0
	}
	readCount := totalReadCount - toolMemoryReadCount
	if readCount < 0 {
		readCount = 0
	}
	out := types.Message{
		Type:              types.MessageTypeCollapsedReadSearch,
		UUID:              "collapsed-" + first.UUID,
		SearchCount:       searchCount,
		ReadCount:         readCount,
		ListCount:         group.listCount,
		ReadFilePaths:     nonMemReadPaths,
		SearchArgs:        append([]string(nil), group.nonMemSearchArgs...),
		LatestDisplayHint: group.latestDisplayHint,
		Messages:          append([]types.Message(nil), group.messages...),
		MemorySearchCount: group.memorySearchCount,
		MemoryReadCount:   memoryReadCount,
		MemoryWriteCount:  group.memoryWriteCount,
	}
	dm := first
	out.DisplayMessage = &dm
	if first.Timestamp != nil {
		ts := *first.Timestamp
		out.Timestamp = &ts
	}
	if group.mcpCallCount > 0 {
		mc := group.mcpCallCount
		out.McpCallCount = &mc
		names := make([]string, 0, len(group.mcpServerNames))
		for s := range group.mcpServerNames {
			names = append(names, s)
		}
		sort.Strings(names)
		out.McpServerNames = names
	}
	if CollapseAllBashFromEnv() && group.bashCount > 0 {
		bc := group.bashCount
		out.BashCount = &bc
		gobc := group.gitOpBashCount
		out.GitOpBashCount = &gobc
	}
	if len(group.commits) > 0 {
		out.Commits = append([]types.GitCommitEntry(nil), group.commits...)
	}
	if len(group.pushes) > 0 {
		out.Pushes = append([]types.GitPushEntry(nil), group.pushes...)
	}
	if len(group.branches) > 0 {
		out.Branches = append([]types.GitBranchEntry(nil), group.branches...)
	}
	if len(group.prs) > 0 {
		out.Prs = append([]types.GitPrEntry(nil), group.prs...)
	}
	if group.hookCount > 0 {
		ht := group.hookTotalMs
		out.HookTotalMs = &ht
		hc := group.hookCount
		out.HookCount = &hc
		out.HookInfos = append([]types.StopHookInfo(nil), group.hookInfos...)
	}
	if len(group.relevantMemories) > 0 {
		out.RelevantMemories = append([]types.MemoryAttachment(nil), group.relevantMemories...)
	}
	return out
}

func collapseReadSearchGroupsInk(messages []types.Message) []types.Message {
	var result []types.Message
	current := createEmptyGroupInk()
	var deferred []types.Message

	flushGroup := func() {
		if len(current.messages) == 0 {
			return
		}
		result = append(result, createCollapsedGroupInk(current))
		for _, d := range deferred {
			result = append(result, d)
		}
		deferred = nil
		current = createEmptyGroupInk()
	}

	for _, msg := range messages {
		if msg.Type == types.MessageTypeCollapsedReadSearch {
			flushGroup()
			result = append(result, msg)
			continue
		}
		if isCollapsibleToolUseInk(msg) {
			toolName, input, toolInfo, ok := getCollapsibleToolInfoGo(msg)
			if !ok {
				flushGroup()
				result = append(result, msg)
				continue
			}
			_ = toolName
			appendToolUseContributionInk(msg, toolName, input, toolInfo, current)
			for _, id := range getToolUseIdsFromMessageInk(msg) {
				current.toolUseIds[id] = struct{}{}
			}
			current.messages = append(current.messages, msg)
			continue
		}
		if isCollapsibleToolResultInk(msg, current.toolUseIds) {
			current.messages = append(current.messages, msg)
			if CollapseAllBashFromEnv() && len(current.bashCommands) > 0 {
				scanBashResultForGitOpsInk(msg, current)
			}
			continue
		}
		if len(current.messages) > 0 && isPreToolHookSummaryMsg(msg) {
			hc := 0
			if msg.HookCount != nil {
				hc = *msg.HookCount
			}
			current.hookCount += hc
			var addMs int64
			if msg.TotalDurationMs != nil {
				addMs = *msg.TotalDurationMs
			} else {
				for _, h := range msg.HookInfos {
					if h.DurationMs != nil {
						addMs += *h.DurationMs
					}
				}
			}
			current.hookTotalMs += addMs
			current.hookInfos = append(current.hookInfos, msg.HookInfos...)
			continue
		}
		if len(current.messages) > 0 && msg.Type == types.MessageTypeAttachment && absorbRelevantMemoriesInk(msg, current) {
			continue
		}
		if shouldSkipMessageInk(msg) {
			if len(current.messages) > 0 && !(msg.Type == types.MessageTypeAttachment && attachmentType(msg) == "nested_memory") {
				deferred = append(deferred, msg)
			} else {
				result = append(result, msg)
			}
			continue
		}
		if isTextBreakerInk(msg) {
			flushGroup()
			result = append(result, msg)
			continue
		}
		if isNonCollapsibleToolUseInk(msg) {
			flushGroup()
			result = append(result, msg)
			continue
		}
		flushGroup()
		result = append(result, msg)
	}
	flushGroup()
	return result
}
