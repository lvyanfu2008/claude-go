package localturn

import (
	"context"
	"encoding/json"

	"goc/ccb-engine/internal/engine"
)

// QueryBridgeParams is a data-only bundle for [QueryBridgeRun] (callers outside ccb-engine cannot reference [engine.ToolRunner]).
type QueryBridgeParams struct {
	RequestID string
	Messages  json.RawMessage
	Tools     json.RawMessage
	System    string
	Cwd       string
	ModelID   string

	SkillExpandUserFollowUp  bool
	FetchSystemPromptIfEmpty bool
	// Runner when non-nil must implement [engine.ToolRunner]; otherwise type is ignored and [StubRunner] is used.
	Runner any
}

func runnerFromAny(v any) engine.ToolRunner {
	if v == nil {
		return nil
	}
	r, ok := v.(engine.ToolRunner)
	if !ok {
		return nil
	}
	return r
}

// QueryBridgeRun is like [RunSubmitUserTurn] but accepts [QueryBridgeParams] and an optional [Runner] via [any].
func QueryBridgeRun(ctx context.Context, p QueryBridgeParams, emit func(StreamEvent)) error {
	pars := Params{
		RequestID:                p.RequestID,
		Messages:                 p.Messages,
		Tools:                    p.Tools,
		System:                   p.System,
		SkillExpandUserFollowUp:  p.SkillExpandUserFollowUp,
		FetchSystemPromptIfEmpty: p.FetchSystemPromptIfEmpty,
		Cwd:                      p.Cwd,
		ModelID:                  p.ModelID,
		Runner:                   runnerFromAny(p.Runner),
	}
	return RunSubmitUserTurn(ctx, pars, emit)
}
