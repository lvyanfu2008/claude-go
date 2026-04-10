package hooks

import (
	"context"
	"runtime"
	"testing"

	"goc/ccb-engine/internal/protocol"
)

func TestExec_Cat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires sh -c")
	}
	ctx := context.Background()
	r, err := Exec(ctx, ExecOptions{
		Command: "cat",
		JSONIn:  `{"hello":"world"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.ExitCode != 0 {
		t.Fatalf("exit %d", r.ExitCode)
	}
	if r.Stdout == "" {
		t.Fatal("empty stdout")
	}
}

func TestExec_PromptRoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires sh -c")
	}
	ctx := context.Background()
	cmd := `read -r _; echo '{"prompt":"p1","message":"m","options":[{"key":"k","label":"L"}]}'; read -r _; echo after`
	r, err := Exec(ctx, ExecOptions{
		Command: cmd,
		JSONIn:  `{}`,
		OnPrompt: func(pr protocol.PromptRequest) (protocol.PromptResponse, error) {
			if pr.Prompt != "p1" {
				t.Errorf("prompt id %q", pr.Prompt)
			}
			return protocol.PromptResponse{PromptResponse: "p1", Selected: "k"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.PromptsHandled != 1 {
		t.Fatalf("prompts %d", r.PromptsHandled)
	}
	if r.ExitCode != 0 {
		t.Fatalf("exit %d stderr=%q", r.ExitCode, r.Stderr)
	}
}
