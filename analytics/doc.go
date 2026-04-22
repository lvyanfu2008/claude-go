// Package analytics mirrors the **local** half of claude-code src/services/analytics:
// queue, sink attach, logEvent / logEventAsync, stripProtoFields, and sampling.
//
// Intentionally **not** ported (require external services, keys, or corp network):
//   - GrowthBook / Statsig live dynamic config fetches
//   - Datadog trackDatadogEvent and feature gates
//   - First-party OpenTelemetry batch export to /api/event_logging/batch
//
// Sampling: [SetEventSamplingGetter] takes precedence — when it is non-nil, its return value is
// used and env GOC_TENGU_EVENT_SAMPLING_CONFIG is ignored. When the getter is nil, set
// GOC_TENGU_EVENT_SAMPLING_CONFIG to a JSON object shaped like TS tengu_event_sampling_config
// (per-event sample_rate 0–1).
//
// Default sink only forwards to [goc/diagnostics.EmitAnalyticsEvent] (stderr / optional files).
package analytics
