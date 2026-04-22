package analytics

import (
	"context"
	"testing"
)

func TestSamplingFromEnvJSON(t *testing.T) {
	t.Cleanup(ResetForTesting)
	ResetForTesting()
	t.Setenv("GOC_TENGU_EVENT_SAMPLING_CONFIG", `{"x":{"sample_rate":0}}`)
	var n int
	AttachAnalyticsSink(&Sink{
		LogEvent: func(name string, metadata map[string]any) { n++ },
	})
	LogEvent("x", nil)
	if n != 0 {
		t.Fatalf("expected env-driven drop, got %d", n)
	}
}

func TestStripProtoFields(t *testing.T) {
	in := map[string]any{"a": 1, "_PROTO_x": "secret", "b": true}
	out := StripProtoFields(in)
	if _, ok := out["_PROTO_x"]; ok {
		t.Fatal("proto key should be stripped")
	}
	if out["a"] != 1 || out["b"] != true {
		t.Fatalf("%#v", out)
	}
}

func TestQueueDrainOnAttach(t *testing.T) {
	t.Cleanup(ResetForTesting)
	ResetForTesting()
	var got []string
	LogEvent("e1", map[string]any{"k": 1})
	LogEventAsync(context.Background(), "e2", map[string]any{"k": 2})
	AttachAnalyticsSink(&Sink{
		LogEvent: func(name string, metadata map[string]any) {
			got = append(got, name)
			_ = metadata
		},
		LogEventAsync: func(ctx context.Context, name string, metadata map[string]any) error {
			got = append(got, name+"-async")
			return nil
		},
	})
	if len(got) != 2 {
		t.Fatalf("drain: %#v", got)
	}
}

func TestSamplingDrop(t *testing.T) {
	t.Cleanup(ResetForTesting)
	ResetForTesting()
	SetEventSamplingGetter(func() map[string]EventSamplingEntry {
		return map[string]EventSamplingEntry{"x": {SampleRate: 0}}
	})
	t.Cleanup(func() { SetEventSamplingGetter(nil) })
	var n int
	AttachAnalyticsSink(&Sink{
		LogEvent: func(name string, metadata map[string]any) { n++ },
	})
	LogEvent("x", nil)
	if n != 0 {
		t.Fatalf("expected drop, got %d calls", n)
	}
}
