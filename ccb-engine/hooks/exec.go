package hooks

import (
	"context"
	"time"

	internalhooks "goc/ccb-engine/internal/hooks"
	"goc/ccb-engine/internal/protocol"
)

type PromptHandler = internalhooks.PromptHandler
type ExecOptions = internalhooks.ExecOptions
type Result = internalhooks.Result

const DefaultTimeout = internalhooks.DefaultTimeout

func Exec(ctx context.Context, opt ExecOptions) (Result, error) {
	return internalhooks.Exec(ctx, opt)
}

func ToInternalPromptHandler(
	h func(protocol.PromptRequest) (protocol.PromptResponse, error),
) PromptHandler {
	return internalhooks.PromptHandler(h)
}

func WithDefaultTimeout(opt *ExecOptions) {
	if opt != nil && opt.Timeout <= 0 {
		opt.Timeout = 120 * time.Second
	}
}
