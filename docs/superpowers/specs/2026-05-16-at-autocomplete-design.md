# @ Autocomplete for Claude-Go TUI — Design

**Date**: 2026-05-16
**Status**: Approved
**Scope**: Add real-time file/folder/agent/MCP-resource suggestion dropdown when user types `@` in the prompt input.

---

## Overview

When the user types `@` in the prompt input, a suggestion list appears above the input area showing fuzzy-matched files, directories, agents, and MCP resources. The user navigates with arrow keys, accepts with Tab (replaces `@token` with the selected path), or submits with Enter (accepts + submits the prompt). Escape dismisses the list.

This brings the Go TUI to parity with the TypeScript `@` autocomplete in `claude-code-best/src/hooks/useTypeahead.tsx`.

## Architecture

Three new modules, integrated into the existing main TUI model:

```
┌─────────────────────────────────────────────────────┐
│  main.go (model)                                    │
│  ┌─────────────────────────────────────────────┐    │
│  │  at_suggest.go (SuggestionUI)                │    │
│  │  - syncAtSuggestions()                       │    │
│  │  - handleAtSuggestKeys()                     │    │
│  │  - renderAtSuggestions()                     │    │
│  │  - applySuggestion()                         │    │
│  └──────────────┬──────────────────────────────┘    │
│                 │                                    │
│  ┌──────────────▼──────────────────────────────┐    │
│  │  suggestions/engine.go (SuggestionEngine)    │    │
│  │  - Update(value, cursor) → SuggestionResult │    │
│  │  - extractCompletionToken()                  │    │
│  │  - merged ranking (file + agent + mcp)       │    │
│  └──────┬──────────────────┬───────────────────┘    │
│         │                  │                        │
│  ┌──────▼──────┐  ┌────────▼────────┐              │
│  │ file_index  │  │ agent + mcp     │              │
│  │ .go         │  │ lookups (inline)│              │
│  │ git ls-files│  │                 │              │
│  │ + fzf matcher│ │                 │              │
│  └─────────────┘  └─────────────────┘              │
└─────────────────────────────────────────────────────┘
```

## Components

### 1. FileIndex (`gou/suggestions/file_index.go`)

In-memory file index with fuzzy search, background-refreshed every 5 seconds.

```
FileIndex {
    entries        []string        // relative paths
    matcher        *fzf.Matcher    // fuzzy matching
    mu             sync.RWMutex
    lastRefresh    time.Time
    refreshThrottle time.Duration  // 5s
}
```

**Index sources** (priority order):
1. `git ls-files --cached --others --exclude-standard` — respects `.gitignore`
2. Fallback: `filepath.Walk` + manual `.gitignore` filtering

**Key methods**:
- `NewFileIndex(cwd string) *FileIndex` — builds initial index
- `Search(query string) []ScoredItem` — fuzzy search, returns top 15
- `GetTopLevelPaths(cwd string) []ScoredItem` — directory listing for empty `@` or path-prefixed tokens (`@./`, `@~/`, `@/`, `@../`)
- `Refresh()` — re-indexes filesystem
- `backgroundRefresh()` — goroutine with 5s throttle

**Fuzzy library**: `github.com/junegunn/fzf` Go bindings, equivalent to TS nucleo.

### 2. SuggestionEngine (`gou/suggestions/engine.go`)

Detects `@` in input, extracts the completion token, generates ranked suggestions.

```
SuggestionEngine {
    fileIndex    *FileIndex
    agents       []AgentDef
    mcpResources []McpResource
    prevValue    string
    dismissed    bool
}
```

**Detection regex** (aligns with TS `HAS_AT_SYMBOL_RE`):
```
(\s|^)@([\p{L}\p{N}_\-./\\()[\]~:]*|"[^"]*"?)$
```

**Key methods**:
- `Update(value string, cursor int) *SuggestionResult` — main entry point
  - If path-like token (`./`, `~/`, `/`, `../`): calls `FileIndex.GetTopLevelPaths()`
  - Otherwise: calls `FileIndex.Search()` + agent search + MCP search
  - Merges and sorts by score, caps at 15 items
- `extractCompletionToken(value, cursor) (token, range)` — extracts text after `@` up to cursor
- `Dismiss()` / `ResetDismissed()` / `IsDismissed()` — Esc suppression until token changes

**SuggestionItem**:
```
type SuggestionItem struct {
    Type  SuggestionType  // file | directory | agent | mcp_resource
    Label string          // display text (filename, agent name, etc.)
    Value string          // replacement value (relative path, agent:type, mcp URI)
    Score float64
    Icon  string          // rendered prefix
}
```

**SuggestionType enum**: `file`, `directory`, `agent`, `mcp_resource`

**Ranking weights**:
1. Exact prefix match → highest
2. Path substring match → medium
3. Description/type match → lowest

### 3. SuggestionUI — Main Model Integration

**Modified file**: `gou/app/main.go` (add ~150 lines)
**New file**: `gou/app/at_suggest.go` (~250 lines)

**New model fields**:
```
suggestionEngine *suggestions.SuggestionEngine
suggestions      []suggestions.SuggestionItem
selectedSuggIdx  int
suggVisible      bool
```

**Core functions** (in `at_suggest.go`):

- `syncAtSuggestions(m *model)` — called after every `m.pr.Update()`. Reads current prompt value and cursor position, calls `suggestionEngine.Update()`, sets `suggVisible`, `suggestions`, resets `selectedSuggIdx` to 0.

- `handleAtSuggestKeys(m *model, msg tea.KeyMsg) tea.Cmd` — handles keyboard when `suggVisible`:
  - **Tab**: `applySuggestion(m, m.suggestions[m.selectedSuggIdx])` — replaces `@token` with selected value, hides list
  - **Enter**: apply + submit prompt
  - **Up** (`↑` / `Ctrl+P`): decrement `selectedSuggIdx` (wrap-around)
  - **Down** (`↓` / `Ctrl+N`): increment `selectedSuggIdx` (wrap-around)
  - **Esc**: `suggVisible = false`, `suggestionEngine.Dismiss()`

- `applySuggestion(m *model, item SuggestionItem)` — locates the `@token` range in the rune buffer via `extractCompletionToken()`, replaces it with `item.Value`, moves cursor to end of inserted text.

- `renderAtSuggestions(m *model) string` — renders up to 6 visible items, centered on `selectedSuggIdx`:
  ```
  📁 src/components/Button.tsx    ← normal
  📁 src/hooks/useAuth.ts         ← selected (highlighted/reverse video)
  📁 src/utils/format.ts          ← normal
  ```
  Returns empty string when `!suggVisible || len(suggestions) == 0`.

**Layout position**: Above the prompt input (footer area), in `main.go View()`:
```
renderAtSuggestions(m)             // @ suggestions (new)
builtinStatusLineView(m)           // status line (existing)
userInputViewWithPromptPrefix(m)   // prompt input (existing)
renderSlashPicker(m)               // slash list (existing)
```

### Coexistence with Slash Command Autocomplete

- `@` detected → `@` suggestions shown in footer, Tab consumed by `@` autocomplete
- `/` at line start → slash list shown below input, Tab consumed by slash autocomplete  
- Both present → `@` above input, slash below input. Tab priority: `@` suggestion if visible, else slash.
- Neither → Tab has no suggestion behavior

## Data Flow

```
User types "@src/comp"
  → prompt.Update(msg) in handleKeyMsg
  → syncAtSuggestions(m)
    → engine.Update("@src/comp", cursor=9)
      → extractCompletionToken → token="src/comp"
      → fileIndex.Search("src/comp") → [Button.tsx, ...]
      → agentSearch("src/comp") → []
      → mcpSearch("src/comp") → []
      → merge + sort → top 15
  → m.suggestions = [Button.tsx, Component.tsx, ...]
  → m.suggVisible = true

User presses Tab
  → handleAtSuggestKeys → applySuggestion(m, Button.tsx)
  → rune buffer: "@src/comp█" → "@src/components/Button.tsx █"
  → suggVisible = false

User presses Enter
  → if suggVisible: apply + submit
  → else: normal Enter behavior
```

## Error Handling

- `git ls-files` fails → fall back to `filepath.Walk` + manual `.gitignore` parsing
- File index refresh fails → keep stale index, log warning, retry on next throttle window
- Permission denied on directory read → skip that directory, continue indexing
- No suggestions found → `suggVisible = false` (don't show empty list)
- Engine panics → recover in `syncAtSuggestions()`, log error, `suggVisible = false`

## Testing Strategy

| Test | What it verifies |
|------|-----------------|
| `file_index_test.go` | `git ls-files` parsing, fuzzy search ranking, `GetTopLevelPaths` directory listing, `.gitignore` filtering, refresh throttle |
| `engine_test.go` | Regex token extraction (`HAS_AT_SYMBOL_RE`), path-like vs fuzzy routing, merged ranking, dismiss/reset cycle |
| `at_suggest_test.go` | `syncAtSuggestions` state transitions, keyboard handler dispatch, `applySuggestion` rune replacement at correct offset, render output format |

## Files Changed / Created

| File | Action | Description |
|------|--------|-------------|
| `gou/suggestions/file_index.go` | **New** | FileIndex with git ls-files + fzf fuzzy search |
| `gou/suggestions/file_index_test.go` | **New** | Tests for file indexing and search |
| `gou/suggestions/engine.go` | **New** | SuggestionEngine: detection, token extraction, merging |
| `gou/suggestions/engine_test.go` | **New** | Tests for engine logic |
| `gou/app/at_suggest.go` | **New** | SuggestionUI: sync, keyboard handling, rendering, apply |
| `gou/app/at_suggest_test.go` | **New** | Tests for UI integration |
| `gou/app/main.go` | **Modify** | Add model fields, wire syncAtSuggestions into handleKeyMsg, add renderAtSuggestions to View() |
| `go.mod` | **Modify** | Add `github.com/junegunn/fzf` dependency |

## Design Decisions

1. **Why fzf over custom matcher**: fzf's Go binding provides the same fuzzy matching quality as TS nucleo. Writing a custom matcher would be premature.
2. **Why 5s throttle on index refresh**: Matches TS behavior. Filesystem state doesn't change frequently enough to warrant faster polling.
3. **Why max 6 visible items**: Matches TS `PromptInputFooterSuggestions.tsx` behavior. Keeps the TUI compact.
4. **Why footer position (above input) instead of overlay**: User preference. Keeps the input visible and avoids z-index complexity in a terminal UI.
5. **Why `@` suggestions take priority over slash Tab**: `@` suggestions appear above the input; slash list appears below. They occupy different screen regions, reducing conflict. When both are visible, `@` consumes Tab because it's closer to the user's current focus.
