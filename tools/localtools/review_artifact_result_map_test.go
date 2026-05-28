package localtools

import (
	"testing"
)

func TestMapReviewArtifactToolResultToAssistantText(t *testing.T) {
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
			name:  "single annotation",
			input: `{"data":{"artifact":"code","annotationCount":1}}`,
			want:  "Review delivered with 1 annotation.",
		},
		{
			name:  "multiple annotations",
			input: `{"data":{"artifact":"code","annotationCount":3}}`,
			want:  "Review delivered with 3 annotations.",
		},
		{
			name:  "zero annotations",
			input: `{"data":{"artifact":"code","annotationCount":0}}`,
			want:  "Review delivered with 0 annotations.",
		},
		{
			name:  "with summary",
			input: `{"data":{"artifact":"code","annotationCount":3,"summary":"Looks good overall"}}`,
			want:  "Review delivered with 3 annotations. Summary: Looks good overall",
		},
		{
			name:  "with title and annotations",
			input: `{"data":{"artifact":"code","title":"main.go","annotationCount":5,"summary":"Needs work"}}`,
			want:  "Review delivered with 5 annotations. Summary: Needs work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapReviewArtifactToolResultToAssistantText(tt.input)
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
