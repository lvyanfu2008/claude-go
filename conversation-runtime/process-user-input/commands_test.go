package processuserinput

import "testing"

func TestLooksLikeSlashCommandName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"", false},
		{"compact", true},
		{"my-skill", true},
		{"ns:tool", true},
		{"snake_case", true},
		{"ask", true},
		{"/no", false},
		{"bad space", false},
		{"bad.dot", false},
	}
	for _, tc := range cases {
		if got := LooksLikeSlashCommandName(tc.name); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.name, got, tc.want)
		}
	}
}
