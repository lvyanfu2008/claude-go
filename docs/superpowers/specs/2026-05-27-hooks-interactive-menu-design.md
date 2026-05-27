# /hooks Interactive Config Menu — Go Implementation Design

## Goal

Align claude-go's `/hooks` slash command with the TS `HooksConfigMenu` — a 4-level interactive read-only menu for browsing configured hooks.

## Current State

Go's `/hooks` only reads `.harness/hooks/` directory and lists files. It is completely disconnected from the actual hooks runtime (`hookexec` package, settings merging, HooksTable).

## Target State

A Bubble Tea sub-model (`hooksConfigMenu`) implementing the same 4-level state machine as TS:

- **Level 0 — SelectEvent**: List all 27 hook events with configured hook counts
- **Level 1 — SelectMatcher**: For events with matcherMetadata, show matchers grouped by source
- **Level 2 — SelectHook**: Show individual hooks for the selected event/matcher, with type labels and source
- **Level 3 — ViewHook**: Read-only detail view of a single hook configuration

## Architecture

### Integration Pattern

Follow the `questionModel` sub-model pattern:

1. New `gouHooksMenuMsg` message type
2. Slash handler sends this message to `tea.Program.Send()`
3. `main.Update()` creates `hooksConfigMenu` sub-model
4. All messages delegated to sub-model while it's active
5. On exit (`done=true`), sub-model set to nil, normal UI resumes

### New Files

| File | Purpose |
|------|---------|
| `gou/app/hooks_config_menu.go` | Bubble Tea sub-model (~500 lines), 4-level state machine |
| `gou/app/hooks_config_styles.go` | Lipgloss style definitions |
| `commands/hooksconfig/manager.go` | Port of TS `hooksConfigManager.ts` — grouping logic, metadata |

### Modified Files

| File | Change |
|------|--------|
| `commands/handlers/hooks.go` | Return interactive trigger instead of directory listing |
| `gou/pui/slash_resolve_demo.go` | Add interactive menu trigger path for slash commands |
| `gou/app/main.go` | Integrate `hooksConfigMenu` sub-model |

### Data Model

```go
type IndividualHookConfig struct {
    Event      HookEvent
    Config     hookstypes.HookCommand
    Matcher    string
    Source     HookSource
    PluginName string
}

type HookSource string  // userSettings | projectSettings | localSettings | pluginHook | sessionHook

type GroupedHooks map[HookEvent]map[string][]IndividualHookConfig
```

Data sourced from `hookexec.MergedHooksFromPaths()` (existing settings merging) plus plugin/session hooks.

### Navigation

- Enter / Space: select and advance
- Esc: go back one level (exit at top level)
- Up/Down / j/k: navigate list items
- Read-only: no editing, footer hints to edit settings.json

## Scope

- Full 4-level interactive menu matching TS behavior
- Read-only display (no hook editing)
- All 27 hook events with metadata
- Source-aware display (User/Project/Local/Plugin/Session)
- Matcher priority sorting (local > project > user > plugin)

## Exclusions

- Hook creation/editing (edit settings.json directly)
- `restrictedByPolicy` / `hooksDisabled` warnings (policy system not yet in Go)
- Plugin name display in SelectHook level (can add later)
