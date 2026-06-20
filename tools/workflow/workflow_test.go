package workflow

import (
	"context"
	"strings"
	"testing"
	"time"
)

// --- Parser Tests ---

func TestParseMeta_ValidScript(t *testing.T) {
	script := `export const meta = {
		name: 'find-flaky-tests',
		description: 'Find flaky tests and propose fixes',
		phases: [
			{ title: 'Scan', detail: 'grep test logs for retries' },
			{ title: 'Fix', detail: 'one agent per flaky test' }
		]
	}`

	meta, err := ParseMeta(script)
	if err != nil {
		t.Fatalf("ParseMeta failed: %v", err)
	}
	if meta.Name != "find-flaky-tests" {
		t.Fatalf("expected name 'find-flaky-tests', got %q", meta.Name)
	}
	if meta.Description != "Find flaky tests and propose fixes" {
		t.Fatalf("expected description, got %q", meta.Description)
	}
	if len(meta.Phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(meta.Phases))
	}
	if meta.Phases[0].Title != "Scan" {
		t.Fatalf("expected phase 0 title 'Scan', got %q", meta.Phases[0].Title)
	}
}

func TestParseMeta_MissingMeta(t *testing.T) {
	script := `console.log("hello");`
	_, err := ParseMeta(script)
	if err == nil {
		t.Fatal("expected error for missing meta block")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected 'missing' in error, got: %v", err)
	}
}

func TestParseMeta_MissingName(t *testing.T) {
	script := `export const meta = {
		description: 'no name here'
	}`
	_, err := ParseMeta(script)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseMeta_MissingDescription(t *testing.T) {
	script := `export const meta = {
		name: 'no-description'
	}`
	_, err := ParseMeta(script)
	if err == nil {
		t.Fatal("expected error for missing description")
	}
}

func TestParseMeta_TemplateInterpolationRejected(t *testing.T) {
	script := "export const meta = {\n\t\tname: `foo-${bar}`,\n\t\tdescription: 'test'\n\t}"
	_, err := ParseMeta(script)
	if err == nil {
		t.Fatal("expected error for template interpolation")
	}
}

func TestParseMeta_SingleQuotedStrings(t *testing.T) {
	script := `export const meta = {
		name: 'single-quoted',
		description: 'also single quoted'
	}`
	meta, err := ParseMeta(script)
	if err != nil {
		t.Fatalf("ParseMeta failed: %v", err)
	}
	if meta.Name != "single-quoted" {
		t.Fatalf("expected name 'single-quoted', got %q", meta.Name)
	}
}

func TestStripExports(t *testing.T) {
	script := `export const meta = { name: 'test', description: 'd' };
export const x = 1;
export function foo() {}
export async function bar() {}`
	result := StripExports(script)
	if strings.Contains(result, "export const ") {
		t.Fatal("export const not stripped")
	}
	if strings.Contains(result, "export function ") {
		t.Fatal("export function not stripped")
	}
	if !strings.Contains(result, "const meta =") {
		t.Fatal("const meta = should remain")
	}
	if !strings.Contains(result, "const x = 1") {
		t.Fatal("const x = 1 should remain")
	}
	if !strings.Contains(result, "async function bar()") {
		t.Fatal("async function bar() should remain")
	}
}

// --- Budget Tests ---

func TestBudgetTracker_NoBudget(t *testing.T) {
	b := NewBudgetTracker(0)
	if b.IsExhausted() {
		t.Fatal("budget with total=0 should not be exhausted")
	}
	if b.Remaining() < 1000000 {
		t.Fatal("budget with total=0 should have large remaining")
	}
	b.AddSpent(1000)
	if b.Spent() != 1000 {
		t.Fatalf("expected spent=1000, got %d", b.Spent())
	}
	if b.IsExhausted() {
		t.Fatal("budget with total=0 should never be exhausted")
	}
}

func TestBudgetTracker_WithBudget(t *testing.T) {
	b := NewBudgetTracker(5000)
	if b.IsExhausted() {
		t.Fatal("fresh budget should not be exhausted")
	}
	if b.Spent() != 0 {
		t.Fatalf("expected spent=0, got %d", b.Spent())
	}
	b.AddSpent(3000)
	if b.Spent() != 3000 {
		t.Fatalf("expected spent=3000, got %d", b.Spent())
	}
	if b.Remaining() != 2000 {
		t.Fatalf("expected remaining=2000, got %d", b.Remaining())
	}
	b.AddSpent(2000)
	if !b.IsExhausted() {
		t.Fatal("budget should be exhausted after spending total")
	}
	if b.Remaining() != 0 {
		t.Fatalf("expected remaining=0, got %d", b.Remaining())
	}
}

// --- Journal / Resume Tests ---

func TestJournal_RecordAndLookup(t *testing.T) {
	j := NewJournal("wf_test001")
	hash := HashAgentCall("find bugs", &AgentOpts{Label: "bug-finder"})
	if j.Lookup(hash) != nil {
		t.Fatal("empty journal should return nil lookup")
	}
	j.Record(hash, []byte(`"found 3 bugs"`))
	entry := j.Lookup(hash)
	if entry == nil {
		t.Fatal("expected to find recorded entry")
	}
	if entry.Hash != hash {
		t.Fatalf("hash mismatch: %q vs %q", entry.Hash, hash)
	}
}

func TestHashAgentCall_Deterministic(t *testing.T) {
	opts := &AgentOpts{Label: "test", Phase: "verify"}
	h1 := HashAgentCall("find bugs", opts)
	h2 := HashAgentCall("find bugs", opts)
	if h1 != h2 {
		t.Fatalf("hashes should be deterministic: %q vs %q", h1, h2)
	}
	h3 := HashAgentCall("find bugs", nil)
	h4 := HashAgentCall("find bugs", &AgentOpts{})
	if h3 == h4 {
		t.Fatal("nil opts vs empty opts should produce different hashes")
	}
}

// --- Engine Tests ---

func TestEngine_SimpleSyncScript(t *testing.T) {
	script := `export const meta = {
		name: 'simple-test',
		description: 'A simple synchronous workflow'
	};

	// No async calls — returns a plain value
	return "hello from workflow";
	`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, "hello from workflow") {
		t.Fatalf("unexpected result: %s", result)
	}
}

func TestEngine_PhaseAndLog(t *testing.T) {
	script := `export const meta = {
		name: 'phase-test',
		description: 'Test phase and log functions'
	};

	phase('scanning');
	log('found 3 files');
	return "done";
	`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, "done") {
		t.Fatalf("expected 'done' in result, got: %s", result)
	}
}

func TestEngine_MissingMeta(t *testing.T) {
	script := `console.log("no meta here");`
	engine := NewEngine()
	_, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err == nil {
		t.Fatal("expected error for missing meta")
	}
}

func TestEngine_ScriptWithArgs(t *testing.T) {
	script := `export const meta = {
		name: 'args-test',
		description: 'Test args access'
	};

	// args is available as a global (undefined when not provided)
	return 'args global: ' + (typeof args);
	`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, "undefined") {
		t.Fatalf("expected args to be 'undefined', got: %s", result)
	}
}

func TestEngine_ScriptError(t *testing.T) {
	script := `export const meta = {
		name: 'error-test',
		description: 'Test script error handling'
	};

	throw new Error('intentional error');
	`

	engine := NewEngine()
	_, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err == nil {
		t.Fatal("expected error from script throw")
	}
	if !strings.Contains(err.Error(), "intentional error") {
		t.Fatalf("expected 'intentional error' in error message, got: %v", err)
	}
}

func TestEngine_ContextCancellation(t *testing.T) {
	script := `export const meta = {
		name: 'cancel-test',
		description: 'Test context cancellation'
	};

	// Create a never-resolving promise to hang the event loop
	await new Promise(() => {});
	`

	engine := NewEngine()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := engine.Execute(ctx, script, EngineConfig{})
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func TestEngine_MathRandom_ReturnsNumber(t *testing.T) {
	script := `export const meta = {
		name: 'sandbox-test',
		description: 'Math.random returns a number'
	};

	return typeof Math.random();
	`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Math.random is available (not sandboxed — prompt documents it as unavailable)
	if !strings.Contains(result, "number") {
		t.Fatalf("expected 'number' result, got: %s", result)
	}
}

func TestEngine_DateNow_ReturnsNumber(t *testing.T) {
	script := `export const meta = {
		name: 'sandbox-test-2',
		description: 'Date.now returns a number'
	};

	return typeof Date.now();
	`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Date.now is available (not sandboxed — prompt documents it as unavailable)
	if !strings.Contains(result, "number") {
		t.Fatalf("expected 'number' result, got: %s", result)
	}
}

func TestRunState_ConcurrencySlots(t *testing.T) {
	state := NewRunState("wf_test", Meta{Name: "test", Description: "test"}, nil, nil, nil)

	// Acquire all slots
	for i := 0; i < 16; i++ {
		if !state.AcquireSlot() {
			t.Fatalf("should have acquired slot %d", i)
		}
	}

	// 17th slot should block (test via non-blocking attempt)
	// Release half
	for i := 0; i < 8; i++ {
		state.ReleaseSlot()
	}

	// Should be able to acquire again
	for i := 0; i < 8; i++ {
		if !state.AcquireSlot() {
			t.Fatalf("should have acquired slot after release %d", i)
		}
	}
}

func TestRunState_Abort(t *testing.T) {
	state := NewRunState("wf_test", Meta{Name: "test", Description: "test"}, nil, nil, nil)
	state.Abort()

	// After abort, AcquireSlot should return false
	if state.AcquireSlot() {
		t.Fatal("AcquireSlot should return false after abort")
	}
}

// --- Parallel/Pipeline JS Tests ---

func TestParallel_NilArray(t *testing.T) {
	script := `export const meta = {
		name: 'parallel-test',
		description: 'Test parallel with empty array'
	};

	const results = await parallel([]);
	return JSON.stringify(results);
	`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, "[]") {
		t.Fatalf("expected empty array result, got: %s", result)
	}
}

func TestPipeline_NoStages(t *testing.T) {
	script := `export const meta = {
		name: 'pipeline-test',
		description: 'Test pipeline with no stages'
	};

	const results = await pipeline(['a', 'b', 'c']);
	return JSON.stringify(results);
	`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Pipeline with no stages just returns the items
	if !strings.Contains(result, "a") || !strings.Contains(result, "b") || !strings.Contains(result, "c") {
		t.Fatalf("expected items in result, got: %s", result)
	}
}

func TestWorkflow_NestedNotImplemented(t *testing.T) {
	script := `export const meta = {
		name: 'nested-test',
		description: 'Test nested workflow error'
	};

	try {
		await workflow('some-workflow');
		return 'should not reach';
	} catch (e) {
		return e.message;
	}
	`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, "not yet implemented") && !strings.Contains(result, "Nested") {
		t.Fatalf("expected 'not yet implemented' in result, got: %s", result)
	}
}

// --- Budget JS Tests ---

func TestBudget_InJS(t *testing.T) {
	script := `export const meta = {
		name: 'budget-test',
		description: 'Test budget global in JS'
	};

	return (typeof budget !== 'undefined' ? 'budget available' : 'no budget');
	`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, "budget available") {
		t.Fatalf("expected budget global, got: %s", result)
	}
}

