package processuserinput

import (
	"context"
	"encoding/json"
	"iter"
	"testing"

	"goc/types"
)

func TestProcessUserInput_hooksIterPreferredOverBatch(t *testing.T) {
	raw, _ := json.Marshal("hello")
	skip := true
	u := "u-hooks-iter-1"
	p := &ProcessUserInputParams{
		Input:           raw,
		Mode:            types.PromptInputModePrompt,
		SkipAttachments: &skip,
		Messages:        nil,
		UUID:            &u,
		PermissionMode:  types.PermissionDefault,
		RuntimeContext:  minimalRuntimeContext(),
	}
	var batchCalled bool
	p.ExecuteUserPromptSubmitHooks = func(context.Context, *ProcessUserInputParams, string) ([]types.AggregatedHookResult, error) {
		batchCalled = true
		return nil, nil
	}
	var pullCount int
	p.ExecuteUserPromptSubmitHooksIter = func(context.Context, *ProcessUserInputParams, string) iter.Seq2[types.AggregatedHookResult, error] {
		return func(yield func(types.AggregatedHookResult, error) bool) {
			pullCount++
			if !yield(types.AggregatedHookResult{}, nil) {
				return
			}
		}
	}
	r, err := ProcessUserInput(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if batchCalled {
		t.Fatal("batch hooks should be skipped when Iter is set")
	}
	if pullCount != 1 {
		t.Fatalf("iter pulls=%d want 1", pullCount)
	}
	if r == nil || !r.ShouldQuery {
		t.Fatalf("expected shouldQuery true, got %+v", r)
	}
}

func TestProcessUserInput_hooksIterBlockingAfterFirst(t *testing.T) {
	raw, _ := json.Marshal("hello")
	skip := true
	u := "u-hooks-iter-2"
	block := types.HookBlockingError{BlockingError: "nope"}
	var stage int
	p := &ProcessUserInputParams{
		Input:           raw,
		Mode:            types.PromptInputModePrompt,
		SkipAttachments: &skip,
		Messages:        nil,
		UUID:            &u,
		PermissionMode:  types.PermissionDefault,
		RuntimeContext:  minimalRuntimeContext(),
		ExecuteUserPromptSubmitHooksIter: func(context.Context, *ProcessUserInputParams, string) iter.Seq2[types.AggregatedHookResult, error] {
			return func(yield func(types.AggregatedHookResult, error) bool) {
				stage++
				if !yield(types.AggregatedHookResult{}, nil) {
					return
				}
				stage++
				if !yield(types.AggregatedHookResult{BlockingError: &block}, nil) {
					return
				}
				stage++
			}
		},
	}
	r, err := ProcessUserInput(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if stage != 2 {
		t.Fatalf("iterator stages=%d want 2 (third yield must not run)", stage)
	}
	if r == nil || r.ShouldQuery {
		t.Fatal("expected blocking hook to set shouldQuery false")
	}
	if len(r.Messages) != 1 {
		t.Fatalf("messages=%d", len(r.Messages))
	}
}

func minimalRuntimeContext() *types.ProcessUserInputContextData {
	return &types.ProcessUserInputContextData{
		ToolUseContext: types.ToolUseContext{
			Options: types.ToolUseContextOptionsData{
				Commands:      []types.Command{},
				MainLoopModel: "claude-sonnet-4-20250514",
			},
		},
	}
}
