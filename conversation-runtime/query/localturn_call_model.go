package query

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"goc/ccb-engine/localturn"
	"goc/gou/ccbhydrate"
	"goc/messagesapi"
)

// LocalTurnCallModelConfig configures [LocalTurnCallModel] (goc/ccb-engine/localturn via [localturn.QueryBridgeRun]).
type LocalTurnCallModelConfig struct {
	// Runner when non-nil must satisfy ccb-engine's [engine.ToolRunner] (e.g. [engine.StubRunner] from inside ccb-engine tests).
	Runner                   any
	SkillExpandUserFollowUp  bool
	FetchSystemPromptIfEmpty bool
	// MessagesNormalize when non-nil overrides [messagesapi.DefaultOptions] for [ccbhydrate.MessagesJSONNormalized].
	MessagesNormalize *messagesapi.Options
}

func normalizeOpts(cfg LocalTurnCallModelConfig) messagesapi.Options {
	if cfg.MessagesNormalize != nil {
		return *cfg.MessagesNormalize
	}
	return messagesapi.DefaultOptions()
}

// buildLocalTurnParams maps [CallModelInput] to [localturn.QueryBridgeParams] (messages JSON + system + tools).
func buildLocalTurnParams(in *CallModelInput, cfg LocalTurnCallModelConfig, requestID string) (localturn.QueryBridgeParams, error) {
	if in == nil {
		return localturn.QueryBridgeParams{}, fmt.Errorf("query: nil CallModelInput")
	}
	msgsJSON, err := ccbhydrate.MessagesJSONNormalized(in.Messages, nil, normalizeOpts(cfg))
	if err != nil {
		return localturn.QueryBridgeParams{}, err
	}
	system := strings.Join([]string(in.SystemPrompt), "\n\n")
	return localturn.QueryBridgeParams{
		RequestID:                requestID,
		Messages:                 msgsJSON,
		Tools:                    in.Tools,
		System:                   system,
		SkillExpandUserFollowUp:  cfg.SkillExpandUserFollowUp,
		FetchSystemPromptIfEmpty: cfg.FetchSystemPromptIfEmpty,
		Cwd:                      in.Cwd,
		ModelID:                  in.ModelID,
		Runner:                   cfg.Runner,
	}, nil
}

// LocalTurnCallModel returns a [QueryDeps.CallModel] implementation that streams
// [localturn.StreamEvent] as JSON in [QueryYield.StreamEvent] (protocol-v1 shape).
func LocalTurnCallModel(cfg LocalTurnCallModelConfig) func(ctx context.Context, in *CallModelInput, emit func(QueryYield) bool) error {
	return func(ctx context.Context, in *CallModelInput, emit func(QueryYield) bool) error {
		rid := "query-" + randomUUID()
		p, err := buildLocalTurnParams(in, cfg, rid)
		if err != nil {
			return err
		}
		return localturn.QueryBridgeRun(ctx, p, func(ev localturn.StreamEvent) {
			raw, err := json.Marshal(ev)
			if err != nil {
				return
			}
			emit(QueryYield{StreamEvent: raw})
		})
	}
}

// ProductionDepsWithLocalTurn returns [ProductionDeps] with [CallModel] set to [LocalTurnCallModel].
func ProductionDepsWithLocalTurn(cfg LocalTurnCallModelConfig) QueryDeps {
	d := ProductionDeps()
	d.CallModel = LocalTurnCallModel(cfg)
	return d
}
