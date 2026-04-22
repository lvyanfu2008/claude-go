package paritytools

import (
	"encoding/json"
	"testing"
)

func TestReviewArtifactFromJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		checkOutput func(t *testing.T, output string)
	}{
		{
			name: "valid input with all fields",
			input: `{
				"artifact": "func test() {\n  return 1\n}",
				"title": "test.go",
				"annotations": [
					{"line": 1, "message": "Missing docstring", "severity": "warning"},
					{"line": 2, "message": "Return value could be constant", "severity": "suggestion"}
				],
				"summary": "Good overall, minor improvements needed"
			}`,
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal output: %v", err)
				}

				data, ok := result["data"].(map[string]interface{})
				if !ok {
					t.Fatalf("output missing data field")
				}

				if artifact, ok := data["artifact"].(string); !ok || artifact != "func test() {\n  return 1\n}" {
					t.Errorf("artifact mismatch: got %v", data["artifact"])
				}

				if title, ok := data["title"].(string); !ok || title != "test.go" {
					t.Errorf("title mismatch: got %v", data["title"])
				}

				if count, ok := data["annotationCount"].(float64); !ok || int(count) != 2 {
					t.Errorf("annotationCount mismatch: got %v", data["annotationCount"])
				}

				if summary, ok := data["summary"].(string); !ok || summary != "Good overall, minor improvements needed" {
					t.Errorf("summary mismatch: got %v", data["summary"])
				}
			},
		},
		{
			name: "valid input with minimal fields",
			input: `{
				"artifact": "print('hello')",
				"annotations": []
			}`,
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal output: %v", err)
				}

				data, ok := result["data"].(map[string]interface{})
				if !ok {
					t.Fatalf("output missing data field")
				}

				if artifact, ok := data["artifact"].(string); !ok || artifact != "print('hello')" {
					t.Errorf("artifact mismatch: got %v", data["artifact"])
				}

				if _, hasTitle := data["title"]; hasTitle {
					t.Errorf("should not have title field")
				}

				if count, ok := data["annotationCount"].(float64); !ok || int(count) != 0 {
					t.Errorf("annotationCount mismatch: got %v", data["annotationCount"])
				}

				if _, hasSummary := data["summary"]; hasSummary {
					t.Errorf("should not have summary field")
				}
			},
		},
		{
			name: "missing artifact",
			input: `{
				"annotations": []
			}`,
			wantErr: true,
		},
		{
			name: "empty artifact",
			input: `{
				"artifact": "   ",
				"annotations": []
			}`,
			wantErr: true,
		},
		{
			name: "annotation missing message",
			input: `{
				"artifact": "code",
				"annotations": [
					{"line": 1, "severity": "warning"}
				]
			}`,
			wantErr: true,
		},
		{
			name: "annotation empty message",
			input: `{
				"artifact": "code",
				"annotations": [
					{"line": 1, "message": "   ", "severity": "warning"}
				]
			}`,
			wantErr: true,
		},
		{
			name: "invalid severity",
			input: `{
				"artifact": "code",
				"annotations": [
					{"line": 1, "message": "test", "severity": "invalid"}
				]
			}`,
			wantErr: true,
		},
		{
			name: "line less than 1",
			input: `{
				"artifact": "code",
				"annotations": [
					{"line": 0, "message": "test"}
				]
			}`,
			wantErr: true,
		},
		{
			name: "valid severities",
			input: `{
				"artifact": "code",
				"annotations": [
					{"message": "info", "severity": "info"},
					{"message": "warning", "severity": "warning"},
					{"message": "error", "severity": "error"},
					{"message": "suggestion", "severity": "suggestion"}
				]
			}`,
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("failed to unmarshal output: %v", err)
				}

				data, ok := result["data"].(map[string]interface{})
				if !ok {
					t.Fatalf("output missing data field")
				}

				if count, ok := data["annotationCount"].(float64); !ok || int(count) != 4 {
					t.Errorf("annotationCount mismatch: got %v", data["annotationCount"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, handled, err := ReviewArtifactFromJSON([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if !handled {
					t.Errorf("expected handled=true when error occurs")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if handled {
				t.Errorf("expected handled=false for successful execution")
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestReviewArtifactAnnotationStruct(t *testing.T) {
	// Test JSON marshaling/unmarshaling of the struct
	annotationJSON := `{"line": 5, "message": "test message", "severity": "warning"}`
	var annotation ReviewArtifactAnnotation
	if err := json.Unmarshal([]byte(annotationJSON), &annotation); err != nil {
		t.Fatalf("failed to unmarshal annotation: %v", err)
	}

	if annotation.Line == nil || *annotation.Line != 5 {
		t.Errorf("Line mismatch: got %v", annotation.Line)
	}
	if annotation.Message != "test message" {
		t.Errorf("Message mismatch: got %s", annotation.Message)
	}
	if annotation.Severity != "warning" {
		t.Errorf("Severity mismatch: got %s", annotation.Severity)
	}

	// Test marshaling back
	marshaled, err := json.Marshal(annotation)
	if err != nil {
		t.Fatalf("failed to marshal annotation: %v", err)
	}

	var annotation2 ReviewArtifactAnnotation
	if err := json.Unmarshal(marshaled, &annotation2); err != nil {
		t.Fatalf("failed to unmarshal marshaled annotation: %v", err)
	}

	if annotation2.Line == nil || *annotation2.Line != 5 {
		t.Errorf("Line mismatch after roundtrip: got %v", annotation2.Line)
	}
	if annotation2.Message != "test message" {
		t.Errorf("Message mismatch after roundtrip: got %s", annotation2.Message)
	}
	if annotation2.Severity != "warning" {
		t.Errorf("Severity mismatch after roundtrip: got %s", annotation2.Severity)
	}
}