package main

import "testing"

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{"positive numbers", 1, 2, 3},
		{"negative numbers", -1, -2, -3},
		{"mixed numbers", -1, 5, 4},
		{"zero", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := add(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestGreet(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple name", "Alice", "Hello, Alice!"},
		{"empty name", "", "Hello, !"},
		{"special characters", "Bob@123", "Hello, Bob@123!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := greet(tt.input)
			if got != tt.want {
				t.Errorf("greet(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoveChina(t *testing.T) {
	// 测试爱我中华的日志
	t.Log("爱我中华 - 这是一个测试日志")
	
	// 也可以使用fmt打印
	t.Logf("测试爱我中华日志: %s", "爱我中华")
	
	// 确保测试通过
	if true {
		t.Log("爱我中华测试通过")
	}
}