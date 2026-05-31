# ink Engine UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure `gou/ink` into a 5-layer engine (core/vdom/layout/render/store) with fiber reconciler, reactive state, virtual scroll, keyboard/mouse input, then migrate existing components.

**Architecture:** Five-layer engine matching claude-code TS patterns: core (terminal I/O), vdom (fiber reconciler + hooks), layout (flexbox + virtual scroll), render (rasterize + paint), store (atoms + selectors + transactions). Components adopt hooks but keep existing visual output.

**Tech Stack:** Go 1.26, `golang.org/x/term`, `goc/gou/virtualscroll`, `goc/gou/theme`, `goc/gou/layout`, `goc/gou/markdown`, `goc/gou/messagerow`

**Spec:** `docs/superpowers/specs/2026-05-31-ink-engine-ui-design.md`

---

## File Structure

```
gou/ink/
  core/                  — Platform abstraction (new)
    terminal.go          — Terminal: raw mode, alt screen, SIGWINCH
    terminal_test.go     — Tests with mock stdin/stdout
    keyboard.go          — KeyboardParser FSM → ParsedKey
    keyboard_test.go     — Test every CSI/Kitty/SS3 sequence
    mouse.go             — SGR 1006 decoder → MouseEvent
    mouse_test.go        — Test mouse event decoding
  vdom/                  — Virtual DOM
    vnode.go             — VNode + Props + Constraints + Context (moved, expanded)
    reconciler.go        — Fiber-based incremental reconciler (rewrite)
    reconciler_test.go   — Diff tests for all effect types
    hooks.go             — useState, useEffect, useMemo, useCallback, useAtom, useSelector
    hooks_test.go        — Hook lifecycle tests
  layout/                — Layout engine
    flexbox.go           — Flexbox layout (moved, +VirtualList handler)
    flexbox_test.go      — Layout tests (moved)
    virtual_scroll.go    — VirtualScrollState wrapping goc/gou/virtualscroll
    virtual_scroll_test.go — Range computation tests
  render/                — Rendering
    screen.go            — Screen + TermCell + CellStyle (moved)
    rasterize.go         — Rasterize (moved, +dirty-region variant)
    paint.go             — DiffEngine extracted (moved from diff.go)
    ansi.go              — ANSI helpers (moved)
  store/                 — Reactive state management
    store.go             — Store with Batch, ScheduleRender (rewrite)
    store_test.go        — Store tests
    atoms.go             — Atom[T] generic atomic state
    atoms_test.go        — Atom tests
    selectors.go         — Selector[T] memoized derived state
    selectors_test.go    — Selector tests
  component.go           — Component type + Context (extracted from vnode.go)
  engine.go              — RenderEngine orchestrating all layers (rewrite)
  engine_test.go         — Integration tests
  compo/                 — UI components (retained, hook adoption)
    repl.go, messages.go, assistant_message.go, user_message.go,
    system_message.go, markdown.go, prompt.go, statusline.go,
    tool_use.go, collapsed_group.go, grouped.go, pipeline.go
```

Old flat-package files that will be deleted after migration:
```
gou/ink/reconciler.go, gou/ink/screen.go, gou/ink/store.go,
gou/ink/flexbox.go, gou/ink/rasterize.go, gou/ink/terminal.go,
gou/ink/diff.go, gou/ink/ansi.go
```

---

### Task 1: Create directory structure and migrate unchanged files

**Files:**
- Create: `gou/ink/core/terminal.go`
- Create: `gou/ink/vdom/vnode.go`
- Create: `gou/ink/render/screen.go`
- Create: `gou/ink/render/ansi.go`
- Create: `gou/ink/layout/flexbox.go`
- Create: `gou/ink/layout/flexbox_test.go`
- Create: `gou/ink/render/rasterize.go`

- [ ] **Step 1: Create subdirectories**

```bash
mkdir -p gou/ink/core gou/ink/vdom gou/ink/layout gou/ink/render gou/ink/store
```

- [ ] **Step 2: Move terminal.go to core/ and update package declaration**

Move `gou/ink/terminal.go` → `gou/ink/core/terminal.go`, change `package ink` to `package core`.

- [ ] **Step 3: Move vnode.go to vdom/**

Move `gou/ink/vnode.go` → `gou/ink/vdom/vnode.go`, change `package ink` to `package vdom`. Update import of theme to reference `goc/gou/theme`. Add `Context` struct expansion:

```go
package vdom

import "goc/gou/theme"

type VNode struct {
	Type     string
	Key      string
	Props    Props
	Children []VNode
	Layout   LayoutResult
}

type Props map[string]interface{}

func (p Props) GetString(key string) string { /* unchanged */ }
func (p Props) GetInt(key string) int { /* unchanged */ }
func (p Props) GetBool(key string) bool { /* unchanged */ }
func (p Props) Get(key string) interface{} { /* unchanged */ }

type LayoutResult struct {
	X, Y         int
	W, H         int
	ContentH     int
	OverflowTop  int
	VisibleRange [2]int
}

type Constraints struct {
	MinW, MaxW int
	MinH, MaxH int
}

func Unbounded() Constraints {
	return Constraints{MinW: 0, MaxW: 1<<31 - 1, MinH: 0, MaxH: 1<<31 - 1}
}

// Context is expanded for hooks and fiber tracking.
type Context struct {
	Theme     *theme.Palette
	Store     StoreReader
	schedule  func()
	
	// Internal: current fiber during reconciliation
	fiber     *Fiber
	hookIndex int
}

type StoreReader interface {
	GetMessages() []Message
	StreamingText() string
	StreamingTools() []StreamingToolUse
	InputValue() string
	CursorPos() int
	IsLoading() bool
	Width() int
	Height() int
	GetMeta(key string) string
}

type Component func(ctx *Context, props Props) VNode

// Forward declarations for messages/types used across packages.
type Message struct {
	UUID          string
	Type          string
	ContentBlocks []ContentBlock
	Meta          map[string]interface{}
}

type ContentBlock struct {
	Type    string
	Content string
	Name    string
	Input   map[string]interface{}
	State   string
	Result  *ContentBlock
	Meta    map[string]interface{}
}

type StreamingToolUse struct {
	UUID  string
	Name  string
	Input map[string]interface{}
}
```

- [ ] **Step 4: Move screen.go to render/**

Move `gou/ink/screen.go` → `gou/ink/render/screen.go`, change `package ink` to `package render`.

- [ ] **Step 5: Move ansi.go to render/**

Move `gou/ink/ansi.go` → `gou/ink/render/ansi.go`, change `package ink` to `package render`.

- [ ] **Step 6: Move rasterize.go to render/**

Move `gou/ink/rasterize.go` → `gou/ink/render/rasterize.go`, change `package ink` to `package render`. Update import of vdom package:

```go
package render

import (
	"image/color"
	"goc/gou/ink/vdom"
)
```

Replace all `VNode` references with `vdom.VNode`, all `CellStyle` with local `CellStyle`, `Props` with `vdom.Props`. Change `rasterizeNode` to accept `*vdom.VNode`.

- [ ] **Step 7: Move flexbox.go to layout/**

Move `gou/ink/flexbox.go` → `gou/ink/layout/flexbox.go`, change `package ink` to `package layout`. Update imports:

```go
package layout

import (
	"goc/gou/ink/render"
	"goc/gou/ink/vdom"
	"github.com/mattn/go-runewidth"
)
```

Replace all `VNode` references with `vdom.VNode`, `Props` with `vdom.Props`, `Constraints` with `vdom.Constraints`, `LayoutResult` with `vdom.LayoutResult`, `Unbounded` with `vdom.Unbounded`. Import render for `stripANSIStr` → use local copy or export from render.

- [ ] **Step 8: Move flexbox_test.go to layout/**

Move `gou/ink/flexbox_test.go` → `gou/ink/layout/flexbox_test.go`. Update package and imports.

- [ ] **Step 9: Remove old files**

```bash
rm gou/ink/terminal.go gou/ink/vnode.go gou/ink/screen.go \
   gou/ink/ansi.go gou/ink/rasterize.go gou/ink/flexbox.go \
   gou/ink/flexbox_test.go
```

- [ ] **Step 10: Compile check**

Run: `cd goc && go build ./gou/ink/core/... ./gou/ink/vdom/... ./gou/ink/render/... ./gou/ink/layout/...`
Expected: compiles successfully (may have import cycle warnings from render depending on vdom — resolve by ensuring render types are self-contained).

- [ ] **Step 11: Commit**

```bash
git add gou/ink/core/ gou/ink/vdom/ gou/ink/render/ gou/ink/layout/
git add -u gou/ink/
git commit -m "refactor(ink): restructure into core/vdom/layout/render subpackages
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 2: Store layer — Atom[T]

**Files:**
- Create: `gou/ink/store/atoms.go`
- Create: `gou/ink/store/atoms_test.go`

- [ ] **Step 1: Write failing tests for Atom[T]**

Create `gou/ink/store/atoms_test.go`:

```go
package store

import (
	"sync"
	"testing"
)

func TestAtomGetSet(t *testing.T) {
	atom := NewAtom(42)
	if atom.Get() != 42 {
		t.Fatalf("expected 42, got %d", atom.Get())
	}
	atom.Set(99)
	if atom.Get() != 99 {
		t.Fatalf("expected 99, got %d", atom.Get())
	}
}

func TestAtomWatch(t *testing.T) {
	atom := NewAtom("hello")
	var mu sync.Mutex
	var received []string
	unsub := atom.Watch(func(v string) {
		mu.Lock()
		received = append(received, v)
		mu.Unlock()
	})
	atom.Set("world")
	atom.Set("foo")
	unsub()
	atom.Set("bar")
	mu.Lock()
	if len(received) != 2 {
		t.Fatalf("expected 2 notifications, got %d: %v", len(received), received)
	}
	if received[0] != "world" || received[1] != "foo" {
		t.Fatalf("unexpected values: %v", received)
	}
	mu.Unlock()
}

func TestAtomWatchMultipleSubscribers(t *testing.T) {
	atom := NewAtom(0)
	count := 0
	var mu sync.Mutex
	inc := func(v int) { mu.Lock(); count++; mu.Unlock() }
	atom.Watch(inc)
	atom.Watch(inc)
	atom.Set(1)
	mu.Lock()
	if count != 2 {
		t.Fatalf("expected 2 increments, got %d", count)
	}
	mu.Unlock()
}

func TestAtomVersion(t *testing.T) {
	atom := NewAtom(0)
	v1 := atom.Version()
	atom.Set(1)
	v2 := atom.Version()
	if v2 <= v1 {
		t.Fatal("version should increment after Set")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd goc && go test ./gou/ink/store/... -run TestAtom`
Expected: FAIL — "undefined: NewAtom"

- [ ] **Step 3: Implement Atom[T]**

Create `gou/ink/store/atoms.go`:

```go
package store

import (
	"sync"
	"sync/atomic"
)

type AtomReader interface {
	Version() uint64
}

type Atom[T any] struct {
	mu       sync.RWMutex
	value    T
	version  uint64
	watchers map[uint64]func(T)
	nextID   uint64
}

func NewAtom[T any](initial T) *Atom[T] {
	return &Atom[T]{
		value:    initial,
		watchers: make(map[uint64]func(T)),
	}
}

func (a *Atom[T]) Get() T {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.value
}

func (a *Atom[T]) Set(val T) {
	a.mu.Lock()
	a.value = val
	atomic.AddUint64(&a.version, 1)
	watchers := make([]func(T), 0, len(a.watchers))
	for _, w := range a.watchers {
		watchers = append(watchers, w)
	}
	a.mu.Unlock()
	for _, w := range watchers {
		w(val)
	}
}

func (a *Atom[T]) Watch(fn func(T)) func() {
	a.mu.Lock()
	id := a.nextID
	a.nextID++
	a.watchers[id] = fn
	a.mu.Unlock()
	return func() {
		a.mu.Lock()
		delete(a.watchers, id)
		a.mu.Unlock()
	}
}

func (a *Atom[T]) Version() uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.version
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd goc && go test ./gou/ink/store/... -run TestAtom -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gou/ink/store/atoms.go gou/ink/store/atoms_test.go
git commit -m "feat(ink/store): add Atom[T] reactive state primitive
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 3: Store layer — Selector[T]

**Files:**
- Create: `gou/ink/store/selectors.go`
- Create: `gou/ink/store/selectors_test.go`

- [ ] **Step 1: Write failing tests for Selector[T]**

Create `gou/ink/store/selectors_test.go`:

```go
package store

import (
	"testing"
)

func TestSelectorBasic(t *testing.T) {
	a := NewAtom(10)
	b := NewAtom(20)
	sel := NewSelector([]AtomReader{a, b}, func() interface{} {
		return a.Get() + b.Get()
	})
	if sel.Get() != 30 {
		t.Fatalf("expected 30, got %v", sel.Get())
	}
}

func TestSelectorMemo(t *testing.T) {
	a := NewAtom(10)
	b := NewAtom(20)
	computes := 0
	sel := NewSelector([]AtomReader{a, b}, func() interface{} {
		computes++
		return a.Get() + b.Get()
	})
	_ = sel.Get()
	_ = sel.Get()
	_ = sel.Get()
	if computes != 1 {
		t.Fatalf("expected 1 computation, got %d", computes)
	}
}

func TestSelectorRecomputeOnChange(t *testing.T) {
	a := NewAtom(10)
	b := NewAtom(20)
	computes := 0
	sel := NewSelector([]AtomReader{a, b}, func() interface{} {
		computes++
		return a.Get() + b.Get()
	})
	_ = sel.Get()
	a.Set(100)
	_ = sel.Get()
	if computes != 2 {
		t.Fatalf("expected 2 computations, got %d", computes)
	}
}

func TestSelectorNoRecomputeOnSameValue(t *testing.T) {
	a := NewAtom(10)
	computes := 0
	sel := NewSelector([]AtomReader{a}, func() interface{} {
		computes++
		return a.Get() * 2
	})
	_ = sel.Get()
	a.Set(10) // same value
	_ = sel.Get()
	// Still recomputes because Atom version increments even on same-value Set
	if computes != 2 {
		t.Fatalf("expected 2 computations (version-based invalidation), got %d", computes)
	}
}

func TestSelectorTyped(t *testing.T) {
	a := NewAtom("hello")
	b := NewAtom("world")
	sel := NewTypedSelector([]AtomReader{a, b}, func() string {
		return a.Get() + " " + b.Get()
	})
	if sel.Get() != "hello world" {
		t.Fatalf("expected 'hello world', got %q", sel.Get())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd goc && go test ./gou/ink/store/... -run TestSelector`
Expected: FAIL — "undefined: NewSelector"

- [ ] **Step 3: Implement Selector[T]**

Create `gou/ink/store/selectors.go`:

```go
package store

import "sync"

type Selector struct {
	deps       []AtomReader
	compute    func() interface{}
	cached     interface{}
	cachedVer  uint64
	mu         sync.RWMutex
}

func NewSelector(deps []AtomReader, compute func() interface{}) *Selector {
	return &Selector{deps: deps, compute: compute}
}

func (s *Selector) Get() interface{} {
	ver := s.combinedVersion()
	s.mu.RLock()
	if ver == s.cachedVer {
		v := s.cached
		s.mu.RUnlock()
		return v
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	// double-check
	ver = s.combinedVersion()
	if ver == s.cachedVer {
		return s.cached
	}
	s.cached = s.compute()
	s.cachedVer = ver
	return s.cached
}

func (s *Selector) combinedVersion() uint64 {
	var v uint64
	for _, d := range s.deps {
		v += d.Version()
	}
	return v
}

func (s *Selector) Version() uint64 {
	return s.combinedVersion()
}

// TypedSelector wraps Selector with type safety.
type TypedSelector[T any] struct {
	raw *Selector
}

func NewTypedSelector[T any](deps []AtomReader, compute func() T) *TypedSelector[T] {
	return &TypedSelector[T]{
		raw: NewSelector(deps, func() interface{} { return compute() }),
	}
}

func (ts *TypedSelector[T]) Get() T {
	return ts.raw.Get().(T)
}

func (ts *TypedSelector[T]) Version() uint64 {
	return ts.raw.Version()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd goc && go test ./gou/ink/store/... -run TestSelector -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gou/ink/store/selectors.go gou/ink/store/selectors_test.go
git commit -m "feat(ink/store): add Selector[T] memoized derived state
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 4: Store layer — Reactive Store with transactions

**Files:**
- Create: `gou/ink/store/store.go`
- Create: `gou/ink/store/store_test.go`

- [ ] **Step 1: Write failing tests for Store**

Create `gou/ink/store/store_test.go`:

```go
package store

import (
	"testing"
	"time"
)

func TestStoreDefineAtom(t *testing.T) {
	s := NewStore()
	msgAtom := s.DefineAtom("messages", []string{})
	if msgAtom.Get()[0] != "" {
		// empty slice
	}
}

func TestStoreBatchCommit(t *testing.T) {
	s := NewStore()
	a := s.DefineAtom("count", 0)
	b := s.DefineAtom("text", "")

	renders := 0
	s.SetOnRender(func() { renders++ })

	s.Batch(func(tx *Transaction) {
		tx.Set(a, 10)
		tx.Set(b, "hello")
	})

	if a.Get() != 10 {
		t.Fatalf("expected 10, got %d", a.Get())
	}
	if b.Get() != "hello" {
		t.Fatalf("expected hello, got %q", b.Get())
	}
	// Batch should trigger exactly one render
	time.Sleep(20 * time.Millisecond)
	if renders != 1 {
		t.Fatalf("expected 1 render after batch, got %d", renders)
	}
}

func TestStoreDefineAtomTyped(t *testing.T) {
	s := NewStore()
	a := DefineAtom[string](s, "name", "default")
	if a.Get() != "default" {
		t.Fatalf("expected default, got %q", a.Get())
	}
	a.Set("alice")
	if a.Get() != "alice" {
		t.Fatalf("expected alice, got %q", a.Get())
	}
}

func TestStoreTransactionAbandon(t *testing.T) {
	s := NewStore()
	a := s.DefineAtom("x", 0)
	renders := 0
	s.SetOnRender(func() { renders++ })

	func() {
		tx := s.Begin()
		tx.Set(a, 999)
		// no commit (tx goes out of scope)
	}()

	if a.Get() != 0 {
		t.Fatalf("expected 0 (abandoned tx), got %d", a.Get())
	}
}

func TestStoreRenderCoalesce(t *testing.T) {
	s := NewStore()
	renders := 0
	s.SetOnRender(func() { renders++ })

	for i := 0; i < 5; i++ {
		s.ScheduleRender()
	}
	time.Sleep(30 * time.Millisecond)
	// Should coalesce to 1 render
	if renders > 2 {
		t.Fatalf("expected 1-2 renders (coalesced), got %d", renders)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd goc && go test ./gou/ink/store/... -run TestStore`
Expected: FAIL — "undefined: NewStore"

- [ ] **Step 3: Implement Store**

Create `gou/ink/store/store.go`:

```go
package store

import (
	"sync"
	"time"
)

type Store struct {
	mu        sync.RWMutex
	atoms     map[string]interface{} // key → *Atom[any]
	subs      map[string]func()
	renderCh  chan struct{}
	onRender  func()
	closeCh   chan struct{}
	txActive  bool
	dirty     map[string]struct{}
}

func NewStore() *Store {
	return &Store{
		atoms:    make(map[string]interface{}),
		subs:     make(map[string]func()),
		renderCh: make(chan struct{}, 1),
		closeCh:  make(chan struct{}),
		dirty:    make(map[string]struct{}),
	}
}

func (s *Store) DefineAtom(key string, initial interface{}) *Atom[interface{}] {
	a := NewAtom[interface{}](initial)
	s.mu.Lock()
	s.atoms[key] = a
	s.mu.Unlock()
	return a
}

func DefineAtom[T any](s *Store, key string, initial T) *Atom[T] {
	a := NewAtom(initial)
	s.mu.Lock()
	s.atoms[key] = a
	s.mu.Unlock()
	return a
}

func (s *Store) GetAtom(key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.atoms[key]
}

func (s *Store) SetOnRender(fn func()) { s.onRender = fn }

func (s *Store) ScheduleRender() {
	select {
	case s.renderCh <- struct{}{}:
	default:
	}
}

func (s *Store) RunRenderLoop() {
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.renderCh:
			if s.onRender != nil {
				s.onRender()
			}
		case <-ticker.C:
		case <-s.closeCh:
			return
		}
	}
}

func (s *Store) Stop() { close(s.closeCh) }

// Begin starts a new transaction.
func (s *Store) Begin() *Transaction {
	s.mu.Lock()
	s.txActive = true
	return &Transaction{store: s, dirty: make(map[string]struct{})}
}

// Batch runs fn in a transaction, commits, and triggers one render.
func (s *Store) Batch(fn func(tx *Transaction)) {
	tx := s.Begin()
	fn(tx)
	tx.Commit()
}

type Transaction struct {
	store   *Store
	dirty   map[string]struct{}
	committed bool
}

func (tx *Transaction) Set(atom interface{}, val interface{}) {
	switch a := atom.(type) {
	case *Atom[interface{}]:
		a.Set(val)
	case interface{ set(interface{}) }:
		a.set(val)
	}
}

func (tx *Transaction) Commit() {
	if tx.committed {
		return
	}
	tx.committed = true
	tx.store.txActive = false
	tx.store.mu.Unlock()
	tx.store.ScheduleRender()
}

// track tracks an atom key as dirty for batched watch notification.
func (s *Store) track(key string) {
	if s.txActive {
		s.dirty[key] = struct{}{}
	}
}
```

- [ ] **Step 4: Run tests, fix iteratively, verify pass**

Run: `cd goc && go test ./gou/ink/store/... -v`
Expected: all store tests PASS

- [ ] **Step 5: Commit**

```bash
git add gou/ink/store/store.go gou/ink/store/store_test.go
git commit -m "feat(ink/store): add reactive Store with Batch transactions
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 5: Core layer — Keyboard parser

**Files:**
- Create: `gou/ink/core/keyboard.go`
- Create: `gou/ink/core/keyboard_test.go`

- [ ] **Step 1: Write tests for KeyboardParser**

Create `gou/ink/core/keyboard_test.go`:

```go
package core

import "testing"

func TestParseBasicKeys(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want ParsedKey
	}{
		{"enter", []byte("\r"), ParsedKey{Key: "enter"}},
		{"tab", []byte("\t"), ParsedKey{Key: "tab"}},
		{"backspace", []byte{127}, ParsedKey{Key: "backspace"}},
		{"ctrl+c", []byte{3}, ParsedKey{Key: "c", Mod: Ctrl}},
		{"escape", []byte{27}, ParsedKey{Key: "esc"}},
		{"letter a", []byte("a"), ParsedKey{Runes: []rune("a")}},
		{"letter A", []byte("A"), ParsedKey{Runes: []rune("A")}},
		{"unicode", []byte("世"), ParsedKey{Runes: []rune("世")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewKeyboardParser()
			got := p.Parse(tt.raw)
			if got.Key != tt.want.Key {
				t.Errorf("Key: got %q, want %q", got.Key, tt.want.Key)
			}
			if got.Mod != tt.want.Mod {
				t.Errorf("Mod: got %v, want %v", got.Mod, tt.want.Mod)
			}
			if tt.want.Runes != nil {
				if string(got.Runes) != string(tt.want.Runes) {
					t.Errorf("Runes: got %q, want %q", string(got.Runes), string(tt.want.Runes))
				}
			}
		})
	}
}

func TestParseCSI(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want ParsedKey
	}{
		{"up", []byte("\x1b[A"), ParsedKey{Key: "up"}},
		{"down", []byte("\x1b[B"), ParsedKey{Key: "down"}},
		{"right", []byte("\x1b[C"), ParsedKey{Key: "right"}},
		{"left", []byte("\x1b[D"), ParsedKey{Key: "left"}},
		{"home", []byte("\x1b[H"), ParsedKey{Key: "home"}},
		{"end", []byte("\x1b[F"), ParsedKey{Key: "end"}},
		{"pgup", []byte("\x1b[5~"), ParsedKey{Key: "pgup"}},
		{"pgdn", []byte("\x1b[6~"), ParsedKey{Key: "pgdn"}},
		{"delete", []byte("\x1b[3~"), ParsedKey{Key: "delete"}},
		{"f1", []byte("\x1bOP"), ParsedKey{Key: "f1"}},
		{"f2", []byte("\x1bOQ"), ParsedKey{Key: "f2"}},
		{"shift+up", []byte("\x1b[1;2A"), ParsedKey{Key: "up", Mod: Shift}},
		{"ctrl+up", []byte("\x1b[1;5A"), ParsedKey{Key: "up", Mod: Ctrl}},
		{"alt+up", []byte("\x1b[1;3A"), ParsedKey{Key: "up", Mod: Alt}},
		{"ctrl+shift+up", []byte("\x1b[1;6A"), ParsedKey{Key: "up", Mod: Ctrl | Shift}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewKeyboardParser()
			got := p.Parse(tt.raw)
			if got.Key != tt.want.Key {
				t.Errorf("Key: got %q, want %q", got.Key, tt.want.Key)
			}
			if got.Mod != tt.want.Mod {
				t.Errorf("Mod: got 0x%x, want 0x%x", got.Mod, tt.want.Mod)
			}
		})
	}
}

func TestParseKitty(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want ParsedKey
	}{
		{"ctrl+o kitty", []byte("\x1b[57399u"), ParsedKey{Key: "o", Mod: Ctrl}},
		{"ctrl+enter kitty", []byte("\x1b[13;5u"), ParsedKey{Key: "enter", Mod: Ctrl}},
		{"alt+enter kitty", []byte("\x1b[13;3u"), ParsedKey{Key: "enter", Mod: Alt}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewKeyboardParser()
			got := p.Parse(tt.raw)
			if got.Key != tt.want.Key || got.Mod != tt.want.Mod {
				t.Errorf("got {%q 0x%x}, want {%q 0x%x}", got.Key, got.Mod, tt.want.Key, tt.want.Mod)
			}
		})
	}
}

func TestParseMultiByteUTF8(t *testing.T) {
	p := NewKeyboardParser()
	input := []byte("hello")
	// KeyboardParser returns one key per byte group — multi-byte UTF8 is in one call
	got := p.Parse(input)
	if string(got.Runes) != "hello" {
		t.Errorf("expected hello, got %q", string(got.Runes))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd goc && go test ./gou/ink/core/... -run TestParse -v`
Expected: FAIL

- [ ] **Step 3: Implement KeyboardParser**

Create `gou/ink/core/keyboard.go`:

```go
package core

const (
	Ctrl  Modifier = 1 << iota
	Alt
	Shift
	Meta
)

type Modifier uint8

type ParsedKey struct {
	Key   string
	Mod   Modifier
	Runes []rune
}

type KeyboardParser struct {
	buf []byte
}

func NewKeyboardParser() *KeyboardParser {
	return &KeyboardParser{buf: make([]byte, 0, 32)}
}

func (p *KeyboardParser) Parse(raw []byte) ParsedKey {
	if len(raw) == 0 {
		return ParsedKey{}
	}
	b := raw[0]

	// Ctrl+letter (1-26)
	if b >= 1 && b <= 26 {
		key := string(rune('a' + b - 1))
		return ParsedKey{Key: key, Mod: Ctrl}
	}

	switch b {
	case 13: // CR
		return ParsedKey{Key: "enter"}
	case 9: // TAB
		return ParsedKey{Key: "tab"}
	case 127: // DEL
		return ParsedKey{Key: "backspace"}
	case 27: // ESC
		if len(raw) == 1 {
			return ParsedKey{Key: "esc"}
		}
		return p.parseEscape(raw[1:])
	default:
		// Printable characters
		runes := []rune(string(raw))
		return ParsedKey{Runes: runes}
	}
}

func (p *KeyboardParser) parseEscape(rest []byte) ParsedKey {
	if len(rest) == 0 {
		return ParsedKey{Key: "esc"}
	}
	switch rest[0] {
	case '[':
		return p.parseCSI(rest[1:])
	case 'O':
		return p.parseSS3(rest[1:])
	default:
		return ParsedKey{Key: "esc", Runes: []rune(string(rest))}
	}
}

func (p *KeyboardParser) parseCSI(rest []byte) ParsedKey {
	if len(rest) == 0 {
		return ParsedKey{}
	}
	// Kitty protocol: CSI <number> u
	if rest[len(rest)-1] == 'u' {
		return p.parseKitty(rest[:len(rest)-1])
	}
	// CSI <number> ~  (PgUp, PgDn, Delete, etc.)
	if rest[len(rest)-1] == '~' {
		return p.parseCSITilde(rest[:len(rest)-1])
	}
	// Simple CSI: final byte is the command
	final := rest[len(rest)-1]
	params := rest[:len(rest)-1]

	mod, param1 := parseCSIParams(params)

	switch final {
	case 'A': return ParsedKey{Key: "up", Mod: mod}
	case 'B': return ParsedKey{Key: "down", Mod: mod}
	case 'C': return ParsedKey{Key: "right", Mod: mod}
	case 'D': return ParsedKey{Key: "left", Mod: mod}
	case 'H': return ParsedKey{Key: "home", Mod: mod}
	case 'F': return ParsedKey{Key: "end", Mod: mod}
	case 'Z': return ParsedKey{Key: "tab", Mod: Shift} // CSI Z = Shift+Tab
	default:
		_ = param1
		return ParsedKey{}
	}
}

func (p *KeyboardParser) parseCSITilde(params []byte) ParsedKey {
	mod, param1 := parseCSIParams(params)
	switch param1 {
	case 2: return ParsedKey{Key: "insert", Mod: mod}
	case 3: return ParsedKey{Key: "delete", Mod: mod}
	case 5: return ParsedKey{Key: "pgup", Mod: mod}
	case 6: return ParsedKey{Key: "pgdn", Mod: mod}
	case 7: return ParsedKey{Key: "home", Mod: mod}
	case 8: return ParsedKey{Key: "end", Mod: mod}
	case 11: return ParsedKey{Key: "f1", Mod: mod}
	case 12: return ParsedKey{Key: "f2", Mod: mod}
	case 13: return ParsedKey{Key: "f3", Mod: mod}
	case 14: return ParsedKey{Key: "f4", Mod: mod}
	case 15: return ParsedKey{Key: "f5", Mod: mod}
	case 17: return ParsedKey{Key: "f6", Mod: mod}
	case 18: return ParsedKey{Key: "f7", Mod: mod}
	case 19: return ParsedKey{Key: "f8", Mod: mod}
	case 20: return ParsedKey{Key: "f9", Mod: mod}
	case 21: return ParsedKey{Key: "f10", Mod: mod}
	case 23: return ParsedKey{Key: "f11", Mod: mod}
	case 24: return ParsedKey{Key: "f12", Mod: mod}
	default: return ParsedKey{}
	}
}

func (p *KeyboardParser) parseSS3(rest []byte) ParsedKey {
	if len(rest) == 0 {
		return ParsedKey{}
	}
	switch rest[0] {
	case 'P': return ParsedKey{Key: "f1"}
	case 'Q': return ParsedKey{Key: "f2"}
	case 'R': return ParsedKey{Key: "f3"}
	case 'S': return ParsedKey{Key: "f4"}
	case 'H': return ParsedKey{Key: "home"}
	case 'F': return ParsedKey{Key: "end"}
	default: return ParsedKey{}
	}
}

func (p *KeyboardParser) parseKitty(params []byte) ParsedKey {
	mod, codepoint := parseCSIParams(params)
	// Map modifier bits (Kitty protocol: 1=shift, 2=alt, 4=ctrl, 8=meta)
	var modBits Modifier
	if mod&1 != 0 { modBits |= Shift }
	if mod&2 != 0 { modBits |= Alt }
	if mod&4 != 0 { modBits |= Ctrl }
	if mod&8 != 0 { modBits |= Meta }

	key := keyNameFromCodepoint(codepoint)
	return ParsedKey{Key: key, Mod: modBits}
}

func parseCSIParams(params []byte) (mod, param1 int) {
	// Parse semicolon-separated numbers
	s := string(params)
	parts := splitByte(s, ';')
	if len(parts) > 1 {
		param1 = atoi(parts[0])
		mod = atoi(parts[1]) - 1 // CSI mod is 1-indexed
	} else if len(parts) == 1 {
		param1 = atoi(parts[0])
		mod = param1 - 1
	}
	return
}

func keyNameFromCodepoint(cp int) string {
	switch cp {
	case 13: return "enter"
	case 9: return "tab"
	case 32: return "space"
	case 127: return "backspace"
	case 27: return "esc"
	default:
		if cp >= 1 && cp <= 26 {
			return string(rune('a' + cp - 1))
		}
		if cp >= 0x20 && cp <= 0x7E {
			return string(rune(cp))
		}
	}
	return ""
}

func splitByte(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		}
	}
	return n
}
```

- [ ] **Step 4: Run tests and fix until pass**

Run: `cd goc && go test ./gou/ink/core/... -run TestParse -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gou/ink/core/keyboard.go gou/ink/core/keyboard_test.go
git commit -m "feat(ink/core): add KeyboardParser with CSI/SS3/Kitty support
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 6: Core layer — Mouse event decoder

**Files:**
- Create: `gou/ink/core/mouse.go`
- Create: `gou/ink/core/mouse_test.go`

- [ ] **Step 1: Write MouseEvent decoder tests**

Create `gou/ink/core/mouse_test.go`:

```go
package core

import "testing"

func TestDecodeMousePress(t *testing.T) {
	ev, ok := DecodeMouse([]byte("\x1b[<0;10;20M"))
	if !ok {
		t.Fatal("expected mouse event")
	}
	if ev.Type != MousePress {
		t.Errorf("expected Press, got %v", ev.Type)
	}
	if ev.X != 10 || ev.Y != 20 {
		t.Errorf("expected (10,20), got (%d,%d)", ev.X, ev.Y)
	}
	if ev.Button != 0 {
		t.Errorf("expected button 0, got %d", ev.Button)
	}
}

func TestDecodeMouseRelease(t *testing.T) {
	ev, ok := DecodeMouse([]byte("\x1b[<0;5;10m"))
	if !ok {
		t.Fatal("expected mouse event")
	}
	if ev.Type != MouseRelease {
		t.Errorf("expected Release, got %v", ev.Type)
	}
}

func TestDecodeMouseWheelUp(t *testing.T) {
	ev, ok := DecodeMouse([]byte("\x1b[<64;10;5M"))
	if !ok {
		t.Fatal("expected mouse event")
	}
	if ev.Type != MouseWheel {
		t.Errorf("expected Wheel, got %v", ev.Type)
	}
	if ev.Button != -1 {
		t.Errorf("expected button -1 (up), got %d", ev.Button)
	}
}

func TestDecodeMouseWheelDown(t *testing.T) {
	ev, ok := DecodeMouse([]byte("\x1b[<65;10;5M"))
	if !ok {
		t.Fatal("expected mouse event")
	}
	if ev.Button != 1 {
		t.Errorf("expected button 1 (down), got %d", ev.Button)
	}
}

func TestDecodeNonMouse(t *testing.T) {
	_, ok := DecodeMouse([]byte("\x1b[A"))
	if ok {
		t.Error("up arrow should not be a mouse event")
	}
	_, ok = DecodeMouse([]byte("hello"))
	if ok {
		t.Error("text should not be a mouse event")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd goc && go test ./gou/ink/core/... -run TestDecodeMouse -v`
Expected: FAIL

- [ ] **Step 3: Implement MouseEvent decoder**

Create `gou/ink/core/mouse.go`:

```go
package core

type MouseEventType uint8

const (
	MousePress   MouseEventType = iota
	MouseRelease
	MouseMove
	MouseWheel
)

type MouseEvent struct {
	Type   MouseEventType
	Button int  // 0=left, 1=middle, 2=right; wheel: -1=up, 1=down
	X, Y   int  // 1-indexed cell coordinates
	Mod    Modifier
}

// DecodeMouse decodes an SGR 1006 mouse sequence.
// Format: ESC [ < Btn ; X ; Y M/m
func DecodeMouse(raw []byte) (MouseEvent, bool) {
	if len(raw) < 6 {
		return MouseEvent{}, false
	}
	if raw[0] != '\x1b' || raw[1] != '[' || raw[2] != '<' {
		return MouseEvent{}, false
	}

	s := string(raw[3:])
	// Split by ';' and extract final char (M/m)
	var nums []int
	start := 0
	var final byte
	for i := 0; i < len(s); i++ {
		if s[i] == ';' {
			nums = append(nums, atoi(s[start:i]))
			start = i + 1
		}
		if s[i] == 'M' || s[i] == 'm' {
			final = s[i]
			nums = append(nums, atoi(s[start:i]))
			break
		}
	}

	if len(nums) < 3 {
		return MouseEvent{}, false
	}

	btn := nums[0]
	ev := MouseEvent{X: nums[1], Y: nums[2]}

	// Extract modifier bits (bits 2-3 of btn field)
	if btn&4 != 0 { ev.Mod |= Shift }
	if btn&8 != 0 { ev.Mod |= Alt }
	if btn&16 != 0 { ev.Mod |= Ctrl }

	// Wheel events: btn >= 64
	if btn >= 64 {
		ev.Type = MouseWheel
		if btn == 64 {
			ev.Button = -1 // up
		} else {
			ev.Button = 1 // down
		}
		return ev, true
	}

	// Motion: btn 32-63
	btnBase := btn & 3
	if btn >= 32 {
		ev.Type = MouseMove
		ev.Button = btnBase
		return ev, true
	}

	// Press/release
	ev.Button = btnBase
	if final == 'm' {
		ev.Type = MouseRelease
	} else {
		ev.Type = MousePress
	}
	return ev, true
}

// IsMouseEvent returns true if raw bytes start an SGR mouse sequence.
func IsMouseEvent(raw []byte) bool {
	return len(raw) >= 4 && raw[0] == '\x1b' && raw[1] == '[' && raw[2] == '<'
}

// IsBracketedPasteStart returns true if raw starts a bracketed paste.
func IsBracketedPasteStart(raw []byte) bool {
	return len(raw) >= 6 && string(raw[:6]) == "\x1b[200~"
}

// IsBracketedPasteEnd returns true if raw ends a bracketed paste.
func IsBracketedPasteEnd(raw []byte) bool {
	return len(raw) >= 6 && string(raw[:6]) == "\x1b[201~"
}
```

- [ ] **Step 4: Run tests and fix until pass**

Run: `cd goc && go test ./gou/ink/core/... -run TestDecodeMouse -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gou/ink/core/mouse.go gou/ink/core/mouse_test.go
git commit -m "feat(ink/core): add SGR 1006 mouse event decoder
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 7: Core layer — Terminal upgrade (alt screen + protocol negotiation)

**Files:**
- Modify: `gou/ink/core/terminal.go` (currently moved from flat package, add alt screen and protocol methods)

- [ ] **Step 1: Read current terminal.go in core/**

Read the file to understand its current state after Task 1 move.

- [ ] **Step 2: Add protocol methods**

Add these methods to the `Terminal` struct in `core/terminal.go`:

```go
// EnableMouse sends SGR mouse tracking enable sequences.
func (t *Terminal) EnableMouse() {
	fmt.Fprint(t.stdout, "\x1b[?1000h\x1b[?1002h\x1b[?1006h")
}

// DisableMouse sends SGR mouse tracking disable sequences.
func (t *Terminal) DisableMouse() {
	fmt.Fprint(t.stdout, "\x1b[?1006l\x1b[?1002l\x1b[?1000l")
}

// EnableKittyKbd sends the Kitty keyboard protocol enable sequence.
func (t *Terminal) EnableKittyKbd() {
	fmt.Fprint(t.stdout, "\x1b[>1u")
	t.restoreFuncs = append(t.restoreFuncs, func() {
		fmt.Fprint(t.stdout, "\x1b[<u")
	})
}

// EnableBracketedPaste sends the bracketed paste enable sequence.
func (t *Terminal) EnableBracketedPaste() {
	fmt.Fprint(t.stdout, "\x1b[?2004h")
	t.restoreFuncs = append(t.restoreFuncs, func() {
		fmt.Fprint(t.stdout, "\x1b[?2004l")
	})
}

// EnterAltScreen switches to the alternate screen buffer.
func (t *Terminal) EnterAltScreen() {
	fmt.Fprint(t.stdout, "\x1b[?1049h")
	t.restoreFuncs = append(t.restoreFuncs, func() {
		fmt.Fprint(t.stdout, "\x1b[?1049l")
	})
}

// Read returns the raw stdin byte channel (renamed from ReadStdin for spec consistency).
func (t *Terminal) Read() <-chan []byte {
	return t.ReadStdin()
}
```

- [ ] **Step 3: Modify Init to make mouse optional**

Change `Init()` so SGR mouse is NOT enabled by default — caller must call `EnableMouse()`:

```go
func (t *Terminal) Init() error {
	fd := int(t.stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("make raw: %w", err)
	}
	t.prevState = state
	t.restoreFuncs = append(t.restoreFuncs, func() {
		term.Restore(fd, state)
	})

	w, h, err := term.GetSize(fd)
	if err == nil {
		t.width, t.height = w, h
	}

	go t.handleSignals()
	return nil
}
```

- [ ] **Step 4: Add ResizeCh for spec consistency**

Add a non-blocking resize channel alongside `ResizeEvents()`:

```go
// ResizeCh returns a channel that receives struct{} on SIGWINCH.
func (t *Terminal) ResizeCh() <-chan struct{} {
	ch := make(chan struct{}, 8)
	go func() {
		for range t.resizeCh {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}()
	return ch
}
```

- [ ] **Step 5: Compile and test**

Run: `cd goc && go build ./gou/ink/core/... && go test ./gou/ink/core/... -v`
Expected: compiles, existing tests pass

- [ ] **Step 6: Commit**

```bash
git add gou/ink/core/terminal.go
git commit -m "feat(ink/core): add alt screen, protocol negotiation, resize channel to Terminal
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 8: VDOM layer — Fiber Reconciler

**Files:**
- Create: `gou/ink/vdom/reconciler.go`
- Create: `gou/ink/vdom/reconciler_test.go`

- [ ] **Step 1: Write Fiber reconciler tests**

Create `gou/ink/vdom/reconciler_test.go`:

```go
package vdom

import "testing"

func TestReconcilerSameTypeNoChange(t *testing.T) {
	old := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "hello"}},
	}}
	new := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "hello"}},
	}}
	r := &FiberReconciler{}
	root := r.Reconcile(old, new)
	if root.effectTag != NoEffect {
		t.Errorf("expected NoEffect, got %v", root.effectTag)
	}
}

func TestReconcilerContentChange(t *testing.T) {
	old := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "hello"}},
	}}
	new := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "world"}},
	}}
	r := &FiberReconciler{}
	root := r.Reconcile(old, new)
	if getEffectChain(root) == NoEffect {
		t.Error("expected effect from content change")
	}
}

func TestReconcilerNewChild(t *testing.T) {
	old := &VNode{Type: "Box", Key: "root", Children: []VNode{}}
	new := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "new"}},
	}}
	r := &FiberReconciler{}
	root := r.Reconcile(old, new)
	hasPlacement := false
	walkEffects(root, func(f *Fiber) {
		if f.effectTag == Placement {
			hasPlacement = true
		}
	})
	if !hasPlacement {
		t.Error("expected Placement effect for new child")
	}
}

func TestReconcilerRemoveChild(t *testing.T) {
	old := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "x"}},
	}}
	new := &VNode{Type: "Box", Key: "root", Children: []VNode{}}
	r := &FiberReconciler{}
	root := r.Reconcile(old, new)
	hasDeletion := false
	walkEffects(root, func(f *Fiber) {
		if f.effectTag == Deletion {
			hasDeletion = true
		}
	})
	if !hasDeletion {
		t.Error("expected Deletion effect for removed child")
	}
}

func TestReconcilerTypeChangeReplaces(t *testing.T) {
	old := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Text", Key: "a", Props: Props{"content": "x"}},
	}}
	new := &VNode{Type: "Box", Key: "root", Children: []VNode{
		{Type: "Box", Key: "a", Props: Props{"direction": "row"}},
	}}
	r := &FiberReconciler{}
	root := r.Reconcile(old, new)
	hasReplace := false
	walkEffects(root, func(f *Fiber) {
		if f.effectTag == Replacement {
			hasReplace = true
		}
	})
	if !hasReplace {
		t.Error("expected Replacement effect for type change")
	}
}

func getEffectChain(f *Fiber) EffectTag {
	tag := f.effectTag
	for c := f.child; c != nil; c = c.sibling {
		if c.effectTag != NoEffect {
			tag = c.effectTag
		}
	}
	return tag
}

func walkEffects(f *Fiber, fn func(*Fiber)) {
	if f.effectTag != NoEffect {
		fn(f)
	}
	for c := f.child; c != nil; c = c.sibling {
		walkEffects(c, fn)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd goc && go test ./gou/ink/vdom/... -run TestReconciler -v`
Expected: FAIL

- [ ] **Step 3: Implement FiberReconciler**

Create `gou/ink/vdom/reconciler.go`:

```go
package vdom

type EffectTag int

const (
	NoEffect    EffectTag = iota
	Placement             // new fiber, needs mount
	Update                // props changed
	Deletion              // fiber removed
	Replacement           // type changed, full replace
)

type Fiber struct {
	vnode     *VNode
	child     *Fiber
	sibling   *Fiber
	returnFiber *Fiber
	effectTag EffectTag
	hooks     []HookState
	deleted   bool
}

type FiberReconciler struct{}

func (r *FiberReconciler) Reconcile(oldRoot, newRoot *VNode) *Fiber {
	if oldRoot == nil {
		f := &Fiber{vnode: newRoot, effectTag: Placement}
		r.buildFiberTree(f, newRoot)
		return f
	}
	if oldRoot.Type != newRoot.Type || oldRoot.Key != newRoot.Key {
		f := &Fiber{vnode: newRoot, effectTag: Replacement}
		r.buildFiberTree(f, newRoot)
		return f
	}
	root := &Fiber{vnode: newRoot}
	r.reconcileChildren(root, oldRoot.Children, newRoot.Children)
	return root
}

func (r *FiberReconciler) buildFiberTree(fiber *Fiber, vnode *VNode) {
	for i := range vnode.Children {
		child := &Fiber{vnode: &vnode.Children[i], effectTag: Placement, returnFiber: fiber}
		if i == 0 {
			fiber.child = child
		} else {
			// find last sibling
			s := fiber.child
			for s.sibling != nil {
				s = s.sibling
			}
			s.sibling = child
		}
		r.buildFiberTree(child, &vnode.Children[i])
	}
}

func (r *FiberReconciler) reconcileChildren(parent *Fiber, oldKids, newKids []VNode) {
	oldByKey := make(map[string]int)
	for i, v := range oldKids {
		if v.Key != "" {
			oldByKey[v.Key] = i
		}
	}
	newByKey := make(map[string]int)
	for i, v := range newKids {
		if v.Key != "" {
			newByKey[v.Key] = i
		}
	}

	var prevSibling *Fiber
	for i, newV := range newKids {
		var childFiber *Fiber

		if oldIdx, ok := oldByKey[newV.Key]; ok && newV.Key != "" {
			oldV := &oldKids[oldIdx]
			if oldV.Type != newV.Type {
				childFiber = &Fiber{vnode: &newV, effectTag: Replacement, returnFiber: parent}
				r.buildFiberTree(childFiber, &newV)
			} else {
				childFiber = &Fiber{vnode: &newV, returnFiber: parent}
				if propsChanged(oldV, &newV) {
					childFiber.effectTag = Update
				}
				r.reconcileChildren(childFiber, oldV.Children, newV.Children)
			}
		} else {
			childFiber = &Fiber{vnode: &newV, effectTag: Placement, returnFiber: parent}
			r.buildFiberTree(childFiber, &newV)
		}

		if i == 0 {
			parent.child = childFiber
		} else {
			prevSibling.sibling = childFiber
		}
		prevSibling = childFiber
	}

	// Mark deletions for keys in old but not in new
	for i, oldV := range oldKids {
		if oldV.Key == "" {
			continue
		}
		if _, ok := newByKey[oldV.Key]; !ok {
			del := &Fiber{vnode: &oldKids[i], effectTag: Deletion, returnFiber: parent}
			del.deleted = true
			// Append to end of sibling list
			if parent.child == nil {
				parent.child = del
			} else {
				s := parent.child
				for s.sibling != nil {
					s = s.sibling
				}
				s.sibling = del
			}
		}
	}
}

func propsChanged(oldV, newV *VNode) bool {
	if oldV.Props == nil && newV.Props == nil {
		return false
	}
	if oldV.Props == nil || newV.Props == nil {
		return true
	}
	// Shallow compare known keys
	keys := []string{"content", "bold", "dim", "italic", "underline", "width", "height", "flexGrow", "direction", "stickyBottom", "color", "bg", "padding", "gap", "alignItems", "justifyContent", "minWidth"}
	for _, k := range keys {
		if oldV.Props[k] != newV.Props[k] {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests and fix until pass**

Run: `cd goc && go test ./gou/ink/vdom/... -run TestReconciler -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gou/ink/vdom/reconciler.go gou/ink/vdom/reconciler_test.go
git commit -m "feat(ink/vdom): add FiberReconciler with key-based diff and effect tags
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 9: VDOM layer — Hooks

**Files:**
- Create: `gou/ink/vdom/hooks.go`
- Create: `gou/ink/vdom/hooks_test.go`

- [ ] **Step 1: Write hooks tests**

Create `gou/ink/vdom/hooks_test.go`:

```go
package vdom

import (
	"sync"
	"testing"
)

func TestUseState(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}

	val, setVal := UseState(ctx, f, 0)
	if val != 0 {
		t.Fatalf("expected 0, got %d", val)
	}
	setVal(42)
	val2, _ := UseState(ctx, f, 0)
	if val2 != 42 {
		t.Fatalf("expected 42 after set, got %d", val2)
	}
}

func TestUseStatePreservesIndex(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}

	UseState(ctx, f, "first")
	UseState(ctx, f, 999)
	// Reset hookIndex (simulating second render)
	ctx.hookIndex = 0
	v1, _ := UseState(ctx, f, "first")
	v2, _ := UseState(ctx, f, 999)
	if v1 != "first" {
		t.Fatalf("expected first, got %v", v1)
	}
	if v2 != 999 {
		t.Fatalf("expected 999, got %d", v2)
	}
}

func TestUseEffect(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}

	var mu sync.Mutex
	effects := 0
	UseEffect(ctx, f, func() func() {
		mu.Lock()
		effects++
		mu.Unlock()
		return nil
	}, nil)
	if effects != 1 {
		t.Fatalf("expected 1 effect run, got %d", effects)
	}
}

func TestUseEffectCleanup(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}

	cleaned := false
	UseEffect(ctx, f, func() func() {
		return func() { cleaned = true }
	}, []interface{}{1})

	// Second render with different deps triggers cleanup + re-run
	ctx.hookIndex = 0
	UseEffect(ctx, f, func() func() {
		return func() {}
	}, []interface{}{2})

	if !cleaned {
		t.Error("expected cleanup to run when deps change")
	}
}

func TestUseMemo(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}

	computes := 0
	result := UseMemo(ctx, f, func() interface{} {
		computes++
		return 42
	}, []interface{}{1, 2})

	if result != 42 {
		t.Fatalf("expected 42, got %v", result)
	}
	if computes != 1 {
		t.Fatalf("expected 1 compute, got %d", computes)
	}

	// Same deps — no recompute
	ctx.hookIndex = 0
	_ = UseMemo(ctx, f, func() interface{} {
		computes++
		return 99
	}, []interface{}{1, 2})

	if computes != 1 {
		t.Fatalf("expected still 1 compute (memo hit), got %d", computes)
	}
}

func TestUseMemoDepsChange(t *testing.T) {
	ctx := &Context{hookIndex: 0}
	f := &Fiber{hooks: make([]HookState, 10)}

	computes := 0
	_ = UseMemo(ctx, f, func() interface{} {
		computes++
		return 10
	}, []interface{}{1})

	ctx.hookIndex = 0
	_ = UseMemo(ctx, f, func() interface{} {
		computes++
		return 20
	}, []interface{}{2})

	if computes != 2 {
		t.Fatalf("expected 2 computes (deps changed), got %d", computes)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd goc && go test ./gou/ink/vdom/... -run "TestUse" -v`
Expected: FAIL

- [ ] **Step 3: Implement Hooks**

Create `gou/ink/vdom/hooks.go`:

```go
package vdom

import "reflect"

type HookState struct {
	state    interface{}
	queue    []func(interface{}) interface{}
	deps     []interface{}
	cleanup  func()
	memoized interface{}
	 effectRun bool
}

// UseState returns stateful value and a setter.
// The setter schedules a re-render (via ctx.schedule).
func UseState[T any](ctx *Context, fiber *Fiber, initial T) (T, func(T)) {
	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx < len(fiber.hooks) && fiber.hooks[idx].state != nil {
		return fiber.hooks[idx].state.(T), func(v T) {
			fiber.hooks[idx].state = v
			if ctx.schedule != nil {
				ctx.schedule()
			}
		}
	}

	// Initialize
	for len(fiber.hooks) <= idx {
		fiber.hooks = append(fiber.hooks, HookState{})
	}
	fiber.hooks[idx].state = initial
	return initial, func(v T) {
		fiber.hooks[idx].state = v
		if ctx.schedule != nil {
			ctx.schedule()
		}
	}
}

// UseEffect runs fn after render, with cleanup on deps change.
func UseEffect(ctx *Context, fiber *Fiber, fn func() func(), deps []interface{}) {
	idx := ctx.hookIndex
	ctx.hookIndex++

	for len(fiber.hooks) <= idx {
		fiber.hooks = append(fiber.hooks, HookState{})
	}

	hs := &fiber.hooks[idx]

	if hs.effectRun && depsEqual(hs.deps, deps) {
		return
	}

	// Run cleanup from previous run
	if hs.cleanup != nil {
		hs.cleanup()
	}

	hs.deps = cloneDeps(deps)
	hs.cleanup = fn()
	hs.effectRun = true
}

// UseMemo returns a memoized value, recomputing only when deps change.
func UseMemo(ctx *Context, fiber *Fiber, fn func() interface{}, deps []interface{}) interface{} {
	idx := ctx.hookIndex
	ctx.hookIndex++

	for len(fiber.hooks) <= idx {
		fiber.hooks = append(fiber.hooks, HookState{})
	}

	hs := &fiber.hooks[idx]

	if hs.effectRun && depsEqual(hs.deps, deps) {
		return hs.memoized
	}

	hs.deps = cloneDeps(deps)
	hs.memoized = fn()
	hs.effectRun = true
	return hs.memoized
}

// UseCallback returns a stable function reference while deps are unchanged.
func UseCallback(ctx *Context, fiber *Fiber, fn interface{}, deps []interface{}) interface{} {
	idx := ctx.hookIndex
	ctx.hookIndex++

	for len(fiber.hooks) <= idx {
		fiber.hooks = append(fiber.hooks, HookState{})
	}

	hs := &fiber.hooks[idx]

	if hs.effectRun && depsEqual(hs.deps, deps) {
		return hs.memoized
	}

	hs.deps = cloneDeps(deps)
	hs.memoized = fn
	hs.effectRun = true
	return fn
}

func depsEqual(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func cloneDeps(deps []interface{}) []interface{} {
	if deps == nil {
		return nil
	}
	out := make([]interface{}, len(deps))
	copy(out, deps)
	return out
}
```

- [ ] **Step 4: Run tests and fix until pass**

Run: `cd goc && go test ./gou/ink/vdom/... -run "TestUse" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add gou/ink/vdom/hooks.go gou/ink/vdom/hooks_test.go
git commit -m "feat(ink/vdom): add useState, useEffect, useMemo, useCallback hooks
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 10: Layout layer — VirtualScrollState + VirtualList VNode

**Files:**
- Create: `gou/ink/layout/virtual_scroll.go`
- Create: `gou/ink/layout/virtual_scroll_test.go`

- [ ] **Step 1: Write VirtualScrollState tests**

Create `gou/ink/layout/virtual_scroll_test.go`:

```go
package layout

import (
	"testing"
	"goc/gou/virtualscroll"
)

func TestVirtualScrollStateComputeRange(t *testing.T) {
	vs := &VirtualScrollState{
		RowHeights: []int{5, 5, 5, 5, 5, 5, 5, 5, 5, 5},
		ScrollTop:  0,
		ViewportH:  20,
		Overscan:   2,
	}
	from, to, offset, total := vs.ComputeRange()
	if from != 0 || to != 6 || offset != 0 || total != 50 {
		t.Errorf("got from=%d to=%d offset=%d total=%d, want 0,6,0,50", from, to, offset, total)
	}
}

func TestVirtualScrollStateStickyBottom(t *testing.T) {
	vs := &VirtualScrollState{
		RowHeights:   []int{5, 5, 5, 5, 5},
		ScrollTop:    5,
		ViewportH:    20,
		Overscan:     2,
		StickyBottom: true,
	}
	vs.UpdateForNewContent(8)
	// Should auto-scroll to bottom
	if vs.ScrollTop <= 5 {
		t.Errorf("expected scrollTop > 5 after stickyBottom update, got %d", vs.ScrollTop)
	}
}

func TestVirtualScrollStateUpdateHeights(t *testing.T) {
	vs := &VirtualScrollState{
		RowHeights: []int{3, 3, 3},
		ViewportH:  10,
		Overscan:   1,
	}
	vs.UpdateHeight(1, 10)
	if vs.RowHeights[1] != 10 {
		t.Errorf("expected height 10, got %d", vs.RowHeights[1])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd goc && go test ./gou/ink/layout/... -run TestVirtualScroll -v`
Expected: FAIL

- [ ] **Step 3: Implement VirtualScrollState**

Create `gou/ink/layout/virtual_scroll.go`:

```go
package layout

import "goc/gou/virtualscroll"

type VirtualScrollState struct {
	RowHeights   []int
	Offsets      []int
	ScrollTop    int
	ViewportH    int
	Overscan     int
	StickyBottom bool
}

func NewVirtualScrollState(rowCount, viewportH int) *VirtualScrollState {
	heights := make([]int, rowCount)
	for i := range heights {
		heights[i] = 1 // default height; will be updated by actual layout
	}
	return &VirtualScrollState{
		RowHeights: heights,
		Offsets:    make([]int, rowCount),
		ViewportH:  viewportH,
		Overscan:   5,
	}
}

func (vs *VirtualScrollState) ComputeRange() (from, to, offsetTop int, totalH int) {
	return virtualscroll.ComputeRange(
		len(vs.RowHeights), vs.RowHeights,
		vs.ScrollTop, vs.ViewportH, vs.Overscan,
	)
}

func (vs *VirtualScrollState) UpdateForNewContent(rowCount int) {
	for len(vs.RowHeights) < rowCount {
		vs.RowHeights = append(vs.RowHeights, 1)
	}
	if vs.StickyBottom {
		total := vs.totalHeight()
		vs.ScrollTop = total - vs.ViewportH
		if vs.ScrollTop < 0 {
			vs.ScrollTop = 0
		}
	}
}

func (vs *VirtualScrollState) UpdateHeight(index, h int) {
	if index < 0 || index >= len(vs.RowHeights) {
		return
	}
	vs.RowHeights[index] = h
}

func (vs *VirtualScrollState) totalHeight() int {
	total := 0
	for _, h := range vs.RowHeights {
		total += h
	}
	return total
}

func (vs *VirtualScrollState) SetRowCount(n int) {
	if n == len(vs.RowHeights) {
		return
	}
	if n > len(vs.RowHeights) {
		for len(vs.RowHeights) < n {
			vs.RowHeights = append(vs.RowHeights, 1)
		}
	} else {
		vs.RowHeights = vs.RowHeights[:n]
	}
}
```

- [ ] **Step 4: Run tests and fix until pass**

Run: `cd goc && go test ./gou/ink/layout/... -run TestVirtualScroll -v`
Expected: PASS (may need to check virtualscroll.ComputeRange signature)

- [ ] **Step 5: Add VirtualList VNode handler to flexbox.go**

In `gou/ink/layout/flexbox.go`, add a `"VirtualList"` case to the `ComputeLayout` function:

```go
func ComputeLayout(node *vdom.VNode, constraints vdom.Constraints) {
	switch node.Type {
	case "Text":
		layoutText(node, constraints)
	case "Box":
		layoutBox(node, constraints)
	case "ScrollBox":
		layoutScrollBox(node, constraints)
	case "VirtualList":
		layoutVirtualList(node, constraints)
	default:
		node.Layout = vdom.LayoutResult{W: 0, H: 0}
	}
}
```

And add the `layoutVirtualList` function using the existing virtualscroll integration:

```go
func layoutVirtualList(node *vdom.VNode, c vdom.Constraints) {
	w := node.Props.GetInt("width")
	if w <= 0 {
		w = c.MaxW
	}

	vs, _ := node.Props["virtualScroll"].(*VirtualScrollState)
	if vs == nil {
		// Fallback: layout all children
		layoutScrollBox(node, c)
		return
	}

	from, to, offsetTop, totalH := vs.ComputeRange()
	if to > len(node.Children) {
		to = len(node.Children)
	}

	contentH := 0
	for i := range node.Children {
		if i >= from && i < to {
			childC := vdom.Constraints{MinW: 0, MaxW: w}
			ComputeLayout(&node.Children[i], childC)
			node.Children[i].Layout.Y = contentH - offsetTop
			node.Children[i].Layout.X = 0
		}
		// Use cached height or default
		h := 1
		if i < len(vs.RowHeights) {
			h = vs.RowHeights[i]
		}
		contentH += h
	}

	h := node.Props.GetInt("height")
	if h <= 0 {
		h = c.MaxH
	}
	if h <= 0 {
		h = contentH
	}

	node.Layout = vdom.LayoutResult{
		W:            w,
		H:            h,
		ContentH:     totalH,
		OverflowTop:  from,
		VisibleRange: [2]int{from, to},
	}
}
```

- [ ] **Step 6: Compile and test**

Run: `cd goc && go build ./gou/ink/layout/... && go test ./gou/ink/layout/... -v`
Expected: compiles, all tests pass

- [ ] **Step 7: Commit**

```bash
git add gou/ink/layout/virtual_scroll.go gou/ink/layout/virtual_scroll_test.go gou/ink/layout/flexbox.go
git commit -m "feat(ink/layout): add VirtualScrollState and VirtualList VNode handler
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 11: Render layer — Paint (extract from diff.go)

**Files:**
- Create: `gou/ink/render/paint.go`
- Delete: `gou/ink/diff.go` (after extraction)

- [ ] **Step 1: Move DiffEngine to render/paint.go**

Create `gou/ink/render/paint.go` with the DiffEngine code from `gou/ink/diff.go`, updated to use `package render` and local types:

```go
package render

import "strings"

type DiffEngine struct {
	cursorX, cursorY int
}

func NewDiffEngine() *DiffEngine {
	return &DiffEngine{}
}

func (d *DiffEngine) Generate(prev, curr *Screen) string {
	var buf strings.Builder
	w := curr.Width
	h := curr.Height

	for row := 0; row < h; row++ {
		prevRow := safeRow(prev, row, w)
		currRow := curr.Cells[row]
		if rowsEqual(prevRow, currRow) {
			continue
		}

		d.moveTo(&buf, row, 0)

		var run []TermCell
		var runStyle CellStyle
		inRun := false

		for col := 0; col < w; col++ {
			prevCell := safeCell(prev, row, col)
			currCell := curr.Cells[row][col]

			if prevCell.Equals(currCell) {
				if inRun {
					d.flushRun(&buf, run, runStyle)
					run = nil
					inRun = false
				}
				d.cursorX = col + 1
				continue
			}

			if !inRun {
				runStyle = currCell.Style
				inRun = true
			} else if !currCell.Style.Equals(runStyle) {
				d.flushRun(&buf, run, runStyle)
				run = nil
				runStyle = currCell.Style
			}
			run = append(run, currCell)
		}
		if inRun {
			d.flushRun(&buf, run, runStyle)
		}
	}
	buf.WriteString(sgrReset())
	return buf.String()
}

func (d *DiffEngine) moveTo(buf *strings.Builder, row, col int) {
	buf.WriteString(cursorTo(row, col))
	d.cursorY = row
	d.cursorX = col
}

func (d *DiffEngine) flushRun(buf *strings.Builder, run []TermCell, style CellStyle) {
	if len(run) == 0 {
		return
	}
	sgr := styleToSGR(style)
	if sgr != "" {
		buf.WriteString(sgr)
	}
	for _, c := range run {
		if c.Rune == 0 {
			buf.WriteByte(' ')
		} else {
			buf.WriteRune(c.Rune)
		}
	}
	buf.WriteString(sgrReset())
	d.cursorX += len(run)
}

func rowsEqual(a, b []TermCell) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equals(b[i]) {
			return false
		}
	}
	return true
}

func safeRow(s *Screen, row, width int) []TermCell {
	if s == nil || row >= len(s.Cells) {
		return make([]TermCell, width)
	}
	return s.Cells[row]
}

func safeCell(s *Screen, row, col int) TermCell {
	if s == nil || row >= len(s.Cells) || col >= len(s.Cells[row]) {
		return TermCell{}
	}
	return s.Cells[row][col]
}
```

- [ ] **Step 2: Delete old diff.go**

```bash
rm gou/ink/diff.go
```

- [ ] **Step 3: Compile**

Run: `cd goc && go build ./gou/ink/render/...`
Expected: compiles

- [ ] **Step 4: Commit**

```bash
git add gou/ink/render/paint.go && git add -u gou/ink/diff.go
git commit -m "refactor(ink/render): extract DiffEngine to render/paint.go
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 12: Engine — New RenderEngine with full pipeline

**Files:**
- Rewrite: `gou/ink/engine.go`
- Rewrite: `gou/ink/engine_test.go`
- Create: `gou/ink/component.go`

- [ ] **Step 1: Write component.go (Component type + Context)**

Create `gou/ink/component.go`:

```go
package ink

import (
	"goc/gou/ink/core"
	"goc/gou/ink/layout"
	"goc/gou/ink/render"
	"goc/gou/ink/store"
	"goc/gou/ink/vdom"
	"goc/gou/theme"
)

// Re-export types for backward compatibility with compo/.
type (
	VNode            = vdom.VNode
	Props            = vdom.Props
	Constraints       = vdom.Constraints
	LayoutResult     = vdom.LayoutResult
	Component        = vdom.Component
	Context           = vdom.Context
	Message          = vdom.Message
	ContentBlock     = vdom.ContentBlock
	StreamingToolUse = vdom.StreamingToolUse
	StoreReader      = vdom.StoreReader

	Screen           = render.Screen
	TermCell         = render.TermCell
	CellStyle        = render.CellStyle
	DiffEngine       = render.DiffEngine

	VirtualScrollState = layout.VirtualScrollState

	Atom[T any]       = store.Atom[T]
	Selector          = store.Selector
	TypedSelector[T any] = store.TypedSelector[T]
	Store             = store.Store
	Transaction       = store.Transaction

	ParsedKey         = core.ParsedKey
	MouseEvent        = core.MouseEvent
	MouseEventType    = core.MouseEventType
	Modifier          = core.Modifier
)

// Re-export constants
const (
	Ctrl  = core.Ctrl
	Alt   = core.Alt
	Shift = core.Shift
	Meta  = core.Meta

	MousePress   = core.MousePress
	MouseRelease = core.MouseRelease
	MouseMove    = core.MouseMove
	MouseWheel   = core.MouseWheel

	NoEffect    = vdom.NoEffect
	Placement   = vdom.Placement
	Update      = vdom.Update
	Deletion    = vdom.Deletion
	Replacement = vdom.Replacement
)

// Re-export functions
var (
	NewTerminal        = core.NewTerminal
	NewKeyboardParser   = core.NewKeyboardParser
	IsMouseEvent       = core.IsMouseEvent
	DecodeMouse        = core.DecodeMouse
	IsBracketedPasteStart = core.IsBracketedPasteStart
	IsBracketedPasteEnd   = core.IsBracketedPasteEnd

	NewScreen          = render.NewScreen
	NewDiffEngine      = render.NewDiffEngine
	Rasterize          = render.Rasterize
	StyleToSGR         = render.StyleToSGR // exported from ansi.go

	ComputeLayout      = layout.ComputeLayout
	NewVirtualScrollState = layout.NewVirtualScrollState

	NewStore           = store.NewStore
	DefineAtom         = store.DefineAtom
	NewSelector        = store.NewSelector
	NewTypedSelector   = store.NewTypedSelector
)
```

- [ ] **Step 2: Write new engine.go**

Rewrite `gou/ink/engine.go`:

```go
package ink

import (
	"goc/gou/ink/core"
	"goc/gou/ink/layout"
	"goc/gou/ink/render"
	"goc/gou/ink/store"
	"goc/gou/ink/vdom"
	"goc/gou/theme"
)

type RenderEngine struct {
	Terminal   *core.Terminal
	Store      *store.Store
	Theme      *theme.Palette
	RootComp   vdom.Component

	reconciler *vdom.FiberReconciler
	prevVTree  *vdom.VNode
	prevScreen *render.Screen
	firstFrame bool

	keyParser  *core.KeyboardParser
	diffEngine *render.DiffEngine

	quitCh     chan struct{}
}

func NewEngine(term *core.Terminal, st *store.Store, pal *theme.Palette, root vdom.Component) *RenderEngine {
	return &RenderEngine{
		Terminal:   term,
		Store:      st,
		Theme:      pal,
		RootComp:   root,
		reconciler: &vdom.FiberReconciler{},
		keyParser:  core.NewKeyboardParser(),
		diffEngine: render.NewDiffEngine(),
		firstFrame: true,
		quitCh:     make(chan struct{}),
	}
}

func (e *RenderEngine) Run() error {
	if err := e.Terminal.Init(); err != nil {
		return err
	}
	defer e.Terminal.Shutdown()

	e.Store.SetOnRender(e.syncRender)

	go e.Store.RunRenderLoop()

	e.syncRender() // first frame

	for {
		select {
		case raw, ok := <-e.Terminal.Read():
			if !ok {
				return nil
			}
			e.handleInput(raw)

		case <-e.Terminal.ResizeCh():
			e.handleResize()

		case <-e.quitCh:
			return nil
		}
	}
}

func (e *RenderEngine) handleInput(raw []byte) {
	switch {
	case core.IsMouseEvent(raw):
		ev, ok := core.DecodeMouse(raw)
		if ok {
			e.onMouse(ev)
		}
	case core.IsBracketedPasteStart(raw):
		// Collect paste content until end marker
		e.onPaste(raw)
	default:
		key := e.keyParser.Parse(raw)
		e.onKey(key)
	}
}

func (e *RenderEngine) onKey(key core.ParsedKey) {
	// Default keybindings
	switch {
	case key.Key == "c" && key.Mod&core.Ctrl != 0:
		e.Quit()
	case key.Key == "enter" && key.Mod == 0:
		e.handleSubmit()
	case key.Key == "enter" && key.Mod&core.Alt != 0:
		e.handleNewline()
	case key.Key == "o" && key.Mod&core.Ctrl != 0:
		e.handleToggleTranscript()
	case key.Key == "up":
		e.scrollBy(-1)
	case key.Key == "down":
		e.scrollBy(1)
	case key.Key == "pgup":
		e.scrollBy(-e.Store.Height() / 2)
	case key.Key == "pgdn":
		e.scrollBy(e.Store.Height() / 2)
	case key.Key == "end":
		e.scrollToBottom()
	default:
		if len(key.Runes) > 0 {
			e.handleRunes(key.Runes)
		}
	}
}

func (e *RenderEngine) onMouse(ev core.MouseEvent) {
	// Stub: store mouse event for component access
}

func (e *RenderEngine) onPaste(raw []byte) {
	// Stub: strip bracketed paste markers
}

func (e *RenderEngine) handleResize() {
	w, h := e.Terminal.Size()
	e.Store.DefineAtom("termWidth", w)
	e.Store.DefineAtom("termHeight", h)
	e.prevScreen = nil
	e.Store.ScheduleRender()
}

func (e *RenderEngine) handleSubmit()         { /* stub — will be connected in integration phase */ }
func (e *RenderEngine) handleNewline()        { /* stub */ }
func (e *RenderEngine) handleToggleTranscript() { /* stub */ }
func (e *RenderEngine) scrollBy(delta int) {
	top := e.scrollTop()
	top += delta
	if top < 0 { top = 0 }
	// set scrollTop atom
}
func (e *RenderEngine) scrollToBottom()       { /* stub */ }
func (e *RenderEngine) handleRunes(r []rune) {
	input := e.inputValue() + string(r)
	// set inputValue atom
}

func (e *RenderEngine) scrollTop() int {
	// read from store atom
	return 0
}
func (e *RenderEngine) inputValue() string {
	// read from store atom
	return ""
}

func (e *RenderEngine) syncRender() {
	w, h := e.Terminal.Size()
	if w <= 0 { w = 80 }
	if h <= 0 { h = 24 }

	ctx := &vdom.Context{
		Theme:    e.Theme,
		schedule: e.Store.ScheduleRender,
	}
	newTree := e.RootComp(ctx, vdom.Props{})

	e.reconciler.Reconcile(e.prevVTree, &newTree)
	e.prevVTree = &newTree

	layout.ComputeLayout(&newTree, vdom.Constraints{MinW: 0, MaxW: w, MinH: 0, MaxH: h})

	cur := render.NewScreen(w, h)
	render.Rasterize(&newTree, cur)

	if e.firstFrame {
		e.Terminal.Write([]byte(eraseDisplay()))
		e.Terminal.Write([]byte(cursorTo(0, 0)))
		e.firstFrame = false
	}

	// Incremental paint via diff
	output := e.diffEngine.Generate(e.prevScreen, cur)
	if output != "" {
		e.Terminal.Write([]byte(output))
	}

	// Position cursor
	cursorY := h - 2
	e.Terminal.Write([]byte(cursorTo(cursorY, 2)))

	// Save screen
	if e.prevScreen == nil || e.prevScreen.Width != w || e.prevScreen.Height != h {
		e.prevScreen = render.NewScreen(w, h)
	} else {
		e.prevScreen.Clear()
	}
	copyScreen(e.prevScreen, cur)
}

func (e *RenderEngine) Quit() {
	close(e.quitCh)
}

func copyScreen(dst, src *render.Screen) {
	minH := dst.Height
	if src.Height < minH { minH = src.Height }
	minW := dst.Width
	if src.Width < minW { minW = src.Width }
	for y := 0; y < minH; y++ {
		copy(dst.Cells[y][:minW], src.Cells[y][:minW])
	}
}
```

- [ ] **Step 3: Write engine_test.go**

Rewrite `gou/ink/engine_test.go`:

```go
package ink

import (
	"testing"
	"goc/gou/theme"
)

func TestEngineCreate(t *testing.T) {
	// Engine creation without a real terminal (unit test only)
	st := NewStore()
	pal, _ := theme.LoadPalette("dark")
	_ = NewEngine(nil, st, pal, func(ctx *Context, p Props) VNode {
		return VNode{Type: "Text", Props: Props{"content": "test"}}
	})
	// Engine created successfully
}

func TestEngineStoreRoundTrip(t *testing.T) {
	st := NewStore()
	messages := DefineAtom(st, "messages", []Message{})
	messages.Set([]Message{{UUID: "1", Type: "user"}})
	if len(messages.Get()) != 1 {
		t.Fatal("expected 1 message")
	}
}
```

- [ ] **Step 4: Compile everything and fix import issues**

Run: `cd goc && go build ./gou/ink/...`
Expected: compiles without errors

- [ ] **Step 5: Run all ink tests**

Run: `cd goc && go test ./gou/ink/... -v`
Expected: all tests pass

- [ ] **Step 6: Commit**

```bash
git add gou/ink/engine.go gou/ink/engine_test.go gou/ink/component.go
git commit -m "feat(ink): rewrite RenderEngine with 5-layer orchestration and new component.go
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 13: Component migration — Tier A (hook adoption)

**Files:**
- Modify: `gou/ink/compo/repl.go`
- Modify: `gou/ink/compo/messages.go`
- Modify: `gou/ink/compo/prompt.go`
- Modify: `gou/ink/compo/markdown.go`
- Modify: `gou/ink/compo/pipeline.go`

- [ ] **Step 1: Update REPL layout** — Replace `"ScrollBox"` with `"VirtualList"` type and add virtualScrollState prop:

```go
// In repl.go, change the ScrollBox child to VirtualList:
{
    Type: "VirtualList", Key: "messages-scroll",
    Props: ink.Props{
        "stickyBottom":   true,
        "flexGrow":       1,
        "virtualScroll":  vsState, // from context
    },
    Children: []ink.VNode{Messages(ctx, p)},
},
```

- [ ] **Step 2: Update Messages** — Use `UseMemo` for the message processing pipeline:

```go
func Messages(ctx *ink.Context, p ink.Props) ink.VNode {
    rawMsgs := /* ... from store ... */
    processed := ink.UseMemo(ctx, func() interface{} {
        return ProcessMessages(rawMsgs)
    }, []interface{}{len(rawMsgs), /* streaming text hash */})
    // ... render children
}
```

- [ ] **Step 3: Update PromptInput** — Read from atoms:

```go
func PromptInput(ctx *ink.Context) ink.VNode {
    val := /* ctx.Store.InputValue() */ 
    cursor := /* ctx.Store.CursorPos() */
    // ... unchanged rendering logic
}
```

- [ ] **Step 4: Update Markdown** — Use `UseMemo` for token cache:

```go
func Markdown(ctx *ink.Context, content string, width int) ink.VNode {
    result := ink.UseMemo(ctx, func() interface{} {
        return renderMarkdown(content, width)
    }, []interface{}{content, width})
    // ...
}
```

- [ ] **Step 5: Update Pipeline** — Convert `ProcessMessages` to work with the selector pattern:

No structural changes needed — the pipeline functions are pure. Just ensure they import from the right packages.

- [ ] **Step 6: Compile and test**

Run: `cd goc && go build ./gou/ink/compo/... && go test ./gou/ink/compo/... -v`
Expected: compiles, tests pass

- [ ] **Step 7: Commit**

```bash
git add gou/ink/compo/
git commit -m "feat(ink/compo): migrate Tier A components to hooks and VirtualList
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 14: Component migration — Tier B (Transcript + Search + Slash + Spinner + Scrollbar)

**Files:**
- Create: `gou/ink/compo/transcript_screen.go`
- Create: `gou/ink/compo/transcript_search.go`
- Create: `gou/ink/compo/slash_picker.go`
- Create: `gou/ink/compo/spinner.go`
- Create: `gou/ink/compo/scrollbar.go`

- [ ] **Step 1: Create TranscriptScreen component** — Port minimal freeze+render from `gou/app/transcript_screen.go`:

Create `gou/ink/compo/transcript_screen.go`:

```go
package compo

import "goc/gou/ink"

func TranscriptScreen(ctx *ink.Context, msgs []ink.Message, searchQuery string, showAll bool) ink.VNode {
	children := make([]ink.VNode, 0)
	matchCount := 0

	for _, msg := range msgs {
		row := MessageRow(ctx, msg)
		if searchQuery != "" {
			// Filter: only show messages containing query
			if !messageContains(msg, searchQuery) {
				continue
			}
			matchCount++
		}
		children = append(children, row)
	}

	header := ink.VNode{
		Type: "Text",
		Props: ink.Props{"content": "TRANSCRIPT — / search  Esc close  ctrl+e show all", "dim": true},
	}

	if searchQuery != "" {
		header.Props["content"] = "Search: " + searchQuery + " (" + itoa(matchCount) + " matches)"
	}

	return ink.VNode{
		Type: "Box", Key: "transcript-screen",
		Props: ink.Props{"direction": "column"},
		Children: append([]ink.VNode{header}, children...),
	}
}

func messageContains(msg ink.Message, query string) bool {
	for _, b := range msg.ContentBlocks {
		if containsString(b.Content, query) {
			return true
		}
		if containsString(b.Name, query) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Create TranscriptSearch component**

Create `gou/ink/compo/transcript_search.go`:

```go
package compo

import "goc/gou/ink"

func TranscriptSearchBar(ctx *ink.Context, query string) ink.VNode {
	return ink.VNode{
		Type: "Box", Key: "transcript-search",
		Props: ink.Props{"direction": "row"},
		Children: []ink.VNode{
			{Type: "Text", Props: ink.Props{"content": "Search: ", "dim": true}},
			{Type: "Text", Props: ink.Props{"content": query, "bold": true}},
		},
	}
}
```

- [ ] **Step 3: Create SlashPicker component**

Create `gou/ink/compo/slash_picker.go`:

```go
package compo

import "goc/gou/ink"

func SlashPicker(ctx *ink.Context, commands []string, filter string, selected int) ink.VNode {
	children := make([]ink.VNode, 0)
	for i, cmd := range commands {
		if filter != "" && !containsString(cmd, filter) {
			continue
		}
		style := ink.Props{"content": "/" + cmd}
		if i == selected {
			style["bold"] = true
			style["bg"] = ctx.Theme.Accent
		}
		children = append(children, ink.VNode{
			Type: "Text", Key: cmd, Props: style,
		})
	}
	return ink.VNode{
		Type: "Box", Key: "slash-picker",
		Props: ink.Props{"direction": "column"},
		Children: children,
	}
}
```

- [ ] **Step 4: Create Spinner component**

Create `gou/ink/compo/spinner.go`:

```go
package compo

import "goc/gou/ink"

func Spinner(ctx *ink.Context, isLoading bool, tip string) ink.VNode {
	if !isLoading { return ink.VNode{Type: "Text"} }
	return ink.VNode{
		Type: "Box", Key: "spinner",
		Props: ink.Props{"direction": "row"},
		Children: []ink.VNode{
			{Type: "Text", Props: ink.Props{"content": "⏳", "color": ctx.Theme.ToolUse}},
			{Type: "Text", Props: ink.Props{"content": " " + tip, "dim": true}},
		},
	}
}
```

- [ ] **Step 5: Create Scrollbar component**

Create `gou/ink/compo/scrollbar.go`:

```go
package compo

import "goc/gou/ink"

func Scrollbar(totalH, viewportH, scrollTop, width int) ink.VNode {
	if totalH <= viewportH {
		return ink.VNode{Type: "Text"}
	}
	thumbH := viewportH * viewportH / totalH
	if thumbH < 1 { thumbH = 1 }
	thumbY := scrollTop * viewportH / totalH

	var cells []string
	for i := 0; i < viewportH; i++ {
		if i >= thumbY && i < thumbY+thumbH {
			cells = append(cells, "█")
		} else {
			cells = append(cells, "░")
		}
	}

	return ink.VNode{
		Type: "Box", Key: "scrollbar",
		Props: ink.Props{"direction": "column", "width": width},
		Children: func() []ink.VNode {
			kids := make([]ink.VNode, len(cells))
			for i, c := range cells {
				kids[i] = ink.VNode{Type: "Text", Props: ink.Props{"content": c, "dim": true}}
			}
			return kids
		}(),
	}
}
```

- [ ] **Step 6: Add helpers**

Add to appropriate util file in compo/:

```go
func containsString(s, substr string) bool {
	import "strings"
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func itoa(n int) string {
	import "strconv"
	return strconv.Itoa(n)
}
```

- [ ] **Step 7: Compile and test**

Run: `cd goc && go build ./gou/ink/compo/...`
Expected: compiles without errors

- [ ] **Step 8: Commit**

```bash
git add gou/ink/compo/transcript_screen.go gou/ink/compo/transcript_search.go \
        gou/ink/compo/slash_picker.go gou/ink/compo/spinner.go gou/ink/compo/scrollbar.go
git commit -m "feat(ink/compo): add Tier B components (Transcript, Search, Slash, Spinner, Scrollbar)
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 15: Integration demo update — wire new engine to GOU_USE_NEW_ENGINE=1

**Files:**
- Modify: `gou/ink/integration/gou_demo.go`

- [ ] **Step 1: Rewrite integration demo to use new 5-layer engine**

Rewrite `gou/ink/integration/gou_demo.go`:

```go
package integration

import (
	"os"
	"goc/gou/ink"
)

func RunNewEngine() bool {
	if os.Getenv("GOU_USE_NEW_ENGINE") != "1" {
		return false
	}

	term := ink.NewTerminal()
	store := ink.NewStore()
	pal := loadPalette()

	// Define atoms
	messages := ink.DefineAtom(store, "messages", []ink.Message{})
	streamingText := ink.DefineAtom(store, "streamingText", "")
	inputValue := ink.DefineAtom(store, "inputValue", "")
	cursorPos := ink.DefineAtom(store, "cursorPos", 0)
	isLoading := ink.DefineAtom(store, "isLoading", false)
	scrollTop := ink.DefineAtom(store, "scrollTop", 0)
	termW := ink.DefineAtom(store, "termWidth", 80)
	termH := ink.DefineAtom(store, "termHeight", 24)

	_ = messages
	_ = streamingText
	_ = inputValue
	_ = cursorPos
	_ = isLoading
	_ = scrollTop
	_ = termW
	_ = termH

	engine := ink.NewEngine(term, store, pal, compo.REPL)

	if err := engine.Terminal.Init(); err != nil {
		panic(err)
	}
	defer engine.Terminal.Shutdown()

	runErr := engine.Run()
	if runErr != nil {
		panic(runErr)
	}
	return true
}

func loadPalette() *theme.Palette {
	pal := theme.ActivePalette()
	if pal == nil {
		pal = &theme.Palette{}
	}
	return pal
}
```

- [ ] **Step 2: Ensure existing `gou/app/main.go` imports still work**

Run: `cd goc && go build ./gou/app/...`
Expected: compiles (app uses Bubble Tea, should be unaffected by ink changes)

- [ ] **Step 3: Run full test suite for ink**

Run: `cd goc && go test ./gou/ink/... -v`
Expected: all tests pass

- [ ] **Step 4: Commit**

```bash
git add gou/ink/integration/gou_demo.go
git commit -m "feat(ink/integration): wire new 5-layer engine to GOU_USE_NEW_ENGINE=1 demo
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

### Task 16: Final integration test and cleanup

**Files:**
- No new files

- [ ] **Step 1: Full build test**

Run: `cd goc && go build ./...`
Expected: entire project compiles

- [ ] **Step 2: Full test suite**

Run: `cd goc && go test ./gou/ink/... ./gou/app/... -v`
Expected: all tests pass

- [ ] **Step 3: Smoke test the new engine (manual)**

Run: `cd goc && GOU_USE_NEW_ENGINE=1 go run ./cmd/claude`
Expected: TUI starts, shows welcome message, accepts text input, Enter shows message in list

- [ ] **Step 4: Verify no regressions in Bubble Tea path**

Run: `cd goc && go run ./cmd/claude` (without GOU_USE_NEW_ENGINE=1)
Expected: existing Bubble Tea TUI works as before

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore(ink): final integration test and cleanup
Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```
