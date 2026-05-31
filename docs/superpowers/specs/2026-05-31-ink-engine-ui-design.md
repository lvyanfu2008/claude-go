# ink Engine UI Design — GOU_USE_NEW_ENGINE=1

## Summary

`claude-go` 的新终端 UI 引擎 (`gou/ink`) 按 TS `claude-code` 模式重构：补齐引擎能力（core / vdom / layout / render / store 五层），组件层第一阶段只做引擎迁移所需的组件和核心交互闭环组件。后端暂不接入。

---

## Architecture: 5-Layer Engine

```
ink/
  core/            — Platform abstraction (fullscreen.ts)
    terminal.go    — raw mode, alt screen, SIGWINCH
    keyboard.go    — Kitty protocol, full CSI parsing
    mouse.go       — SGR 1006 mouse tracking
    paste.go       — bracketed paste (DEC 2004)

  vdom/            — Virtual DOM (React Ink)
    vnode.go       — VNode (retained)
    reconciler.go  — Fiber-style incremental reconciler
    hooks.go       — useState, useEffect, useMemo, useCallback

  layout/          — Layout engine
    flexbox.go     — Flexbox (retained, +cache layer)
    virtual_scroll.go — VirtualScrollState + VirtualList (useVirtualScroll)

  render/          — Rendering
    rasterize.go   — Cell rasterization (retained)
    screen.go      — Cell buffer (retained)
    paint.go       — Diff-based paint (dirty-rect upgrade)

  store/           — State management (src/state/)
    store.go       — Reactive store, transactions, subscriptions
    atoms.go       — Atom[T] atomic state
    selectors.go   — Selector[T] derived + memoized

  engine.go        — Orchestrates all subsystems
  component.go     — Component type + Context
```

### TS Mapping

| Go package | TS equivalent |
|-----------|---------------|
| `core/terminal.go` | `fullscreen.ts` |
| `core/keyboard.go` | `keybindings/` + `useInput()` |
| `vdom/reconciler.go` | React Ink reconciler |
| `vdom/hooks.go` | React hooks |
| `layout/virtual_scroll.go` | `useVirtualScroll.ts` |
| `store/store.go` | `src/state/` |

---

## core: Terminal Platform Abstraction

### Terminal

```go
Terminal {
    Init()    → raw mode + optional alt screen + SIGWINCH listener
    Shutdown()→ restore cooked mode + rmcup + stop SIGWINCH

    Read()    → chan []byte  (raw stdin stream)
    Write([]byte)
    Size()    → (w, h int)
    ResizeCh  → chan struct{}

    EnableMouse()         → DECSET 1006 (SGR mouse)
    DisableMouse()
    EnableKittyKbd()      → CSI > 1 u + query flags
    DisableKittyKbd()
    EnableBracketedPaste()→ DECSET 2004
}
```

### Keyboard

Input pipeline: raw bytes → KeyboardParser (FSM) → ParsedKey

```go
ParsedKey { Key string; Mod Modifiers; Runes []rune }
```

Modifiers: `Ctrl | Alt | Shift | Meta` bitmask.

Supported sequences:
- Basic: `\r`(Enter), `\x7f`(Backspace), `\x03`(Ctrl+C), `\t`(Tab), `\x1b`(Esc)
- CSI: `\x1b[A`(Up), `\x1b[1;2A`(Shift+Up), `\x1b[1;5A`(Ctrl+Up), etc.
- Kitty: `\x1b[57399u`(Ctrl+o), `\x1b[13;5u`(Ctrl+Enter), etc.
- SS3: `\x1bOA` etc. (function key variants)
- Full modifier-key matrix across arrow/function/Tab/Enter/Backspace

### Mouse

SGR 1006 format: `\x1b[<Btn;X;YM` (press) / `\x1b[<Btn;X;Ym` (release) / `\x1b[<64;X;YM` (wheel up)

```go
MouseEvent { Type MouseEventType; Button int; X, Y int; Mod Modifiers }
```

### Paste

Bracketed paste mode (DEC 2004): `\x1b[200~`...content...`\x1b[201~`

---

## vdom: Fiber Reconciler + Hooks

### FiberReconciler

Two-phase schedule (render + commit), 16ms frame budget:

```
Fiber {
    vnode     *VNode
    child     *Fiber
    sibling   *Fiber
    return    *Fiber
    effectTag EffectTag   // Placement | Update | Deletion | NoEffect
    hooks     []HookState
}
```

Diff algorithm:
- Different type → replace subtree
- Same key + same type → retain fiber, update props
- Different key → reorder children by key
- `shouldComponentUpdate` → shallow prop comparison (not hardcoded field list)

### Hooks

```go
func UseState[T any](ctx *Context, initial T) (T, func(T))
func UseEffect(ctx *Context, fn func() func(), deps []interface{})
func UseMemo[T any](ctx *Context, fn func() T, deps []interface{}) T
func UseCallback(ctx *Context, fn interface{}, deps []interface{}) interface{}
func UseAtom[T any](ctx *Context, atom *Atom[T]) T
func UseSelector[T any](ctx *Context, sel *Selector[T]) T
```

Hook state stored on fiber by index; runtime checks hook count consistency.

---

## layout: Virtual Scroll

### VirtualScrollState

```go
VirtualScrollState {
    RowHeights   []int
    Offsets      []int
    ScrollTop    int
    ViewportH    int
    Overscan     int
    StickyBottom bool
}
```

Reuses existing `gou/virtualscroll.ComputeRange` for the core algorithm.

### VirtualList VNode

Only layouts children in `[visibleFrom, visibleTo]`; off-screen children get height-only placeholders. Height measured via: initial estimate (Markdown + wrap) → actual heights from `child.Layout.H` backfilled into `RowHeights` cache.

---

## store: Reactive State

### Atom[T]

```go
Atom[T] { value T; watchers map[uint64]func(T) }
```

### Selector[T]

```go
Selector[T] { deps []AtomReader; compute func() T; cached T; version uint64 }
```

Auto-memoizes: re-computes only when dependency versions change.

### Defined atoms

```
messages, streamingText, streamingTools, scrollTop,
inputValue, cursorPos, isLoading, termWidth, termHeight,
uiScreen, transcriptSearchQuery, transcriptShowAll,
permissionMode, mouseEnabled
```

### Defined selectors

```
visibleMessages    — pipeline(messages, streamingText, streamingTools)
messageRowHeights  — measureRowHeights(msgs, termWidth)
scrollMax          — max(0, totalHeight - viewportH)
viewportMessages   — sliceVisible(msgs, heights, scrollTop, termHeight)
```

### Transactions

```go
store.Batch(func(tx *Transaction) {
    tx.Set(store.Messages, newMessages)
    tx.Set(store.StreamingText, text)
    // single render flush on commit
})
```

---

## Engine Main Loop

```
stdin → Terminal.Read() → EventClassifier
                              │
                   ┌──────────┼──────────┐
                   ▼          ▼          ▼
              MouseEvent  PasteEvent  ParsedKey
                   │          │          │
                   └──────────┼──────────┘
                              ▼
                   InputRouter (keybindings lookup)
                              │
                              ▼
                   store.Batch(atom writes)
                              │
                              ▼
                   scheduleRender() [async, 16ms coalesce]
                              │
                              ▼
                   reconcile → layout → rasterize → paint
```

### Render pipeline (4 phases)

1. **reconcile** — diff rootFiber against new VNode, mark effectTags
2. **layoutDirty** — only layout dirty fibers; VirtualList only layouts visible children
3. **rasterizeDirty** — only rasterize changed regions
4. **paintDirtyRect** — row-by-row cell diff; only emit changed rows to stdout

Performance targets: idle 0ms, input <8ms, streaming token <8ms, first-frame/resize <30ms, 1000-msg scroll <8ms.

### Input routing

Keybindings stored as `map[ParsedKey]Action`; actions write atoms via `tx`. Default bindings cover: Enter(submit), Alt+Enter(newline), Ctrl+O(transcript), F2(slash picker), /(search), Esc(dismiss), arrows(scroll), PgUp/Dn, End, Ctrl+C(quit), etc.

---

## Components: Phase 1 Scope

### Tier A — Direct migration (existing ink/compo, engine upgrade only)

REPL layout (ScrollBox→VirtualList), Messages, MessageRow, AssistantMessage, UserMessage, SystemMessage, Markdown (+UseMemo), PromptInput (+UseAtom), StatusLine (+UseAtom), Pipeline (→Selector).

### Tier B — Ported from `gou/app` (existing Bubble Tea implementation)

| Component | Source | Function |
|-----------|--------|----------|
| TranscriptScreen | `transcript_screen.go` | Frozen history view, search, dump, external editor |
| TranscriptSearch | `transcript_search.go` | Search bar UI, match highlight, result count |
| SlashPicker | `slash_picker.go` | F2 popup, input filter, Tab insert |
| Spinner | `spinner_row.go` | Tool execution spinner + tips |
| Scrollbar | `message_scrollbar.go` | Scrollbar indicator |

### Tier C — New from scratch (TS parity, deferred)

CodeHighlight (`HighlightedCode/`), FileEditDiff (`FileEditToolDiff.tsx`), ThinkingToggle, ToolResultBlock, PermissionPills, StreamingText — deferred to later phases.

### Deferred to backend-integration phase

PermissionModal, QuestionUI, AgentFooter, AtSuggest, TaskList.

---

## File Plan

```
gou/ink/
  core/
    terminal.go       — NEW
    keyboard.go        — NEW
    keyboard_test.go   — NEW
    mouse.go           — NEW
  vdom/
    vnode.go           — RETAINED (minor: Context expansion)
    reconciler.go      — REWRITE (fiber-style)
    reconciler_test.go — REWRITE
    hooks.go           — NEW
    hooks_test.go      — NEW
  layout/
    flexbox.go         — RETAINED (minor: VirtualList handler)
    flexbox_test.go    — RETAINED
    virtual_scroll.go  — NEW
    virtual_scroll_test.go — NEW
  render/
    rasterize.go       — RETAINED (minor: dirty-region rasterize)
    screen.go          — RETAINED
    paint.go           — NEW (extracted from engine.go)
  store/
    store.go           — REWRITE (reactive + transactions)
    store_test.go      — REWRITE
    atoms.go           — NEW
    atoms_test.go      — NEW
    selectors.go       — NEW
  engine.go            — REWRITE (orchestration of all layers)
  engine_test.go       — REWRITE
  component.go         — NEW (extracted from vnode.go)
  terminal.go          — RETAINED (re-exported from core/)
  reconciler.go        — DELETED (moved to vdom/)
  screen.go            — DELETED (moved to render/)
  store.go             — DELETED (moved to store/)
  flexbox.go           — DELETED (moved to layout/)
  rasterize.go         — DELETED (moved to render/)
  diff.go              — RETAINED (internal to paint)
  ansi.go              — RETAINED (used by keyboard + rasterize)

gou/ink/compo/         — RETAINED (tier A migration, minor hook adoption)
gou/app/               — UNCHANGED (existing Bubble Tea app)
```
