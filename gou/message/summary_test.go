package message

import (
	"testing"
)

func TestGenerateToolResultSummary(t *testing.T) {
	tests := []struct {
		name     string
		block    map[string]interface{}
		expected string
	}{
		{
			name: "Empty result",
			block: map[string]interface{}{
				"content": "",
			},
			expected: "[Empty result]",
		},
		{
			name: "Structured read with 1 line",
			block: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": `{"type":"text","file":{"numLines":1,"content":"test"}}`,
					},
				},
			},
			expected: "Read 1 file (1 line)",
		},
		{
			name: "Structured read with multiple lines",
			block: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": `{"type":"text","file":{"numLines":5,"content":"line1\nline2\nline3\nline4\nline5"}}`,
					},
				},
			},
			expected: "Read 1 file (5 lines)",
		},
		{
			name: "Grep result with 1 match",
			block: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Found 1 match",
					},
				},
			},
			expected: "Found 1 match",
		},
		{
			name: "Grep result with multiple matches",
			block: map[string]interface{}{
				"content": `[{"type":"text","text":"Found 3 matches across 2 files"}]`,
			},
			expected: "Found 3 matches",
		},
		{
			name: "File listing",
			block: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "file1.txt\nfile2.txt\nfile3.txt",
					},
				},
			},
			expected: "Listed 3 items",
		},
		{
			name: "Single line text",
			block: map[string]interface{}{
				"content": `[{"type":"text","text":"Hello world"}]`,
			},
			expected: "[Text result]",
		},
		{
			name: "Multi-line text",
			block: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Line 1\nLine 2\nLine 3",
					},
				},
			},
			expected: "[Text: 3 lines]",
		},
		{
			name: "Empty array",
			block: map[string]interface{}{
				"content": []interface{}{},
			},
			expected: "[Empty result]",
		},
		{
			name: "Single item array",
			block: map[string]interface{}{
				"content": []interface{}{map[string]interface{}{"type": "text", "text": "test"}},
			},
			expected: "[Text result]",
		},
		{
			name: "Multiple items array",
			block: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{"type": "text", "text": "item1"},
					map[string]interface{}{"type": "text", "text": "item2"},
				},
			},
			expected: "[2 results]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateToolResultSummary(tt.block)
			if result != tt.expected {
				t.Errorf("GenerateToolResultSummary() = %v, want %v", result, tt.expected)
			}
		})
	}
}