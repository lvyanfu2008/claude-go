package paritytools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTasksV2_CreateListGetUpdateDelete(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", tmp)
	t.Setenv("CLAUDE_CODE_TASK_LIST_ID", "parity-v2-test")
	t.Setenv("GOU_DEMO_NON_INTERACTIVE", "") // clear if inherited
	t.Setenv("CLAUDE_CODE_NON_INTERACTIVE", "")

	cfg := Config{SessionID: "ignored-when-env-list-id-set"}

	createRaw := []byte(`{"subject":"Alpha","description":"do alpha","activeForm":"Doing alpha"}`)
	out, isErr, err := Run(ctx, "TaskCreate", createRaw, cfg)
	if err != nil || isErr {
		t.Fatalf("TaskCreate: err=%v isErr=%v out=%s", err, isErr, out)
	}
	var createOut struct {
		Data struct {
			Task struct {
				ID      string `json:"id"`
				Subject string `json:"subject"`
			} `json:"task"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &createOut); err != nil {
		t.Fatal(err)
	}
	id := strings.TrimSpace(createOut.Data.Task.ID)
	if id != "1" || createOut.Data.Task.Subject != "Alpha" {
		t.Fatalf("unexpected create payload: %+v", createOut)
	}

	listOut, isErr, err := Run(ctx, "TaskList", []byte("{}"), cfg)
	if err != nil || isErr {
		t.Fatalf("TaskList: err=%v isErr=%v out=%s", err, isErr, listOut)
	}
	if !strings.Contains(listOut, `"id":"1"`) || !strings.Contains(listOut, `"status":"pending"`) {
		t.Fatalf("TaskList: %s", listOut)
	}

	getRaw := []byte(`{"taskId":"` + id + `"}`)
	getOut, isErr, err := Run(ctx, "TaskGet", getRaw, cfg)
	if err != nil || isErr {
		t.Fatalf("TaskGet: err=%v isErr=%v out=%s", err, isErr, getOut)
	}
	if !strings.Contains(getOut, `"description":"do alpha"`) {
		t.Fatalf("TaskGet: %s", getOut)
	}

	updRaw := []byte(`{"taskId":"` + id + `","status":"in_progress"}`)
	upOut, isErr, err := Run(ctx, "TaskUpdate", updRaw, cfg)
	if err != nil || isErr {
		t.Fatalf("TaskUpdate status: err=%v isErr=%v out=%s", err, isErr, upOut)
	}
	if !strings.Contains(upOut, `"success":true`) || !strings.Contains(upOut, `"status"`) {
		t.Fatalf("TaskUpdate: %s", upOut)
	}

	getOut2, _, _ := Run(ctx, "TaskGet", getRaw, cfg)
	if !strings.Contains(getOut2, `"status":"in_progress"`) {
		t.Fatalf("expected in_progress after update: %s", getOut2)
	}

	delRaw := []byte(`{"taskId":"` + id + `","status":"deleted"}`)
	delOut, isErr, err := Run(ctx, "TaskUpdate", delRaw, cfg)
	if err != nil || isErr {
		t.Fatalf("TaskUpdate delete: err=%v isErr=%v out=%s", err, isErr, delOut)
	}
	if !strings.Contains(delOut, `"deleted"`) {
		t.Fatalf("TaskUpdate delete: %s", delOut)
	}

	getOut3, _, _ := Run(ctx, "TaskGet", getRaw, cfg)
	if !strings.Contains(getOut3, `"task":null`) {
		t.Fatalf("expected null task after delete: %s", getOut3)
	}

	// New task should not reuse id 1 (high water mark).
	out2, _, err := Run(ctx, "TaskCreate", []byte(`{"subject":"Beta","description":"b"}`), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, `"id":"2"`) {
		t.Fatalf("expected id 2 after delete, got %s", out2)
	}
}

func TestTasksV2_DisabledNonInteractive(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", tmp)
	t.Setenv("CLAUDE_CODE_TASK_LIST_ID", "parity-v2-off")
	t.Setenv("GOU_DEMO_NON_INTERACTIVE", "1")
	t.Setenv("CLAUDE_CODE_ENABLE_TASKS", "")

	cfg := Config{}
	_, isErr, err := Run(ctx, "TaskList", []byte("{}"), cfg)
	if err == nil || !isErr {
		t.Fatalf("expected error when disabled, err=%v isErr=%v", err, isErr)
	}
	if !strings.Contains(err.Error(), "Todo v2 tools disabled") {
		t.Fatalf("unexpected err: %v", err)
	}
}
