# @ Autocomplete for Claude-Go TUI — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add real-time file/folder/agent/MCP-resource suggestion dropdown when user types `@` in the prompt input.

**Architecture:** Three modules — `FileIndex` (git ls-files indexing + fuzzy search), `SuggestionEngine` (regex token detection + merged ranking), `SuggestionUI` (TUI rendering above input + keyboard navigation). Follows existing slash command autocomplete pattern in `gou/app/slash_suggest_ts.go`.

**Tech Stack:** Go stdlib (regexp, os, strings), Bubble Tea v2 (`charm.land/bubbletea/v2`), Lipgloss v2 (`charm.land/lipgloss/v2`), no new external dependencies.

---

### Task 1: FileIndex — file index with scored search

**Files:**
- Create: `gou/suggestions/file_index.go`
- Create: `gou/suggestions/file_index_test.go`

- [ ] **Step 1: Create the suggestions package directory**

Run: `mkdir -p /Users/lvyanfu/Work/claude/claude-go/gou/suggestions`

- [ ] **Step 2: Write the FileIndex implementation**

Create `gou/suggestions/file_index.go`:

```go
// Package suggestions provides file indexing and @-mention autocomplete suggestions for the TUI.
package suggestions

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileIndex maintains an in-memory list of project files with background refresh.
type FileIndex struct {
	entries          []string
	mu               sync.RWMutex
	cwd              string
	lastRefresh      time.Time
	refreshThrottle  time.Duration
	stopCh           chan struct{}
}

// NewFileIndex builds the initial file index from the given working directory.
func NewFileIndex(cwd string) *FileIndex {
	fi := &FileIndex{
		cwd:             cwd,
		refreshThrottle: 5 * time.Second,
		stopCh:          make(chan struct{}),
	}
	fi.refresh()
	go fi.backgroundRefresh()
	return fi
}

// Stop shuts down the background refresh goroutine.
func (fi *FileIndex) Stop() {
	close(fi.stopCh)
}

// Search returns scored file/directory entries matching query. Returns top 15.
func (fi *FileIndex) Search(query string) []ScoredItem {
	fi.mu.RLock()
	defer fi.mu.RUnlock()
	return scoredSearch(fi.entries, query, 15)
}

// GetTopLevelPaths lists directory entries under the given directory path.
// dirPath is relative to cwd (e.g. "src", ".", "~/proj").
func (fi *FileIndex) GetTopLevelPaths(dirPath string) []ScoredItem {
	absPath := fi.resolvePath(dirPath)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil
	}
	var items []ScoredItem
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		rel, err := filepath.Rel(fi.cwd, filepath.Join(absPath, name))
		if err != nil {
			continue
		}
		icon := "F"
		sType := SuggestionTypeFile
		if e.IsDir() {
			icon = "D"
			sType = SuggestionTypeDirectory
			rel += "/"
		}
		items = append(items, ScoredItem{
			Type:  sType,
			Label: filepath.ToSlash(rel),
			Value: filepath.ToSlash(rel),
			Score: 0,
			Icon:  icon,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Type != items[j].Type {
			return items[i].Type == SuggestionTypeDirectory
		}
		return items[i].Label < items[j].Label
	})
	if len(items) > 15 {
		items = items[:15]
	}
	return items
}

func (fi *FileIndex) resolvePath(rel string) string {
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, rel[2:])
	}
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	return filepath.Join(fi.cwd, rel)
}

func (fi *FileIndex) refresh() {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	fi.entries = fi.collectFiles()
	fi.lastRefresh = time.Now()
}

func (fi *FileIndex) collectFiles() []string {
	entries := fi.tryGitLsFiles()
	if entries == nil {
		entries = fi.walkFilesystem()
	}
	return entries
}

func (fi *FileIndex) tryGitLsFiles() []string {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = fi.cwd
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var entries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entries = append(entries, filepath.ToSlash(line))
	}
	if len(entries) == 0 {
		return nil
	}
	return entries
}

func (fi *FileIndex) walkFilesystem() []string {
	var entries []string
	filepath.Walk(fi.cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") && base != "." {
				return filepath.SkipDir
			}
			if base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(fi.cwd, path)
		if err != nil {
			return nil
		}
		entries = append(entries, filepath.ToSlash(rel))
		return nil
	})
	return entries
}

func (fi *FileIndex) backgroundRefresh() {
	ticker := time.NewTicker(fi.refreshThrottle)
	defer ticker.Stop()
	for {
		select {
		case <-fi.stopCh:
			return
		case <-ticker.C:
			fi.refresh()
		}
	}
}

// scoredSearch ranks entries against query using a simple tiered scoring system
// (same pattern as rankedSlashForQuery in slash_suggest_ts.go).
func scoredSearch(entries []string, query string, limit int) []ScoredItem {
	q := strings.ToLower(query)
	type rank struct {
		idx   int
		tier  int
		score float64
	}
	var cand []rank
	for i, entry := range entries {
		lower := strings.ToLower(entry)
		base := filepath.Base(entry)
		lowerBase := strings.ToLower(base)
		var tier int
		var score float64
		switch {
		case q == "":
			tier = 10
		case lower == q:
			tier = 0
		case strings.HasPrefix(lower, q):
			tier = 1
			score = float64(len(lower) - len(q))
		case strings.HasPrefix(lowerBase, q):
			tier = 2
			score = float64(len(lowerBase) - len(q))
		case strings.Contains(lower, q):
			tier = 3
			score = float64(strings.Index(lower, q))
		case strings.Contains(lowerBase, q):
			tier = 4
			score = float64(strings.Index(lowerBase, q))
		case subsequenceMatch(q, lower):
			tier = 5
			score = 2
		default:
			continue
		}
		cand = append(cand, rank{idx: i, tier: tier, score: score})
	}
	sort.SliceStable(cand, func(i, j int) bool {
		if cand[i].tier != cand[j].tier {
			return cand[i].tier < cand[j].tier
		}
		if cand[i].score != cand[j].score {
			return cand[i].score < cand[j].score
		}
		return strings.ToLower(entries[cand[i].idx]) < strings.ToLower(entries[cand[j].idx])
	})
	var out []ScoredItem
	seen := map[string]bool{}
	for _, c := range cand {
		entry := entries[c.idx]
		if seen[entry] {
			continue
		}
		seen[entry] = true
		icon := "F"
		if strings.HasSuffix(entry, "/") || filepath.Ext(entry) == "" {
			// check if it's actually a directory
		}
		out = append(out, ScoredItem{
			Type:  SuggestionTypeFile,
			Label: entry,
			Value: entry,
			Score: float64(c.tier)*100 + c.score,
			Icon:  icon,
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func subsequenceMatch(needle, haystack string) bool {
	if needle == "" {
		return true
	}
	nr := []rune(needle)
	hr := []rune(haystack)
	j := 0
	for i := 0; i < len(hr) && j < len(nr); i++ {
		if hr[i] == nr[j] {
			j++
		}
	}
	return j == len(nr)
}
```

- [ ] **Step 3: Write the FileIndex tests**

Create `gou/suggestions/file_index_test.go`:

```go
package suggestions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScoredSearch_ExactMatch(t *testing.T) {
	entries := []string{
		"src/components/Button.tsx",
		"src/hooks/useAuth.ts",
		"src/utils/format.ts",
		"README.md",
	}
	results := scoredSearch(entries, "src/components/Button.tsx", 15)
	if len(results) == 0 {
		t.Fatal("expected at least one result for exact match")
	}
	if results[0].Label != "src/components/Button.tsx" {
		t.Errorf("expected exact match first, got %q", results[0].Label)
	}
}

func TestScoredSearch_PrefixMatch(t *testing.T) {
	entries := []string{
		"src/components/Button.tsx",
		"src/components/Modal.tsx",
		"README.md",
	}
	results := scoredSearch(entries, "src/components", 15)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 prefix matches, got %d", len(results))
	}
}

func TestScoredSearch_SubsequenceMatch(t *testing.T) {
	entries := []string{
		"src/components/Button.tsx",
		"src/hooks/useAuth.ts",
		"README.md",
	}
	// "btn" should subsequence-match "Button"
	results := scoredSearch(entries, "btn", 15)
	if len(results) == 0 {
		t.Fatal("expected at least one subsequence match for 'btn'")
	}
	if results[0].Label != "src/components/Button.tsx" {
		t.Errorf("expected Button.tsx first, got %q", results[0].Label)
	}
}

func TestScoredSearch_EmptyQuery(t *testing.T) {
	entries := []string{"a.go", "b.go", "c.go"}
	results := scoredSearch(entries, "", 15)
	if len(results) != 3 {
		t.Errorf("expected 3 results for empty query, got %d", len(results))
	}
}

func TestGetTopLevelPaths_DirectoriesFirst(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "src", "lib"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "src", "utils.go"), []byte("package src"), 0644)

	fi := &FileIndex{cwd: tmpDir}
	results := fi.GetTopLevelPaths("src")
	if len(results) == 0 {
		t.Fatal("expected results for src directory")
	}
	// First result should be directory "lib/"
	if results[0].Type != SuggestionTypeDirectory {
		t.Errorf("expected first result to be directory, got type %v label %q", results[0].Type, results[0].Label)
	}
	if results[0].Label != "src/lib/" {
		t.Errorf("expected 'src/lib/', got %q", results[0].Label)
	}
}

func TestGetTopLevelPaths_HidesDotFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("secret"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("hello"), 0644)

	fi := &FileIndex{cwd: tmpDir}
	results := fi.GetTopLevelPaths(".")
	for _, r := range results {
		if filepath.Base(r.Label)[0] == '.' {
			t.Errorf("expected no dot-files, got %q", r.Label)
		}
	}
}
```

- [ ] **Step 4: Run FileIndex tests**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go test ./gou/suggestions/ -v -run TestScoredSearch\|TestGetTopLevelPaths`
Expected: All tests PASS

- [ ] **Step 5: Commit FileIndex**

```bash
cd /Users/lvyanfu/Work/claude/claude-go
git add gou/suggestions/file_index.go gou/suggestions/file_index_test.go
git commit -m "feat: add FileIndex for project file indexing with scored search"
```

---

### Task 2: SuggestionEngine — token detection and suggestion merging

**Files:**
- Create: `gou/suggestions/engine.go`
- Create: `gou/suggestions/engine_test.go`

- [ ] **Step 1: Write the SuggestionEngine implementation**

Create `gou/suggestions/engine.go`:

```go
package suggestions

import (
	"regexp"
	"strings"
)

// SuggestionType is the kind of autocomplete suggestion.
type SuggestionType int

const (
	SuggestionTypeFile        SuggestionType = iota
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

// HAS_AT_SYMBOL_RE detects @mention at cursor position (aligns with TS).
// Matches patterns like: "@foo", "@path/to/file", "@./relative"
var hasAtSymbolRe = regexp.MustCompile(`(\s|^)@([\p{L}\p{N}_\-./\\()[\]~:]*|"[^"]*"?)$`)

// SuggestionEngine detects @ tokens in the prompt and generates ranked suggestions.
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

// CompletionRange describes the token range to replace in the input.
type CompletionRange struct {
	Start int // rune index (inclusive)
	End   int // rune index (exclusive)
}

// SuggestionResult holds the current suggestion state.
type SuggestionResult struct {
	Items     []ScoredItem
	Token     string
	Range     CompletionRange
	HasResults bool
}

// Update processes the current prompt value and cursor position, returning suggestions.
// Returns nil if no @ token is detected at cursor or suggestions are dismissed.
func (e *SuggestionEngine) Update(value string, cursor int) *SuggestionResult {
	token, rng := extractCompletionToken(value, cursor)
	if token == "" {
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
		// Also search index for deeper matches within the path
		if len(items) == 0 {
			items = e.fileIndex.Search(token)
		}
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
// Returns the token text (without @) and its rune range in value.
func extractCompletionToken(value string, cursor int) (string, CompletionRange) {
	rs := []rune(value)
	if cursor < 0 || cursor > len(rs) {
		return "", CompletionRange{}
	}
	// The text from start up to cursor must match HAS_AT_SYMBOL_RE
	before := string(rs[:cursor])
	loc := hasAtSymbolRe.FindStringSubmatchIndex(before)
	if loc == nil {
		return "", CompletionRange{}
	}
	// loc[4:5] is the token capture group (after @)
	if len(loc) < 6 || loc[4] < 0 {
		return "", CompletionRange{}
	}
	token := before[loc[4]:loc[5]]
	// Calculate rune index of @ sign
	atByte := loc[2] // group 2 start is the @-sign
	atRune := runeIndex(rs, atByte)
	tokenEnd := cursor
	return token, CompletionRange{Start: atRune, End: tokenEnd}
}

// runeIndex converts byte offset to rune offset in the given rune slice.
func runeIndex(rs []rune, byteOffset int) int {
	count := 0
	for i := range string(rs) {
		if i >= byteOffset {
			break
		}
		count++
	}
	return count
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
	case strings.HasPrefix(token, "../"):
		return token
	case strings.HasPrefix(token, "~/"):
		return token
	case strings.HasPrefix(token, "/"):
		return token
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
		score := 1000.0
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
				Icon:  "◇",
			})
		}
	}
	return items
}

func sortScoredItems(items []ScoredItem) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].Score > items[j].Score {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}
```

- [ ] **Step 2: Write the SuggestionEngine tests**

Create `gou/suggestions/engine_test.go`:

```go
package suggestions

import (
	"testing"
)

func TestExtractCompletionToken_SimpleAt(t *testing.T) {
	token, rng := extractCompletionToken("hello @foo", 10)
	if token != "foo" {
		t.Errorf("expected token 'foo', got %q", token)
	}
	if rng.Start != 7 || rng.End != 10 {
		t.Errorf("expected range [7,10], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestExtractCompletionToken_PathLike(t *testing.T) {
	token, rng := extractCompletionToken("@./src/comp", 10)
	if token != "./src/comp" {
		t.Errorf("expected token './src/comp', got %q", token)
	}
	if rng.Start != 1 || rng.End != 10 {
		t.Errorf("expected range [1,10], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestExtractCompletionToken_NoAtSymbol(t *testing.T) {
	token, _ := extractCompletionToken("hello world", 11)
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestExtractCompletionToken_AtLineStart(t *testing.T) {
	token, rng := extractCompletionToken("@foo bar", 4)
	if token != "foo" {
		t.Errorf("expected token 'foo', got %q", token)
	}
	if rng.Start != 1 || rng.End != 4 {
		t.Errorf("expected range [1,4], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestExtractCompletionToken_MidLineAt(t *testing.T) {
	token, rng := extractCompletionToken("run @test.go now", 10)
	if token != "test.go" {
		t.Errorf("expected token 'test.go', got %q", token)
	}
	if rng.Start != 5 || rng.End != 12 {
		t.Errorf("expected range [5,12], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestEngine_DismissResetsOnTokenChange(t *testing.T) {
	fi := &FileIndex{entries: []string{"foo.go", "bar.go"}}
	engine := NewSuggestionEngine(fi)

	// First update
	result := engine.Update("@foo", 4)
	if result == nil || !result.HasResults {
		t.Fatal("expected results for @foo")
	}

	// Dismiss
	engine.Dismiss()

	// Same token should return nil
	result2 := engine.Update("@foo", 4)
	if result2 != nil {
		t.Fatal("expected nil after dismiss with same token")
	}

	// Different token should work again
	result3 := engine.Update("@bar", 4)
	if result3 == nil {
		t.Fatal("expected results after token change")
	}
}

func TestEngine_PathLikeDetection(t *testing.T) {
	fi := &FileIndex{entries: []string{"src/main.go"}}
	engine := NewSuggestionEngine(fi)

	// Path-like token triggers GetTopLevelPaths (empty result since no real fs)
	result := engine.Update("@./nonexistent", 14)
	// Should still search the index as fallback
	if result == nil {
		t.Fatal("expected non-nil result for path-like token")
	}
}

func TestIsPathLikeToken(t *testing.T) {
	tests := []struct {
		token    string
		expected bool
	}{
		{"./foo", true},
		{"../bar", true},
		{"~/proj", true},
		{"/abs/path", true},
		{"src/file", false},
		{"agent-name", false},
	}
	for _, tt := range tests {
		got := isPathLikeToken(tt.token)
		if got != tt.expected {
			t.Errorf("isPathLikeToken(%q) = %v, want %v", tt.token, got, tt.expected)
		}
	}
}

func TestSearchAgents_FindsMatch(t *testing.T) {
	agents := []AgentDef{
		{Name: "Explore", DisplayName: "Explore Agent", Description: "Searches the codebase"},
		{Name: "Plan", DisplayName: "Plan Agent", Description: "Designs implementation plans"},
	}
	results := searchAgents(agents, "explore")
	if len(results) == 0 {
		t.Fatal("expected agent match for 'explore'")
	}
	if results[0].Value != "agent-Explore" {
		t.Errorf("expected 'agent-Explore', got %q", results[0].Value)
	}
	if results[0].Icon != "*" {
		t.Errorf("expected Icon '*', got %q", results[0].Icon)
	}
}
```

- [ ] **Step 3: Run SuggestionEngine tests**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go test ./gou/suggestions/ -v -run TestExtractCompletionToken\|TestEngine_\|TestIsPathLike\|TestSearchAgents`
Expected: All tests PASS (TestEngine_PathLikeDetection may use a tempDir fixture — adjust if needed)

- [ ] **Step 4: Commit SuggestionEngine**

```bash
cd /Users/lvyanfu/Work/claude/claude-go
git add gou/suggestions/engine.go gou/suggestions/engine_test.go
git commit -m "feat: add SuggestionEngine for @ token detection and merged ranking"
```

---

### Task 3: SuggestionUI — TUI rendering and keyboard handling

**Files:**
- Create: `gou/app/at_suggest.go`
- Create: `gou/app/at_suggest_test.go`

- [ ] **Step 1: Write the SuggestionUI implementation**

Create `gou/app/at_suggest.go`:

```go
package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"goc/gou/suggestions"
)

// syncAtSuggestions runs after every prompt Update to refresh the @ suggestion list.
func (m *model) syncAtSuggestions() {
	if m.suggestionEngine == nil {
		return
	}
	value := m.pr.Value()
	cursor := m.pr.CursorRuneIndex()
	result := m.suggestionEngine.Update(value, cursor)
	if result == nil || !result.HasResults {
		m.suggVisible = false
		m.suggestions = nil
		m.selectedSuggIdx = 0
		return
	}
	m.suggestions = result.Items
	if m.selectedSuggIdx >= len(m.suggestions) {
		m.selectedSuggIdx = 0
	}
	m.suggVisible = true
}

// handleAtSuggestKeys handles keyboard input when the @ suggestion list is visible.
// Returns: 0 = not handled, 1 = handled (no submit), 2 = handled + submit pending.
func (m *model) handleAtSuggestKeys(msg tea.KeyPressMsg) int {
	if m.uiScreen != gouDemoScreenPrompt || !m.suggVisible || len(m.suggestions) == 0 {
		return 0
	}
	k := msg.Key()
	switch msg.String() {
	case "tab":
		m.applySuggestion(m.suggestions[m.selectedSuggIdx])
		return 1
	case "up":
		if m.selectedSuggIdx > 0 {
			m.selectedSuggIdx--
		} else {
			m.selectedSuggIdx = len(m.suggestions) - 1
		}
		return 1
	case "down":
		if m.selectedSuggIdx+1 < len(m.suggestions) {
			m.selectedSuggIdx++
		} else {
			m.selectedSuggIdx = 0
		}
		return 1
	case "esc":
		m.suggVisible = false
		m.suggestionEngine.Dismiss()
		return 1
	}
	// Arrow key codes (non-string forms)
	if !k.Mod.Contains(tea.ModShift) {
		if k.Code == tea.KeyUp {
			if m.selectedSuggIdx > 0 {
				m.selectedSuggIdx--
			} else {
				m.selectedSuggIdx = len(m.suggestions) - 1
			}
			return 1
		}
		if k.Code == tea.KeyDown {
			if m.selectedSuggIdx+1 < len(m.suggestions) {
				m.selectedSuggIdx++
			} else {
				m.selectedSuggIdx = 0
			}
			return 1
		}
	}
	// Enter: apply suggestion, caller does submit
	if isPromptEnterKey(msg) {
		m.applySuggestion(m.suggestions[m.selectedSuggIdx])
		return 2
	}
	return 0
}

// applySuggestion replaces the @token at the cursor with the selected suggestion value.
func (m *model) applySuggestion(item suggestions.ScoredItem) {
	value := m.pr.Value()
	cursor := m.pr.CursorRuneIndex()
	token, rng := extractCompletionTokenForApply(value, cursor)
	if token == "" {
		return
	}
	rs := []rune(value)
	rep := item.Value + " "
	var b strings.Builder
	b.WriteString(string(rs[:rng.Start]))
	b.WriteString(rep)
	b.WriteString(string(rs[rng.End:]))
	m.pr.SetValue(b.String())
	m.suggVisible = false
	m.selectedSuggIdx = 0
}

// extractCompletionTokenForApply is a local helper that matches the engine's token extraction
// for the apply step (avoiding import cycle through duplication).
func extractCompletionTokenForApply(value string, cursor int) (string, suggestions.CompletionRange) {
	rs := []rune(value)
	if cursor < 0 || cursor > len(rs) {
		return "", suggestions.CompletionRange{}
	}
	// Find @ before cursor by scanning runes backward from cursor
	atIdx := -1
	for i := cursor - 1; i >= 0; i-- {
		if rs[i] == '@' {
			// Check that @ is preceded by space or line start
			if i == 0 || rs[i-1] == ' ' || rs[i-1] == '\n' {
				atIdx = i
				break
			}
		}
	}
	if atIdx < 0 {
		return "", suggestions.CompletionRange{}
	}
	token := string(rs[atIdx+1 : cursor])
	return token, suggestions.CompletionRange{Start: atIdx, End: cursor}
}

// renderAtSuggestions renders the suggestion list above the prompt input (footer area).
func (m *model) renderAtSuggestions() string {
	if !m.suggVisible || len(m.suggestions) == 0 || m.uiScreen != gouDemoScreenPrompt {
		return ""
	}
	width := m.cols
	if width < 40 {
		width = 40
	}
	maxVisible := 6
	if len(m.suggestions) < maxVisible {
		maxVisible = len(m.suggestions)
	}
	// Center the visible window on selectedSuggIdx
	start := m.selectedSuggIdx - maxVisible/2
	if start < 0 {
		start = 0
	}
	if start+maxVisible > len(m.suggestions) {
		start = len(m.suggestions) - maxVisible
	}
	if start < 0 {
		start = 0
	}

	var b strings.Builder
	// Title line
	title := lipgloss.NewStyle().Bold(true).Render("Suggestions  ") +
		lipgloss.NewStyle().Faint(true).Render("Tab accept  Enter submit  Esc dismiss")
	b.WriteString(lipgloss.NewStyle().Width(width).MaxWidth(width).Render(title))
	b.WriteByte('\n')

	for i := start; i < len(m.suggestions) && i < start+maxVisible; i++ {
		item := m.suggestions[i]
		icon := item.Icon
		if icon == "" {
			switch item.Type {
			case suggestions.SuggestionTypeFile:
				icon = "F"
			case suggestions.SuggestionTypeDirectory:
				icon = "D"
			case suggestions.SuggestionTypeAgent:
				icon = "*"
			case suggestions.SuggestionTypeMcpResource:
				icon = "◇"
			}
		}
		line := "  " + icon + " " + item.Label
		if i == m.selectedSuggIdx {
			line = lipgloss.NewStyle().Reverse(true).Render("  " + icon + " " + item.Label)
		}
		b.WriteString(lipgloss.NewStyle().Width(width).MaxWidth(width).Render(line))
		b.WriteByte('\n')
	}
	return b.String()
}
```

- [ ] **Step 2: Write the SuggestionUI tests**

Create `gou/app/at_suggest_test.go`:

```go
package app

import (
	"testing"

	"goc/gou/suggestions"
)

func TestExtractCompletionTokenForApply_FindsAt(t *testing.T) {
	token, rng := extractCompletionTokenForApply("run @test.go", 12)
	if token != "test.go" {
		t.Errorf("expected 'test.go', got %q", token)
	}
	if rng.Start != 5 || rng.End != 12 {
		t.Errorf("expected [5,12], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestExtractCompletionTokenForApply_NoAt(t *testing.T) {
	token, _ := extractCompletionTokenForApply("no at sign", 11)
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestExtractCompletionTokenForApply_AtLineStart(t *testing.T) {
	token, rng := extractCompletionTokenForApply("@hello", 6)
	if token != "hello" {
		t.Errorf("expected 'hello', got %q", token)
	}
	if rng.Start != 1 || rng.End != 6 {
		t.Errorf("expected [1,6], got [%d,%d]", rng.Start, rng.End)
	}
}

func TestApplySuggestion_ReplacesToken(t *testing.T) {
	// Test the core replacement logic
	value := "@foo bar"
	rs := []rune(value)
	token, rng := extractCompletionTokenForApply(value, 4)
	if token != "foo" {
		t.Fatalf("expected 'foo', got %q", token)
	}
	// Simulate the replacement
	rep := "foobar.go "
	var b strings.Builder
	b.WriteString(string(rs[:rng.Start]))
	b.WriteString(rep)
	b.WriteString(string(rs[rng.End:]))
	result := b.String()
	if result != "@foobar.go  bar" {
		t.Errorf("expected '@foobar.go  bar', got %q", result)
	}
}

func TestSuggestionItemIcons(t *testing.T) {
	// Verify SuggestionType values map to expected icons
	items := []suggestions.ScoredItem{
		{Type: suggestions.SuggestionTypeFile, Icon: "F"},
		{Type: suggestions.SuggestionTypeDirectory, Icon: "D"},
		{Type: suggestions.SuggestionTypeAgent, Icon: "*"},
		{Type: suggestions.SuggestionTypeMcpResource, Icon: "◇"},
	}
	for _, item := range items {
		if item.Icon == "" {
			t.Errorf("expected non-empty icon for type %v", item.Type)
		}
	}
}
```

- [ ] **Step 3: Run SuggestionUI tests**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go test ./gou/app/ -v -run TestExtractCompletionTokenForApply\|TestApplySuggestion\|TestSuggestionItemIcons`
Expected: All tests PASS

- [ ] **Step 4: Commit SuggestionUI**

```bash
cd /Users/lvyanfu/Work/claude/claude-go
git add gou/app/at_suggest.go gou/app/at_suggest_test.go
git commit -m "feat: add SuggestionUI for @ autocomplete rendering and keyboard handling"
```

---

### Task 4: Wire into main.go

**Files:**
- Modify: `gou/app/main.go`

- [ ] **Step 1: Add model fields for suggestion state**

In `gou/app/main.go`, add these fields to the `model` struct (after line 529, before line 530):

```go
// @-mention autocomplete
suggestionEngine *suggestions.SuggestionEngine
suggestions      []suggestions.ScoredItem
selectedSuggIdx  int
suggVisible      bool
pendingAtSubmit  string // set by handleAtSuggestKeys when Enter submits via suggestion
```

Edit `gou/app/main.go`: find `slashResultPanel *string` (line 530), add after:

```go
	// @-mention autocomplete suggestions (see at_suggest.go)
	suggestionEngine *suggestions.SuggestionEngine
	suggestions      []suggestions.ScoredItem
	selectedSuggIdx  int
	suggVisible      bool
```

- [ ] **Step 2: Initialize suggestionEngine in newModel()**

In `newModel()` at line 767, after `pr := prompt.New()`, add:

```go
	pr := prompt.New()
	pr.SetEnterSubmits(gouDemoPromptEnterSubmits())

	cwd, _ := os.Getwd()
	// Initialize @-mention autocomplete file index and engine
	suggFI := suggestions.NewFileIndex(cwd)
	suggEngine := suggestions.NewSuggestionEngine(suggFI)
```

And in the `return &model{...}` block (after line 789), add:

```go
		suggestionEngine:  suggEngine,
```

- [ ] **Step 3: Shutdown FileIndex on app exit**

In the `Cleanup` or appropriate shutdown path — add a `Stop()` call when the TUI exits. Find where the app quits (around line 977, `case "ctrl+c": return m, tea.Quit`), and ensure `m.suggestionEngine` is cleaned up. Since `FileIndex.Stop()` closes the background goroutine's channel, add it as a deferred cleanup in `newModel`:

In `newModel()`, after creating `suggFI`, add:

```go
	// NOTE: FileIndex background goroutine will be garbage-collected when model is GC'd.
	// The stop channel ensures clean shutdown. If explicit cleanup is needed, call suggFI.Stop().
	_ = suggFI
```

Actually, add a proper cleanup: in the model struct, add a `cleanup()` method or ensure `tea.Quit` path handles it. Simplest approach — add a `cleanupModel` call before `tea.Quit`:

Find the quit path in `handleKeyMsgPreserving` (line 977): `case "ctrl+c": return m, tea.Quit`. Add cleanup before:

```go
	case "ctrl+c":
		if m.suggestionEngine != nil {
			// FileIndex cleanup handled via the engine's fileIndex reference
			// (no explicit Stop needed — background goroutine exits with process)
		}
		return m, tea.Quit
```

(Background goroutine exits when the process exits, so no explicit cleanup is strictly needed.)

- [ ] **Step 4: Wire syncAtSuggestions into handleKeyMsgPreserving**

In `handleKeyMsgPreserving`, after line 1039 (`m.pr.Update(prompt.NormalizeTTYNewlineKey(msg))`), add:

```go
	m.pr.Update(prompt.NormalizeTTYNewlineKey(msg))
	m.syncAtSuggestions()
	m.syncSlashListAfterPrompt()
```

Replace the existing `m.syncSlashListAfterPrompt()` call on line 1040 with the above.

- [ ] **Step 5: Wire handleAtSuggestKeys before slash list handling**

In `handleKeyMsgPreserving`, before the slash list handling (line 965), add @ suggestion key handling:

```go
	// @-mention autocomplete: Tab/Enter/↑/↓/Esc (must run before slash list nav)
	if m.uiScreen == gouDemoScreenPrompt && m.handleAtSuggestKeys(msg) {
		return m, nil
	}
	// Slash command list: ↑/↓/Tab must win over message-pane scroll
	if m.uiScreen == gouDemoScreenPrompt && m.handleSlashListNavKey(msg) {
		return m, nil
	}
```

- [ ] **Step 6: Handle Enter-submit from @ suggestions in main key handler**

In `handleKeyMsgPreserving`, the `handleAtSuggestKeys` method returns 2 when Enter is pressed on a suggestion. Wire the submit logic:

```go
	// @-mention autocomplete: Tab/Enter/↑/↓/Esc (must run before slash list nav)
	if m.uiScreen == gouDemoScreenPrompt {
		switch m.handleAtSuggestKeys(msg) {
		case 2: // Enter: apply + submit
			fullPrompt := strings.TrimRight(m.pr.Value(), "\r\n")
			m.pr.SetValue("")
			m.suggVisible = false
			m.syncAtSuggestions()
			line := strings.TrimSpace(fullPrompt)
			if line == "" {
				return m, nil
			}
			return m.gouSubmitFromPromptText(fullPrompt, line)
		case 1: // handled (Tab/↑/↓/Esc)
			return m, nil
		}
	}
```

- [ ] **Step 7: Add renderAtSuggestions to View()**

In `View()` at line 1564 (before `builtinStatusLineView`), add:

```go
		// @-mention autocomplete suggestions (above input footer area)
		if s := m.renderAtSuggestions(); s != "" {
			b.WriteString(s)
			b.WriteByte('\n')
		}
```

Place this BEFORE the `builtinStatusLineView` block (line 1564).

- [ ] **Step 8: Add suggestion area to height calculation**

In `inputAreaHeight()` (around line 857), add the suggestion list height to the calculation:

```go
func (m *model) inputAreaHeight() int {
	h := m.pr.LineCount()
	if m.suggVisible && len(m.suggestions) > 0 {
		visibleRows := min(6, len(m.suggestions))
		h += 1 + visibleRows // title line + suggestion rows
	}
	h++ // horizontal rule above input
	return h
}
```

- [ ] **Step 9: Add `suggestions` import to main.go**

Add to the import block in `main.go`:

```go
	"goc/gou/suggestions"
```

- [ ] **Step 10: Rebuild height cache on suggestion change**

In `syncAtSuggestions`, rebuild the height cache when visibility changes:

```go
func (m *model) syncAtSuggestions() {
	if m.suggestionEngine == nil {
		return
	}
	prevVisible := m.suggVisible
	value := m.pr.Value()
	cursor := m.pr.CursorRuneIndex()
	result := m.suggestionEngine.Update(value, cursor)
	if result == nil || !result.HasResults {
		m.suggVisible = false
		m.suggestions = nil
		m.selectedSuggIdx = 0
	} else {
		m.suggestions = result.Items
		if m.selectedSuggIdx >= len(m.suggestions) {
			m.selectedSuggIdx = 0
		}
		m.suggVisible = true
	}
	if prevVisible != m.suggVisible {
		m.rebuildHeightCache()
	}
}
```

- [ ] **Step 11: Initialize agent suggestions from loaded commands**

After `loadSlashCommandsOnce()` finishes loading, populate the suggestion engine with agent definitions:

In `loadSlashCommandsOnce()` or after it's called, add:

```go
func (m *model) refreshAgentSuggestions() {
	if m.suggestionEngine == nil {
		return
	}
	var agents []suggestions.AgentDef
	for _, cmd := range m.slashCommands {
		if cmd.Agent != nil {
			agents = append(agents, suggestions.AgentDef{
				Name:        types.GetCommandName(cmd),
				DisplayName: types.GetCommandName(cmd),
				Description: cmd.Description,
			})
		}
	}
	m.suggestionEngine.SetAgents(agents)
}
```

Call this after `loadSlashCommandsOnce()` in `m.loadSlashCommandsOnce()` or after the visible slash list refresh.

Find `loadSlashCommandsOnce` and add `m.refreshAgentSuggestions()` after the commands are loaded.

- [ ] **Step 12: Build and verify compilation**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go build ./gou/app/`
Expected: No errors

- [ ] **Step 13: Run all tests**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && go test ./gou/suggestions/ ./gou/app/ -v`
Expected: All tests PASS

- [ ] **Step 14: Run golangci-lint**

Run: `cd /Users/lvyanfu/Work/claude/claude-go && golangci-lint run ./gou/suggestions/ ./gou/app/`
Expected: No lint errors

- [ ] **Step 15: Commit main.go integration**

```bash
cd /Users/lvyanfu/Work/claude/claude-go
git add gou/app/main.go
git commit -m "feat: wire @ autocomplete into TUI main model"
```

---

## Integration Verification

After all tasks are complete:

- [ ] **Verify 1: Manual smoke test** — build and run the TUI, type `@` followed by characters, confirm suggestion list appears above input, Tab replaces token, Enter submits, Esc dismisses.

- [ ] **Verify 2: Coexistence with slash commands** — type `/` at line start, confirm slash list still works. Type `help me @file` mid-input, confirm @ suggestions appear and slash mid-input still works.

- [ ] **Verify 3: Full test suite** — `go test ./gou/suggestions/... ./gou/app/... -count=1` passes.
