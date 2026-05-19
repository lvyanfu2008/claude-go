package localtools

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"goc/tools/tool"
	"goc/types"
)

const toolSummaryMaxLenBehaviors = 50

func init() {
	registerBehaviors()
}

func registerBehaviors() {
	// Read
	tool.RegisterBehaviors("Read", &tool.Behaviors{
		ActivityDescriber:       &readActivityDescriber{},
		SearchOrReadChecker:     &readSearchOrReadChecker{},
		SearchTextExtractor:     &readSearchTextExtractor{},
		ResultTruncationChecker: &readResultTruncationChecker{},
	})

	// Write
	tool.RegisterBehaviors("Write", &tool.Behaviors{
		ActivityDescriber: &writeActivityDescriber{},
	})

	// Edit
	tool.RegisterBehaviors("Edit", &tool.Behaviors{
		ActivityDescriber: &editActivityDescriber{},
	})

	// Grep
	tool.RegisterBehaviors("Grep", &tool.Behaviors{
		ActivityDescriber:   &grepActivityDescriber{},
		SearchOrReadChecker: &grepSearchOrReadChecker{},
	})

	// Glob
	tool.RegisterBehaviors("Glob", &tool.Behaviors{
		ActivityDescriber:   &globActivityDescriber{},
		SearchOrReadChecker: &globSearchOrReadChecker{},
	})

	// Bash (also registered as BashZog)
	bashBehaviors := &tool.Behaviors{
		ActivityDescriber:   &bashActivityDescriber{},
		SearchOrReadChecker: &bashSearchOrReadChecker{},
	}
	tool.RegisterBehaviors("Bash", bashBehaviors)
	tool.RegisterBehaviors("BashZog", bashBehaviors)

	// WebFetch
	tool.RegisterBehaviors("WebFetch", &tool.Behaviors{
		ActivityDescriber: &webFetchActivityDescriber{},
	})

	// WebSearch
	tool.RegisterBehaviors("WebSearch", &tool.Behaviors{
		ActivityDescriber: &webSearchActivityDescriber{},
	})

	// NotebookEdit
	tool.RegisterBehaviors("NotebookEdit", &tool.Behaviors{
		ActivityDescriber: &notebookEditActivityDescriber{},
	})

	// Agent / Task
	agentBehaviors := &tool.Behaviors{
		ActivityDescriber: &agentActivityDescriber{},
	}
	tool.RegisterBehaviors("Agent", agentBehaviors)
	tool.RegisterBehaviors("Task", agentBehaviors)

	// PowerShell
	tool.RegisterBehaviors("PowerShell", &tool.Behaviors{
		ActivityDescriber: &powershellActivityDescriber{},
	})
}

// --- helpers ---

func displayPathForActivity(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	// Simplified: just clean the path (full cwd-relative logic is in messagerow.DisplayPathForActivity)
	return filepath.Clean(p)
}

func inputMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

func strFromInput(raw json.RawMessage, key string) string {
	m := inputMap(raw)
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func truncateSummary(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= toolSummaryMaxLenBehaviors {
		return s
	}
	runes := []rune(s)
	if len(runes) > toolSummaryMaxLenBehaviors-1 {
		runes = runes[:toolSummaryMaxLenBehaviors-1]
	}
	return string(runes) + "…"
}

// --- Read tool ---

type readActivityDescriber struct{}

func (r *readActivityDescriber) GetActivityDescription(input json.RawMessage) string {
	fp := strFromInput(input, "file_path")
	if fp == "" {
		return "Reading file"
	}
	return "Reading " + truncateSummary(displayPathForActivity(fp))
}

type readSearchOrReadChecker struct{}

func (r *readSearchOrReadChecker) IsSearchOrReadCommand(input json.RawMessage) *types.SearchOrReadCollapse {
	return &types.SearchOrReadCollapse{IsRead: true}
}

type readSearchTextExtractor struct{}

func (r *readSearchTextExtractor) ExtractSearchText(output json.RawMessage) string {
	var out struct {
		File struct {
			Content string `json:"content"`
		} `json:"file"`
	}
	if err := json.Unmarshal(output, &out); err != nil {
		return ""
	}
	return out.File.Content
}

type readResultTruncationChecker struct{}

func (r *readResultTruncationChecker) IsResultTruncated(output json.RawMessage) bool {
	var out struct {
		File struct {
			Content   string `json:"content"`
			TotalLines int    `json:"totalLines"`
			NumLines   int    `json:"numLines"`
		} `json:"file"`
	}
	if err := json.Unmarshal(output, &out); err != nil {
		return false
	}
	return out.File.NumLines > 0 && out.File.NumLines < out.File.TotalLines
}

// --- Write tool ---

type writeActivityDescriber struct{}

func (w *writeActivityDescriber) GetActivityDescription(input json.RawMessage) string {
	fp := strFromInput(input, "file_path")
	if fp == "" {
		return "Writing file"
	}
	return "Writing " + truncateSummary(displayPathForActivity(fp))
}

// --- Edit tool ---

type editActivityDescriber struct{}

func (e *editActivityDescriber) GetActivityDescription(input json.RawMessage) string {
	fp := strFromInput(input, "file_path")
	if fp == "" {
		return "Editing file"
	}
	return "Editing " + truncateSummary(displayPathForActivity(fp))
}

// --- Grep tool ---

type grepActivityDescriber struct{}

func (g *grepActivityDescriber) GetActivityDescription(input json.RawMessage) string {
	pat := strFromInput(input, "pattern")
	if pat == "" {
		return "Searching"
	}
	return "Searching for " + truncateSummary(pat)
}

type grepSearchOrReadChecker struct{}

func (g *grepSearchOrReadChecker) IsSearchOrReadCommand(input json.RawMessage) *types.SearchOrReadCollapse {
	return &types.SearchOrReadCollapse{IsSearch: true}
}

// --- Glob tool ---

type globActivityDescriber struct{}

func (g *globActivityDescriber) GetActivityDescription(input json.RawMessage) string {
	pat := strFromInput(input, "pattern")
	if pat == "" {
		return "Finding files"
	}
	return "Finding " + truncateSummary(pat)
}

type globSearchOrReadChecker struct{}

func (g *globSearchOrReadChecker) IsSearchOrReadCommand(input json.RawMessage) *types.SearchOrReadCollapse {
	return &types.SearchOrReadCollapse{IsSearch: true}
}

// --- Bash tool ---

type bashActivityDescriber struct{}

func (b *bashActivityDescriber) GetActivityDescription(input json.RawMessage) string {
	cmd := strFromInput(input, "command")
	if cmd == "" {
		return "Running command"
	}
	desc := strFromInput(input, "description")
	if strings.TrimSpace(desc) == "" {
		desc = truncateSummary(cmd)
	} else {
		desc = truncateSummary(desc)
	}
	return "Running " + desc
}

type bashSearchOrReadChecker struct{}

func (b *bashSearchOrReadChecker) IsSearchOrReadCommand(input json.RawMessage) *types.SearchOrReadCollapse {
	cmd := strFromInput(input, "command")
	return classifyBashCommand(cmd)
}

// --- WebFetch tool ---

type webFetchActivityDescriber struct{}

func (w *webFetchActivityDescriber) GetActivityDescription(input json.RawMessage) string {
	u := strFromInput(input, "url")
	if u == "" {
		return "Fetching web page"
	}
	return "Fetching " + truncateSummary(u)
}

// --- WebSearch tool ---

type webSearchActivityDescriber struct{}

func (w *webSearchActivityDescriber) GetActivityDescription(input json.RawMessage) string {
	q := strFromInput(input, "query")
	if q == "" {
		return "Searching the web"
	}
	return "Searching for " + truncateSummary(q)
}

// --- NotebookEdit tool ---

type notebookEditActivityDescriber struct{}

func (n *notebookEditActivityDescriber) GetActivityDescription(input json.RawMessage) string {
	np := strFromInput(input, "notebook_path")
	if np == "" {
		return "Editing notebook"
	}
	return "Editing notebook " + truncateSummary(displayPathForActivity(np))
}

// --- Agent / Task tool ---

type agentActivityDescriber struct{}

func (a *agentActivityDescriber) GetActivityDescription(input json.RawMessage) string {
	d := strFromInput(input, "description")
	if strings.TrimSpace(d) == "" {
		return "Running task"
	}
	return d
}

// --- PowerShell tool ---

type powershellActivityDescriber struct{}

func (p *powershellActivityDescriber) GetActivityDescription(input json.RawMessage) string {
	cmd := strFromInput(input, "command")
	if cmd == "" {
		return "Running command"
	}
	desc := strFromInput(input, "description")
	if strings.TrimSpace(desc) == "" {
		desc = truncateSummary(cmd)
	} else {
		desc = truncateSummary(desc)
	}
	return "Running " + desc
}

// classifyBashCommand mirrors TS isSearchOrReadBashCommand for registering Bash.
func classifyBashCommand(cmd string) *types.SearchOrReadCollapse {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}
	base := parts[0]
	// Remove leading path components (e.g. /usr/bin/grep → grep)
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}

	searchCmds := map[string]bool{
		"grep": true, "rg": true, "find": true, "fd": true,
		"ag": true, "ack": true, "git": true,
	}
	readCmds := map[string]bool{
		"cat": true, "head": true, "tail": true, "less": true,
		"more": true, "wc": true, "nl": true,
	}
	listCmds := map[string]bool{
		"ls": true, "tree": true, "dir": true, "du": true,
		"df": true, "stat": true,
	}

	if listCmds[base] {
		return &types.SearchOrReadCollapse{IsSearch: true, IsList: true}
	}
	if searchCmds[base] {
		return &types.SearchOrReadCollapse{IsSearch: true}
	}
	if readCmds[base] {
		return &types.SearchOrReadCollapse{IsRead: true}
	}
	return nil
}
