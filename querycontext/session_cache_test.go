package querycontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goc/claudemd"
)

func TestMain(m *testing.M) {
	ClearAllContextCaches()
	claudemd.ClearMemoryFileCaches()
	os.Exit(m.Run())
}

func TestUserContextMemoizationAndClear(t *testing.T) {
	ClearAllContextCaches()
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(mdPath, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	a1, err := BuildUserContext(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(a1["claudeMd"], "v1") {
		t.Fatalf("first read: %#v", a1)
	}

	if err := os.WriteFile(mdPath, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	a2, err := BuildUserContext(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(a2["claudeMd"], "v1") {
		t.Fatalf("memo should keep v1, got %#v", a2)
	}

	// getMemoryFiles remains memoized until claudemd clear (TS: pair with resetGetMemoryFilesCache).
	ClearUserContextCache()
	aStale, err := BuildUserContext(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(aStale["claudeMd"], "v1") {
		t.Fatalf("user context clear alone should not bust getMemoryFiles memo, got %#v", aStale)
	}
	ClearUserContextCache()
	claudemd.ClearMemoryFileCaches()
	a3, err := BuildUserContext(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(a3["claudeMd"], "v2") {
		t.Fatalf("after user + memory clear expected v2, got %#v", a3)
	}
}

func TestClearUserAndSystemContextCachesKeepsGitMemo(t *testing.T) {
	ClearAllContextCaches()
	ctx := context.Background()
	cwd := t.TempDir()

	g1 := BuildGitStatusSnapshot(ctx, cwd)
	g2 := BuildGitStatusSnapshot(ctx, cwd)
	if g1 != g2 {
		t.Fatalf("git memo: g1=%q g2=%q", g1, g2)
	}

	s1 := BuildSystemContext(ctx, cwd, nil)
	ClearUserAndSystemContextCaches()
	s2 := BuildSystemContext(ctx, cwd, nil)
	if len(s1) != len(s2) {
		t.Fatalf("system rebuild len mismatch s1=%v s2=%v", s1, s2)
	}

	g3 := BuildGitStatusSnapshot(ctx, cwd)
	if g1 != g3 {
		t.Fatalf("git cache should survive user+system clear: g1=%q g3=%q", g1, g3)
	}

	ClearAllContextCaches()
	g4 := BuildGitStatusSnapshot(ctx, cwd)
	if g1 != g4 {
		// git output can legitimately differ if timing/commands change; both empty is fine.
		t.Logf("git after full clear (informational): was %q now %q", g1, g4)
	}
}
