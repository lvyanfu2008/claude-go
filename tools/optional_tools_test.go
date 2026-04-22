package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func decodeData(t *testing.T, out string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data object in output: %v", payload)
	}
	return data
}

func TestOptionalToolsNoLongerUseUnavailableErrors(t *testing.T) {
	repoDir := t.TempDir()
	if out, err := exec.Command("git", "-C", repoDir, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, string(out))
	}
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	if out, err := exec.Command("git", "-C", repoDir, "add", ".").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v (%s)", err, string(out))
	}
	if out, err := exec.Command("git", "-C", repoDir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "seed").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v (%s)", err, string(out))
	}
	cfg := Config{
		WorkDir:     repoDir,
		ProjectRoot: repoDir,
		SessionID:   "session-test",
	}

	cases := []struct {
		name string
		run  func() (string, bool, error)
		key  string
	}{
		{
			name: "VerifyPlanExecution",
			run: func() (string, bool, error) {
				return VerifyPlanExecutionFromJSON([]byte(`{"plan_summary":"ok","all_steps_completed":true}`))
			},
			key: "verified",
		},
		{
			name: "TerminalCapture",
			run: func() (string, bool, error) {
				return TerminalCaptureFromJSON([]byte(`{"lines":10}`))
			},
			key: "content",
		},
		{
			name: "LSP",
			run: func() (string, bool, error) {
				return LSPFromJSON([]byte(`{"operation":"hover","filePath":"a.go"}`))
			},
			key: "result",
		},
		{
			name: "EnterWorktree",
			run: func() (string, bool, error) {
				return EnterWorktreeFromJSON([]byte(`{"name":"wt-test"}`), cfg)
			},
			key: "worktreePath",
		},
		{
			name: "ExitWorktree",
			run: func() (string, bool, error) {
				return ExitWorktreeFromJSON([]byte(`{"action":"keep"}`), cfg)
			},
			key: "action",
		},
		{
			name: "TeamCreate",
			run: func() (string, bool, error) {
				return TeamCreateFromJSON([]byte(`{"team_name":"alpha"}`), cfg)
			},
			key: "team_name",
		},
		{
			name: "TeamDelete",
			run: func() (string, bool, error) {
				return TeamDeleteFromJSON([]byte(`{}`), cfg)
			},
			key: "success",
		},
		{
			name: "Config",
			run: func() (string, bool, error) {
				return ConfigFromJSON([]byte(`{"setting":"theme","value":"dark"}`), cfg)
			},
			key: "success",
		},
		{
			name: "Tungsten",
			run: func() (string, bool, error) {
				return TungstenFromJSON([]byte(`{}`))
			},
			key: "error",
		},
		{
			name: "SuggestBackgroundPR",
			run: func() (string, bool, error) {
				return SuggestBackgroundPRFromJSON([]byte(`{}`))
			},
			key: "suggested",
		},
		{
			name: "WebBrowser",
			run: func() (string, bool, error) {
				return WebBrowserFromJSON([]byte(`{"url":"https://example.com"}`))
			},
			key: "content",
		},
		{
			name: "RemoteTrigger",
			run: func() (string, bool, error) {
				t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
				return RemoteTriggerFromJSON([]byte(`{"action":"list"}`))
			},
			key: "",
		},
		{
			name: "Monitor",
			run: func() (string, bool, error) {
				return MonitorFromJSON(context.Background(), []byte(`{"command":"echo hi","description":"test monitor"}`), cfg)
			},
			key: "taskId",
		},
		{
			name: "Workflow",
			run: func() (string, bool, error) {
				return WorkflowFromJSON([]byte(`{"workflow":"build"}`))
			},
			key: "output",
		},
		{
			name: "Snip",
			run: func() (string, bool, error) {
				return SnipFromJSON([]byte(`{"message_ids":["1","2"],"reason":"compact"}`))
			},
			key: "snipped_count",
		},
		{
			name: "SendUserFile",
			run: func() (string, bool, error) {
				return SendUserFileFromJSON([]byte(`{"file_path":"/tmp/a.txt"}`))
			},
			key: "sent",
		},
		{
			name: "PushNotification",
			run: func() (string, bool, error) {
				return PushNotificationFromJSON([]byte(`{"title":"t","body":"b"}`))
			},
			key: "sent",
		},
		{
			name: "SubscribePR",
			run: func() (string, bool, error) {
				return SubscribePRFromJSON([]byte(`{"repo":"o/r","pr_number":1}`))
			},
			key: "subscribed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, isErr, err := tc.run()
			if tc.name == "RemoteTrigger" {
				if err == nil {
					t.Fatalf("expected auth error for RemoteTrigger")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if isErr {
				t.Fatalf("unexpected tool error flag")
			}
			data := decodeData(t, out)
			if _, ok := data[tc.key]; !ok {
				t.Fatalf("missing expected key %q in data: %v", tc.key, data)
			}
		})
	}
}

func TestLSPFromJSONReadsFileContent(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	src := "package main\n\nfunc HelloWorld() {}\n\nfunc main() { HelloWorld() }\n"
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	out, isErr, err := LSPFromJSON([]byte(`{"operation":"documentSymbol","filePath":"` + file + `"}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if isErr {
		t.Fatalf("unexpected error flag")
	}
	data := decodeData(t, out)
	result, _ := data["result"].(string)
	if !strings.Contains(result, "func HelloWorld") {
		t.Fatalf("expected symbol listing to include HelloWorld, got: %q", result)
	}
}

func TestListPeersFromJSONIncludeSelf(t *testing.T) {
	t.Setenv("CLAUDE_CODE_MESSAGING_SOCKET_PATH", "/tmp/claude.sock")
	out, isErr, err := ListPeersFromJSON([]byte(`{"include_self":true}`), Config{WorkDir: "/tmp"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if isErr {
		t.Fatalf("unexpected error flag")
	}
	data := decodeData(t, out)
	peers, ok := data["peers"].([]any)
	if !ok || len(peers) != 1 {
		t.Fatalf("expected one self peer, got: %v", data["peers"])
	}
	p0, _ := peers[0].(map[string]any)
	if p0["address"] != "uds:/tmp/claude.sock" {
		t.Fatalf("unexpected self address: %v", p0["address"])
	}
}
