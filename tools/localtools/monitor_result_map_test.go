package localtools

import (
	"testing"
)

func TestMapMonitorToolResultToAssistantText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "not JSON",
			input:   "hello",
			wantErr: true,
		},
		{
			name:    "missing taskId",
			input:   `{"data":{"outputFile":"/tmp/out"}}`,
			wantErr: true,
		},
		{
			name:  "valid output",
			input: `{"data":{"taskId":"monitor-123","outputFile":"/tmp/tasks/monitor-123.output"}}`,
			want:  "Monitor started (task monitor-123). Output file: /tmp/tasks/monitor-123.output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapMonitorToolResultToAssistantText(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
