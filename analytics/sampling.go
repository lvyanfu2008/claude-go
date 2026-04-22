package analytics

import (
	"math/rand"
	"sync"
)

// EventSamplingEntry mirrors TS EventSamplingConfig[eventName].sample_rate.
type EventSamplingEntry struct {
	SampleRate float64
}

// SamplingGetter returns per-event sample rates (0–1). Empty or missing events → 100% log.
// Set via [SetEventSamplingGetter] for GrowthBook parity; default is nil (always log at 100%).
var (
	samplingMu     sync.RWMutex
	samplingGetter func() map[string]EventSamplingEntry
)

// SetEventSamplingGetter installs a provider for tengu_event_sampling_config parity (optional).
func SetEventSamplingGetter(f func() map[string]EventSamplingEntry) {
	samplingMu.Lock()
	defer samplingMu.Unlock()
	samplingGetter = f
}

func getSamplingConfig() map[string]EventSamplingEntry {
	samplingMu.RLock()
	g := samplingGetter
	samplingMu.RUnlock()
	if g == nil {
		return nil
	}
	return g()
}

// ShouldSampleEvent mirrors TS shouldSampleEvent return semantics for the sink:
//   - drop=true: do not log (sample roll failed for partial rate)
//   - addSampleRate=false: log metadata as-is (100% or invalid config)
//   - addSampleRate=true: merge sample_rate into metadata before sinks
func ShouldSampleEvent(eventName string) (addSampleRate bool, sampleRate float64, drop bool) {
	cfg := getSamplingConfig()
	if cfg == nil {
		return false, 0, false
	}
	ev, ok := cfg[eventName]
	if !ok {
		return false, 0, false
	}
	r := ev.SampleRate
	if r < 0 || r > 1 {
		return false, 0, false
	}
	if r >= 1 {
		return false, 0, false
	}
	if r <= 0 {
		return false, 0, true
	}
	if rand.Float64() < r { //nolint:gosec // parity with TS Math.random for sampling
		return true, r, false
	}
	return false, 0, true
}
