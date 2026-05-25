package hookexec

import "testing"

func TestExtractPluginRoot(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{
			cmd:  `"/Users/lv/.harness/plugins/cache/claude-plugins-official/superpowers/5.1.0/hooks/run-hook.cmd" session-start`,
			want: "/Users/lv/.harness/plugins/cache/claude-plugins-official/superpowers/5.1.0",
		},
		{
			cmd:  `/home/user/.harness/plugins/cache/local/my-plugin/2.0.0/hooks/session-start`,
			want: "/home/user/.harness/plugins/cache/local/my-plugin/2.0.0",
		},
		{
			cmd:  `C:\Users\test\.harness\plugins\cache\claude-plugins-official\superpowers\5.1.0\hooks\run-hook.cmd session-start`,
			want: "C:/Users/test/.harness/plugins/cache/claude-plugins-official/superpowers/5.1.0",
		},
		{
			cmd:  `"C:\Users\test\.harness\plugins\cache\local\my-plugin\2.0.0\hooks\start.cmd"`,
			want: "C:/Users/test/.harness/plugins/cache/local/my-plugin/2.0.0",
		},
		{
			cmd:  `echo hello`,
			want: "",
		},
	}
	for _, tt := range tests {
		got := extractPluginRoot(tt.cmd)
		if got != tt.want {
			t.Errorf("extractPluginRoot(%q) = %q, want %q", tt.cmd, got, tt.want)
		}
	}
}
