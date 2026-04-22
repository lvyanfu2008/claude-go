package analytics

import (
	"context"

	"goc/diagnostics"
)

// InitializeAnalyticsSink attaches the default bridge sink (TS initializeAnalyticsSink).
// Only forwards to diagnostics — no Datadog/1P network calls. Call after [goc/diagnostics.InitAnalytics].
func InitializeAnalyticsSink() {
	AttachAnalyticsSink(&Sink{
		LogEvent: bridgeLogEvent,
		LogEventAsync: func(ctx context.Context, name string, metadata map[string]any) error {
			_ = ctx
			bridgeLogEvent(name, metadata)
			return nil
		},
	})
}

func bridgeLogEvent(name string, metadata map[string]any) {
	safe := StripProtoFields(shallowClone(metadata))
	diagnostics.EmitAnalyticsEvent(name, safe)
}
