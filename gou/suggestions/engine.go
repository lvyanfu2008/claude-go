package suggestions

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// SuggestionType is the kind of autocomplete suggestion.
type SuggestionType int

const (
	SuggestionTypeFile SuggestionType = iota
	SuggestionTypeDirectory
	SuggestionTypeAgent
	SuggestionTypeMcpResource
)

// ScoredItem is a single suggestion result.
type ScoredItem struct {
	Type  SuggestionType
	Label string  // display text
	Value string  // replacement text
	Score float64 // lower is better
	Icon  string  // single-char prefix
}

// AgentDef is a minimal agent definition for suggestion matching.
type AgentDef struct {
	Name        string
	DisplayName string
	Description string
}

// McpResource is a minimal MCP resource for suggestion matching.
type McpResource struct {
	Server string
	URI    string
}

// hasAtSymbolRe detects @mention at cursor position (aligns with TS HAS_AT_SYMBOL_RE).
// Matches patterns like: "@foo", "@path/to/file", "@./relative"
var hasAtSymbolRe = regexp.MustCompile(`(\s|^)@([\p{L}\p{N}_\-./\\()[\]~:]*|"[^"]*"?)$`)

// SuggestionEngine detects @ tokens in the prompt and generates ranked suggestions.
// All methods are designed for single-goroutine use (Bubble Tea TUI event loop).
// SetAgents and SetMcpResources should be called from the same goroutine as Update.
type SuggestionEngine struct {
	fileIndex    *FileIndex
	agents       []AgentDef
	mcpResources []McpResource
	prevToken    string
	dismissed    bool
}

// NewSuggestionEngine creates a suggestion engine with the given file index.
func NewSuggestionEngine(fileIndex *FileIndex) *SuggestionEngine {
	return &SuggestionEngine{
		fileIndex: fileIndex,
	}
}

// SetAgents sets the agent definitions for suggestion matching.
func (e *SuggestionEngine) SetAgents(agents []AgentDef) {
	e.agents = agents
}

// SetMcpResources sets the MCP resources for suggestion matching.
func (e *SuggestionEngine) SetMcpResources(resources []McpResource) {
	e.mcpResources = resources
}

// FileIndex returns the underlying file index (for lifecycle management like Stop).
func (e *SuggestionEngine) FileIndex() *FileIndex {
	return e.fileIndex
}

// CompletionRange describes the token range to replace in the input.
type CompletionRange struct {
	Start int // rune index (inclusive)
	End   int // rune index (exclusive)
}

// SuggestionResult holds the current suggestion state.
type SuggestionResult struct {
	Items      []ScoredItem
	Token      string
	Range      CompletionRange
	HasResults bool
}

// Update processes the current prompt value and cursor position, returning suggestions.
// Returns nil if no @ token is detected at cursor or suggestions are dismissed.
func (e *SuggestionEngine) Update(value string, cursor int) *SuggestionResult {
	token, rng, matched := extractCompletionToken(value, cursor)
	if !matched {
		e.prevToken = ""
		e.dismissed = false
		return nil
	}
	// Re-enable when token changes
	if e.dismissed && token == e.prevToken {
		return nil
	}
	e.dismissed = false
	e.prevToken = token

	isPathLike := isPathLikeToken(token)
	var items []ScoredItem

	if isPathLike {
		dirPath := resolvePathToken(token)
		items = e.fileIndex.GetTopLevelPaths(dirPath)
		if len(items) == 0 {
			items = e.fileIndex.Search(token)
		}
	} else if token == "" {
		items = e.fileIndex.GetTopLevelPaths(".")
	} else {
		items = e.fileIndex.Search(token)
	}

	// Merge agent suggestions
	agentItems := searchAgents(e.agents, token)
	items = append(items, agentItems...)

	// Merge MCP resource suggestions
	mcpItems := searchMcpResources(e.mcpResources, token)
	items = append(items, mcpItems...)

	// Sort merged results by score
	sortScoredItems(items)

	if len(items) > 15 {
		items = items[:15]
	}

	return &SuggestionResult{
		Items:      items,
		Token:      token,
		Range:      rng,
		HasResults: len(items) > 0,
	}
}

// Dismiss hides suggestions until the token changes.
func (e *SuggestionEngine) Dismiss() {
	e.dismissed = true
}

// IsDismissed reports whether suggestions are currently dismissed.
func (e *SuggestionEngine) IsDismissed() bool {
	return e.dismissed
}

// ResetDismissed clears the dismissed state.
func (e *SuggestionEngine) ResetDismissed() {
	e.dismissed = false
}

// extractCompletionToken finds the @token at the cursor position.
// Returns the token text (without @), its rune range in value, and whether @ was matched.
// When @ is matched but no filter text follows, token is "" and matched is true.
func extractCompletionToken(value string, cursor int) (string, CompletionRange, bool) {
	rs := []rune(value)
	if cursor < 0 || cursor > len(rs) {
		return "", CompletionRange{}, false
	}
	before := string(rs[:cursor])
	loc := hasAtSymbolRe.FindStringSubmatchIndex(before)
	if loc == nil {
		return "", CompletionRange{}, false
	}
	// loc[4:5] is the token capture group (after @). Groups: 0=full, 1=prefix, 2=@, 3=@, 4=token start, 5=token end
	if len(loc) < 6 || loc[4] < 0 {
		return "", CompletionRange{}, false
	}
	token := before[loc[4]:loc[5]]
	tokenStartByte := loc[4]
	atRune := utf8.RuneCountInString(before[:tokenStartByte])
	tokenEnd := cursor
	return token, CompletionRange{Start: atRune, End: tokenEnd}, true
}

func isPathLikeToken(token string) bool {
	return strings.HasPrefix(token, "./") ||
		strings.HasPrefix(token, "../") ||
		strings.HasPrefix(token, "~/") ||
		strings.HasPrefix(token, "/")
}

func resolvePathToken(token string) string {
	switch {
	case strings.HasPrefix(token, "./"):
		return strings.TrimPrefix(token, "./")
	default:
		return token
	}
}

func searchAgents(agents []AgentDef, query string) []ScoredItem {
	q := strings.ToLower(query)
	var items []ScoredItem
	for _, a := range agents {
		name := strings.ToLower(a.Name)
		disp := strings.ToLower(a.DisplayName)
		desc := strings.ToLower(a.Description)
		var score float64
		matched := false
		switch {
		case name == q:
			score = 0
			matched = true
		case strings.HasPrefix(name, q):
			score = float64(len(name) - len(q))
			matched = true
		case strings.HasPrefix(disp, q):
			score = 10 + float64(len(disp)-len(q))
			matched = true
		case strings.Contains(name, q):
			score = 50 + float64(strings.Index(name, q))
			matched = true
		case strings.Contains(desc, q):
			score = 100 + float64(strings.Index(desc, q))
			matched = true
		}
		if matched {
			label := a.DisplayName
			if label == "" {
				label = a.Name
			}
			items = append(items, ScoredItem{
				Type:  SuggestionTypeAgent,
				Label: label,
				Value: "agent-" + a.Name,
				Score: score,
				Icon:  "*",
			})
		}
	}
	return items
}

func searchMcpResources(resources []McpResource, query string) []ScoredItem {
	q := strings.ToLower(query)
	var items []ScoredItem
	for _, r := range resources {
		uri := strings.ToLower(r.URI)
		if strings.Contains(uri, q) {
			items = append(items, ScoredItem{
				Type:  SuggestionTypeMcpResource,
				Label: r.Server + ":" + r.URI,
				Value: r.Server + ":" + r.URI,
				Score: float64(strings.Index(uri, q)),
				Icon: "◇",
			})
		}
	}
	return items
}

func sortScoredItems(items []ScoredItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Score < items[j].Score
	})
}
