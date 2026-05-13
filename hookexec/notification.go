package hookexec

import (
	"context"
	"encoding/json"

	"goc/types"
)

const hookEventNotification = "Notification"

type notificationHookInput struct {
	BaseHookInput
	Message          string `json:"message"`
	NotificationType string `json:"notification_type"`
}

// RunNotificationHooks executes Notification command hooks.
// Mirrors TS executeNotificationHooks in src/utils/hooks.ts.
func RunNotificationHooks(
	ctx context.Context,
	table HooksTable,
	workDir string,
	base BaseHookInput,
	message, notificationType string,
	batchTimeoutMs int,
) ([]types.AggregatedHookResult, error) {
	if HooksDisabled() || ShouldDisableAllHooksIncludingManaged() || ShouldSkipHookDueToTrust() {
		return nil, nil
	}

	in := notificationHookInput{
		BaseHookInput:   base,
		Message:         message,
		NotificationType: notificationType,
	}
	in.HookEventName = hookEventNotification

	jsonIn, err := marshalHookInput(in)
	if err != nil {
		return nil, err
	}

	var hookInput map[string]any
	if err := json.Unmarshal([]byte(jsonIn), &hookInput); err != nil {
		return nil, err
	}
	if len(CommandHooksForHookInput(table, hookInput)) == 0 {
		return nil, nil
	}

	wd := trimOrDot(workDir)
	results := ExecuteCommandHooksOutsideREPLParallel(OutsideReplCommandParams{
		Ctx:       ctx,
		WorkDir:   wd,
		Hooks:     table,
		JSONInput: jsonIn,
		TimeoutMs: batchTimeoutMs,
	})

	var agg []types.AggregatedHookResult
	for _, r := range results {
		agg = append(agg, hookAggregate(r, randomUUID(), hookEventNotification, r.Command)...)
	}
	return agg, nil
}
