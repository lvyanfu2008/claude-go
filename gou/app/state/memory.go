package state

import (
	"context"

	"goc/conversation-runtime/query"
	"goc/services/autodream"
	"goc/services/extractmemories"
	"goc/services/sessionmemory"
)

type Memory struct {
	AutoDream   *autodream.State
	ExtractMem  *extractmemories.State
	SessionMem  *sessionmemory.State
	SessionHook func(ctx context.Context, params query.QueryCompleteParams)
	LastGuidance  string
	LastUserCtx   map[string]string
	LastSystemCtx map[string]string
}
