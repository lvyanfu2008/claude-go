package analytics

import (
	"encoding/json"
	"os"
	"strings"
)

// EnvEventSamplingConfig is the env GOC_TENGU_EVENT_SAMPLING_CONFIG JSON shape (TS tengu_event_sampling_config).
// Example: {"tengu_input_prompt":{"sample_rate":0.25}}
type EnvEventSamplingConfig map[string]struct {
	SampleRate float64 `json:"sample_rate"`
}

func parseSamplingConfigFromEnv() map[string]EventSamplingEntry {
	raw := strings.TrimSpace(os.Getenv("GOC_TENGU_EVENT_SAMPLING_CONFIG"))
	if raw == "" {
		return nil
	}
	var parsed EnvEventSamplingConfig
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}
	if len(parsed) == 0 {
		return nil
	}
	out := make(map[string]EventSamplingEntry, len(parsed))
	for k, v := range parsed {
		out[k] = EventSamplingEntry{SampleRate: v.SampleRate}
	}
	return out
}
