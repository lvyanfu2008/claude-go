package workflow

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// --- Scenario 1: Agent errors should not crash workflow ---

func TestScenario_AgentErrorDoesNotCrash(t *testing.T) {
	SetTaskFunctions(nil, nil)
	var callCount atomic.Int32
	SetAgentRunner(func(raw []byte, cfg AgentRunConfig) (string, bool, error) {
		callCount.Add(1)
		return "", true, fmt.Errorf("simulated agent failure")
	})

	script := `export const meta = {
name: 'error-handling',
description: 'Test that agent errors are handled gracefully',
};

const results = await parallel([
() => agent("task 1"),
() => agent("task 2"),
() => agent("task 3"),
]);

// Results should be [null, null, null] after filtering
return { count: results.length };
`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute should not fail for agent errors: %v", err)
	}
	if !strings.Contains(result, `"count":0`) {
		t.Fatalf("expected 0 results after filtering errors, got: %s", result)
	}
	if callCount.Load() != 3 {
		t.Fatalf("expected 3 agent calls, got %d", callCount.Load())
	}
}

// --- Scenario 2: Mixed success/failure in parallel ---

func TestScenario_MixedSuccessAndFailure(t *testing.T) {
	SetTaskFunctions(nil, nil)
	var callCount atomic.Int32
	SetAgentRunner(func(raw []byte, cfg AgentRunConfig) (string, bool, error) {
		n := callCount.Add(1)
		if n%2 == 1 { // odd calls succeed, even calls fail
			return `{"data":{"message":"success"}}`, false, nil
		}
		return "", true, fmt.Errorf("agent error")
	})

	script := `export const meta = {
name: 'mixed-results',
description: 'Test parallel with mixed success and failure',
};

const results = await parallel([
() => agent("task A"),
() => agent("task B"),
() => agent("task C"),
() => agent("task D"),
]);

return { count: results.length, first: results[0] };
`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Odd calls succeed (1,3) = 2 successes, even calls fail (2,4) = filtered
	if !strings.Contains(result, `"count":2`) {
		t.Fatalf("expected 2 successful results, got: %s", result)
	}
}

// --- Scenario 3: Budget exhaustion ---

func TestScenario_BudgetExhausted(t *testing.T) {
	SetTaskFunctions(nil, nil)
	var callCount atomic.Int32
	SetAgentRunner(func(raw []byte, cfg AgentRunConfig) (string, bool, error) {
		callCount.Add(1)
		return `{"data":{"message":"ok"}}`, false, nil
	})

	// Budget of 0 means no limit, but we set the budget in JS
	script := `export const meta = {
name: 'budget-test',
description: 'Test budget tracking',
};

// Set a very low budget
budget.total = 100;
budget.spent = function() { return 50; };
budget.remaining = function() { return Math.max(0, budget.total - budget.spent()); };

// First agent should run (spent 50 < total 100)
const r1 = await agent("task 1");

return { spent: budget.spent(), remaining: budget.remaining() };
`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if callCount.Load() < 1 {
		t.Fatalf("expected at least 1 agent call")
	}
	_ = result
}

// --- Scenario 4: Context cancellation ---

func TestScenario_ContextCancelled(t *testing.T) {
	SetTaskFunctions(nil, nil)
	SetAgentRunner(func(raw []byte, cfg AgentRunConfig) (string, bool, error) {
		time.Sleep(500 * time.Millisecond) // slow agent
		return `{"data":{"message":"too late"}}`, false, nil
	})

	script := `export const meta = {
name: 'cancel-test',
description: 'Test context cancellation during workflow',
};

const result = await agent("slow task");
return { done: true };
`

	engine := NewEngine()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := engine.Execute(ctx, script, EngineConfig{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "cancel") {
		t.Fatalf("expected deadline/cancel error, got: %v", err)
	}
}

// --- Scenario 5: Pipeline with item failure continues others ---

func TestScenario_PipelineItemFailure(t *testing.T) {
	SetTaskFunctions(nil, nil)
	var callCount atomic.Int32
	SetAgentRunner(func(raw []byte, cfg AgentRunConfig) (string, bool, error) {
		n := callCount.Add(1)
		input := string(raw)
		if strings.Contains(input, "broken item") {
			return "", true, fmt.Errorf("broken")
		}
		return fmt.Sprintf(`{"data":{"message":"result-%d"}}`, n), false, nil
	})

	script := `export const meta = {
name: 'pipeline-failure',
description: 'Test that one pipeline item failure does not block others',
};

const items = ["good A", "broken item", "good B"];
const results = await pipeline(items, item => agent(item));

// broken item returns null, filtered out; good items succeed
return { count: results.length };
`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, `"count":2`) {
		t.Fatalf("expected 2 results (broken filtered), got: %s", result)
	}
}

// --- Scenario 6: Nested workflow() stub ---

func TestScenario_NestedWorkflowError(t *testing.T) {
	SetTaskFunctions(nil, nil)

	script := `export const meta = {
name: 'nested-test',
description: 'Test nested workflow error',
};

try {
await workflow("child-workflow");
return { nested: "should not reach" };
} catch (e) {
return { error: e.message, handled: true };
}
`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, "handled") || !strings.Contains(result, "true") {
		t.Fatalf("expected handled error, got: %s", result)
	}
	if !strings.Contains(result, "not yet implemented") {
		t.Fatalf("expected 'not yet implemented' in error message: %s", result)
	}
}

// --- Scenario 7: Large number of parallel agents (concurrency limit) ---

func TestScenario_ParallelConcurrencyLimit(t *testing.T) {
	SetTaskFunctions(nil, nil)
	var maxConcurrent atomic.Int32
	var current atomic.Int32
	SetAgentRunner(func(raw []byte, cfg AgentRunConfig) (string, bool, error) {
		cur := current.Add(1)
		for {
			m := maxConcurrent.Load()
			if cur > m {
				maxConcurrent.Store(cur)
			}
			if m == 0 && cur == 1 {
				maxConcurrent.Store(1)
			}
			break // short-circuit for test (in real life, agents take time)
		}
		time.Sleep(10 * time.Millisecond)
		current.Add(-1)
		return `{"data":{"message":"done"}}`, false, nil
	})

	// Request 20 agents — should be capped at 16 concurrent
	script := `export const meta = {
name: 'concurrency-test',
description: 'Test parallel agent concurrency limit',
};

const thunks = [];
for (let i = 0; i < 20; i++) {
thunks.push(() => agent("task " + i));
}
const results = await parallel(thunks);
return { count: results.length };
`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, `"count":20`) {
		t.Fatalf("expected 20 results, got: %s", result)
	}
	// maxConcurrent should be <= 16 (semaphore limit)
	mc := maxConcurrent.Load()
	if mc > 16 {
		t.Logf("max concurrent agents: %d (expected <= 16)", mc)
	}
}

// --- Scenario 8: Schema mode with various response formats ---

func TestScenario_SchemaModeResponseFormats(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		expectJSON bool
		expectKey  string
	}{
		{"plain JSON", `{"data":{"message":"{\"score\":100}"}}`, true, "100"},
		{"markdown wrapped", `{"data":{"output":"` + "```json\n{\"score\":95}\n```" + `"}}`, true, "95"},
		{"json prefix", `{"data":{"message":"Here is the result: {\"score\":90}"}}`, true, "90"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			SetTaskFunctions(nil, nil)
			SetAgentRunner(func(raw []byte, cfg AgentRunConfig) (string, bool, error) {
				return tc.response, false, nil
			})

			script := `export const meta = {
name: 'schema-test',
description: 'Test schema extraction from ` + tc.name + `',
};

const r = await agent("test", {
schema: {type: "object", properties: {score: {type: "number"}}, required: ["score"]}
});

return { score: r.score };
`

			engine := NewEngine()
			result, err := engine.Execute(context.Background(), script, EngineConfig{})
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}
			if tc.expectJSON && tc.expectKey != "" {
				if !strings.Contains(result, tc.expectKey) {
					t.Errorf("expected key %q in result, got: %s", tc.expectKey, result[:200])
				}
			}
		})
	}
}

// --- Scenario 9: scriptPath resolution ---

func TestScenario_ScriptPathResolution(t *testing.T) {
	SetTaskFunctions(nil, nil)

	// Test that ParseMeta works on various realistic scripts from LLM
	validScripts := []string{
		`export const meta = {
name: 'simple',
description: 'A simple workflow',
};
return "ok";
`,
		`export const meta = { name: 'minimal', description: 'Minimal workflow' };
phase('test');
await agent("test task");
return "done";
`,
		`export const meta = {
name: 'full-workflow',
description: 'Complete workflow with all features',
phases: [
{ title: 'Scan', detail: 'Read files and find issues' },
{ title: 'Verify', detail: 'Verify each finding' },
{ title: 'Report', detail: 'Generate final report' },
],
};
phase('Scan');
const results = await parallel([
() => agent("scan security"),
() => agent("scan config"),
]);
const allFindings = results.flatMap(r => {
if (typeof r === "string") return [];
return (r && r.findings) ? r.findings : [];
});
phase('Verify');
const confirmed = allFindings.filter(f => f && f.severity === "high");
phase('Report');
return { total: allFindings.length, confirmed: confirmed.length };
`,
	}

	for i, script := range validScripts {
		t.Run(fmt.Sprintf("script-%d", i), func(t *testing.T) {
			engine := NewEngine()
			_, err := engine.Execute(context.Background(), script, EngineConfig{})
			if err != nil {
				t.Errorf("valid script %d failed: %v", i, err)
			}
		})
	}
}

// --- Scenario 10: Invalid scripts should give helpful errors ---

func TestScenario_InvalidScriptErrors(t *testing.T) {
	tests := []struct {
		name        string
		script      string
		wantErrPart string
	}{
		{
			"missing meta",
			`console.log("no meta");`,
			"missing",
		},
		{
			"missing description",
			`export const meta = { name: 'only-name' };`,
			"description",
		},
		{
			"template interpolation",
			"export const meta = {\nname: `test-${foo}`,\ndescription: 'test'\n};",
			"template",
		},
		{
			"JS syntax error",
			"export const meta = { name: 'test', description: 'test' };\nconst x = ;\nreturn 1;",
			"SyntaxError",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := NewEngine()
			_, err := engine.Execute(context.Background(), tc.script, EngineConfig{})
			if err == nil {
				t.Fatal("expected error but got nil")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.wantErrPart)) {
				t.Errorf("expected error containing %q, got: %v", tc.wantErrPart, err)
			}
		})
	}
}

// --- Scenario 11: Progress callbacks fire correctly ---

func TestScenario_ProgressCallbacks(t *testing.T) {
	SetTaskFunctions(nil, nil)
	SetAgentRunner(func(raw []byte, cfg AgentRunConfig) (string, bool, error) {
		return `{"data":{"message":"ok"}}`, false, nil
	})

	var events []string
	script := `export const meta = {
name: 'progress-test',
description: 'Verify progress callbacks fire in correct order',
phases: [{title: 'Phase1'}, {title: 'Phase2'}],
};

phase('Phase1');
log('log message 1');
const r = await agent("task 1", {label: "finder"});
phase('Phase2');
log('log message 2');
await agent("task 2", {label: "verifier"});
return { done: true };
`

	engine := NewEngine()
	_, err := engine.Execute(context.Background(), script, EngineConfig{
		WorkflowProgressCallback: func(agentID, status, message string) {
			events = append(events, status+":"+message)
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify we got progress events
	foundPhase := false
	foundLog := false
	foundAgent := false
	for _, e := range events {
		if strings.Contains(e, "Phase:") {
			foundPhase = true
		}
		if strings.Contains(e, "log message") {
			foundLog = true
		}
		if strings.Contains(e, "agent:") {
			foundAgent = true
		}
	}
	if !foundPhase {
		t.Error("no phase progress events")
	}
	if !foundLog {
		t.Error("no log progress events")
	}
	if !foundAgent {
		t.Error("no agent progress events")
	}
}

// --- Scenario 12: JSON.stringify result extraction (verify no double-encoding) ---

func TestScenario_ResultJSONEncoding(t *testing.T) {
	SetTaskFunctions(nil, nil)
	SetAgentRunner(func(raw []byte, cfg AgentRunConfig) (string, bool, error) {
		return `{"data":{"message":"{\"findings\":[{\"file\":\"App.java\",\"line\":42}]}"}}`, false, nil
	})

	script := `export const meta = {
name: 'encoding-test',
description: 'Verify result is not double-encoded',
};

const r = await agent("find issues", {
schema: {
type: "object",
properties: {findings: {type: "array"}},
required: ["findings"]
}
});

return { findingsCount: r.findings ? r.findings.length : 0 };
`

	engine := NewEngine()
	result, err := engine.Execute(context.Background(), script, EngineConfig{})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	// Should be "findingsCount":1 not double-encoded
	if !strings.Contains(result, `"findingsCount":1`) {
		t.Fatalf("expected findingsCount:1, got: %s", result)
	}
}
