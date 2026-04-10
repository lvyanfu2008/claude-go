package commands

import (
	"context"
	"errors"
	"testing"
)

func TestLoadAllCommandsAsync_matchesLoadAllCommands(t *testing.T) {
	ClearLoadAllCommandsCache()
	ctx := context.Background()
	cwd := t.TempDir()
	opts := DefaultLoadOptions()

	syncOut, err := LoadAllCommands(ctx, cwd, opts)
	if err != nil {
		t.Fatal(err)
	}
	ClearLoadAllCommandsCache()
	res := <-LoadAllCommandsAsync(ctx, cwd, opts)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if len(res.Commands) != len(syncOut) {
		t.Fatalf("async len %d sync len %d", len(res.Commands), len(syncOut))
	}
	for i := range syncOut {
		if res.Commands[i].Name != syncOut[i].Name {
			t.Fatalf("i=%d async=%q sync=%q", i, res.Commands[i].Name, syncOut[i].Name)
		}
	}
}

func TestGetSkillsAsync_matchesGetSkills(t *testing.T) {
	ctx := context.Background()
	cwd := t.TempDir()
	opts := DefaultLoadOptions()

	syncBatch := GetSkills(ctx, cwd, opts)
	res := <-GetSkillsAsync(ctx, cwd, opts)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if len(res.Batch.BundledSkills) != len(syncBatch.BundledSkills) {
		t.Fatal("bundled mismatch")
	}
}

func TestLoadAllCommands_ctxCanceledBeforeAsync(t *testing.T) {
	ClearLoadAllCommandsCache()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := <-LoadAllCommandsAsync(ctx, t.TempDir(), DefaultLoadOptions())
	if !errors.Is(res.Err, context.Canceled) {
		t.Fatalf("want context.Canceled, got err=%v", res.Err)
	}
	if res.Commands != nil {
		t.Fatalf("want nil commands on error, got len=%d", len(res.Commands))
	}
}
