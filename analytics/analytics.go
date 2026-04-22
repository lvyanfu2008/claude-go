// Package analytics mirrors claude-code src/services/analytics (queue, sink attach,
// logEvent / logEventAsync, stripProtoFields, sampling hooks). Default sink forwards
// to [goc/diagnostics.EmitAnalyticsEvent] after sampling and _PROTO_ stripping.
package analytics

import (
	"context"
	"sync"
)

// Sink mirrors TS AnalyticsSink (sync + async entrypoints).
type Sink struct {
	LogEvent      func(eventName string, metadata map[string]any)
	LogEventAsync func(ctx context.Context, eventName string, metadata map[string]any) error
}

type queuedEvent struct {
	name  string
	meta  map[string]any
	async bool
}

var (
	mu   sync.Mutex
	sink *Sink
	q    []queuedEvent
)

func shallowClone(m map[string]any) map[string]any {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// AttachAnalyticsSink installs the global sink and drains queued events (TS attachAnalyticsSink).
// Idempotent: second call is a no-op.
func AttachAnalyticsSink(s *Sink) {
	if s == nil || s.LogEvent == nil {
		return
	}
	mu.Lock()
	if sink != nil {
		mu.Unlock()
		return
	}
	sink = s
	pending := append([]queuedEvent(nil), q...)
	q = nil
	mu.Unlock()
	drainQueue(pending, s)
}

func drainQueue(events []queuedEvent, s *Sink) {
	for _, e := range events {
		if e.async && s.LogEventAsync != nil {
			_ = s.LogEventAsync(context.Background(), e.name, e.meta)
			continue
		}
		s.LogEvent(e.name, e.meta)
	}
}

// LogEvent enqueues until a sink is attached, then routes to the sink (TS logEvent).
func LogEvent(eventName string, metadata map[string]any) {
	md := shallowClone(metadata)
	if add, rate, drop := ShouldSampleEvent(eventName); drop {
		return
	} else if add {
		if md == nil {
			md = make(map[string]any, 1)
		}
		md["sample_rate"] = rate
	}

	mu.Lock()
	s := sink
	if s == nil {
		q = append(q, queuedEvent{name: eventName, meta: md, async: false})
		mu.Unlock()
		return
	}
	mu.Unlock()
	s.LogEvent(eventName, md)
}

// LogEventAsync mirrors TS logEventAsync (queue + sink); default sink may ignore ctx.
func LogEventAsync(ctx context.Context, eventName string, metadata map[string]any) error {
	md := shallowClone(metadata)
	if add, rate, drop := ShouldSampleEvent(eventName); drop {
		return nil
	} else if add {
		if md == nil {
			md = make(map[string]any, 1)
		}
		md["sample_rate"] = rate
	}

	mu.Lock()
	s := sink
	if s == nil {
		q = append(q, queuedEvent{name: eventName, meta: md, async: true})
		mu.Unlock()
		return nil
	}
	fnAsync := s.LogEventAsync
	fnSync := s.LogEvent
	mu.Unlock()
	if fnAsync != nil {
		return fnAsync(ctx, eventName, md)
	}
	if fnSync != nil {
		fnSync(eventName, md)
	}
	return nil
}

// ResetForTesting clears sink and queue (TS _resetForTesting).
func ResetForTesting() {
	mu.Lock()
	defer mu.Unlock()
	sink = nil
	q = nil
}
