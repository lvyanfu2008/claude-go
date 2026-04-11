package query

import (
	"context"
	"encoding/json"
	"testing"

	"goc/types"
)

func TestRunApplyToolResultBudget_defaultPassthrough(t *testing.T) {
	deps := &QueryDeps{}
	in := &ToolResultBudgetInput{Messages: []types.Message{{UUID: "x"}}}
	got, err := runApplyToolResultBudget(context.Background(), deps, in)
	if err != nil || len(got) != 1 || got[0].UUID != "x" {
		t.Fatalf("got %#v err %v", got, err)
	}
}

func TestRunApplyToolResultBudget_defaultReapply(t *testing.T) {
	deps := &QueryDeps{}
	inner, _ := json.Marshal(map[string]any{
		"role":    "user",
		"content": []any{map[string]any{"type": "tool_result", "tool_use_id": "t1", "content": "BIG"}},
	})
	ms := []types.Message{{Type: types.MessageTypeUser, UUID: "u", Message: inner}}
	state := json.RawMessage(`{"replacements":{"t1":"small"}}`)
	in := &ToolResultBudgetInput{Messages: ms, ContentReplacementState: state}
	got, err := runApplyToolResultBudget(context.Background(), deps, in)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %#v err %v", got, err)
	}
	var env struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(got[0].Message, &env); err != nil {
		t.Fatal(err)
	}
	var blocks []map[string]any
	if err := json.Unmarshal(env.Content, &blocks); err != nil {
		t.Fatal(err)
	}
	if blocks[0]["content"] != "small" {
		t.Fatalf("%#v", blocks[0])
	}
}

func TestRunApplyToolResultBudget_customDep(t *testing.T) {
	deps := &QueryDeps{
		ApplyToolResultBudget: func(ctx context.Context, in *ToolResultBudgetInput) ([]types.Message, error) {
			return []types.Message{{UUID: "custom"}}, nil
		},
	}
	in := &ToolResultBudgetInput{Messages: []types.Message{{UUID: "orig"}}}
	got, err := runApplyToolResultBudget(context.Background(), deps, in)
	if err != nil || len(got) != 1 || got[0].UUID != "custom" {
		t.Fatalf("got %#v err %v", got, err)
	}
}

func TestRunSnipCompact_noopWhenNil(t *testing.T) {
	deps := &QueryDeps{}
	got, err := runSnipCompact(context.Background(), deps, &SnipCompactInput{Messages: []types.Message{{UUID: "s"}}})
	if err != nil || got != nil {
		t.Fatalf("got %#v err %v", got, err)
	}
}

func TestRunMicrocompact_noopWhenNil(t *testing.T) {
	deps := &QueryDeps{}
	in := &MicrocompactInput{Messages: []types.Message{{UUID: "1"}}}
	got, err := runMicrocompact(context.Background(), deps, in)
	if err != nil || len(got) != 1 || got[0].UUID != "1" {
		t.Fatalf("got %#v err %v", got, err)
	}
}

func TestRunMicrocompact_invokesDep(t *testing.T) {
	deps := &QueryDeps{
		Microcompact: func(ctx context.Context, in *MicrocompactInput) (*MicrocompactResult, error) {
			return &MicrocompactResult{Messages: []types.Message{{UUID: "mc"}}}, nil
		},
	}
	in := &MicrocompactInput{Messages: []types.Message{{UUID: "1"}}}
	got, err := runMicrocompact(context.Background(), deps, in)
	if err != nil || len(got) != 1 || got[0].UUID != "mc" {
		t.Fatalf("got %#v err %v", got, err)
	}
}

func TestRunMicrocompact_nilResultUsesInput(t *testing.T) {
	deps := &QueryDeps{
		Microcompact: func(ctx context.Context, in *MicrocompactInput) (*MicrocompactResult, error) {
			return nil, nil
		},
	}
	in := &MicrocompactInput{Messages: []types.Message{{UUID: "keep"}}}
	got, err := runMicrocompact(context.Background(), deps, in)
	if err != nil || len(got) != 1 || got[0].UUID != "keep" {
		t.Fatalf("got %#v err %v", got, err)
	}
}

func TestRunAutocompact_replacesWhenPostMessages(t *testing.T) {
	deps := &QueryDeps{
		Autocompact: func(ctx context.Context, in *AutocompactInput) (*AutocompactResult, error) {
			if in.CacheSafe.ForkContextMessages == nil {
				t.Fatal("expected fork messages")
			}
			return &AutocompactResult{
				WasCompacted: true,
				PostMessages: []types.Message{{UUID: "post"}},
			}, nil
		},
	}
	in := &AutocompactInput{
		Messages:       []types.Message{{UUID: "pre"}},
		ToolUseContext: &types.ToolUseContext{},
		CacheSafe: CacheSafeParams{
			ForkContextMessages: []types.Message{{UUID: "pre"}},
		},
	}
	got, res, err := runAutocompact(context.Background(), deps, in)
	if err != nil || len(got) != 1 || got[0].UUID != "post" || res == nil || !res.WasCompacted {
		t.Fatalf("got %#v res %#v err %v", got, res, err)
	}
}

func TestRunAutocompact_returnsResultWhenNotCompacted(t *testing.T) {
	deps := &QueryDeps{
		Autocompact: func(ctx context.Context, in *AutocompactInput) (*AutocompactResult, error) {
			return &AutocompactResult{
				WasCompacted:        false,
				ConsecutiveFailures: 2,
				UpdatedTracking:     json.RawMessage(`{"cf":2}`),
			}, nil
		},
	}
	in := &AutocompactInput{Messages: []types.Message{{UUID: "x"}}}
	got, res, err := runAutocompact(context.Background(), deps, in)
	if err != nil || len(got) != 1 || got[0].UUID != "x" {
		t.Fatalf("messages %#v err %v", got, err)
	}
	if res == nil || res.ConsecutiveFailures != 2 {
		t.Fatalf("res %#v", res)
	}
}

func TestApplyAutocompactSideEffects(t *testing.T) {
	st := State{ToolUseContext: types.ToolUseContext{}}
	applyAutocompactSideEffects(&st, &AutocompactResult{
		UpdatedTracking:                json.RawMessage(`{"turnCounter":3}`),
		UpdatedContentReplacementState: json.RawMessage(`{"replacements":{"a":"b"}}`),
	})
	if string(st.AutoCompactTracking) != `{"turnCounter":3}` {
		t.Fatalf("tracking %s", st.AutoCompactTracking)
	}
	if string(st.ToolUseContext.ContentReplacementState) != `{"replacements":{"a":"b"}}` {
		t.Fatalf("crs %s", st.ToolUseContext.ContentReplacementState)
	}
	applyAutocompactSideEffects(&st, nil)
}
