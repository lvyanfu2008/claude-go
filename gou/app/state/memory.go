package state

import "context"

type Memory struct {
	AutoDream     interface{}
	ExtractMem    interface{}
	SessionMem    interface{}
	SessionHook   func(ctx context.Context, params interface{})
	LastGuidance  string
	LastUserCtx   map[string]string
	LastSystemCtx map[string]string
}
